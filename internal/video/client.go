package video

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const maxMetadataResponseSize = 16 << 20

// Client uses one cookie-aware HTTP session for page, metadata, and media requests.
type Client struct {
	httpClient *http.Client
	cookie     string
	userAgent  string
}

// NewClient creates a TikTok video client. The supplied cookie is optional.
func NewClient(options ClientOptions) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	userAgent := strings.TrimSpace(options.UserAgent)
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	return &Client{
		cookie:    normalizeCookie(options.Cookie),
		userAgent: userAgent,
		httpClient: &http.Client{
			Jar: jar,
			CheckRedirect: func(request *http.Request, previous []*http.Request) error {
				if len(previous) >= 10 {
					return errors.New("too many redirects")
				}
				if request.URL.Scheme != "https" || !isAllowedTikTokHost(request.URL.Hostname()) {
					return fmt.Errorf("TikTok redirected to a disallowed host: %s", request.URL.Redacted())
				}
				return nil
			},
		},
	}, nil
}

func (client *Client) fetchMetadata(
	ctx context.Context,
	target string,
	accept string,
	referer string,
) ([]byte, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	client.setHeaders(request, accept, referer)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	finalURL := response.Request.URL.String()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, finalURL, fmt.Errorf("HTTP %s", response.Status)
	}
	body, err := readLimited(response.Body, maxMetadataResponseSize)
	if err != nil {
		return nil, finalURL, err
	}
	return body, finalURL, nil
}

func (client *Client) setHeaders(request *http.Request, accept, referer string) {
	request.Header.Set("User-Agent", client.userAgent)
	request.Header.Set("Accept", accept)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Cache-Control", "no-cache")
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	if client.cookie != "" {
		request.Header.Set("Cookie", client.cookie)
	}
}

func parseTikTokURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" || !isTikTokPageHost(parsed.Hostname()) {
		return nil, errors.New("only HTTPS URLs under tiktok.com are accepted")
	}
	if parsed.User != nil {
		return nil, errors.New("URL must not contain a username or password")
	}
	return parsed, nil
}

func isTikTokPageHost(host string) bool {
	return domainMatches(host, "tiktok.com")
}

func isAllowedTikTokHost(host string) bool {
	for _, domain := range []string{
		"tiktok.com",
		"tiktokcdn.com",
		"tiktokv.com",
		"ibytedtos.com",
		"byteoversea.com",
	} {
		if domainMatches(host, domain) {
			return true
		}
	}
	return false
}

func domainMatches(host, domain string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func normalizeCookie(cookie string) string {
	cookie = strings.TrimSpace(cookie)
	if len(cookie) >= 7 && strings.EqualFold(cookie[:7], "cookie:") {
		cookie = strings.TrimSpace(cookie[7:])
	}
	return cookie
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds the %d MiB limit", limit>>20)
	}
	return body, nil
}
