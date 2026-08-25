package tiktokcrawler

import (
	"context"
	"testing"
	"time"
)

func TestDetectKind(t *testing.T) {
	tests := []struct {
		url  string
		want Kind
		fail bool
	}{
		{"https://www.tiktok.com/@example/video/1234567890123456789", KindVideo, false},
		{"https://www.tiktok.com/@example/video/1234567890123456789?lang=en", KindVideo, false},
		{"https://www.tiktok.com/@example/photo/1234567890123456789", KindPhoto, false},
		{"https://www.tiktok.com/@example/live", KindLive, false},
		{"https://www.tiktok.com/shortdrama/episode/7665073849083368469/1", KindShortDrama, false},
		{"https://example.com/@example/video/123", "", true},
		{"https://www.tiktok.com/@example", "", true},
	}
	for _, test := range tests {
		kind, err := DetectKind(test.url)
		if test.fail {
			if err == nil {
				t.Errorf("DetectKind(%q) succeeded with %q", test.url, kind)
			}
			continue
		}
		if err != nil || kind != test.want {
			t.Errorf("DetectKind(%q) = %q, %v; want %q", test.url, kind, err, test.want)
		}
	}
}

func TestNewClientExposesSubClients(t *testing.T) {
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.Video() == nil || client.Photo() == nil || client.Livestream() == nil || client.ShortDrama() == nil {
		t.Fatal("NewClient must expose all sub-clients")
	}
}

func TestClientResolveDetectsKindBeforeNetwork(t *testing.T) {
	client, err := NewClient(ClientOptions{Cookie: "ttwid=nope", Headers: map[string]string{"X-Test": "1"}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// These URLs fail fast at DetectKind before any network request is made.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := client.Resolve(ctx, "https://example.com/not-tiktok"); err == nil {
		t.Fatal("Resolve() succeeded for a non-TikTok URL")
	}
	if _, err := client.Resolve(ctx, "https://www.tiktok.com/@example"); err == nil {
		t.Fatal("Resolve() succeeded for an unsupported TikTok URL")
	}
}
