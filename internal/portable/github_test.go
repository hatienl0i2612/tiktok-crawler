package portable

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadAssetRejectsTruncatedAndOversizedBodies(t *testing.T) {
	t.Parallel()
	for _, payload := range []string{"short", "longer-than-expected"} {
		t.Run(payload, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				_, _ = writer.Write([]byte(payload))
			}))
			t.Cleanup(server.Close)
			hash := sha256.Sum256([]byte("expected"))
			asset := Asset{URL: server.URL, Size: 8, Digest: "sha256:" + hex.EncodeToString(hash[:])}
			path := filepath.Join(t.TempDir(), "binary")
			err := DownloadAsset(context.Background(), server.Client(), asset, path, false)
			if err == nil || !strings.Contains(err.Error(), "size mismatch") {
				t.Fatalf("DownloadAsset() error = %v", err)
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed download was retained: %v", err)
			}
		})
	}
}

func TestExecutableDownloadTrustBoundary(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"http://github.com/binary", "https://github.com.evil.test/binary", "://invalid"} {
		err := DownloadAsset(context.Background(), HTTPClient(), Asset{
			URL: target, Size: 1, Digest: "sha256:" + strings.Repeat("0", 64),
		}, filepath.Join(t.TempDir(), "binary"), true)
		if err == nil || !strings.Contains(err.Error(), "untrusted") {
			t.Errorf("DownloadAsset(%q) error = %v", target, err)
		}
	}
	client := HTTPClient()
	request, _ := http.NewRequest(http.MethodGet, "https://evil.test/binary", nil)
	if err := client.CheckRedirect(request, nil); err == nil {
		t.Fatal("download allowed an untrusted redirect")
	}
	request, _ = http.NewRequest(http.MethodGet, "https://release-assets.githubusercontent.com/binary", nil)
	if err := client.CheckRedirect(request, nil); err != nil {
		t.Fatalf("GitHub asset redirect rejected: %v", err)
	}
}
