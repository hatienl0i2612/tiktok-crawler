package photo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestResolvePhotoPost(t *testing.T) {
	t.Parallel()
	const photoID = "7666697358540852500"
	firstURL := "https://v16-webapp.tiktokcdn.com/image/first.jpeg?expire=1787850190"
	watermarkedURL := "https://v16-webapp.tiktokcdn.com/image/first-watermarked.jpeg?expire=1787850190"
	secondURL := "https://v16-webapp.tiktokcdn.com/image/second.webp?expire=1787850190"
	audioURL := "https://v16-webapp.tiktokcdn.com/audio/sound.mp3?expire=1787850190"
	body, err := json.Marshal(playerResponse{Items: []playerItem{{
		IDStr:       photoID,
		AwemeType:   150,
		Description: "Photo Post description",
		Region:      "VN",
		AuthorInfo:  playerAuthor{UniqueID: "creator", Nickname: "Creator", SecretID: "secret"},
		ImagePostInfo: imagePostInfo{Images: []postImage{
			{DisplayImage: imageAddress{Width: 1080, Height: 1350, URLList: []string{firstURL, " https://backup.tiktokcdn.com/image/first.jpeg "}}, OwnerWatermarkImage: imageAddress{Width: 1080, Height: 1350, URLList: []string{watermarkedURL}}},
			{DisplayImage: imageAddress{Width: 720, Height: 900, URLList: []string{secondURL}}},
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	embedBody := []byte(`<script id="__FRONTITY_CONNECT_STATE__" type="application/json">{"source":{"data":{"photo":{"code":200,"videoData":{"itemInfos":{"id":"` + photoID + `"},"musicInfos":{"musicId":"sound-id","musicName":"Embed sound","authorName":"Sound author","playUrl":["` + audioURL + `"]}}}}}}</script>`)
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	inputURL := "https://www.tiktok.com/@creator/photo/" + photoID
	client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Referer") != inputURL {
			t.Fatalf("Referer = %q, want %q", request.Header.Get("Referer"), inputURL)
		}
		switch request.URL.Path {
		case "/player/api/v1/items":
			if request.URL.Query().Get("item_ids") != photoID {
				t.Fatalf("item_ids = %q", request.URL.Query().Get("item_ids"))
			}
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), ContentLength: int64(len(body)), Request: request}, nil
		case "/embed/v2/" + photoID:
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(embedBody))), ContentLength: int64(len(embedBody)), Request: request}, nil
		default:
			return nil, errors.New("unexpected request: " + request.URL.String())
		}
	})

	result, err := client.Resolve(context.Background(), inputURL)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Post.ID != photoID || result.Post.Author.UniqueID != "creator" || len(result.Images) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	firstImage := result.Images[0]
	if firstImage.Index != 1 || len(firstImage.Media) != 2 || firstImage.Media[0].Watermarked || !firstImage.Media[1].Watermarked {
		t.Fatalf("unexpected first image media: %+v", firstImage)
	}
	if firstImage.Media[0].Type != "image" || firstImage.Media[0].Format != "jpeg" || firstImage.Media[0].Quality != "1350p" || firstImage.Media[0].ExpiresAt == nil {
		t.Fatalf("unexpected normalized image: %+v", firstImage.Media[0])
	}
	if result.Images[1].Media[0].Format != "webp" {
		t.Fatalf("second image format = %q", result.Images[1].Media[0].Format)
	}
	if len(result.Post.Music.PlaybackURLs) != 1 || result.Post.Music.PlaybackURLs[0] != audioURL {
		t.Fatalf("unexpected music playback URLs: %+v", result.Post.Music)
	}
}

func TestResolvePhotoStoryFromEmbedFallback(t *testing.T) {
	t.Parallel()
	const storyID = "7677796567234940178"
	const createdAt = "1787626328"
	imageURL := "https://p16-common-sign.tiktokcdn.com/story.jpeg?x-expires=1787824800"
	backupURL := "https://p19-common-sign.tiktokcdn.com/story.jpeg?x-expires=1787824800"
	audioURL := "https://v16m.tiktokcdn.com/story-sound.mp4"

	state := embedState{}
	state.Source.Data = map[string]embedPage{
		"/embed/v2/" + storyID: {
			Code: http.StatusOK,
			VideoData: embedPhoto{
				Item: embedItem{
					ID:              storyID,
					CreateTime:      createdAt,
					AuthorID:        "author-id",
					Covers:          []string{imageURL},
					CoversOrigin:    []string{imageURL},
					DiggCount:       48,
					ShareCount:      1,
					PlayCount:       611,
					CommentCount:    0,
					LocationCreated: "VN",
				},
				Author: embedAuthor{
					SecUID:       "author-secret",
					UserID:       "author-id",
					UniqueID:     "dongmaulacviet1",
					Nickname:     "Dòng Máu Lạc Việt",
					Signature:    "profile signature",
					CoversLarger: []string{"https://p16-common-sign.tiktokcdn.com/avatar.jpeg"},
				},
				AuthorStats: embedAuthorStatistics{
					FollowingCount: 495,
					FollowerCount:  85700,
					HeartCount:     2300000,
					VideoCount:     516,
					DiggCount:      6145,
				},
				Music: embedMusic{
					MusicID:    "7140488448951700251",
					MusicName:  "Giang Hải Không Độ Nàng Remix",
					AuthorName: "Theanh28 Music",
					PlayURL:    []string{audioURL},
				},
				ImagePostInfo: embedImagePostInfo{DisplayImages: []embedImage{{
					Height: 2560, Width: 1920, URLList: []string{imageURL, backupURL},
				}}},
			},
		},
	}
	stateBody, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	embedBody := `<script id="__FRONTITY_CONNECT_STATE__" type="application/json">` + string(stateBody) + `</script>`

	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	inputURL := "https://www.tiktok.com/@dongmaulacviet1/photo/" + storyID
	requestCount := 0
	client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		if request.Header.Get("Referer") != inputURL {
			t.Fatalf("Referer = %q, want %q", request.Header.Get("Referer"), inputURL)
		}
		switch request.URL.Path {
		case "/player/api/v1/items":
			body := `{"status_code":0,"status_msg":"","items":[]}`
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: request}, nil
		case "/embed/v2/" + storyID:
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(embedBody)), ContentLength: int64(len(embedBody)), Request: request}, nil
		default:
			return nil, errors.New("unexpected request: " + request.URL.String())
		}
	})

	result, err := client.Resolve(context.Background(), inputURL)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
	if !result.IsStory || len(result.Sources) != 1 || result.Sources[0] != "embed/v2" {
		t.Fatalf("unexpected Story source metadata: %+v", result)
	}
	if result.Post.ID != storyID || result.Post.Author.UniqueID != "dongmaulacviet1" || result.Post.CreatedAt == nil {
		t.Fatalf("unexpected Story post: %+v", result.Post)
	}
	if result.Post.Statistics.PlayCount != 611 || result.Post.Statistics.LikeCount != 48 || result.Post.Author.Statistics.FollowerCount != 85700 {
		t.Fatalf("unexpected Story statistics: %+v, author: %+v", result.Post.Statistics, result.Post.Author.Statistics)
	}
	if len(result.Post.Music.PlaybackURLs) != 1 || result.Post.Music.PlaybackURLs[0] != audioURL {
		t.Fatalf("unexpected Story music: %+v", result.Post.Music)
	}
	if len(result.Images) != 1 || len(result.Images[0].Media) != 1 {
		t.Fatalf("unexpected Story images: %+v", result.Images)
	}
	image := result.Images[0].Media[0]
	if image.Width != 1920 || image.Height != 2560 || image.URL != imageURL || len(image.BackupURLs) != 1 || image.BackupURLs[0] != backupURL || image.ExpiresAt == nil {
		t.Fatalf("unexpected Story image variant: %+v", image)
	}
}

func TestPhotoResolverHelpers(t *testing.T) {
	t.Parallel()
	parsed, err := url.Parse("https://www.tiktok.com/@creator/photo/123456789?lang=en")
	if err != nil || photoIDFromURL(parsed) != "123456789" {
		t.Fatalf("photoIDFromURL() = %q, %v", photoIDFromURL(parsed), err)
	}
	invalid, _ := url.Parse("https://www.tiktok.com/@creator/photo/not-a-number")
	if photoIDFromURL(invalid) != "" {
		t.Fatal("invalid Photo Post ID was accepted")
	}
	if imageFormat("https://v16.tiktokcdn.com/image") != "jpeg" {
		t.Fatal("imageFormat did not fall back to jpeg")
	}
}
