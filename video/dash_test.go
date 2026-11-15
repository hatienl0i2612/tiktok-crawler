package video

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const dashVideoPage = `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{
 "__DEFAULT_SCOPE__":{"webapp.video-detail":{"statusCode":0,"itemInfo":{"itemStruct":{
  "id":"7702790327672884500","author":{"uniqueId":"nghienthitnuong"},
  "music":{"playUrl":"https://v16.tiktokcdn.com/unrelated-music.mp3"},
  "video":{"format":"mp4","width":576,"height":1024,"duration":145,
   "downloadAddr":"https://v16.tiktokcdn.com/watermark.mp4",
   "bitrateInfo":[
    {"Format":"mp4","CodecType":"h264","Bitrate":1784946,"GearName":"normal_540_0",
     "PlayAddr":{"Width":576,"Height":1024,"UrlList":["https://v16.tiktokcdn.com/muxed.mp4"]}},
    {"Format":"dash","CodecType":"h265_hvc1","Bitrate":1052102,"GearName":"adapt_lowest_1080_1",
     "VideoExtra":"{\"audio_file_id\":\"high-audio\"}",
     "PlayAddr":{"Width":1080,"Height":1920,"DataSize":"15646593","UrlList":["https://v16.tiktokcdn.com/media-video-hvc1/"]}},
    {"Format":"dash","CodecType":"h265_hvc1","Bitrate":545148,"GearName":"adapt_540_1",
     "VideoExtra":"{\"audio_file_id\":\"low-audio\"}",
     "PlayAddr":{"Width":576,"Height":1024,"UrlList":["https://v16.tiktokcdn.com/low-video.mp4"]}}
   ],
   "bitrateAudioInfo":[
    {"FileId":"low-audio","Bitrate":49071,"AudioDataSize":891595,
     "UrlList":{"MainUrl":"https://v16.tiktokcdn.com/low-audio.m4a"}},
    {"FileHash":"high-audio","Bitrate":64958,"AudioDataSize":"1183178",
     "UrlList":{"MainUrl":"https://v16.tiktokcdn.com/high-audio.m4a?expire=1794912835",
                "BackupUrl":"https://v19.tiktokcdn.com/high-audio.m4a",
                "FallbackUrl":"https://www.tiktok.com/aweme/v1/play/?file_id=high-audio"}}
   ]
  }
 }}}}
}</script>`

func TestResolveDASHVideoPairsAudio(t *testing.T) {
	t.Parallel()
	for _, playerStatus := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(http.StatusText(playerStatus), func(t *testing.T) {
			client, err := NewClient(ClientOptions{})
			if err != nil {
				t.Fatal(err)
			}
			client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				status, body := http.StatusOK, dashVideoPage
				switch request.URL.Path {
				case "/player/api/v1/items":
					status, body = playerStatus, `{"items":[{"id_str":"7702790327672884500","video_info":{"url_list":["https://v16.tiktokcdn.com/player-video-only.mp4"]}}]}`
				case "/embed/v2/7702790327672884500":
					status, body = http.StatusForbidden, "denied"
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
			})
			result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@nghienthitnuong/video/7702790327672884500")
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Media) != 4 {
				t.Fatalf("media = %+v", result.Media)
			}
			best := result.Media[0]
			if best.Width != 1080 || best.Height != 1920 || best.Format != "mp4" || best.Codec != "h265" || best.Audio == nil {
				t.Fatalf("best profile = %+v", best)
			}
			audio := best.Audio
			if audio.URI != "high-audio" || audio.Type != "audio" || audio.Size != 1183178 || audio.Bitrate != 64958 || audio.ExpiresAt == nil {
				t.Fatalf("audio track = %+v", audio)
			}
			if !strings.Contains(audio.URL, "high-audio.m4a") || len(audio.BackupURLs) != 2 || !strings.Contains(audio.BackupURLs[1], "/aweme/v1/play/") {
				t.Fatalf("audio backup order = %+v", audio)
			}
			if result.Media[1].Audio != nil || result.Media[1].Codec != "h264" || result.Media[2].Audio.URI != "low-audio" || result.Media[3].Audio != nil {
				t.Fatalf("audio associations leaked between profiles: %+v", result.Media)
			}
		})
	}
}

func TestDASHAudioMissingMetadataRemainsRequired(t *testing.T) {
	t.Parallel()
	for _, extra := range []string{"", "{", `{}`, `{"audio_file_id":"missing"}`} {
		audio := makeDASHAudio(extra, nil)
		if audio == nil || audio.Type != "audio" || audio.URL != "" || len(audio.BackupURLs) != 0 {
			t.Fatalf("makeDASHAudio(%q) = %+v", extra, audio)
		}
	}
}

func TestMakePlaybackMediaRequiresDASHAudio(t *testing.T) {
	t.Parallel()
	for _, info := range []playerVideoInfo{
		{Meta: playerVideoMeta{Format: "dash"}, URLList: []string{"https://v16.tiktokcdn.com/video"}},
		{Profiles: []playerProfile{{Format: "dash", PlayAddr: playerPlayAddress{URLList: []string{"https://v16.tiktokcdn.com/video"}}}}},
	} {
		variants := makePlaybackMedia(info)
		if len(variants) != 1 || variants[0].Format != "mp4" || variants[0].Audio == nil {
			t.Fatalf("DASH profile lost its required audio track: %+v", variants)
		}
	}
}

func TestResolveDASHPageWithoutVideoKeepsPlayerMedia(t *testing.T) {
	t.Parallel()
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{"__DEFAULT_SCOPE__":{"webapp.video-detail":{"itemInfo":{"itemStruct":{"id":"7702790327672884500","video":{"downloadAddr":"https://v16.tiktokcdn.com/watermark","bitrateAudioInfo":[{"FileHash":"audio"}]}}}}}}</script>`
		if request.URL.Path == "/player/api/v1/items" {
			body = `{"items":[{"id_str":"7702790327672884500","video_info":{"url_list":["https://v16.tiktokcdn.com/player.mp4"]}}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@nghienthitnuong/video/7702790327672884500")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Media) != 2 || result.Media[0].URL != "https://v16.tiktokcdn.com/player.mp4" || result.Media[0].Audio != nil {
		t.Fatalf("incomplete page replaced working player media: %+v", result.Media)
	}
}
