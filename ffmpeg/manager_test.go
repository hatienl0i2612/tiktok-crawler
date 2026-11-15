package ffmpeg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hatienl0i2612/tiktok-crawler/internal/portable"
)

func TestEnsureUsesFFmpegFromPath(t *testing.T) {
	t.Parallel()
	manager := NewManager(Options{})
	manager.lookPath = func(name string) (string, error) {
		if name != "ffmpeg" {
			t.Fatalf("LookPath(%q)", name)
		}
		return "/usr/local/bin/ffmpeg", nil
	}
	path, err := manager.Ensure(context.Background())
	if err != nil || path != "/usr/local/bin/ffmpeg" {
		t.Fatalf("Ensure() = %q, %v", path, err)
	}
}

func TestEnsureDownloadsVerifiesAndCaches(t *testing.T) {
	t.Parallel()
	for _, goos := range []string{"darwin", "linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			manager, requests := releaseManager(t, goos, "")
			var status bytes.Buffer
			manager.status = &status
			path, err := manager.Ensure(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			payload, err := os.ReadFile(path)
			if err != nil || string(payload) != "portable-ffmpeg" {
				t.Fatalf("executable = %q, %v", payload, err)
			}
			if !manager.isExecutable(path) || requests.Load() != 2 {
				t.Fatalf("invalid executable or request count: %s, %d", path, requests.Load())
			}
			if goos == "windows" && filepath.Ext(path) != ".exe" {
				t.Fatalf("Windows executable = %q", path)
			}
			cached, err := manager.Ensure(context.Background())
			if err != nil || cached != path || requests.Load() != 2 {
				t.Fatalf("cached Ensure() = %q, %v, requests=%d", cached, err, requests.Load())
			}
			if !strings.Contains(status.String(), "installing a portable build") || !strings.Contains(status.String(), "Portable FFmpeg is ready") {
				t.Fatalf("status = %q", status.String())
			}
			assertNoStaging(t, manager.tempDir)
		})
	}
}

func TestEnsureRejectsUnverifiedDownloadAndCleansUp(t *testing.T) {
	t.Parallel()
	for _, digest := range []string{"invalid", "sha256:" + strings.Repeat("0", 64)} {
		t.Run(digest, func(t *testing.T) {
			manager, _ := releaseManager(t, "darwin", digest)
			if _, err := manager.Ensure(context.Background()); err == nil || !strings.Contains(err.Error(), "SHA-256") {
				t.Fatalf("Ensure() error = %v", err)
			}
			assertNoStaging(t, manager.tempDir)
			if _, err := os.Stat(filepath.Join(manager.tempDir, "tiktok-crawler", "ffmpeg", "darwin-amd64")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed installation published a cache: %v", err)
			}
		})
	}
}

func TestEnsureCancellation(t *testing.T) {
	t.Parallel()
	manager, requests := releaseManager(t, "linux", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.Ensure(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ensure() error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatal("cancelled install sent a request")
	}
	assertNoStaging(t, manager.tempDir)
}

func TestAssetPlatformMatrix(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ goos, goarch, want string }{
		{"darwin", "arm64", "ffmpeg-darwin-arm64"},
		{"darwin", "amd64", "ffmpeg-darwin-x64"},
		{"linux", "amd64", "ffmpeg-linux-x64"},
		{"linux", "arm64", "ffmpeg-linux-arm64"},
		{"linux", "arm", "ffmpeg-linux-arm"},
		{"linux", "386", "ffmpeg-linux-ia32"},
		{"windows", "amd64", "ffmpeg-win32-x64"},
	} {
		manager := &Manager{goos: test.goos, goarch: test.goarch}
		if got, err := manager.assetName(); err != nil || got != test.want {
			t.Errorf("%s/%s = %q, %v; want %q", test.goos, test.goarch, got, err, test.want)
		}
	}
	manager := &Manager{goos: "freebsd", goarch: "amd64"}
	if _, err := manager.assetName(); err == nil || !strings.Contains(err.Error(), "install ffmpeg in PATH manually") {
		t.Fatalf("unsupported platform error = %v", err)
	}
}

func releaseManager(t *testing.T, goos, digest string) (*Manager, *atomic.Int32) {
	t.Helper()
	payload := []byte("portable-ffmpeg")
	if digest == "" {
		hash := sha256.Sum256(payload)
		digest = "sha256:" + hex.EncodeToString(hash[:])
	}
	requests := &atomic.Int32{}
	manager := NewManager(Options{})
	manager.goos, manager.goarch = goos, "amd64"
	manager.tempDir = t.TempDir()
	manager.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	name, err := manager.assetName()
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path == "/release" {
			_ = json.NewEncoder(writer).Encode(portable.Release{TagName: "test", Assets: []portable.Asset{{
				Name: name, Size: int64(len(payload)), Digest: digest, URL: server.URL + "/binary",
			}}})
		} else {
			_, _ = writer.Write(payload)
		}
	}))
	t.Cleanup(server.Close)
	manager.releaseURL, manager.httpClient, manager.enforceGitHub = server.URL+"/release", server.Client(), false
	return manager, requests
}

func assertNoStaging(t *testing.T, root string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "tiktok-crawler", "ffmpeg", ".ffmpeg-install-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("staging directories remain: %v, %v", matches, err)
	}
}
