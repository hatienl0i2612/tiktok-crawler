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

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
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
	if !looksLikeMedia("video", "video/mp4; charset=binary", nil) || !looksLikeMedia("video", "application/octet-stream", []byte{0, 0, 0, 20, 'f', 't', 'y', 'p'}) {
		t.Fatal("video response was not recognized")
	}
	if looksLikeMedia("video", "text/html", []byte("<html>")) {
		t.Fatal("HTML was recognized as video")
	}
	if !looksLikeMedia("image", "image/jpeg", nil) || !looksLikeMedia("image", "application/octet-stream", []byte{0xff, 0xd8, 0xff, 0xe0}) {
		t.Fatal("image response was not recognized")
	}
	if !looksLikeMedia("image", "application/octet-stream", []byte{0, 0, 0, 20, 'f', 't', 'y', 'p', 'a', 'v', 'i', 'f'}) {
		t.Fatal("AVIF image response was not recognized")
	}
	if looksLikeMedia("image", "text/html", []byte("<html>")) {
		t.Fatal("HTML was recognized as image")
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

func TestDownloadAllDownloadsImageCollection(t *testing.T) {
	t.Parallel()
	payloads := map[string][]byte{
		"/one": {0xff, 0xd8, 0xff, 0xe0, 1, 2, 3, 4},
		"/two": {0xff, 0xd8, 0xff, 0xe0, 5, 6, 7, 8},
	}
	session, err := tiktok.NewSession(tiktok.SessionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		payload, ok := payloads[request.URL.Path]
		if !ok {
			t.Fatalf("unexpected URL: %s", request.URL)
		}
		if request.Header.Get("Referer") != "https://www.tiktok.com/@creator/photo/123" {
			t.Fatalf("unexpected Referer: %q", request.Header.Get("Referer"))
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{"Content-Type": []string{"image/jpeg"}},
			Body:          io.NopCloser(bytes.NewReader(payload)),
			ContentLength: int64(len(payload)),
			Request:       request,
		}, nil
	})

	outputDirectory := filepath.Join(t.TempDir(), "images")
	var starts []BatchStart
	var progress []BatchProgress
	result, err := DownloadAll(context.Background(), session, []BatchItem{
		{Variants: []media.Variant{{Type: "image", Format: "jpeg", Quality: "1350p", URL: "https://v16.tiktokcdn.com/one"}}},
		{Variants: []media.Variant{{Type: "image", Format: "jpeg", Quality: "1350p", URL: "https://v16.tiktokcdn.com/two"}}},
	}, FileInfo{Author: "creator", VideoID: "123", Referer: "https://www.tiktok.com/@creator/photo/123"}, BatchOptions{
		OutputDir: outputDirectory,
		OnStart: func(start BatchStart) {
			starts = append(starts, start)
		},
		Progress: func(update BatchProgress) {
			progress = append(progress, update)
		},
	})
	if err != nil {
		t.Fatalf("DownloadAll: %v", err)
	}
	if len(result.Downloads) != 2 || len(starts) != 2 || starts[0].Index != 1 || starts[1].Index != 2 {
		t.Fatalf("unexpected batch result: downloads=%+v starts=%+v", result, starts)
	}
	if len(progress) < 4 || progress[0].Index != 1 || progress[len(progress)-1].Index != 2 {
		t.Fatalf("unexpected batch progress: %+v", progress)
	}
	for index, download := range result.Downloads {
		if filepath.Dir(download.Path) != outputDirectory {
			t.Fatalf("output directory = %q, want %q", filepath.Dir(download.Path), outputDirectory)
		}
		written, err := os.ReadFile(download.Path)
		if err != nil || !bytes.Equal(written, payloads[[]string{"/one", "/two"}[index]]) {
			t.Fatalf("download %d = %q, %v", index, written, err)
		}
	}
}

func TestDownloadAllUsesPerVideoMetadataAndQuality(t *testing.T) {
	t.Parallel()
	payload := []byte{0, 0, 0, 20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}
	session, err := tiktok.NewSession(tiktok.SessionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Referer") != "https://www.tiktok.com/@creator/video/222" {
			t.Fatalf("Referer = %q", request.Header.Get("Referer"))
		}
		return &http.Response{
			StatusCode: http.StatusOK, Status: "200 OK",
			Header: http.Header{"Content-Type": []string{"video/mp4"}},
			Body:   io.NopCloser(bytes.NewReader(payload)), ContentLength: int64(len(payload)), Request: request,
		}, nil
	})

	directory := t.TempDir()
	var start BatchStart
	result, err := DownloadAll(context.Background(), session, []BatchItem{{
		File: FileInfo{VideoID: "222", Referer: "https://www.tiktok.com/@creator/video/222"},
		Variants: []media.Variant{
			{Type: "video", Codec: "h264", Format: "mp4", Quality: "720p", Height: 720, URL: "https://v16.tiktokcdn.com/720"},
			{Type: "video", Codec: "h265", Format: "mp4", Quality: "1080p", Height: 1080, URL: "https://v16.tiktokcdn.com/1080"},
		},
	}}, FileInfo{Author: "creator"}, BatchOptions{
		OutputDir: directory, Quality: "720p", OnStart: func(value BatchStart) { start = value },
	})
	if err != nil {
		t.Fatalf("DownloadAll: %v", err)
	}
	if len(result.Downloads) != 1 || filepath.Base(result.Downloads[0].Path) != "creator_222_no_watermark_720p_h264.mp4" {
		t.Fatalf("unexpected downloads: %+v", result)
	}
	if start.File.Author != "creator" || start.File.VideoID != "222" || start.Media.Quality != "720p" {
		t.Fatalf("unexpected start: %+v", start)
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
