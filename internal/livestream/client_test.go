package livestream

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

func TestParseTikTokURL(t *testing.T) {
	t.Parallel()

	valid, err := parseTikTokURL("  https://www.tiktok.com/@example/live?lang=en  ")
	if err != nil {
		t.Fatalf("parse valid URL: %v", err)
	}
	if valid.Hostname() != "www.tiktok.com" {
		t.Fatalf("unexpected hostname: %q", valid.Hostname())
	}

	invalid := []string{
		"http://www.tiktok.com/@example/live",
		"https://tiktok.com.example.org/@example/live",
		"https://user:password@www.tiktok.com/@example/live",
		"not a URL",
	}
	for _, rawURL := range invalid {
		rawURL := rawURL
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()
			if _, err := parseTikTokURL(rawURL); err == nil {
				t.Fatalf("parseTikTokURL(%q) unexpectedly succeeded", rawURL)
			}
		})
	}
}

func TestUsernameFromLiveURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "standard", path: "/@Example/live", want: "Example"},
		{name: "escaped", path: "/@name%20with%20spaces/LIVE", want: "name with spaces"},
		{name: "missing username", path: "/live", want: ""},
		{name: "not live", path: "/@example/video/123", want: ""},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := url.Parse("https://www.tiktok.com" + test.path)
			if err != nil {
				t.Fatalf("parse fixture URL: %v", err)
			}
			if got := usernameFromLiveURL(parsed); got != test.want {
				t.Fatalf("usernameFromLiveURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseSIGIState(t *testing.T) {
	t.Parallel()

	html := []byte(`<html><script type="application/json" id="SIGI_STATE">
		{"LiveRoom":{"liveRoomUserInfo":{"user":{"id":"42","uniqueId":"example"},"liveRoom":{"title":"Test room"}}}}
	</script></html>`)
	info, err := parseSIGIState(html)
	if err != nil {
		t.Fatalf("parse SIGI_STATE: %v", err)
	}
	if info.User.ID != "42" || info.User.UniqueID != "example" || info.LiveRoom.Title != "Test room" {
		t.Fatalf("unexpected room info: %+v", info)
	}

	for name, body := range map[string][]byte{
		"missing": []byte(`<html></html>`),
		"invalid": []byte(`<script id="SIGI_STATE">{</script>`),
		"no user": []byte(`<script id="SIGI_STATE">{"LiveRoom":{"liveRoomUserInfo":{}}}</script>`),
	} {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseSIGIState(body); err == nil {
				t.Fatal("parseSIGIState() unexpectedly succeeded")
			}
		})
	}
}

func TestClientHeadersAndRedirectPolicy(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientOptions{Cookie: " Cookie: session=abc ", UserAgent: " test-agent "})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	request, _ := http.NewRequest(http.MethodGet, "https://www.tiktok.com/@example/live", nil)
	client.setHeaders(request, "application/json", "https://www.tiktok.com/")
	if got := request.Header.Get("User-Agent"); got != "test-agent" {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := request.Header.Get("Cookie"); got != "session=abc" {
		t.Fatalf("Cookie = %q", got)
	}
	if got := request.Header.Get("Referer"); got != "https://www.tiktok.com/" {
		t.Fatalf("Referer = %q", got)
	}

	allowed, _ := http.NewRequest(http.MethodGet, "https://m.tiktok.com/redirected", nil)
	if err := client.httpClient.CheckRedirect(allowed, nil); err != nil {
		t.Fatalf("allow TikTok redirect: %v", err)
	}
	disallowed, _ := http.NewRequest(http.MethodGet, "https://example.org/redirected", nil)
	if err := client.httpClient.CheckRedirect(disallowed, nil); err == nil {
		t.Fatal("external redirect was unexpectedly allowed")
	}
	previous := make([]*http.Request, 10)
	if err := client.httpClient.CheckRedirect(allowed, previous); err == nil {
		t.Fatal("redirect limit was not enforced")
	}
}

func TestResolveLiveRoom(t *testing.T) {
	t.Parallel()

	streamData := mustStreamData(t, "h264", "origin", "https://pull-hls.tiktokcdn.com/live.m3u8?expire=1787850190")
	apiBody, err := json.Marshal(apiResponse{Data: roomInfo{
		User:  wireUser{ID: "42", UniqueID: "Example", Nickname: "Example User", RoomID: "99", Status: 2},
		Stats: wireStats{FollowerCount: 100},
		LiveRoom: wireLiveRoom{
			Title:      "Fixture room",
			Status:     2,
			StartTime:  1_700_000_000,
			StreamID:   "stream-1",
			StreamData: streamContainer{PullData: pullData{StreamData: streamData}},
			Stats:      wireLiveStats{EnterCount: 25, UserCount: 10},
		},
	}})
	if err != nil {
		t.Fatalf("marshal API fixture: %v", err)
	}

	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body []byte
		switch request.URL.Path {
		case "/@example/live":
			body = []byte(`<script id="SIGI_STATE">{"LiveRoom":{"liveRoomUserInfo":{"user":{"id":"page","uniqueId":"example"}}}}</script>`)
		case "/api-live/user/room":
			if got := request.URL.Query().Get("uniqueId"); got != "example" {
				t.Fatalf("uniqueId query = %q", got)
			}
			body = apiBody
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

	result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@example/live")
	if err != nil {
		t.Fatalf("resolve live room: %v", err)
	}
	if result.Source != "api-live/user/room" || result.User.ID != "42" || !result.Live.IsLive {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Live.StartTime == nil || !result.Live.StartTime.Equal(time.Unix(1_700_000_000, 0).UTC()) {
		t.Fatalf("unexpected start time: %v", result.Live.StartTime)
	}
	if len(result.Streams) != 1 || result.Streams[0].Protocol != "hls" {
		t.Fatalf("unexpected streams: %+v", result.Streams)
	}
}

func TestResolveError(t *testing.T) {
	t.Parallel()

	err := resolveError(errors.New("page failed"), nil, errors.New("API failed"))
	if !strings.Contains(err.Error(), "live page: page failed") || !strings.Contains(err.Error(), "room API: API failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
