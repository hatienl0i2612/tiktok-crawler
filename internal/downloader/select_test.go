package downloader

import (
	"strings"
	"testing"

	"tiktok-crawler/internal/media"
)

func TestSelect(t *testing.T) {
	t.Parallel()
	variants := []media.Variant{
		{Codec: "h264", Quality: "720p", Width: 720, Height: 720, Bitrate: 800, URL: "h264-720"},
		{Codec: "h264", Quality: "1080p", Width: 1080, Height: 1080, Bitrate: 1_000, URL: "h264-1080"},
		{Codec: "h265", Quality: "1920p", Width: 1080, Height: 1920, Bitrate: 2_000, URL: "h265-1920"},
		{Watermarked: true, Codec: "unknown", Quality: "1080p", Width: 1080, Height: 1080, URL: "watermark"},
	}
	for _, test := range []struct {
		name    string
		options Options
		wantURL string
	}{
		{name: "best quality", options: Options{Quality: "best"}, wantURL: "h265-1920"},
		{name: "exact height", options: Options{Quality: "720p"}, wantURL: "h264-720"},
		{name: "watermark", options: Options{Quality: "best", Watermarked: true}, wantURL: "watermark"},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, err := selectVariant(variants, test.options)
			if err != nil || selected.URL != test.wantURL {
				t.Fatalf("Select() = %#v, %v; want %q", selected, err, test.wantURL)
			}
		})
	}
}

func TestSelectErrorsAndHelpers(t *testing.T) {
	t.Parallel()
	variants := []media.Variant{{Codec: "h264", Quality: "720p", Height: 720, URL: "video"}}
	if _, err := selectVariant(variants, Options{Quality: "full-hd"}); err == nil {
		t.Fatal("selectVariant() accepted an invalid quality")
	}
	if _, err := selectVariant(variants, Options{Quality: "best", Watermarked: true}); err == nil || !strings.Contains(err.Error(), "available variants:") {
		t.Fatalf("unexpected no-match error: %v", err)
	}
}
