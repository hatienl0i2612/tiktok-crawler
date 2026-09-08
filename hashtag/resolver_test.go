package hashtag

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

func TestResolveHashtag(t *testing.T) {
	t.Parallel()

	var state embedState
	state.Source.Data = map[string]embedPage{
		"/embed/tag/chatgpt": {
			PageName: "tag", PlaylistType: "hashtag", PlaylistEndpoint: "chatgpt",
			EmbedInfo: embedInfo{
				ID: "7172185891421552645", ViewCount: 102169282526, VideoCount: 7699749,
				StatusCode: 0, Code: 200,
			},
			VideoList: []embedVideo{
				{
					ID: "7614425026158136589", Description: "Speaking meow #chatgpt", Width: 720, Height: 1280,
					Ratio: "720p", PlayAddress: "https://v16.tiktokv.com/video.mp4", PlayCount: 47900000,
					CoverURL: "https://p16.tiktokcdn.com/cover.jpeg", AuthorUniqueID: "notnadialy",
				},
				{
					ID: "7319904028123548958", Description: "Photo mode", CoverURL: "https://p16.tiktokcdn.com/photomode.jpeg",
					PlayCount: 48300000, AuthorUniqueID: "photoauthor",
				},
				{
					ID: "private", PlayAddress: "https://v16.tiktokv.com/private.mp4", PrivateItem: true, AuthorUniqueID: "privateauthor",
				},
			},
		},
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	body := `<script id="__FRONTITY_CONNECT_STATE__">` + string(stateJSON) + `</script>`

	client, err := NewClient(ClientOptions{Cookie: "ttwid=test"})
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	client.session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.URL.Path != "/embed/tag/chatgpt" {
			return nil, errors.New("unexpected request: " + request.URL.String())
		}
		if request.Header.Get("Referer") != "https://www.tiktok.com/tag/chatgpt" || request.Header.Get("Cookie") != "ttwid=test" {
			t.Errorf("unexpected headers: %v", request.Header)
		}
		return &http.Response{
			StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(body)), Request: request,
		}, nil
	})

	result, err := client.Resolve(context.Background(), "https://www.tiktok.com/tag/chatgpt")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if requests != 1 || len(result.Sources) != 1 || result.Sources[0] != "tag embed" {
		t.Fatalf("unexpected requests/sources: requests=%d sources=%v", requests, result.Sources)
	}
	if result.Hashtag.ID != "7172185891421552645" || result.Hashtag.Name != "chatgpt" || result.Hashtag.ViewCount != 102169282526 || result.Hashtag.VideoCount != 7699749 {
		t.Fatalf("unexpected hashtag: %+v", result.Hashtag)
	}
	if result.Listing.ReturnedCount != 1 || result.Listing.TotalCount != 7699749 || result.Listing.SkippedNonVideoCount != 1 || !result.Listing.EmbedOnly {
		t.Fatalf("unexpected listing: %+v", result.Listing)
	}
	wantURL := "https://www.tiktok.com/@notnadialy/video/7614425026158136589"
	if len(result.VideoURLs) != 1 || result.VideoURLs[0] != wantURL || len(result.Videos) != 1 || result.Videos[0].URL != wantURL {
		t.Fatalf("unexpected videos: URLs=%v videos=%+v", result.VideoURLs, result.Videos)
	}
	if result.Videos[0].PlayCount != 47900000 || len(result.Videos[0].CoverURLs) != 1 {
		t.Fatalf("unexpected video metadata: %+v", result.Videos[0])
	}
}

func TestResolveHashtagRejectsUnrelatedEmbed(t *testing.T) {
	t.Parallel()
	state := `{"source":{"data":{"other":{"pageName":"tag","playlistType":"hashtag","playlistEndpoint":"other"}}}}`
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	client.session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `<script id="__FRONTITY_CONNECT_STATE__">` + state + `</script>`
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	if _, err := client.Resolve(context.Background(), "https://www.tiktok.com/tag/chatgpt"); err == nil {
		t.Fatal("Resolve() accepted embed data for a different hashtag")
	}
}

func TestTagNameFromURL(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		path string
		want string
	}{
		{path: "/tag/chatgpt", want: "chatgpt"},
		{path: "/tag/ChatGPT/", want: "ChatGPT"},
		{path: "/tag/tr%C3%AD-tu%E1%BB%87", want: "trí-tuệ"},
		{path: "/tag/foo%2Fbar", want: ""},
		{path: "/tag/foo%5Cbar", want: ""},
		{path: "/tag/", want: ""},
		{path: "/tag/chatgpt/video", want: ""},
		{path: "/explore", want: ""},
	} {
		parsed, err := url.Parse("https://www.tiktok.com" + test.path)
		if err != nil {
			t.Fatal(err)
		}
		if got := tagNameFromURL(parsed); got != test.want {
			t.Errorf("tagNameFromURL(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}
