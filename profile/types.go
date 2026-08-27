// Package profile resolves public TikTok creator profiles and the recent video
// links exposed by TikTok's creator embed.
package profile

import "time"

// ClientOptions configures the HTTP session used for profile and video requests.
type ClientOptions struct {
	Cookie    string
	UserAgent string
	Headers   map[string]string
}

// Result contains public profile metadata and canonical links for the recent
// videos exposed by TikTok's creator embed.
type Result struct {
	InputURL  string    `json:"input_url"`
	FinalURL  string    `json:"final_url"`
	Sources   []string  `json:"sources"`
	FetchedAt time.Time `json:"fetched_at"`
	Warnings  []string  `json:"warnings,omitempty"`
	User      User      `json:"user"`
	Listing   Listing   `json:"listing"`
	VideoURLs []string  `json:"video_urls"`
	Videos    []Video   `json:"videos"`
}

// User contains public TikTok creator metadata.
type User struct {
	ID             string         `json:"id"`
	SecUID         string         `json:"sec_uid,omitempty"`
	UniqueID       string         `json:"unique_id"`
	Nickname       string         `json:"nickname"`
	Signature      string         `json:"signature,omitempty"`
	Verified       bool           `json:"verified"`
	PrivateAccount bool           `json:"private_account"`
	AvatarURLs     []string       `json:"avatar_urls,omitempty"`
	BioLink        string         `json:"bio_link,omitempty"`
	Statistics     UserStatistics `json:"statistics"`
}

// UserStatistics contains public creator counters.
type UserStatistics struct {
	FollowingCount int64 `json:"following_count"`
	FollowerCount  int64 `json:"follower_count"`
	HeartCount     int64 `json:"heart_count"`
	VideoCount     int64 `json:"video_count"`
	DiggCount      int64 `json:"digg_count"`
	FriendCount    int64 `json:"friend_count"`
}

// Listing explains the scope of the creator embed video list.
type Listing struct {
	ReturnedCount int   `json:"returned_count"`
	TotalCount    int64 `json:"total_count,omitempty"`
	LatestOnly    bool  `json:"latest_only"`
}

// Video is one recent public video advertised by the creator embed.
type Video struct {
	ID             string   `json:"id"`
	URL            string   `json:"url"`
	Description    string   `json:"description"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	Quality        string   `json:"quality,omitempty"`
	PlayCount      int64    `json:"play_count"`
	Private        bool     `json:"private"`
	CoverURLs      []string `json:"cover_urls,omitempty"`
	AuthorUniqueID string   `json:"author_unique_id"`
}
