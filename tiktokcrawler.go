// Package tiktokcrawler is the high-level entry point for the TikTok Crawler
// library. It detects the kind of a TikTok URL and resolves it through the
// specialized video, photo, profile, livestream, or shortdrama clients.
//
// # Quick start
//
//	config := tiktokcrawler.ClientOptions{
//		Cookie:  "ttwid=...; sessionid=...",
//		Headers: map[string]string{"User-Agent": "my-agent"},
//	}
//	client, err := tiktokcrawler.NewClient(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	result, err := client.Resolve(ctx, "https://www.tiktok.com/@example/video/123")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// For more focused control, import the subpackages directly (video, photo,
// profile, livestream, shortdrama, cookies, downloader).
package tiktokcrawler

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/hatienl0i2612/tiktok-crawler/livestream"
	"github.com/hatienl0i2612/tiktok-crawler/photo"
	"github.com/hatienl0i2612/tiktok-crawler/profile"
	"github.com/hatienl0i2612/tiktok-crawler/shortdrama"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

var (
	livePathPattern       = regexp.MustCompile(`^/@[^/]+/live/?$`)
	videoPathPattern      = regexp.MustCompile(`^/@[^/]+/video/[0-9]+/?$`)
	photoPathPattern      = regexp.MustCompile(`^/@[^/]+/photo/[0-9]+/?$`)
	shortDramaPathPattern = regexp.MustCompile(`^/shortdrama/episode/[0-9]+/[1-9][0-9]*/?$`)
	profilePathPattern    = regexp.MustCompile(`^/@[^/]+/?$`)
)

// Kind describes the type of a TikTok content URL.
type Kind string

const (
	// KindVideo is a TikTok video post or video Story under a /video/<id> URL.
	KindVideo Kind = "video"
	// KindPhoto is a TikTok Photo Post or photo Story under a /photo/<id> URL.
	KindPhoto Kind = "photo"
	// KindLive is a TikTok LIVE room.
	KindLive Kind = "livestream"
	// KindShortDrama is a TikTok Short Drama episode.
	KindShortDrama Kind = "shortdrama"
	// KindProfile is a public TikTok creator profile under an /@username URL.
	KindProfile Kind = "profile"
)

// DetectKind validates a TikTok URL and returns its content kind. It returns
// an error for non-TikTok URLs or URLs without a supported content path.
func DetectKind(rawURL string) (Kind, error) {
	parsed, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid TikTok URL %q", rawURL)
	}
	switch {
	case livePathPattern.MatchString(parsed.EscapedPath()):
		return KindLive, nil
	case videoPathPattern.MatchString(parsed.EscapedPath()):
		return KindVideo, nil
	case photoPathPattern.MatchString(parsed.EscapedPath()):
		return KindPhoto, nil
	case shortDramaPathPattern.MatchString(parsed.EscapedPath()):
		return KindShortDrama, nil
	case profilePathPattern.MatchString(parsed.EscapedPath()):
		return KindProfile, nil
	default:
		return "", fmt.Errorf("URL must be a TikTok video, Photo Post, profile, Short Drama episode, or LIVE URL: %q", rawURL)
	}
}

// ClientOptions configures the HTTP sessions used by the high-level client.
type ClientOptions struct {
	Cookie  string
	Headers map[string]string
}

// Client is a high-level client that detects the URL kind and resolves it.
type Client struct {
	video      *video.Client
	photo      *photo.Client
	profile    *profile.Client
	livestream *livestream.Client
	shortdrama *shortdrama.Client
}

// NewClient creates a high-level client whose sub-clients share the same
// cookie and header configuration.
func NewClient(options ClientOptions) (*Client, error) {
	videoClient, err := video.NewClient(video.ClientOptions{Cookie: options.Cookie, Headers: options.Headers})
	if err != nil {
		return nil, err
	}
	photoClient, err := photo.NewClient(photo.ClientOptions{Cookie: options.Cookie, Headers: options.Headers})
	if err != nil {
		return nil, err
	}
	profileClient, err := profile.NewClient(profile.ClientOptions{Cookie: options.Cookie, Headers: options.Headers})
	if err != nil {
		return nil, err
	}
	liveClient, err := livestream.NewClient(livestream.ClientOptions{Cookie: options.Cookie, Headers: options.Headers})
	if err != nil {
		return nil, err
	}
	dramaClient, err := shortdrama.NewClient(shortdrama.ClientOptions{Cookie: options.Cookie, Headers: options.Headers})
	if err != nil {
		return nil, err
	}
	return &Client{video: videoClient, photo: photoClient, profile: profileClient, livestream: liveClient, shortdrama: dramaClient}, nil
}

// Video returns the underlying video client for focused resolution or download.
func (client *Client) Video() *video.Client { return client.video }

// Photo returns the underlying Photo Post client for focused resolution or download.
func (client *Client) Photo() *photo.Client { return client.photo }

// Profile returns the underlying creator profile client.
func (client *Client) Profile() *profile.Client { return client.profile }

// Livestream returns the underlying livestream client for focused resolution.
func (client *Client) Livestream() *livestream.Client { return client.livestream }

// ShortDrama returns the underlying short drama client for focused resolution.
func (client *Client) ShortDrama() *shortdrama.Client { return client.shortdrama }

// Result is a tagged union of every possible resolve result.
type Result struct {
	Kind       Kind
	Video      *video.Result
	Photo      *photo.Result
	Profile    *profile.Result
	Livestream *livestream.Result
	ShortDrama *shortdrama.Result
}

// Resolve detects the kind of rawURL and resolves it with the matching client.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	kind, err := DetectKind(rawURL)
	if err != nil {
		return nil, err
	}
	switch kind {
	case KindVideo:
		result, err := client.video.Resolve(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindVideo, Video: result}, nil
	case KindPhoto:
		result, err := client.photo.Resolve(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindPhoto, Photo: result}, nil
	case KindProfile:
		result, err := client.profile.Resolve(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindProfile, Profile: result}, nil
	case KindLive:
		result, err := client.livestream.Resolve(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindLive, Livestream: result}, nil
	case KindShortDrama:
		result, err := client.shortdrama.Resolve(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindShortDrama, ShortDrama: result}, nil
	default:
		return nil, errors.New("unsupported TikTok content kind")
	}
}
