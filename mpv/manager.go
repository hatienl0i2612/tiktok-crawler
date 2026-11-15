// Package mpv locates or installs a portable mpv executable and launches it
// for TikTok LIVE playback.
package mpv

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/hatienl0i2612/tiktok-crawler/internal/portable"
)

const (
	stableReleaseURL = "https://api.github.com/repos/mpv-player/mpv/releases/latest"
	linuxReleaseURL  = "https://api.github.com/repos/pkgforge-dev/mpv-AppImage/releases/latest"
	maxExtractedSize = 1 << 30
)

// Options configures portable mpv discovery and installation.
type Options struct {
	// Status receives installation progress. It may be nil.
	Status io.Writer
}

// PlayOptions configures the mpv process.
type PlayOptions struct {
	Referer string
	Title   string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Manager discovers, installs, and launches mpv.
type Manager struct {
	status           io.Writer
	goos             string
	goarch           string
	tempDir          string
	lookPath         func(string) (string, error)
	httpClient       *http.Client
	stableReleaseURL string
	linuxReleaseURL  string
	enforceGitHub    bool
}

type githubRelease = portable.Release
type githubAsset = portable.Asset

type archiveKind string

const (
	archiveZIP      archiveKind = "zip"
	archiveAppImage archiveKind = "appimage"
)

type platformSpec struct {
	releaseURL  string
	assetSuffix string
	kind        archiveKind
	binaryName  string
	binaryPath  string
}

// NewManager creates an mpv manager for the current operating system.
func NewManager(options Options) *Manager {
	return &Manager{
		status:           options.Status,
		goos:             runtime.GOOS,
		goarch:           runtime.GOARCH,
		tempDir:          os.TempDir(),
		lookPath:         exec.LookPath,
		httpClient:       portable.HTTPClient(),
		stableReleaseURL: stableReleaseURL,
		linuxReleaseURL:  linuxReleaseURL,
		enforceGitHub:    true,
	}
}

// Ensure returns mpv from PATH or installs a verified portable build under
// the operating system's temporary directory and returns that executable.
func (manager *Manager) Ensure(ctx context.Context) (string, error) {
	if manager == nil {
		return "", errors.New("mpv manager is not configured")
	}
	if path, err := manager.lookPath("mpv"); err == nil && path != "" {
		return path, nil
	}
	spec, err := manager.platform()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(manager.tempDir, "tiktok-crawler", "mpv", manager.goos+"-"+manager.goarch)
	if executable := findExecutable(cacheDir, spec); executable != "" {
		return executable, nil
	}
	if _, statErr := os.Stat(cacheDir); statErr == nil {
		return "", fmt.Errorf("portable mpv cache is incomplete: %s; remove that path and retry", cacheDir)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("inspect portable mpv cache: %w", statErr)
	}

	if err := os.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
		return "", fmt.Errorf("create portable mpv cache parent: %w", err)
	}
	manager.report("mpv was not found in PATH; installing a portable build in %s\n", cacheDir)
	release, err := portable.FetchRelease(ctx, manager.httpClient, spec.releaseURL)
	if err != nil {
		return "", err
	}
	asset, err := selectAsset(release, spec)
	if err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(filepath.Dir(cacheDir), ".mpv-install-*")
	if err != nil {
		return "", fmt.Errorf("create portable mpv staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	manager.report("Downloading %s (%s)\n", asset.Name, release.TagName)
	archivePath := filepath.Join(staging, filepath.Base(asset.Name))
	if err := portable.DownloadAsset(ctx, manager.httpClient, asset, archivePath, manager.enforceGitHub); err != nil {
		return "", err
	}
	installRoot := filepath.Join(staging, "files")
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		return "", err
	}
	switch spec.kind {
	case archiveZIP:
		if err := extractZIP(archivePath, installRoot); err != nil {
			return "", fmt.Errorf("extract portable mpv: %w", err)
		}
	case archiveAppImage:
		target := filepath.Join(installRoot, spec.binaryName)
		if err := os.Rename(archivePath, target); err != nil {
			return "", fmt.Errorf("install mpv AppImage: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported mpv archive type %q", spec.kind)
	}
	executable := findExecutable(installRoot, spec)
	if executable == "" {
		return "", fmt.Errorf("asset %s did not contain the expected mpv executable", asset.Name)
	}
	if manager.goos != "windows" {
		if err := os.Chmod(executable, 0o755); err != nil {
			return "", fmt.Errorf("make portable mpv executable: %w", err)
		}
	}
	if err := os.Rename(installRoot, cacheDir); err != nil {
		if existing := findExecutable(cacheDir, spec); existing != "" {
			manager.report("Portable mpv is ready: %s\n", existing)
			return existing, nil
		}
		return "", fmt.Errorf("publish portable mpv installation: %w", err)
	}
	executable = findExecutable(cacheDir, spec)
	manager.report("Portable mpv is ready: %s\n", executable)
	return executable, nil
}

// Play opens one HTTPS TikTok stream with mpv and waits until the player exits.
func (manager *Manager) Play(ctx context.Context, executable, streamURL string, options PlayOptions) error {
	parsed, err := url.Parse(strings.TrimSpace(streamURL))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return errors.New("mpv requires a valid HTTPS stream URL")
	}
	if strings.TrimSpace(executable) == "" {
		return errors.New("mpv executable path is empty")
	}
	command := exec.CommandContext(ctx, executable, playArguments(parsed.String(), options)...)
	if strings.EqualFold(filepath.Ext(executable), ".AppImage") {
		command.Env = append(os.Environ(), "APPIMAGE_EXTRACT_AND_RUN=1")
	}
	command.Stdin = options.Stdin
	command.Stdout = options.Stdout
	command.Stderr = options.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("mpv playback failed: %w", err)
	}
	return nil
}

func playArguments(streamURL string, options PlayOptions) []string {
	arguments := []string{"--force-window=yes", "--keep-open=no"}
	if title := strings.TrimSpace(options.Title); title != "" {
		arguments = append(arguments, "--title="+title)
	}
	if referer := strings.TrimSpace(options.Referer); referer != "" {
		arguments = append(arguments, "--referrer="+referer)
	}
	return append(arguments, "--", streamURL)
}

func (manager *Manager) platform() (platformSpec, error) {
	spec := platformSpec{releaseURL: manager.stableReleaseURL, kind: archiveZIP}
	switch manager.goos + "/" + manager.goarch {
	case "darwin/arm64":
		spec.assetSuffix = "-macos-14-arm.zip"
		spec.binaryPath = filepath.FromSlash("mpv.app/Contents/MacOS/mpv")
	case "darwin/amd64":
		spec.assetSuffix = "-macos-15-intel.zip"
		spec.binaryPath = filepath.FromSlash("mpv.app/Contents/MacOS/mpv")
	case "windows/amd64":
		spec.assetSuffix, spec.binaryName = "-x86_64-pc-windows-msvc.zip", "mpv.exe"
	case "windows/arm64":
		spec.assetSuffix, spec.binaryName = "-aarch64-pc-windows-msvc.zip", "mpv.exe"
	case "windows/386":
		spec.assetSuffix, spec.binaryName = "-i686-w64-mingw32.zip", "mpv.exe"
	case "linux/amd64":
		spec.releaseURL, spec.assetSuffix = manager.linuxReleaseURL, "-anylinux-x86_64.AppImage"
		spec.kind, spec.binaryName = archiveAppImage, "mpv.AppImage"
	case "linux/arm64":
		spec.releaseURL, spec.assetSuffix = manager.linuxReleaseURL, "-anylinux-aarch64.AppImage"
		spec.kind, spec.binaryName = archiveAppImage, "mpv.AppImage"
	default:
		return platformSpec{}, fmt.Errorf("automatic portable mpv installation is not supported on %s/%s; install mpv in PATH manually", manager.goos, manager.goarch)
	}
	return spec, nil
}

func selectAsset(release githubRelease, spec platformSpec) (githubAsset, error) {
	var matches []githubAsset
	for _, asset := range release.Assets {
		if filepath.Base(asset.Name) == asset.Name && strings.HasSuffix(asset.Name, spec.assetSuffix) {
			matches = append(matches, asset)
		}
	}
	if len(matches) != 1 {
		return githubAsset{}, fmt.Errorf("mpv release %s has %d assets ending in %q; expected exactly one", release.TagName, len(matches), spec.assetSuffix)
	}
	asset := matches[0]
	if err := portable.ValidateAsset(asset); err != nil {
		return githubAsset{}, fmt.Errorf("mpv asset %s: %w", asset.Name, err)
	}
	return asset, nil
}

func extractZIP(archivePath, destination string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	var extracted int64
	for _, entry := range archive.File {
		cleanName := filepath.Clean(filepath.FromSlash(entry.Name))
		if cleanName == "." || filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive contains unsafe path %q", entry.Name)
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive contains unsupported symbolic link %q", entry.Name)
		}
		target := filepath.Join(destination, cleanName)
		relative, err := filepath.Rel(destination, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive path escapes destination: %q", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if entry.UncompressedSize64 > uint64(maxExtractedSize-extracted) {
			return fmt.Errorf("extracted mpv files exceed the %d MiB limit", maxExtractedSize>>20)
		}
		extracted += int64(entry.UncompressedSize64)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		mode := entry.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		destinationFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			source.Close()
			return err
		}
		written, copyErr := io.Copy(destinationFile, io.LimitReader(source, int64(entry.UncompressedSize64)+1))
		closeErr := destinationFile.Close()
		sourceErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		if written != int64(entry.UncompressedSize64) {
			return fmt.Errorf("archive entry %q size mismatch", entry.Name)
		}
		if closeErr != nil {
			return closeErr
		}
		if sourceErr != nil {
			return sourceErr
		}
	}
	return nil
}

func findExecutable(root string, spec platformSpec) string {
	if spec.binaryPath != "" {
		candidate := filepath.Join(root, spec.binaryPath)
		if executableFile(candidate) {
			return candidate
		}
	}
	var matches []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		matchesPath := spec.binaryPath != "" && strings.HasSuffix(filepath.Clean(path), spec.binaryPath)
		matchesName := spec.binaryName != "" && strings.EqualFold(entry.Name(), spec.binaryName)
		if (matchesPath || matchesName) && executableFile(path) {
			matches = append(matches, path)
		}
		return nil
	})
	sort.Slice(matches, func(i, j int) bool {
		if len(matches[i]) != len(matches[j]) {
			return len(matches[i]) < len(matches[j])
		}
		return matches[i] < matches[j]
	})
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func executableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func (manager *Manager) report(format string, arguments ...any) {
	if manager.status != nil {
		_, _ = fmt.Fprintf(manager.status, format, arguments...)
	}
}
