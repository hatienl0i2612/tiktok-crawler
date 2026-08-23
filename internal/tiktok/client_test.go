package tiktok

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseURL(t *testing.T) {
	t.Parallel()
	parsed, err := ParseURL(" https://m.tiktok.com/@example/video/123 ")
	if err != nil || parsed.Hostname() != "m.tiktok.com" {
		t.Fatalf("ParseURL() = %v, %v", parsed, err)
	}
	for _, rawURL := range []string{
		"http://www.tiktok.com/@example/video/123",
		"https://tiktok.com.example.org/@example/video/123",
		"https://user:password@www.tiktok.com/@example/video/123",
		"not a URL",
	} {
		if _, err := ParseURL(rawURL); err == nil {
			t.Errorf("ParseURL(%q) unexpectedly succeeded", rawURL)
		}
	}
}

func TestAllowedHosts(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"tiktok.com", "v16.tiktokcdn.com", "api.tiktokv.com.", "media.ibytedtos.com", "video.byteoversea.com"} {
		if !IsAllowedHost(host) {
			t.Errorf("expected host %q to be allowed", host)
		}
	}
	for _, host := range []string{"example.org", "tiktokcdn.com.example.org", "eviltiktok.com"} {
		if IsAllowedHost(host) {
			t.Errorf("expected host %q to be rejected", host)
		}
	}
	if IsPageHost("v16.tiktokcdn.com") {
		t.Fatal("a CDN host must not be accepted as a TikTok page host")
	}
}

func TestSessionHeadersCookieAndRedirectPolicy(t *testing.T) {
	t.Parallel()
	session, err := NewSession(SessionOptions{Cookie: " Cookie: session=abc; msToken=token ", UserAgent: " test-agent "})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	request, _ := http.NewRequest(http.MethodGet, "https://www.tiktok.com/", nil)
	session.SetHeaders(request, "application/json", "https://www.tiktok.com/referer")
	if request.Header.Get("User-Agent") != "test-agent" || request.Header.Get("Cookie") != "session=abc; msToken=token" {
		t.Fatalf("unexpected headers: %v", request.Header)
	}
	if got := session.CookieValue("msToken"); got != "token" {
		t.Fatalf("CookieValue() = %q", got)
	}
	allowed, _ := http.NewRequest(http.MethodGet, "https://v16.tiktokcdn.com/video", nil)
	if err := session.HTTPClient().CheckRedirect(allowed, nil); err != nil {
		t.Fatalf("allow TikTok redirect: %v", err)
	}
	disallowed, _ := http.NewRequest(http.MethodGet, "https://example.org/video", nil)
	if err := session.HTTPClient().CheckRedirect(disallowed, nil); err == nil {
		t.Fatal("external redirect was unexpectedly allowed")
	}
	if err := session.HTTPClient().CheckRedirect(allowed, make([]*http.Request, 10)); err == nil {
		t.Fatal("redirect limit was not enforced")
	}
}

func TestSessionCustomHeadersOverrideDefaults(t *testing.T) {
	t.Parallel()
	session, err := NewSession(SessionOptions{
		Cookie:    "session=abc",
		UserAgent: "default-agent",
		Headers: map[string]string{
			"user-agent":      "custom-agent",
			"Accept-Language": "fr-FR,fr;q=0.9",
			"X-Custom":        "yes",
			"":                "ignored",
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := session.UserAgent(); got != "custom-agent" {
		t.Fatalf("UserAgent() = %q, want the custom header value", got)
	}
	request, _ := http.NewRequest(http.MethodGet, "https://www.tiktok.com/", nil)
	session.SetHeaders(request, "application/json", "https://www.tiktok.com/referer")
	if got := request.Header.Get("User-Agent"); got != "custom-agent" {
		t.Fatalf("User-Agent header = %q", got)
	}
	if got := request.Header.Get("X-Custom"); got != "yes" {
		t.Fatalf("X-Custom header = %q", got)
	}
	if got := request.Header.Get("Accept-Language"); got != "fr-FR,fr;q=0.9" {
		t.Fatalf("Accept-Language header = %q, want the custom override", got)
	}
	if got := request.Header.Get("Referer"); got != "https://www.tiktok.com/referer" {
		t.Fatalf("Referer header = %q, want the per-request value", got)
	}
	if got := request.Header.Get("Cookie"); got != "session=abc" {
		t.Fatalf("Cookie header = %q, want the configured cookie", got)
	}
}

func TestSharedHelpers(t *testing.T) {
	t.Parallel()
	body, err := ReadLimited(strings.NewReader("abc"), 3)
	if err != nil || string(body) != "abc" {
		t.Fatalf("ReadLimited() = %q, %v", body, err)
	}
	if _, err := ReadLimited(strings.NewReader("abcd"), 3); err == nil {
		t.Fatal("ReadLimited() accepted an oversized response")
	}
	if got := NormalizeCodec("HEVC"); got != "h265" {
		t.Fatalf("NormalizeCodec() = %q", got)
	}
	expires := ExpiryFromURL("https://v16.tiktokcdn.com/video?expire=1787850190")
	if expires == nil || !expires.Equal(time.Unix(1_787_850_190, 0).UTC()) {
		t.Fatalf("ExpiryFromURL() = %v", expires)
	}
}
