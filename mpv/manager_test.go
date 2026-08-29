package mpv

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestEnsureUsesMPVFromPath(t *testing.T) {
	manager := newTestManager(t, "darwin", "arm64")
	manager.lookPath = func(name string) (string, error) {
		if name != "mpv" {
			t.Fatalf("LookPath(%q)", name)
		}
		return "/usr/local/bin/mpv", nil
	}
	path, err := manager.Ensure(context.Background())
	if err != nil || path != "/usr/local/bin/mpv" {
		t.Fatalf("Ensure() = %q, %v", path, err)
	}
}

func TestEnsureDownloadsVerifiesAndCachesWindowsZIP(t *testing.T) {
	archive := zipBytes(t, map[string]string{
		"mpv/mpv.exe":    "portable-mpv",
		"mpv/helper.dll": "dependency",
	})
	manager, requests, status := managerWithReleaseServer(t, "windows", "amd64", archive, "mpv-v0.41.0-x86_64-pc-windows-msvc.zip", "")

	path, err := manager.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure(): %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "portable-mpv" {
		t.Fatalf("installed executable = %q, %v", data, err)
	}
	if *requests != 2 {
		t.Fatalf("HTTP requests = %d, want release + asset", *requests)
	}
	if !strings.Contains(status.String(), "installing a portable build") || !strings.Contains(status.String(), "Portable mpv is ready") {
		t.Fatalf("status output = %q", status.String())
	}

	cached, err := manager.Ensure(context.Background())
	if err != nil || cached != path {
		t.Fatalf("cached Ensure() = %q, %v", cached, err)
	}
	if *requests != 2 {
		t.Fatalf("cached Ensure made another HTTP request: %d", *requests)
	}
}

func TestEnsureInstallsLinuxAppImage(t *testing.T) {
	image := []byte("fake-appimage")
	manager, _, _ := managerWithReleaseServer(t, "linux", "arm64", image, "mpv-v0.41.0-anylinux-aarch64.AppImage", "")
	path, err := manager.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure(): %v", err)
	}
	if filepath.Base(path) != "mpv.AppImage" {
		t.Fatalf("installed path = %q", path)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("AppImage is not executable: mode=%v err=%v", info.Mode(), err)
	}
}

func TestFindExecutableSupportsNestedMacOSBundle(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "mpv-v0.41.0", "mpv.app", "Contents", "MacOS", "mpv")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("mpv"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec, err := newTestManager(t, "darwin", "arm64").platform()
	if err != nil {
		t.Fatal(err)
	}
	if got := findExecutable(root, spec); got != executable {
		t.Fatalf("findExecutable() = %q, want %q", got, executable)
	}
}

func TestEnsureRejectsChecksumMismatch(t *testing.T) {
	manager, _, _ := managerWithReleaseServer(t, "linux", "amd64", []byte("corrupted"), "mpv-v0.41.0-anylinux-x86_64.AppImage", "sha256:"+strings.Repeat("0", 64))
	_, err := manager.Ensure(context.Background())
	if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("Ensure() error = %v", err)
	}
}

func TestExtractZIPRejectsTraversal(t *testing.T) {
	archive := zipBytes(t, map[string]string{"../mpv.exe": "bad"})
	archivePath := filepath.Join(t.TempDir(), "mpv.zip")
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	err := extractZIP(archivePath, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("extractZIP() error = %v", err)
	}
}

func TestPlatformMatrix(t *testing.T) {
	tests := []struct {
		goos, goarch string
		suffix       string
		kind         archiveKind
		fail         bool
	}{
		{"darwin", "arm64", "-macos-14-arm.zip", archiveZIP, false},
		{"darwin", "amd64", "-macos-15-intel.zip", archiveZIP, false},
		{"windows", "amd64", "-x86_64-pc-windows-msvc.zip", archiveZIP, false},
		{"windows", "arm64", "-aarch64-pc-windows-msvc.zip", archiveZIP, false},
		{"windows", "386", "-i686-w64-mingw32.zip", archiveZIP, false},
		{"linux", "amd64", "-anylinux-x86_64.AppImage", archiveAppImage, false},
		{"linux", "arm64", "-anylinux-aarch64.AppImage", archiveAppImage, false},
		{"freebsd", "amd64", "", "", true},
	}
	for _, test := range tests {
		t.Run(test.goos+"-"+test.goarch, func(t *testing.T) {
			manager := newTestManager(t, test.goos, test.goarch)
			spec, err := manager.platform()
			if test.fail {
				if err == nil {
					t.Fatalf("platform() = %+v", spec)
				}
				return
			}
			if err != nil || spec.assetSuffix != test.suffix || spec.kind != test.kind {
				t.Fatalf("platform() = %+v, %v", spec, err)
			}
		})
	}
}

func TestPlayArguments(t *testing.T) {
	got := playArguments("https://pull.tiktokcdn.com/live.m3u8", PlayOptions{
		Title: " TikTok LIVE @example ", Referer: " https://www.tiktok.com/@example/live ",
	})
	want := []string{
		"--force-window=yes", "--keep-open=no",
		"--title=TikTok LIVE @example", "--referrer=https://www.tiktok.com/@example/live",
		"--", "https://pull.tiktokcdn.com/live.m3u8",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("playArguments() = %q, want %q", got, want)
	}
}

func TestPlayRejectsUntrustedStream(t *testing.T) {
	manager := newTestManager(t, "linux", "amd64")
	for _, streamURL := range []string{"http://pull.tiktokcdn.com/live.m3u8", "https://example.com/live.m3u8"} {
		if err := manager.Play(context.Background(), "/bin/false", streamURL, PlayOptions{}); err == nil {
			t.Fatalf("Play(%q) succeeded", streamURL)
		}
	}
}

func newTestManager(t *testing.T, goos, goarch string) *Manager {
	t.Helper()
	manager := NewManager(Options{})
	manager.goos = goos
	manager.goarch = goarch
	manager.tempDir = t.TempDir()
	manager.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	return manager
}

func managerWithReleaseServer(t *testing.T, goos, goarch string, asset []byte, assetName, digestOverride string) (*Manager, *int32, *bytes.Buffer) {
	t.Helper()
	hash := sha256.Sum256(asset)
	digest := "sha256:" + hex.EncodeToString(hash[:])
	if digestOverride != "" {
		digest = digestOverride
	}
	var requests int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&requests, 1)
		switch request.URL.Path {
		case "/release":
			_ = json.NewEncoder(writer).Encode(githubRelease{
				TagName: "v0.41.0",
				Assets: []githubAsset{{
					Name: assetName, URL: server.URL + "/asset", Size: int64(len(asset)), Digest: digest,
				}},
			})
		case "/asset":
			_, _ = writer.Write(asset)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	status := &bytes.Buffer{}
	manager := newTestManager(t, goos, goarch)
	manager.status = status
	manager.stableReleaseURL = server.URL + "/release"
	manager.linuxReleaseURL = server.URL + "/release"
	manager.httpClient = server.Client()
	manager.enforceGitHub = false
	return manager, &requests, status
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
