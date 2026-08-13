package livestream

import "time"

const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"

// ClientOptions configures the HTTP session used to resolve a live room.
type ClientOptions struct {
	Cookie    string
	UserAgent string
}

// Result contains live-room metadata and every playback URL exposed by TikTok.
type Result struct {
	InputURL  string    `json:"input_url"`
	FinalURL  string    `json:"final_url"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
	User      User      `json:"user"`
	Live      Live      `json:"live"`
	Streams   []Stream  `json:"streams"`
}

// User describes the owner of a TikTok live room.
type User struct {
	ID            string `json:"id"`
	UniqueID      string `json:"unique_id"`
	Nickname      string `json:"nickname"`
	RoomID        string `json:"room_id"`
	Status        int    `json:"status"`
	FollowerCount int64  `json:"follower_count"`
}

// Live describes the current state of a TikTok live room.
type Live struct {
	IsLive      bool       `json:"is_live"`
	Status      int        `json:"status"`
	Title       string     `json:"title"`
	StreamID    string     `json:"stream_id"`
	ViewerCount int64      `json:"viewer_count"`
	EnterCount  int64      `json:"enter_count"`
	StartTime   *time.Time `json:"start_time,omitempty"`
}

// Stream is one signed playback URL for a quality, codec, protocol, and CDN line.
type Stream struct {
	Codec      string     `json:"codec"`
	Quality    string     `json:"quality"`
	Line       string     `json:"line"`
	Protocol   string     `json:"protocol"`
	URL        string     `json:"url"`
	Resolution string     `json:"resolution,omitempty"`
	Bitrate    int64      `json:"bitrate,omitempty"`
	CDN        string     `json:"cdn,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// SelectOptions filters the stream list before the best matching stream is chosen.
type SelectOptions struct {
	Codec   string
	Quality string
	Format  string
}
