// Package photo resolves TikTok Photo Posts and their downloadable images.
package photo

import (
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

// ClientOptions configures the HTTP session used to crawl TikTok Photo Posts.
// It is shared with the regular video resolver because both content types use
// TikTok's web player endpoint.
type ClientOptions = video.ClientOptions

// Result contains normalized Photo Post metadata and every image source.
type Result struct {
	InputURL  string      `json:"input_url"`
	FinalURL  string      `json:"final_url"`
	Sources   []string    `json:"sources"`
	FetchedAt time.Time   `json:"fetched_at"`
	Warnings  []string    `json:"warnings,omitempty"`
	Post      video.Video `json:"post"`
	Images    []Image     `json:"images"`
}

// Image is one ordered image in a TikTok Photo Post.
type Image struct {
	Index int             `json:"index"`
	Media []media.Variant `json:"media"`
}
