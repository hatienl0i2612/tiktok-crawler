// Package ffmpeg locates or installs a portable FFmpeg executable for media muxing.
package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/hatienl0i2612/tiktok-crawler/internal/portable"
)

const releaseURL = "https://api.github.com/repos/eugeneware/ffmpeg-static/releases/latest"

// Options configures portable FFmpeg installation messages.
type Options struct {
	Status io.Writer
}

// Manager discovers and installs FFmpeg for the current platform.
type Manager struct {
	status        io.Writer
	goos          string
	goarch        string
	tempDir       string
	lookPath      func(string) (string, error)
	httpClient    *http.Client
	releaseURL    string
	enforceGitHub bool
}

// NewManager creates an FFmpeg manager for the current operating system.
func NewManager(options Options) *Manager {
	return &Manager{
		status: options.Status, goos: runtime.GOOS, goarch: runtime.GOARCH,
		tempDir: os.TempDir(), lookPath: exec.LookPath, httpClient: portable.HTTPClient(),
		releaseURL: releaseURL, enforceGitHub: true,
	}
}

// Ensure prefers FFmpeg in PATH, then a cached portable build, and installs one if needed.
func (manager *Manager) Ensure(ctx context.Context) (string, error) {
	if manager == nil {
		return "", errors.New("FFmpeg manager is not configured")
	}
	if path, err := manager.lookPath("ffmpeg"); err == nil && path != "" {
		return path, nil
	}
	assetName, err := manager.assetName()
	if err != nil {
		return "", err
	}
	binaryName := "ffmpeg"
	if manager.goos == "windows" {
		binaryName += ".exe"
	}
	cacheDir := filepath.Join(manager.tempDir, "tiktok-crawler", "ffmpeg", manager.goos+"-"+manager.goarch)
	executable := filepath.Join(cacheDir, binaryName)
	if manager.isExecutable(executable) {
		return executable, nil
	}
	if _, err := os.Stat(cacheDir); err == nil {
		return "", fmt.Errorf("portable FFmpeg cache is incomplete: %s; remove that path and retry", cacheDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect portable FFmpeg cache: %w", err)
	}
	manager.report("FFmpeg was not found in PATH; installing a portable build in %s\n", cacheDir)
	release, err := portable.FetchRelease(ctx, manager.httpClient, manager.releaseURL)
	if err != nil {
		return "", fmt.Errorf("fetch FFmpeg release: %w", err)
	}
	var asset portable.Asset
	for _, candidate := range release.Assets {
		if candidate.Name == assetName {
			asset = candidate
			break
		}
	}
	if asset.Name == "" {
		return "", fmt.Errorf("FFmpeg release %s has no %s asset; install ffmpeg in PATH manually", release.TagName, assetName)
	}
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
		return "", fmt.Errorf("create portable FFmpeg cache parent: %w", err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(cacheDir), ".ffmpeg-install-*")
	if err != nil {
		return "", fmt.Errorf("create portable FFmpeg staging directory: %w", err)
	}
	defer os.RemoveAll(staging)
	manager.report("Downloading %s (%s)\n", asset.Name, release.TagName)
	staged := filepath.Join(staging, binaryName)
	if err := portable.DownloadAsset(ctx, manager.httpClient, asset, staged, manager.enforceGitHub); err != nil {
		return "", fmt.Errorf("download portable FFmpeg: %w", err)
	}
	if manager.goos != "windows" {
		if err := os.Chmod(staged, 0o755); err != nil {
			return "", fmt.Errorf("make portable FFmpeg executable: %w", err)
		}
	}
	if err := os.Rename(staging, cacheDir); err != nil && !manager.isExecutable(executable) {
		return "", fmt.Errorf("publish portable FFmpeg installation: %w", err)
	}
	manager.report("Portable FFmpeg is ready: %s\n", executable)
	return executable, nil
}

func (manager *Manager) assetName() (string, error) {
	platform := manager.goos
	architecture := manager.goarch
	switch manager.goos + "/" + manager.goarch {
	case "darwin/amd64", "linux/amd64", "windows/amd64":
		architecture = "x64"
	case "darwin/arm64", "linux/arm64", "linux/arm":
	case "linux/386":
		architecture = "ia32"
	default:
		return "", fmt.Errorf("automatic portable FFmpeg installation is not supported on %s/%s; install ffmpeg in PATH manually", manager.goos, manager.goarch)
	}
	if platform == "windows" {
		platform = "win32"
	}
	return "ffmpeg-" + platform + "-" + architecture, nil
}

func (manager *Manager) isExecutable(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && (manager.goos == "windows" || info.Mode().Perm()&0o111 != 0)
}

func (manager *Manager) report(format string, arguments ...any) {
	if manager.status != nil {
		_, _ = fmt.Fprintf(manager.status, format, arguments...)
	}
}
