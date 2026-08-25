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
