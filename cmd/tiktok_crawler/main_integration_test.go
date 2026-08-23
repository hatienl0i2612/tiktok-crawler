package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"

	"tiktok-crawler/internal/livestream"
	"tiktok-crawler/internal/shortdrama"
	"tiktok-crawler/internal/video"
)

func TestRunVideoJSONIntegration(t *testing.T) {
	if os.Getenv("TIKTOK_VIDEO_INTEGRATION") != "1" {
		t.Skip("set TIKTOK_VIDEO_INTEGRATION=1 to run against TikTok")
	}
	const inputURL = "https://www.tiktok.com/@forever0404_/video/7671176369300327700"
	var stdout, stderr bytes.Buffer
	if err := run([]string{inputURL, "-json"}, &stdout, &stderr); err != nil {
		t.Fatalf("run -json %s: %v", inputURL, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	var result video.Result
	decoder := json.NewDecoder(&stdout)
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("decode JSON output: %v\noutput: %s", err, stdout.String())
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("expected exactly one JSON document, got: %v", err)
	}
	if result.InputURL != inputURL || result.Video.ID != "7671176369300327700" || !strings.EqualFold(result.Video.Author.UniqueID, "forever0404_") {
		t.Fatalf("unexpected video result: %+v", result.Video)
	}
	if len(result.Media) == 0 {
		t.Fatal("expected at least one downloadable media profile")
	}
	for _, media := range result.Media {
		parsed, err := url.Parse(media.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			t.Fatalf("invalid media URL %q: %v", media.URL, err)
		}
	}
}

func TestRunLiveJSONIntegration(t *testing.T) {
	if os.Getenv("TIKTOK_LIVE_INTEGRATION") != "1" {
		t.Skip("set TIKTOK_LIVE_INTEGRATION=1 to run against TikTok")
	}
	const inputURL = "https://www.tiktok.com/@weathernewslive/live"
	var stdout, stderr bytes.Buffer
	if err := run([]string{inputURL, "-quality", "720p", "-json"}, &stdout, &stderr); err != nil {
		t.Fatalf("run -json %s: %v", inputURL, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	var result livestream.Result
	decoder := json.NewDecoder(&stdout)
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("decode JSON output: %v\noutput: %s", err, stdout.String())
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("expected exactly one JSON document, got: %v", err)
	}
	if result.InputURL != inputURL || !strings.EqualFold(result.User.UniqueID, "weathernewslive") || !result.Live.IsLive || len(result.Streams) == 0 {
		t.Fatalf("unexpected LIVE result: %+v", result)
	}
}

func TestRunShortDramaJSONIntegration(t *testing.T) {
	if os.Getenv("TIKTOK_SHORT_DRAMA_INTEGRATION") != "1" {
		t.Skip("set TIKTOK_SHORT_DRAMA_INTEGRATION=1 to run against TikTok")
	}

	const inputURL = "https://www.tiktok.com/shortdrama/episode/7665073849083368469/1"
	var stdout, stderr bytes.Buffer
	// Short Drama needs a valid browser msToken cookie. Use -cookies-from-browser
	// (or -cookies-file) with a browser that is logged in to TikTok.
	if err := run([]string{inputURL, "-json", "-cookies-from-browser", "chrome"}, &stdout, &stderr); err != nil {
		t.Fatalf("run -json %s: %v", inputURL, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	var result shortdrama.Result
	if err := json.NewDecoder(&stdout).Decode(&result); err != nil {
		t.Fatalf("decode JSON output: %v\noutput: %s", err, stdout.String())
	}
	if result.ShortDrama == nil || result.ShortDrama.ID != "7665073849083368469" || result.ShortDrama.Episode != 1 {
		t.Fatalf("unexpected Short Drama metadata: %#v", result.ShortDrama)
	}
	if result.Video.ID == "" || len(result.Media) == 0 {
		t.Fatalf("expected episode video and media: %#v", result)
	}
}
