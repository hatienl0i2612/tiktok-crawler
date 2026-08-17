package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "https://www.tiktok.com"
	maxResponseSize    = 16 << 20
	discoveryUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 Version/18.5 Mobile/15E148 Safari/604.1"
)

// Client searches TikTok through one cookie-aware web session.
type Client struct {
	httpClient *http.Client
	baseURL    string
	cookie     string
	userAgent  string
}

// NewClient creates a TikTok video search client. The supplied cookie is optional.
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
		baseURL:   defaultBaseURL,
		cookie:    normalizeCookie(options.Cookie),
		userAgent: userAgent,
		httpClient: &http.Client{
			Jar: jar,
			CheckRedirect: func(request *http.Request, previous []*http.Request) error {
				if len(previous) >= 10 {
					return errors.New("too many redirects")
				}
				if request.URL.Scheme != "https" || !isTikTokHost(request.URL.Hostname()) {
					return fmt.Errorf("TikTok redirected to a disallowed host: %s", request.URL.Redacted())
				}
				return nil
			},
		},
	}, nil
}

// Search returns canonical TikTok video URLs for one result page.
func (client *Client) Search(ctx context.Context, options Options) ([]string, error) {
	options, locale, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}

	pageURL, err := client.searchPageURL(options.Keyword)
	if err != nil {
		return nil, err
	}
	pageBody, err := client.fetch(ctx, pageURL, "text/html,application/xhtml+xml", "", locale.browserLanguage)
	if err != nil {
		return nil, fmt.Errorf("fetch TikTok search page: %w", err)
	}

	pageContext := parsePageContext(pageBody)
	requestURL, err := client.searchAPIURL(options, locale, pageContext)
	if err != nil {
		return nil, err
	}
	apiBody, err := client.fetch(
		ctx,
		requestURL,
		"application/json, text/plain, */*",
		pageURL,
		resolvedBrowserLanguage(locale, pageContext),
	)
	if err == nil && len(apiBody) > 0 {
		links, parseErr := parseSearchResponse(apiBody)
		if parseErr == nil {
			return links, nil
		}
		err = parseErr
	} else if err == nil {
		err = errors.New("TikTok returned an empty API response")
	}

	if options.PageIndex == 0 {
		links := parseVideoLinks(pageBody)
		if len(links) == 0 {
			discoveryURL, urlErr := client.discoveryPageURL(options.Keyword, locale)
			if urlErr == nil {
				discoveryBody, fetchErr := client.fetchAs(
					ctx,
					discoveryURL,
					"text/html,application/xhtml+xml",
					pageURL,
					resolvedBrowserLanguage(locale, pageContext),
					discoveryUserAgent,
				)
				if fetchErr == nil {
					links = parseVideoLinks(discoveryBody)
				}
			}
		}
		if len(links) > options.PageSize {
			links = links[:options.PageSize]
		}
		if len(links) > 0 {
			return links, nil
		}
	}
	return nil, fmt.Errorf(
		"search TikTok videos: %w; TikTok may be rate-limiting this network or requiring a verification cookie",
		err,
	)
}

func (client *Client) discoveryPageURL(keyword string, locale localeOptions) (string, error) {
	base, err := url.Parse(client.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse TikTok base URL: %w", err)
	}
	if keyword == "" {
		base.Path = "/channel/trending"
	} else {
		base.Path = "/discover/" + keyword
		base.RawPath = "/discover/" + url.PathEscape(keyword)
	}
	if locale.language != "" {
		query := base.Query()
		query.Set("lang", locale.language)
		base.RawQuery = query.Encode()
	}
	return base.String(), nil
}

func (client *Client) searchPageURL(keyword string) (string, error) {
	base, err := url.Parse(client.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse TikTok base URL: %w", err)
	}
	if keyword == "" {
		base.Path = "/foryou"
		return base.String(), nil
	}
	base.Path = "/search/video"
	query := base.Query()
	query.Set("q", keyword)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (client *Client) searchAPIURL(options Options, locale localeOptions, page pageContext) (string, error) {
	base, err := url.Parse(client.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse TikTok base URL: %w", err)
	}
	if options.Keyword == "" {
		base.Path = "/api/recommend/item_list/"
	} else {
		base.Path = "/api/search/item/full/"
	}

	offset := options.PageIndex * options.PageSize
	language := page.Language
	if language == "" {
		language = "en"
	}
	region := page.Region
	if locale.language != "" {
		language = locale.language
	}
	if locale.region != "" {
		region = locale.region
	}

	deviceID := page.DeviceID
	if deviceID == "" {
		deviceID = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	createdAt := page.WebIDCreatedTime
	if createdAt == "" {
		createdAt = strconv.FormatInt(time.Now().Unix(), 10)
	}
	browserLanguage := resolvedBrowserLanguage(locale, page)

	query := base.Query()
	query.Set("WebIdLastTime", createdAt)
	query.Set("aid", "1988")
	query.Set("app_language", language)
	query.Set("app_name", "tiktok_web")
	query.Set("browser_language", browserLanguage)
	query.Set("browser_name", "Mozilla")
	query.Set("browser_online", "true")
	query.Set("browser_platform", browserPlatform())
	query.Set("browser_version", strings.TrimPrefix(client.userAgent, "Mozilla/"))
	query.Set("channel", "tiktok_web")
	query.Set("cookie_enabled", "true")
	query.Set("count", strconv.Itoa(options.PageSize))
	query.Set("cursor", strconv.Itoa(offset))
	query.Set("data_collection_enabled", "false")
	query.Set("device_id", deviceID)
	query.Set("device_platform", "web_pc")
	query.Set("focus_state", "true")
	query.Set("history_len", "2")
	query.Set("is_fullscreen", "false")
	query.Set("is_page_visible", "true")
	query.Set("os", browserOS())
	query.Set("priority_region", "")
	query.Set("referer", "")
	if region != "" {
		query.Set("region", region)
	}
	query.Set("screen_height", "1080")
	query.Set("screen_width", "1920")
	query.Set("tz_name", time.Now().Location().String())
	query.Set("user_is_login", strconv.FormatBool(client.cookie != ""))
	query.Set("webcast_language", language)
	query.Set("X-Bogus", "1")

	if options.Keyword == "" {
		query.Set("from_page", "fyp")
		query.Set("type", "1")
	} else {
		query.Set("from_page", "search")
		query.Set("is_non_personalized_search", "0")
		query.Set("keyword", options.Keyword)
		query.Set("offset", strconv.Itoa(offset))
	}
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (client *Client) fetch(ctx context.Context, target, accept, referer, language string) ([]byte, error) {
	return client.fetchAs(ctx, target, accept, referer, language, client.userAgent)
}

func (client *Client) fetchAs(ctx context.Context, target, accept, referer, language, userAgent string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", accept)
	if language == "" {
		language = "en-US"
	}
	request.Header.Set("Accept-Language", language+",en;q=0.8")
	request.Header.Set("Cache-Control", "no-cache")
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	if client.cookie != "" {
		request.Header.Set("Cookie", client.cookie)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP %s", response.Status)
	}
	return readLimited(response.Body)
}

func readLimited(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxResponseSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("response exceeds the %d MiB limit", maxResponseSize>>20)
	}
	return body, nil
}

func browserPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "MacIntel"
	case "windows":
		return "Win32"
	default:
		return "Linux x86_64"
	}
}

func browserOS() string {
	if runtime.GOOS == "darwin" {
		return "mac"
	}
	return runtime.GOOS
}

func normalizeCookie(cookie string) string {
	cookie = strings.TrimSpace(cookie)
	if len(cookie) >= 7 && strings.EqualFold(cookie[:7], "cookie:") {
		cookie = strings.TrimSpace(cookie[7:])
	}
	return cookie
}

func isTikTokHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "tiktok.com" || strings.HasSuffix(host, ".tiktok.com")
}
