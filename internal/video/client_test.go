package video

import (
	"net/http"
	"strings"
	"testing"
)

func TestParseTikTokURL(t *testing.T) {
	t.Parallel()

	parsed, err := parseTikTokURL(" https://m.tiktok.com/@example/video/123 ")
	if err != nil {
		t.Fatalf("parse valid URL: %v", err)
	}
	if parsed.Hostname() != "m.tiktok.com" {
		t.Fatalf("hostname = %q", parsed.Hostname())
	}

	invalid := []string{
		"http://www.tiktok.com/@example/video/123",
		"https://tiktok.com.example.org/@example/video/123",
		"https://user:password@www.tiktok.com/@example/video/123",
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

func TestAllowedTikTokHosts(t *testing.T) {
	t.Parallel()

	for _, host := range []string{
		"tiktok.com",
		"v16.tiktokcdn.com",
		"api.tiktokv.com.",
		"media.ibytedtos.com",
		"video.byteoversea.com",
	} {
		if !isAllowedTikTokHost(host) {
			t.Errorf("expected host %q to be allowed", host)
		}
	}
	for _, host := range []string{
		"example.org",
		"tiktokcdn.com.example.org",
		"eviltiktok.com",
	} {
		if isAllowedTikTokHost(host) {
			t.Errorf("expected host %q to be rejected", host)
		}
	}
	if isTikTokPageHost("v16.tiktokcdn.com") {
		t.Fatal("a CDN host must not be accepted as a TikTok page host")
	}
}

func TestClientHeadersAndRedirectPolicy(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientOptions{Cookie: "COOKIE: session=abc", UserAgent: " test-agent "})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	request, _ := http.NewRequest(http.MethodGet, "https://www.tiktok.com/@example/video/123", nil)
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

	allowed, _ := http.NewRequest(http.MethodGet, "https://v16.tiktokcdn.com/video", nil)
	if err := client.httpClient.CheckRedirect(allowed, nil); err != nil {
		t.Fatalf("allow TikTok CDN redirect: %v", err)
	}
	disallowed, _ := http.NewRequest(http.MethodGet, "https://example.org/video", nil)
	if err := client.httpClient.CheckRedirect(disallowed, nil); err == nil {
		t.Fatal("external redirect was unexpectedly allowed")
	}
	if err := client.httpClient.CheckRedirect(allowed, make([]*http.Request, 10)); err == nil {
		t.Fatal("redirect limit was not enforced")
	}
}

func TestReadLimited(t *testing.T) {
	t.Parallel()

	body, err := readLimited(strings.NewReader("abc"), 3)
	if err != nil || string(body) != "abc" {
		t.Fatalf("readLimited() = %q, %v", body, err)
	}
	if _, err := readLimited(strings.NewReader("abcd"), 3); err == nil {
		t.Fatal("readLimited() accepted an oversized response")
	}
}
