// Package shortdrama resolves and downloads TikTok Short Drama episodes.
package shortdrama

import (
	"time"

	"tiktok-crawler/internal/media"
	"tiktok-crawler/internal/video"
)

const DefaultUserAgent = video.DefaultUserAgent

// ClientOptions configures the HTTP session used for Short Drama requests.
type ClientOptions struct {
	Cookie    string
	UserAgent string
}

// Result contains series, episode, and downloadable media metadata.
type Result struct {
	InputURL   string          `json:"input_url"`
	FinalURL   string          `json:"final_url"`
	Sources    []string        `json:"sources"`
	FetchedAt  time.Time       `json:"fetched_at"`
	Warnings   []string        `json:"warnings,omitempty"`
	ShortDrama *ShortDrama     `json:"short_drama"`
	Video      video.Video     `json:"video"`
	Media      []media.Variant `json:"media"`
}

// ShortDrama describes the series and selected episode.
type ShortDrama struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Episode       int    `json:"episode"`
	EpisodeCount  int    `json:"episode_count"`
	IsLimitedFree bool   `json:"is_limited_free"`
}
