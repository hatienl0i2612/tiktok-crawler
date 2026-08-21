package video

import (
	"context"
	"strings"
	"testing"
)

func TestDownloadResolvedSelectsMediaAndReportsStart(t *testing.T) {
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	result := &Result{
		FinalURL: "https://www.tiktok.com/@example/video/123",
		Video:    Video{ID: "123", Author: Author{UniqueID: "example"}},
		Media:    []Media{{Codec: "h264", Quality: "720p", Height: 720}},
	}
	var started DownloadStart
	_, err = client.DownloadResolved(context.Background(), result, ResolvedDownloadOptions{
		Quality: "720p",
		OnStart: func(value DownloadStart) {
			started = value
		},
	})
	if err == nil || !strings.Contains(err.Error(), "selected media has no download URL") {
		t.Fatalf("DownloadResolved error = %v", err)
	}
	if started.Result != result || started.Media == nil || started.Media.Quality != "720p" {
		t.Fatalf("unexpected start event: %#v", started)
	}
	if !strings.Contains(started.OutputPath, "example_123_no_watermark_720p_h264.mp4") {
		t.Fatalf("default output path = %q", started.OutputPath)
	}
}
