package video

import (
	"context"

	"tiktok-crawler/internal/tiktok"
)

const maxMetadataResponseSize = 16 << 20

// Client resolves regular TikTok video metadata using a shared HTTP session.
type Client struct {
	session *tiktok.Session
}

// NewClient creates a TikTok video client. The supplied cookie is optional.
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

// Session returns the HTTP session used to resolve and download this video's media.
func (client *Client) Session() *tiktok.Session {
	return client.session
}

func (client *Client) fetchMetadata(
	ctx context.Context,
	target string,
	accept string,
	referer string,
) ([]byte, string, error) {
	return client.session.Fetch(ctx, target, accept, referer, maxMetadataResponseSize)
}
