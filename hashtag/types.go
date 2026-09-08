// Package hashtag resolves TikTok hashtag pages and the public video links
// exposed by TikTok's tag embed.
package hashtag

import "time"

// ClientOptions configures the HTTP session used for hashtag and video requests.
type ClientOptions struct {
	Cookie    string
	UserAgent string
	Headers   map[string]string
}

// Result contains hashtag metadata and canonical links for the public videos
// exposed by TikTok's tag embed.
type Result struct {
	InputURL  string    `json:"input_url"`
	FinalURL  string    `json:"final_url"`
	Sources   []string  `json:"sources"`
	FetchedAt time.Time `json:"fetched_at"`
	Hashtag   Hashtag   `json:"hashtag"`
	Listing   Listing   `json:"listing"`
	VideoURLs []string  `json:"video_urls"`
	Videos    []Video   `json:"videos"`
}

// Hashtag contains public metadata for one TikTok hashtag.
type Hashtag struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CoverURL    string `json:"cover_url,omitempty"`
	ViewCount   int64  `json:"view_count,omitempty"`
	VideoCount  int64  `json:"video_count,omitempty"`
}

// Listing explains the scope of the tag embed video list.
type Listing struct {
	ReturnedCount        int   `json:"returned_count"`
	TotalCount           int64 `json:"total_count,omitempty"`
	SkippedNonVideoCount int   `json:"skipped_non_video_count,omitempty"`
	EmbedOnly            bool  `json:"embed_only"`
}

// Video is one public video advertised by the tag embed.
type Video struct {
	ID             string   `json:"id"`
	URL            string   `json:"url"`
	Description    string   `json:"description"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	Quality        string   `json:"quality,omitempty"`
	PlayCount      int64    `json:"play_count"`
	CoverURLs      []string `json:"cover_urls,omitempty"`
	AuthorUniqueID string   `json:"author_unique_id"`
}
