package shortdrama

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestShortDramaEpisodeFromURL(t *testing.T) {
	parsed, err := url.Parse("https://www.tiktok.com/shortdrama/episode/7665073849083368469/12")
	if err != nil {
		t.Fatal(err)
	}
	dramaID, episode, ok := episodeFromURL(parsed)
	if !ok || dramaID != "7665073849083368469" || episode != 12 {
		t.Fatalf("episodeFromURL() = %q, %d, %t", dramaID, episode, ok)
	}
}

func TestBuildShortDramaResult(t *testing.T) {
	const fixture = `{
		"id":"7665074054730091796",
		"desc":"Episode one",
		"author":{"id":"10","uniqueId":"example","nickname":"Example"},
		"stats":{"playCount":100,"diggCount":20,"commentCount":3,"shareCount":4},
		"dramaInfo":{"dramaID":"7665073849083368469","dramaName":"Example Drama","numVideos":80,"DramaVideoData":{"EpisodeNumber":1,"TranslatedDescription":"Translated episode"}},
		"video":{"duration":178,"width":1080,"height":1920,"playAddr":"https://v16.tiktokcdn.com/episode.mp4?expire=1787300000"}
	}`
	var item shortDramaItem
	if err := json.Unmarshal([]byte(fixture), &item); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	result := buildShortDramaResult("input", "final", "7665073849083368469", 1, item)
	if result.ShortDrama == nil || result.ShortDrama.Name != "Example Drama" || result.ShortDrama.Episode != 1 {
		t.Fatalf("unexpected Short Drama metadata: %#v", result.ShortDrama)
	}
	if result.Video.Description != "Translated episode" || result.Video.Author.UniqueID != "example" {
		t.Fatalf("unexpected video metadata: %#v", result.Video)
	}
	if len(result.Media) != 1 || result.Media[0].URL != "https://v16.tiktokcdn.com/episode.mp4?expire=1787300000" {
		t.Fatalf("unexpected media: %#v", result.Media)
	}
}

func TestNewClientUsesShortDramaDefaults(t *testing.T) {
	client, err := NewClient(ClientOptions{Cookie: "ttwid=abc; msToken=token-value; sessionid=xyz"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if got := client.session.CookieValue("msToken"); got != "token-value" {
		t.Fatalf("CookieValue() = %q", got)
	}
	if client.session.UserAgent() != DefaultUserAgent {
		t.Fatalf("UserAgent() = %q", client.session.UserAgent())
	}
}
