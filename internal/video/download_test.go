package video

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadUsesBackupAndReportsProgress(t *testing.T) {
	t.Parallel()

	payload := []byte{0, 0, 0, 20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 1, 2, 3, 4}
	requests := 0
	client := &Client{
		cookie:    "session=abc",
		userAgent: "test-agent",
		httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requests++
			if request.Header.Get("Referer") != "https://www.tiktok.com/@creator/video/123" {
				t.Fatalf("Referer = %q", request.Header.Get("Referer"))
			}
			if request.Header.Get("Cookie") != "session=abc" {
				t.Fatalf("Cookie = %q", request.Header.Get("Cookie"))
			}
			if request.URL.Path == "/primary" {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Status:     "403 Forbidden",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("forbidden")),
					Request:    request,
				}, nil
			}
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Header:        http.Header{"Content-Type": []string{"application/octet-stream"}},
				Body:          io.NopCloser(bytes.NewReader(payload)),
				ContentLength: int64(len(payload)),
				Request:       request,
			}, nil
		})},
	}

	outputPath := filepath.Join(t.TempDir(), "video.mp4")
	var progress []DownloadProgress
	result, err := client.Download(context.Background(), Media{
		URL:        "https://v16.tiktokcdn.com/primary",
		BackupURLs: []string{"https://v16.tiktokcdn.com/backup"},
	}, DownloadOptions{
		OutputPath: outputPath,
		Referer:    "https://www.tiktok.com/@creator/video/123",
		Progress: func(update DownloadProgress) {
			progress = append(progress, update)
		},
	})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
	if result.Path != outputPath || result.Bytes != int64(len(payload)) || !strings.HasSuffix(result.SourceURL, "/backup") {
		t.Fatalf("unexpected result: %+v", result)
	}
	written, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if !bytes.Equal(written, payload) {
		t.Fatalf("downloaded payload = %v", written)
	}
	if len(progress) < 2 || progress[0].DownloadedBytes != 0 || progress[len(progress)-1].DownloadedBytes != int64(len(payload)) {
		t.Fatalf("unexpected progress: %+v", progress)
	}
}

func TestDownloadValidation(t *testing.T) {
	t.Parallel()

	client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{"Content-Type": []string{"text/html"}},
			Body:          io.NopCloser(strings.NewReader("not a video")),
			ContentLength: 11,
			Request:       request,
		}, nil
	})}}
	temporaryDirectory := t.TempDir()
	existingPath := filepath.Join(temporaryDirectory, "existing.mp4")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o600); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	tests := []struct {
		name    string
		media   Media
		options DownloadOptions
	}{
		{name: "empty output", media: Media{URL: "https://v16.tiktokcdn.com/video"}},
		{name: "existing output", media: Media{URL: "https://v16.tiktokcdn.com/video"}, options: DownloadOptions{OutputPath: existingPath}},
		{name: "missing URL", options: DownloadOptions{OutputPath: filepath.Join(temporaryDirectory, "missing.mp4")}},
		{name: "disallowed host", media: Media{URL: "https://example.org/video"}, options: DownloadOptions{OutputPath: filepath.Join(temporaryDirectory, "disallowed.mp4")}},
		{name: "non-video response", media: Media{URL: "https://v16.tiktokcdn.com/video"}, options: DownloadOptions{OutputPath: filepath.Join(temporaryDirectory, "not-video.mp4")}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := client.Download(context.Background(), test.media, test.options); err == nil {
				t.Fatal("Download() unexpectedly succeeded")
			}
		})
	}
}

func TestDefaultFilename(t *testing.T) {
	t.Parallel()

	result := &Result{Video: Video{ID: "123/456", Author: Author{UniqueID: " @creator name "}}}
	media := &Media{Quality: "1080p", Codec: "h264", Format: "mp4"}
	if got, want := DefaultFilename(result, media), "creator_name_123_456_no_watermark_1080p_h264.mp4"; got != want {
		t.Fatalf("DefaultFilename() = %q, want %q", got, want)
	}
	media.Watermarked = true
	if got := DefaultFilename(result, media); !strings.Contains(got, "_watermark_") {
		t.Fatalf("watermarked filename = %q", got)
	}
	if got := DefaultFilename(nil, nil); got != "video_unknown_no_watermark_unknown_unknown.mp4" {
		t.Fatalf("nil filename = %q", got)
	}
}

func TestDownloadHelpers(t *testing.T) {
	t.Parallel()

	if !looksLikeVideo("video/mp4; charset=binary", nil) {
		t.Fatal("video content type was not recognized")
	}
	if !looksLikeVideo("application/octet-stream", []byte{0, 0, 0, 20, 'f', 't', 'y', 'p'}) {
		t.Fatal("MP4 header was not recognized")
	}
	if looksLikeVideo("text/html", []byte("<html>")) {
		t.Fatal("HTML was recognized as video")
	}
	if got := sanitizeFilenamePart("  bad///name...  "); got != "bad_name" {
		t.Fatalf("sanitizeFilenamePart() = %q", got)
	}
	if got := redactURL("https://user:secret@v16.tiktokcdn.com/video"); strings.Contains(got, "secret") {
		t.Fatalf("redactURL() exposed password: %q", got)
	}
}
