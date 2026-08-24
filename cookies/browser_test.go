package cookies

import (
	"net/http"
	"strings"
	"testing"
)

func TestBrowserLoader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		{"chrome", true},
		{"Chrome", true},
		{"  FIREFOX  ", true},
		{"edge-dev", true},
		{"Edge Dev", true},
		{"opera-gx", true},
		{"safari", true},
		{"librewolf", true},
		{"", false},
		{"netscape", false},
	}
	for _, test := range tests {
		got := loaderFor(test.name)
		if (got != nil) != test.want {
			t.Errorf("browserLoader(%q) != nil = %v, want %v", test.name, got != nil, test.want)
		}
	}
}

func TestFormatCookieHeader(t *testing.T) {
	t.Parallel()
	cookies := []*http.Cookie{
		{Name: "ttwid", Value: "1%7Cv1"},
		{Name: "sessionid", Value: "abc123"},
		{Name: "expired_only", Value: ""},
		nil,
		{Name: "", Value: "ignored"},
	}
	got := FormatCookieHeader(cookies)
	if want := "ttwid=1%7Cv1; sessionid=abc123; expired_only="; got != want {
		t.Fatalf("FormatCookieHeader() = %q, want %q", got, want)
	}
}

func TestLoadTikTokCookieHeaderRejectsUnsupportedBrowser(t *testing.T) {
	t.Parallel()
	_, err := LoadTikTokCookieHeader("internet-explorer")
	if err == nil {
		t.Fatal("LoadTikTokCookieHeader() succeeded for an unsupported browser")
	}
	if !strings.Contains(err.Error(), "supported browsers") {
		t.Fatalf("error %q does not list the supported browsers", err)
	}
}
