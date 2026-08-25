// Package photo resolves TikTok Photo Posts and photo Stories and returns their
// downloadable images.
package photo

import (
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

// ClientOptions configures the HTTP session used to crawl TikTok Photo Posts
// and photo Stories. It shares the regular video resolver's web session.
type ClientOptions = video.ClientOptions

// Result contains normalized Photo Post or photo Story metadata and every image source.
type Result struct {
	InputURL  string    `json:"input_url"`
	FinalURL  string    `json:"final_url"`
	Sources   []string  `json:"sources"`
	FetchedAt time.Time `json:"fetched_at"`
	Warnings  []string  `json:"warnings,omitempty"`
	// IsStory is true when TikTok's player omits the item and the /photo URL
	// resolves through the photo Story embed fallback.
	IsStory bool        `json:"is_story,omitempty"`
	Post    video.Video `json:"post"`
	Images  []Image     `json:"images"`
}

// Image is one ordered image in a TikTok Photo Post or photo Story.
type Image struct {
	Index int             `json:"index"`
	Media []media.Variant `json:"media"`
}
