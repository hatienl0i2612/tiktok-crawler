package video

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestResolveVideo(t *testing.T) {
	t.Parallel()

	const videoID = "7670393762513718549"
	playbackURL := "https://v16-webapp.tiktokcdn.com/play.mp4?expire=1787850190"
	watermarkURL := "https://v16-webapp.tiktokcdn.com/download.mp4?expire=1787850190"
	playerBody, err := json.Marshal(playerResponse{Items: []playerItem{{
		IDStr:       videoID,
		Description: "Player description",
		Region:      "US",
		AuthorInfo: playerAuthor{
			UniqueID: "creator",
			Nickname: "Creator",
			SecretID: "secret",
		},
		StatisticsInfo: playerStatistics{DiggCount: 10, CommentCount: 2, ShareCount: 1},
		VideoInfo: playerVideoInfo{
			Meta: playerVideoMeta{Duration: 12, Width: 720, Height: 1280, Format: "MP4"},
			Profiles: []playerProfile{{
				CodecType: "h264",
				Bitrate:   1_000_000,
				FPS:       30,
				GearName:  "normal_720",
				PlayAddr: playerPlayAddress{
					Width: 720, Height: 1280, DataSize: 1234, URI: "video-uri",
					URLList: []string{playbackURL},
				},
			}},
		},
	}}})
	if err != nil {
		t.Fatalf("marshal player fixture: %v", err)
	}

	var embedState frontityState
	embedState.Source.Data = map[string]embedPageData{
		"video": {
			Code: http.StatusOK,
			VideoData: embedVideoData{
				ItemInfos: embedItem{
					ID: videoID, Text: "Embed description", CreateTime: "1700000000",
					PlayCount: 500, DiggCount: 12, CommentCount: 3, ShareCount: 2,
					Video: embedVideo{VideoMeta: embedVideoMeta{Width: 720, Height: 1280, Duration: 13}},
				},
				AuthorInfos:       embedAuthor{UserID: "author-id", UniqueID: "creator", Nickname: "Creator"},
				ChallengeInfoList: []embedChallenge{{ChallengeID: "1", ChallengeName: "testing"}},
			},
		},
	}
	embedJSON, err := json.Marshal(embedState)
	if err != nil {
		t.Fatalf("marshal embed fixture: %v", err)
	}
	pageBody := []byte(`<html><script type="application/json">{"downloadAddr":"` + watermarkURL + `"}</script></html>`)
	embedBody := []byte(`<script id="__FRONTITY_CONNECT_STATE__">` + string(embedJSON) + `</script>`)

	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	client.session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body []byte
		switch request.URL.Path {
		case "/@creator/video/" + videoID:
			body = pageBody
		case "/player/api/v1/items":
			if got := request.URL.Query().Get("item_ids"); got != videoID {
				t.Fatalf("item_ids query = %q", got)
			}
			body = playerBody
		case "/embed/v2/" + videoID:
			body = embedBody
		default:
			return nil, errors.New("unexpected request: " + request.URL.String())
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader(string(body))),
			ContentLength: int64(len(body)),
			Request:       request,
		}, nil
	})

	inputURL := "https://www.tiktok.com/@creator/video/" + videoID
	result, err := client.Resolve(context.Background(), inputURL)
	if err != nil {
		t.Fatalf("resolve video: %v", err)
	}
	if result.Video.ID != videoID || result.Video.Description != "Embed description" {
		t.Fatalf("unexpected video metadata: %+v", result.Video)
	}
	if result.Video.Statistics.PlayCount != 500 || result.Video.Statistics.LikeCount != 12 {
		t.Fatalf("unexpected statistics: %+v", result.Video.Statistics)
	}
	if result.Video.CreatedAt == nil || result.Video.CreatedAt.Unix() != 1_700_000_000 {
		t.Fatalf("unexpected creation time: %v", result.Video.CreatedAt)
	}
	if len(result.Sources) != 3 || len(result.Media) != 2 {
		t.Fatalf("sources=%v media=%+v", result.Sources, result.Media)
	}
	if result.Media[0].Watermarked || !result.Media[1].Watermarked || result.Media[1].URL != watermarkURL {
		t.Fatalf("unexpected media ordering: %+v", result.Media)
	}
}

func TestVideoIDFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{path: "/@creator/video/123456789", want: "123456789"},
		{path: "/embed/video/987654321/", want: "987654321"},
		{path: "/@creator/video/not-a-number", want: ""},
		{path: "/@creator/photo/123456789", want: ""},
	}
	for _, test := range tests {
		parsed, err := url.Parse("https://www.tiktok.com" + test.path)
		if err != nil {
			t.Fatalf("parse fixture URL: %v", err)
		}
		if got := videoIDFromURL(parsed); got != test.want {
			t.Errorf("videoIDFromURL(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}

func TestMakePlaybackMedia(t *testing.T) {
	t.Parallel()

	profileURL := "https://v16.tiktokcdn.com/profile.mp4?expire=1787850190"
	media := makePlaybackMedia(playerVideoInfo{
		Meta: playerVideoMeta{Format: "MP4"},
		Profiles: []playerProfile{
			{CodecType: "bytevc1", PlayAddr: playerPlayAddress{Width: 1080, Height: 1920, URLList: []string{profileURL, " https://backup.tiktokcdn.com/video "}}},
			{CodecType: "h264", PlayAddr: playerPlayAddress{Height: 720}},
		},
	})
	if len(media) != 1 || media[0].Codec != "h265" || media[0].Quality != "1920p" {
		t.Fatalf("unexpected profile media: %+v", media)
	}
	if len(media[0].BackupURLs) != 1 || media[0].ExpiresAt == nil {
		t.Fatalf("unexpected profile URLs: %+v", media[0])
	}

	fallback := makePlaybackMedia(playerVideoInfo{
		URI:     "fallback-uri",
		URLList: []string{"https://v16.tiktokcdn.com/fallback.mp4"},
		Meta:    playerVideoMeta{Width: 576, Height: 1024},
	})
	if len(fallback) != 1 || fallback[0].Codec != "h264" || fallback[0].Format != "mp4" {
		t.Fatalf("unexpected fallback media: %+v", fallback)
	}
}

func TestExtractDownloadURLs(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"downloadAddr": {"urlList": [
			"https://v16.tiktokcdn.com/a.mp4",
			"https://v16.tiktokcdn.com/a.mp4",
			"https://example.org/not-allowed.mp4"
		]},
		"nested": {"download_url": "https://api.tiktokv.com/b.mp4"},
		"playAddr": "https://v16.tiktokcdn.com/not-a-download.mp4"
	}`)
	urls := extractDownloadURLs(body)
	if len(urls) != 2 || urls[0] != "https://api.tiktokv.com/b.mp4" || urls[1] != "https://v16.tiktokcdn.com/a.mp4" {
		t.Fatalf("unexpected download URLs: %v", urls)
	}
	html := append([]byte(`<script>not JSON</script><script type="application/json">`), body...)
	html = append(html, []byte(`</script>`)...)
	if got := extractDownloadURLsFromHTML(html); len(got) != 2 {
		t.Fatalf("HTML download URLs: %v", got)
	}
	if got := extractDownloadURLs([]byte(`{`)); got != nil {
		t.Fatalf("invalid JSON returned URLs: %v", got)
	}
}

func TestMakeCaptions(t *testing.T) {
	t.Parallel()

	captions := makeCaptions([]playerCaption{
		{Language: "en", CaptionFormat: "webvtt", URLList: []string{"https://v16.tiktokcdn.com/en.vtt"}, Expire: 1_700_000_000, IsAutoGenerated: true},
		{Language: "vi"},
	})
	if len(captions) != 1 || captions[0].URL != "https://v16.tiktokcdn.com/en.vtt" || captions[0].ExpiresAt == nil {
		t.Fatalf("unexpected captions: %+v", captions)
	}
}

func TestResolverHelpers(t *testing.T) {
	t.Parallel()

	if got := qualityName(0); got != "unknown" {
		t.Fatalf("qualityName(0) = %q", got)
	}
	if got := qualityName(1080); got != "1080p" {
		t.Fatalf("qualityName(1080) = %q", got)
	}
	if got := normalizeCodec("AVC1"); got != "h264" {
		t.Fatalf("normalizeCodec(AVC1) = %q", got)
	}
	if got := normalizeCodec(""); got != "unknown" {
		t.Fatalf("normalizeCodec(empty) = %q", got)
	}
	expires := expiryFromURL("https://v16.tiktokcdn.com/video?x-m-expire=1787850190000")
	if expires == nil || !expires.Equal(time.Unix(1_787_850_190, 0).UTC()) {
		t.Fatalf("unexpected expiry: %v", expires)
	}
	values := uniqueStrings([]string{" a ", "", "a", "b"})
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("uniqueStrings() = %v", values)
	}
}

func TestFlexibleInt64(t *testing.T) {
	t.Parallel()

	for _, input := range []string{`123`, `"123"`} {
		var value flexibleInt64
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}
		if value != 123 {
			t.Fatalf("value = %d, want 123", value)
		}
	}
	var nullValue flexibleInt64 = 9
	if err := json.Unmarshal([]byte(`null`), &nullValue); err != nil || nullValue != 0 {
		t.Fatalf("unmarshal null = %d, %v", nullValue, err)
	}
	var invalid flexibleInt64
	if err := json.Unmarshal([]byte(`"abc"`), &invalid); err == nil {
		t.Fatal("invalid flexible integer unexpectedly succeeded")
	}
}
