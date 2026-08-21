package downloader

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tiktok-crawler/internal/media"
	"tiktok-crawler/internal/tiktok"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestDownloadUsesBackupAndReportsProgress(t *testing.T) {
	t.Parallel()
	payload := []byte{0, 0, 0, 20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 1, 2, 3, 4}
	requests := 0
	session, err := tiktok.NewSession(tiktok.SessionOptions{Cookie: "session=abc", UserAgent: "test-agent"})
	if err != nil {
		t.Fatal(err)
	}
	session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.Header.Get("Referer") != "https://www.tiktok.com/@creator/video/123" || request.Header.Get("Cookie") != "session=abc" {
			t.Fatalf("unexpected request headers: %v", request.Header)
		}
		if request.URL.Path == "/primary" {
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden")), Request: request}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/octet-stream"}}, Body: io.NopCloser(bytes.NewReader(payload)), ContentLength: int64(len(payload)), Request: request}, nil
	})

	outputPath := filepath.Join(t.TempDir(), "video.mp4")
	var progress []DownloadProgress
	result, err := Download(context.Background(), session, []media.Variant{{
		URL: "https://v16.tiktokcdn.com/primary", BackupURLs: []string{"https://v16.tiktokcdn.com/backup"},
	}}, FileInfo{Referer: "https://www.tiktok.com/@creator/video/123"}, Options{
		OutputPath: outputPath,
		Progress:   func(update DownloadProgress) { progress = append(progress, update) },
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if requests != 2 || result.Path != outputPath || result.Bytes != int64(len(payload)) {
		t.Fatalf("unexpected result: requests=%d result=%+v", requests, result)
	}
	written, err := os.ReadFile(outputPath)
	if err != nil || !bytes.Equal(written, payload) {
		t.Fatalf("downloaded payload = %v, %v", written, err)
	}
	if len(progress) < 2 || progress[0].DownloadedBytes != 0 || progress[len(progress)-1].DownloadedBytes != int64(len(payload)) {
		t.Fatalf("unexpected progress: %+v", progress)
	}
}

func TestDownloadHelpers(t *testing.T) {
	t.Parallel()
	if !looksLikeVideo("video/mp4; charset=binary", nil) || !looksLikeVideo("application/octet-stream", []byte{0, 0, 0, 20, 'f', 't', 'y', 'p'}) {
		t.Fatal("video response was not recognized")
	}
	if looksLikeVideo("text/html", []byte("<html>")) {
		t.Fatal("HTML was recognized as video")
	}
	if got := sanitizeFilenamePart("  bad///name...  "); got != "bad_name" {
		t.Fatalf("sanitizeFilenamePart() = %q", got)
	}
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	outputPath, err := resolveOutputPath("~/Downloads/output.mp4")
	if err != nil || outputPath != filepath.Join(homeDirectory, "Downloads", "output.mp4") {
		t.Fatalf("resolveOutputPath() = %q, %v", outputPath, err)
	}
	filename := defaultFilename(FileInfo{Author: " @creator name ", VideoID: "123/456"}, &media.Variant{Quality: "1080p", Codec: "h264", Format: "mp4"})
	if filename != "creator_name_123_456_no_watermark_1080p_h264.mp4" {
		t.Fatalf("DefaultFilename() = %q", filename)
	}
}

func TestDownloadValidation(t *testing.T) {
	t.Parallel()
	session, err := tiktok.NewSession(tiktok.SessionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{"Content-Type": []string{"text/html"}},
			Body:          io.NopCloser(strings.NewReader("not a video")),
			ContentLength: 11,
			Request:       request,
		}, nil
	})
	directory := t.TempDir()
	existingPath := filepath.Join(directory, "existing.mp4")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		variant media.Variant
		output  string
	}{
		{name: "empty output", variant: media.Variant{URL: "https://v16.tiktokcdn.com/video"}},
		{name: "existing output", variant: media.Variant{URL: "https://v16.tiktokcdn.com/video"}, output: existingPath},
		{name: "missing URL", output: filepath.Join(directory, "missing.mp4")},
		{name: "disallowed host", variant: media.Variant{URL: "https://example.org/video"}, output: filepath.Join(directory, "disallowed.mp4")},
		{name: "non-video response", variant: media.Variant{URL: "https://v16.tiktokcdn.com/video"}, output: filepath.Join(directory, "not-video.mp4")},
		{name: "missing directory", variant: media.Variant{URL: "https://v16.tiktokcdn.com/video"}, output: filepath.Join(directory, "missing", "video.mp4")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Download(context.Background(), session, []media.Variant{test.variant}, FileInfo{}, Options{OutputPath: test.output}); err == nil {
				t.Fatal("Download() unexpectedly succeeded")
			}
		})
	}
}

func TestDownloadExpandsHomeOutputPath(t *testing.T) {
	t.Parallel()
	session, err := tiktok.NewSession(tiktok.SessionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var started DownloadStart
	_, err = Download(context.Background(), session, []media.Variant{{Codec: "h265", Quality: "1920p"}}, FileInfo{}, Options{
		OutputPath: "~/Downloads/output.mp4",
		Quality:    "best",
		OnStart:    func(value DownloadStart) { started = value },
	})
	if err == nil {
		t.Fatal("Download() unexpectedly succeeded")
	}
	homeDirectory, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Fatal(homeErr)
	}
	want := filepath.Join(homeDirectory, "Downloads", "output.mp4")
	if started.OutputPath != want {
		t.Fatalf("start output path = %q, want %q", started.OutputPath, want)
	}
}
