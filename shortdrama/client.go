package shortdrama

import "github.com/hatienl0i2612/tiktok-crawler/tiktok"

const maxMetadataResponseSize = 16 << 20

// Client resolves TikTok Short Drama episodes using a shared HTTP session.
type Client struct {
	session *tiktok.Session
}

// NewClient creates a TikTok Short Drama client.
func NewClient(options ClientOptions) (*Client, error) {
	userAgent := options.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	session, err := tiktok.NewSession(tiktok.SessionOptions{Cookie: options.Cookie, UserAgent: userAgent, Headers: options.Headers})
	if err != nil {
		return nil, err
	}
	return &Client{session: session}, nil
}

// Session returns the HTTP session used to resolve and download this episode's media.
func (client *Client) Session() *tiktok.Session {
	return client.session
}
