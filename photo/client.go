package photo

import (
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

const maxMetadataResponseSize = 16 << 20

// Client resolves TikTok Photo Posts and photo Stories using the same mobile
// web session as the regular video resolver.
type Client struct {
	session *tiktok.Session
}

// NewClient creates a TikTok photo client. The supplied cookie is optional.
func NewClient(options ClientOptions) (*Client, error) {
	videoClient, err := video.NewClient(video.ClientOptions(options))
	if err != nil {
		return nil, err
	}
	return &Client{session: videoClient.Session()}, nil
}

// Session returns the HTTP session used to resolve and download photo images.
func (client *Client) Session() *tiktok.Session {
	return client.session
}
