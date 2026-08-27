package profile

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

func TestResolveProfile(t *testing.T) {
	t.Parallel()

	var hydration hydrationState
	hydration.DefaultScope.UserDetail.UserInfo = hydrationUserInfo{
		User: hydrationUser{
			ID: "7191834833537254401", SecUID: "profile-sec-uid", UniqueID: "forever0404_",
			Nickname: "Một Ngày Buồn", Signature: "profile bio", AvatarThumb: "https://p16-common-sign.tiktokcdn.com/thumb.jpeg",
		},
		StatsV2: hydrationStatistics{
			FollowingCount: 90, FollowerCount: 430866, HeartCount: 74623291, VideoCount: 8429, FriendCount: 85,
		},
	}
	hydrationJSON, err := json.Marshal(hydration)
	if err != nil {
		t.Fatal(err)
	}
	pageBody := []byte(`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">` + string(hydrationJSON) + `</script>`)

	var embed embedState
	embed.Source.Data = map[string]embedPage{
		"/embed/@forever0404_": {
			PageName: "creator", PlaylistType: "creator", PlaylistID: "forever0404_",
			UserInfo: embedUser{ID: "7191834833537254401", UniqueID: "forever0404_", Nickname: "Embed nickname", Code: 200},
			VideoList: []embedVideo{
				{ID: "7677951094584003860", Description: "newest", Width: 576, Height: 1024, Ratio: "540p", CoverURL: "https://p16-common-sign.tiktokcdn.com/cover.jpeg", PlayCount: 34300, AuthorUniqueID: "forever0404_"},
				{ID: "7677940967663488277", Description: "second", Width: 576, Height: 1024, Ratio: "540p", PlayCount: 13700, AuthorUniqueID: "forever0404_"},
				{ID: "private", PrivateItem: true, AuthorUniqueID: "forever0404_"},
			},
		},
	}
	embedJSON, err := json.Marshal(embed)
	if err != nil {
		t.Fatal(err)
	}
	embedBody := []byte(`<script id="__FRONTITY_CONNECT_STATE__">` + string(embedJSON) + `</script>`)

	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pageRequests, embedRequests := 0, 0
	client.session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body []byte
		switch request.URL.Path {
		case "/@forever0404_":
			pageRequests++
			body = pageBody
		case "/embed/@forever0404_":
			embedRequests++
			body = embedBody
		default:
			return nil, errors.New("unexpected request: " + request.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(string(body))), ContentLength: int64(len(body)), Request: request,
		}, nil
	})

	result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@forever0404_")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if pageRequests != 1 || embedRequests != 1 || len(result.Sources) != 2 || len(result.Warnings) != 0 {
		t.Fatalf("unexpected requests/result sources: page=%d embed=%d result=%+v", pageRequests, embedRequests, result)
	}
	if result.User.UniqueID != "forever0404_" || result.User.SecUID != "profile-sec-uid" || result.User.Statistics.VideoCount != 8429 || result.User.Statistics.FollowerCount != 430866 {
		t.Fatalf("unexpected user: %+v", result.User)
	}
	if result.Listing.ReturnedCount != 2 || result.Listing.TotalCount != 8429 || !result.Listing.LatestOnly || len(result.VideoURLs) != 2 || len(result.Videos) != 2 {
		t.Fatalf("unexpected listing: %+v URLs=%v videos=%+v", result.Listing, result.VideoURLs, result.Videos)
	}
	wantURL := "https://www.tiktok.com/@forever0404_/video/7677951094584003860"
	if result.VideoURLs[0] != wantURL || result.Videos[0].URL != wantURL || result.Videos[0].PlayCount != 34300 {
		t.Fatalf("unexpected first video: %+v", result.Videos[0])
	}
}

func TestResolveProfileUsesEmbedWhenHydrationUnavailable(t *testing.T) {
	t.Parallel()
	var embed embedState
	embed.Source.Data = map[string]embedPage{
		"creator": {PlaylistType: "creator", PlaylistID: "creator", UserInfo: embedUser{UniqueID: "creator", Code: 200}},
	}
	embedJSON, _ := json.Marshal(embed)
	embedBody := `<script id="__FRONTITY_CONNECT_STATE__">` + string(embedJSON) + `</script>`

	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	client.session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `<html></html>`
		if strings.HasPrefix(request.URL.Path, "/embed/") {
			body = embedBody
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@creator")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.UniqueID != "creator" || len(result.Warnings) != 1 || len(result.Sources) != 1 || result.Sources[0] != "creator embed" {
		t.Fatalf("unexpected fallback result: %+v", result)
	}
}

func TestUsernameFromURL(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		path string
		want string
	}{
		{path: "/@forever0404_", want: "forever0404_"},
		{path: "/@creator/", want: "creator"},
		{path: "/@creator/video/123", want: ""},
		{path: "/explore", want: ""},
	} {
		parsed, err := url.Parse("https://www.tiktok.com" + test.path)
		if err != nil {
			t.Fatal(err)
		}
		if got := usernameFromURL(parsed); got != test.want {
			t.Errorf("usernameFromURL(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}
