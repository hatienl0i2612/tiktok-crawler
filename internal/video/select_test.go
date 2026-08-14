package video

import (
	"strings"
	"testing"
)

func TestSelectMedia(t *testing.T) {
	t.Parallel()

	media := []Media{
		{Watermarked: false, Codec: "h264", Quality: "720p", Width: 720, Height: 720, Bitrate: 800, URL: "h264-720"},
		{Watermarked: false, Codec: "h264", Quality: "1080p", Width: 1080, Height: 1080, Bitrate: 1_000, URL: "h264-1080"},
		{Watermarked: false, Codec: "h265", Quality: "1080p", Width: 1080, Height: 1080, Bitrate: 2_000, URL: "h265-1080"},
		{Watermarked: true, Codec: "unknown", Quality: "1080p", Width: 1080, Height: 1080, URL: "watermark"},
	}

	tests := []struct {
		name    string
		options SelectOptions
		wantURL string
	}{
		{name: "default codec", options: SelectOptions{Quality: "best"}, wantURL: "h264-1080"},
		{name: "automatic codec", options: SelectOptions{Codec: "auto", Quality: "best"}, wantURL: "h265-1080"},
		{name: "exact height", options: SelectOptions{Codec: "h264", Quality: "720p"}, wantURL: "h264-720"},
		{name: "watermark", options: SelectOptions{Codec: "auto", Quality: "best", Watermarked: true}, wantURL: "watermark"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			selected, err := SelectMedia(media, test.options)
			if err != nil {
				t.Fatalf("select media: %v", err)
			}
			if selected.URL != test.wantURL {
				t.Fatalf("selected URL = %q, want %q", selected.URL, test.wantURL)
			}
		})
	}
}

func TestSelectMediaErrors(t *testing.T) {
	t.Parallel()

	media := []Media{{Codec: "h264", Quality: "720p", Height: 720, URL: "video"}}
	for name, options := range map[string]SelectOptions{
		"codec":   {Codec: "vp9", Quality: "best"},
		"quality": {Codec: "h264", Quality: "full-hd"},
	} {
		name, options := name, options
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := SelectMedia(media, options); err == nil {
				t.Fatal("SelectMedia() unexpectedly succeeded")
			}
		})
	}
	_, err := SelectMedia(media, SelectOptions{Codec: "h265", Quality: "best"})
	if err == nil || !strings.Contains(err.Error(), "available variants:") {
		t.Fatalf("unexpected no-match error: %v", err)
	}
	if got := availableMediaSummary(nil); got != "none" {
		t.Fatalf("availableMediaSummary(nil) = %q", got)
	}
}
