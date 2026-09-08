package hashtag

import (
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

const maxHashtagResponseSize = 16 << 20

// Client resolves hashtag pages and reuses one video client/session when their
// returned canonical video links are downloaded.
type Client struct {
	video   *video.Client
	session *tiktok.Session
}

// NewClient creates a public TikTok hashtag client.
func NewClient(options ClientOptions) (*Client, error) {
	videoClient, err := video.NewClient(video.ClientOptions{
		Cookie: options.Cookie, UserAgent: options.UserAgent, Headers: options.Headers,
	})
	if err != nil {
		return nil, err
	}
	return &Client{video: videoClient, session: videoClient.Session()}, nil
}

// Video returns the video client sharing this hashtag client's HTTP session.
func (client *Client) Video() *video.Client { return client.video }

// Session returns the cookie-aware session shared by hashtag and video requests.
func (client *Client) Session() *tiktok.Session { return client.session }
