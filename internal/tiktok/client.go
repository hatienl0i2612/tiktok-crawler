// Package tiktok contains shared HTTP and URL primitives for TikTok crawlers.
package tiktok

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

const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"

// SessionOptions configures a shared cookie-aware TikTok HTTP session.
type SessionOptions struct {
	Cookie    string
	UserAgent string
}

// Session sends requests with consistent TikTok headers and redirect policy.
type Session struct {
	httpClient *http.Client
	cookie     string
	userAgent  string
}

// NewSession creates a cookie-aware TikTok HTTP session.
func NewSession(options SessionOptions) (*Session, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	userAgent := strings.TrimSpace(options.UserAgent)
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	return &Session{
		cookie:    NormalizeCookie(options.Cookie),
		userAgent: userAgent,
		httpClient: &http.Client{
			Jar: jar,
			CheckRedirect: func(request *http.Request, previous []*http.Request) error {
				if len(previous) >= 10 {
					return errors.New("too many redirects")
				}
				if request.URL.Scheme != "https" || !IsAllowedHost(request.URL.Hostname()) {
					return fmt.Errorf("TikTok redirected to a disallowed host: %s", request.URL.Redacted())
				}
				return nil
			},
		},
	}, nil
}

// Fetch executes a GET request and reads a size-limited response body.
func (session *Session) Fetch(
	ctx context.Context,
	target string,
	accept string,
	referer string,
	limit int64,
) ([]byte, string, error) {
	response, err := session.Get(ctx, target, accept, referer)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	finalURL := response.Request.URL.String()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, finalURL, fmt.Errorf("HTTP %s", response.Status)
	}
	body, err := ReadLimited(response.Body, limit)
	if err != nil {
		return nil, finalURL, err
	}
	return body, finalURL, nil
}

// Get executes a GET request with the session's standard TikTok headers.
func (session *Session) Get(ctx context.Context, target, accept, referer string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	session.SetHeaders(request, accept, referer)
	return session.httpClient.Do(request)
}

// SetHeaders applies the standard headers used by TikTok web requests.
func (session *Session) SetHeaders(request *http.Request, accept, referer string) {
	request.Header.Set("User-Agent", session.userAgent)
	request.Header.Set("Accept", accept)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Cache-Control", "no-cache")
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	if session.cookie != "" {
		request.Header.Set("Cookie", session.cookie)
	}
}

// HTTPClient returns the underlying client for transport customization in tests.
func (session *Session) HTTPClient() *http.Client {
	return session.httpClient
}

// UserAgent returns the normalized User-Agent used by the session.
func (session *Session) UserAgent() string {
	return session.userAgent
}

// CookieValue returns one value from the configured Cookie header.
func (session *Session) CookieValue(name string) string {
	for _, pair := range strings.Split(session.cookie, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if ok && strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// NormalizeCookie removes whitespace and an optional Cookie: prefix.
func NormalizeCookie(cookie string) string {
	cookie = strings.TrimSpace(cookie)
	if len(cookie) >= 7 && strings.EqualFold(cookie[:7], "cookie:") {
		cookie = strings.TrimSpace(cookie[7:])
	}
	return cookie
}

// ReadLimited reads at most limit bytes and rejects oversized responses.
func ReadLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds the %d MiB limit", limit>>20)
	}
	return body, nil
}

// ParseURL validates an HTTPS TikTok page URL.
func ParseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" || !IsPageHost(parsed.Hostname()) {
		return nil, errors.New("only HTTPS URLs under tiktok.com are accepted")
	}
	if parsed.User != nil {
		return nil, errors.New("URL must not contain a username or password")
	}
	return parsed, nil
}

// IsPageHost reports whether host belongs to tiktok.com.
func IsPageHost(host string) bool {
	return DomainMatches(host, "tiktok.com")
}

// IsAllowedHost reports whether host is a known TikTok page or media domain.
func IsAllowedHost(host string) bool {
	for _, domain := range []string{
		"tiktok.com",
		"tiktokcdn.com",
		"tiktokv.com",
		"ibytedtos.com",
		"byteoversea.com",
	} {
		if DomainMatches(host, domain) {
			return true
		}
	}
	return false
}

// DomainMatches performs an exact or subdomain-safe DNS suffix match.
func DomainMatches(host, domain string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	return host == domain || strings.HasSuffix(host, "."+domain)
}
