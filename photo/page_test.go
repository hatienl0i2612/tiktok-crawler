package photo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

const desktopPhotoPage = `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">{
 "__DEFAULT_SCOPE__":{"webapp.video-detail":{"statusCode":0,"itemInfo":{"itemStruct":{
  "id":"7704668607598562581","desc":"Photo page description","createTime":"1700000000","locationCreated":"VN",
  "author":{"id":"author-id","secUid":"secret","uniqueId":"nhatuiminhne","nickname":"Creator","verified":true,"avatarThumb":"https://p16.tiktokcdn.com/avatar.jpeg"},
  "authorStats":{"followerCount":"123"},"stats":{"playCount":"500","diggCount":12,"commentCount":"3","shareCount":2},
  "music":{"id":"music-id","title":"Song","authorName":"Artist","playUrl":"https://v16.tiktokcdn.com/music.mp3"},
  "imagePost":{"images":[
   {"imageWidth":986,"imageHeight":1232,"imageURL":{"urlList":["https://p16.tiktokcdn.com/first.jpeg?x-expires=1794913200","https://p19.tiktokcdn.com/first.jpeg"]}},
   {"imageWidth":1,"imageHeight":1,"imageURL":{"urlList":[]}},
   {"imageWidth":720,"imageHeight":900,"imageURL":{"urlList":["https://p16.tiktokcdn.com/second.webp"]}}
  ],"cover":{"imageURL":{"urlList":["https://p16.tiktokcdn.com/cover.jpeg"]}}}
 }}}}
}</script>`

func TestResolvePhotoWhenPlayerUnavailable(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"forbidden", http.StatusForbidden, "denied"},
		{"invalid JSON", http.StatusOK, "<html>challenge</html>"},
		{"API error", http.StatusOK, `{"status_code":10216,"status_msg":"unavailable"}`},
		{"empty items", http.StatusOK, `{"status_code":0,"items":[]}`},
		{"item without images", http.StatusOK, `{"items":[{"id_str":"7704668607598562581"}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(ClientOptions{Cookie: "session=test"})
			if err != nil {
				t.Fatal(err)
			}
			inputURL := "https://www.tiktok.com/@nhatuiminhne/photo/7704668607598562581?lang=vi"
			pageRequests := 0
			client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.Header.Get("Referer") != inputURL || request.Header.Get("Cookie") != "session=test" {
					t.Fatalf("fallback lost session headers: %v", request.Header)
				}
				status, body := http.StatusOK, desktopPhotoPage
				switch request.URL.Path {
				case "/player/api/v1/items":
					status, body = test.status, test.body
				case "/@nhatuiminhne/video/7704668607598562581":
					pageRequests++
					if request.URL.Query().Get("lang") != "vi" {
						t.Fatal("fallback dropped URL query parameters")
					}
				case "/embed/v2/7704668607598562581":
					if test.name != "item without images" {
						t.Fatal("a valid page fallback must not depend on embed")
					}
					status, body = http.StatusForbidden, "denied"
				default:
					t.Fatalf("unexpected request: %s", request.URL)
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
			})
			result, err := client.Resolve(context.Background(), inputURL)
			if err != nil {
				t.Fatal(err)
			}
			if pageRequests != 1 || result.IsStory || result.InputURL != inputURL || result.FinalURL != inputURL {
				t.Fatalf("pageRequests=%d, result=%+v", pageRequests, result)
			}
			post := result.Post
			if post.ID != "7704668607598562581" || post.Description != "Photo page description" || post.Author.ID != "author-id" || post.Author.UniqueID != "nhatuiminhne" || !post.Author.Verified || post.Author.Statistics.FollowerCount != 123 {
				t.Fatalf("page metadata = %+v", post)
			}
			if post.CreatedAt == nil || post.CreatedAt.Unix() != 1700000000 || post.Statistics.PlayCount != 500 || post.Statistics.LikeCount != 12 || post.Statistics.CommentCount != 3 || post.Statistics.ShareCount != 2 {
				t.Fatalf("page timestamps and counters = %+v", post)
			}
			if post.Music.ID != "music-id" || len(post.Music.PlaybackURLs) != 1 || post.Music.PlaybackURLs[0] != "https://v16.tiktokcdn.com/music.mp3" || len(post.Covers.Default) != 1 {
				t.Fatalf("page music and covers = %+v, %+v", post.Music, post.Covers)
			}
			if len(result.Sources) != 1 || result.Sources[0] != "__UNIVERSAL_DATA_FOR_REHYDRATION__" || len(result.Images) != 2 || result.Images[0].Index != 1 || result.Images[1].Index != 3 {
				t.Fatalf("page sources and image order = %+v", result)
			}
			first := result.Images[0].Media[0]
			if first.Width != 986 || first.Height != 1232 || first.Format != "jpeg" || first.ExpiresAt == nil || first.Watermarked || len(first.BackupURLs) != 1 || first.BackupURLs[0] != "https://p19.tiktokcdn.com/first.jpeg" {
				t.Fatalf("page image = %+v", first)
			}
			if test.name == "empty items" && len(result.Warnings) != 0 || test.name != "empty items" && len(result.Warnings) != 1 {
				t.Fatalf("fallback warnings = %v", result.Warnings)
			}
		})
	}
}

func TestResolvePhotoRejectsUnavailablePageFallback(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"<html>CSR shell</html>",
		`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{</script>`,
		`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{"__DEFAULT_SCOPE__":{}}</script>`,
		strings.Replace(desktopPhotoPage, `"statusCode":0`, `"statusCode":10204`, 1),
		strings.ReplaceAll(desktopPhotoPage, "7704668607598562581", "123456789"),
		strings.ReplaceAll(desktopPhotoPage, `"imageURL"`, `"unavailableURL"`),
	} {
		client, err := NewClient(ClientOptions{})
		if err != nil {
			t.Fatal(err)
		}
		client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
			status, response := http.StatusForbidden, "denied"
			if strings.Contains(request.URL.Path, "/video/") {
				status, response = http.StatusOK, body
			}
			return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response)), Request: request}, nil
		})
		result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@nhatuiminhne/photo/7704668607598562581")
		if result != nil || err == nil || !strings.Contains(err.Error(), "HTTP Forbidden") || !strings.Contains(err.Error(), "page fallback failed") || !strings.Contains(err.Error(), "embed fallback failed") {
			t.Fatalf("result=%+v error=%v", result, err)
		}
	}
}

func TestResolvePhotoCancellationStopsFallback(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"player", "page"} {
		t.Run(stage, func(t *testing.T) {
			client, err := NewClient(ClientOptions{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client.Session().HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if strings.HasPrefix(request.URL.Path, "/embed/") || stage == "player" && strings.Contains(request.URL.Path, "/video/") {
					t.Fatal("fallback continued after cancellation")
				}
				if stage == "player" || strings.Contains(request.URL.Path, "/video/") {
					cancel()
					return nil, context.Canceled
				}
				return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("denied")), Request: request}, nil
			})
			if result, err := client.Resolve(ctx, "https://www.tiktok.com/@nhatuiminhne/photo/7704668607598562581"); result != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}
