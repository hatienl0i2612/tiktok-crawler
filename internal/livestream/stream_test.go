package livestream

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func mustStreamData(t *testing.T, codec, quality, streamURL string) string {
	t.Helper()
	metadata, err := json.Marshal(map[string]any{
		"VCodec":     codec,
		"resolution": "1920x1080",
		"vbitrate":   "9200000",
		"cdn_name":   "fixture-cdn",
	})
	if err != nil {
		t.Fatalf("marshal stream metadata: %v", err)
	}
	payload, err := json.Marshal(map[string]any{
		"data": map[string]any{
			quality: map[string]any{
				"main": map[string]any{
					"sdk_params": string(metadata),
					"hls":        streamURL,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal stream data: %v", err)
	}
	return string(payload)
}

func TestParseStreamData(t *testing.T) {
	t.Parallel()

	streamURL := "https://pull-hls.tiktokcdn.com/live.m3u8?expire=1787850190"
	streams, err := parseStreamData(mustStreamData(t, "bytevc1", "origin", streamURL), "h264")
	if err != nil {
		t.Fatalf("parse stream data: %v", err)
	}
	if len(streams) != 1 {
		t.Fatalf("stream count = %d, want 1", len(streams))
	}
	stream := streams[0]
	if stream.Codec != "h265" || stream.Resolution != "1920x1080" || stream.Bitrate != 9_200_000 {
		t.Fatalf("unexpected stream metadata: %+v", stream)
	}
	if stream.CDN != "fixture-cdn" || stream.URL != streamURL {
		t.Fatalf("unexpected stream endpoint: %+v", stream)
	}
	wantExpiry := time.Unix(1_787_850_190, 0).UTC()
	if stream.ExpiresAt == nil || !stream.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("ExpiresAt = %v, want %v", stream.ExpiresAt, wantExpiry)
	}

	for name, encoded := range map[string]string{
		"invalid JSON": `{`,
		"empty data":   `{"data":{}}`,
	} {
		name, encoded := name, encoded
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseStreamData(encoded, "h264"); err == nil {
				t.Fatal("parseStreamData() unexpectedly succeeded")
			}
		})
	}
}

func TestCollectStreamsDeduplicatesAndReportsMalformedCodecs(t *testing.T) {
	t.Parallel()

	streamURL := "https://pull-hls.tiktokcdn.com/live.m3u8"
	encoded := mustStreamData(t, "h264", "hd", streamURL)
	streams, parseErrors := collectStreams(roomInfo{LiveRoom: wireLiveRoom{
		StreamData:     streamContainer{PullData: pullData{StreamData: encoded}},
		HEVCStreamData: streamContainer{PullData: pullData{StreamData: encoded}},
	}})
	if len(parseErrors) != 0 {
		t.Fatalf("unexpected parse errors: %v", parseErrors)
	}
	if len(streams) != 1 {
		t.Fatalf("stream count = %d, want one deduplicated stream", len(streams))
	}

	streams, parseErrors = collectStreams(roomInfo{LiveRoom: wireLiveRoom{
		StreamData:     streamContainer{PullData: pullData{StreamData: encoded}},
		HEVCStreamData: streamContainer{PullData: pullData{StreamData: `{`}},
	}})
	if len(streams) != 1 || len(parseErrors) != 1 || !strings.HasPrefix(parseErrors[0], "h265:") {
		t.Fatalf("streams=%+v errors=%v", streams, parseErrors)
	}
}

func TestSelectStream(t *testing.T) {
	t.Parallel()

	streams := []Stream{
		{Codec: "h264", Quality: "ao", Line: "main", Protocol: "flv", URL: "audio"},
		{Codec: "h264", Quality: "hd_60", Line: "main", Protocol: "hls", URL: "hd60"},
		{Codec: "h264", Quality: "origin", Line: "backup", Protocol: "flv", URL: "origin-h264"},
		{Codec: "h265", Quality: "origin", Line: "main", Protocol: "hls", URL: "origin-h265"},
	}

	tests := []struct {
		name    string
		options SelectOptions
		wantURL string
	}{
		{name: "best", options: SelectOptions{Codec: "auto", Quality: "best", Format: "auto"}, wantURL: "origin-h264"},
		{name: "exact codec", options: SelectOptions{Codec: "h265", Quality: "best", Format: "hls"}, wantURL: "origin-h265"},
		{name: "normalized quality", options: SelectOptions{Codec: "h264", Quality: "hd60", Format: "hls"}, wantURL: "hd60"},
		{name: "audio only", options: SelectOptions{Codec: "h264", Quality: "ao", Format: "flv"}, wantURL: "audio"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			selected, err := SelectStream(streams, test.options)
			if err != nil {
				t.Fatalf("select stream: %v", err)
			}
			if selected.URL != test.wantURL {
				t.Fatalf("selected URL = %q, want %q", selected.URL, test.wantURL)
			}
		})
	}

	_, err := SelectStream(streams, SelectOptions{Codec: "h265", Quality: "hd", Format: "flv"})
	if err == nil || !strings.Contains(err.Error(), "available streams:") {
		t.Fatalf("unexpected no-match error: %v", err)
	}
}

func TestStreamHelpers(t *testing.T) {
	t.Parallel()

	if got := normalizeCodec("H.264/AVC"); got != "h264" {
		t.Fatalf("normalizeCodec() = %q", got)
	}
	if got := normalizeCodec("HEVC"); got != "h265" {
		t.Fatalf("normalizeCodec() = %q", got)
	}
	expires := expiryFromURL("https://cdn.tiktokcdn.com/live?x-expires=1787850190000")
	if expires == nil || expires.Unix() != 1_787_850_190 {
		t.Fatalf("unexpected expiry: %v", expires)
	}
	if expires := expiryFromURL("not a URL"); expires != nil {
		t.Fatalf("invalid URL produced expiry: %v", expires)
	}
}
