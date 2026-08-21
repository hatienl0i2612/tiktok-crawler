package main

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"tiktok-crawler/internal/livestream"
)

func TestParseOptionsDetectsVideoAndLiveURLs(t *testing.T) {
	t.Setenv("TIKTOK_COOKIE", "")

	const videoURL = "https://www.tiktok.com/@forever0404_/video/7671176369300327700"
	const liveURL = "https://www.tiktok.com/@weathernewslive/live"
	tests := []struct {
		name string
		args []string
		want options
	}{
		{
			name: "video with flags after URL",
			args: []string{videoURL, "-watermark", "-quality", "720p", "-output", "result.mp4"},
			want: options{inputURL: videoURL, content: contentTypeVideo, output: "result.mp4", watermark: true, quality: "720p", userAgent: livestream.DefaultUserAgent, timeout: 20 * time.Second},
		},
		{
			name: "live ignores video-only options",
			args: []string{liveURL, "-quality", "720p", "-watermark", "-output", "unused.mp4", "-json"},
			want: options{inputURL: liveURL, content: contentTypeLive, output: "unused.mp4", watermark: true, quality: "720p", json: true, userAgent: livestream.DefaultUserAgent, timeout: 20 * time.Second},
		},
		{
			name: "live flags before URL",
			args: []string{"-verbose", "-timeout", "30s", "-user-agent", "test-agent", liveURL},
			want: options{inputURL: liveURL, content: contentTypeLive, quality: "best", verbose: true, userAgent: "test-agent", timeout: 30 * time.Second},
		},
		{
			name: "URL flag",
			args: []string{"-json", "-url", videoURL},
			want: options{inputURL: videoURL, content: contentTypeVideo, quality: "best", json: true, userAgent: livestream.DefaultUserAgent, timeout: 20 * time.Second},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got, err := parseOptions(test.args, &stderr)
			if err != nil {
				t.Fatalf("parseOptions(%q): %v\nstderr: %s", test.args, err, stderr.String())
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("options = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		url  string
		want contentType
		fail bool
	}{
		{"https://www.tiktok.com/@example/live", contentTypeLive, false},
		{"https://www.tiktok.com/@example/video/1234567890123456789?lang=en", contentTypeVideo, false},
		{"https://example.com/@example/live", "", true},
		{"https://www.tiktok.com/@example", "", true},
	}
	for _, test := range tests {
		got, err := detectContentType(test.url)
		if test.fail {
			if err == nil {
				t.Errorf("detectContentType(%q) succeeded with %q", test.url, got)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Errorf("detectContentType(%q) = %q, %v; want %q", test.url, got, err, test.want)
		}
	}
}
