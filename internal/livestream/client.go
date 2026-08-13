package livestream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maxResponseSize = 16 << 20

var sigiStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']SIGI_STATE["'][^>]*>(.*?)</script\s*>`)

// Client resolves TikTok live pages using a shared cookie-aware HTTP session.
type Client struct {
	httpClient *http.Client
	cookie     string
	userAgent  string
}

// NewClient creates a resolver client. The supplied cookie is optional.
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
				if request.URL.Scheme != "https" || !isTikTokHost(request.URL.Hostname()) {
					return fmt.Errorf("TikTok redirected to a disallowed host: %s", request.URL.Redacted())
				}
				return nil
			},
		},
	}, nil
}

// Resolve fetches a TikTok live page and returns all exposed playback streams.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	inputURL, err := parseTikTokURL(rawURL)
	if err != nil {
		return nil, err
	}

	pageBody, fetchedURL, pageErr := client.fetch(
		ctx,
		inputURL.String(),
		"text/html,application/xhtml+xml",
		"",
	)
	finalURL := inputURL.String()
	if fetchedURL != "" {
		finalURL = fetchedURL
	}

	var pageInfo roomInfo
	var sigiErr error
	if pageErr == nil {
		pageInfo, sigiErr = parseSIGIState(pageBody)
	}

	username := usernameFromLiveURL(inputURL)
	if final, parseErr := url.Parse(finalURL); parseErr == nil {
		if redirectedUsername := usernameFromLiveURL(final); redirectedUsername != "" {
			username = redirectedUsername
		}
	}
	if pageInfo.User.UniqueID != "" {
		username = pageInfo.User.UniqueID
	}
	if username == "" {
		if pageErr != nil {
			return nil, fmt.Errorf("fetch live page: %w", pageErr)
		}
		return nil, errors.New("URL must match https://www.tiktok.com/@username/live")
	}

	apiInfo, apiErr := client.fetchUserRoom(ctx, username, finalURL)
	info, source := roomInfo{}, ""
	switch {
	case apiErr == nil && hasUser(apiInfo):
		info, source = apiInfo, "api-live/user/room"
	case hasUser(pageInfo):
		info, source = pageInfo, "SIGI_STATE"
	default:
		return nil, resolveError(pageErr, sigiErr, apiErr)
	}

	streams, streamErrors := collectStreams(info)
	if len(streams) == 0 {
		details := ""
		if len(streamErrors) > 0 {
			details = ": " + strings.Join(streamErrors, "; ")
		}
		return nil, fmt.Errorf(
			"no playback URLs found (user status=%d, room status=%d)%s; the live may have ended or may require a suitable cookie or region",
			info.User.Status,
			info.LiveRoom.Status,
			details,
		)
	}

	return makeResult(inputURL.String(), finalURL, source, info, streams), nil
}

func (client *Client) fetchUserRoom(ctx context.Context, username, referer string) (roomInfo, error) {
	endpoint, _ := url.Parse("https://www.tiktok.com/api-live/user/room")
	query := endpoint.Query()
	query.Set("uniqueId", strings.ToLower(username))
	query.Set("webcast_language", "en")
	query.Set("aid", "1988")
	query.Set("sourceType", "54")
	query.Set("staleTime", "600000")
	endpoint.RawQuery = query.Encode()

	body, _, err := client.fetch(ctx, endpoint.String(), "application/json, text/plain, */*", referer)
	if err != nil {
		return roomInfo{}, err
	}
	var response apiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return roomInfo{}, fmt.Errorf("decode room API response: %w", err)
	}
	status := response.StatusCode
	if response.StatusCodeSnake != 0 {
		status = response.StatusCodeSnake
	}
	if status != 0 {
		message := response.Prompts
		if message == "" {
			message = response.Message
		}
		return roomInfo{}, fmt.Errorf("TikTok returned status %d: %s", status, message)
	}
	if !hasUser(response.Data) {
		return roomInfo{}, errors.New("room API returned an empty user")
	}
	return response.Data, nil
}

func (client *Client) fetch(
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
	body, err := readLimited(response.Body)
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
	if parsed.Scheme != "https" || !isTikTokHost(parsed.Hostname()) {
		return nil, errors.New("only HTTPS URLs under tiktok.com are accepted")
	}
	if parsed.User != nil {
		return nil, errors.New("URL must not contain a username or password")
	}
	return parsed, nil
}

func isTikTokHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "tiktok.com" || strings.HasSuffix(host, ".tiktok.com")
}

func usernameFromLiveURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "@") || !strings.EqualFold(parts[1], "live") {
		return ""
	}
	username, err := url.PathUnescape(strings.TrimPrefix(parts[0], "@"))
	if err != nil {
		return ""
	}
	return username
}

func parseSIGIState(html []byte) (roomInfo, error) {
	match := sigiStatePattern.FindSubmatch(html)
	if len(match) != 2 {
		return roomInfo{}, errors.New("SIGI_STATE script was not found")
	}
	var state sigiState
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return roomInfo{}, fmt.Errorf("decode SIGI_STATE: %w", err)
	}
	info := state.LiveRoom.LiveRoomUserInfo
	if !hasUser(info) {
		return roomInfo{}, errors.New("SIGI_STATE does not contain liveRoomUserInfo")
	}
	return info, nil
}

func hasUser(info roomInfo) bool {
	return info.User.ID != "" || info.User.UniqueID != ""
}

func resolveError(pageErr, sigiErr, apiErr error) error {
	parts := make([]string, 0, 3)
	if pageErr != nil {
		parts = append(parts, "live page: "+pageErr.Error())
	} else if sigiErr != nil {
		parts = append(parts, "SIGI_STATE: "+sigiErr.Error())
	}
	if apiErr != nil {
		parts = append(parts, "room API: "+apiErr.Error())
	}
	if len(parts) == 0 {
		parts = append(parts, "TikTok returned no room data")
	}
	return errors.New("unable to resolve live room: " + strings.Join(parts, "; "))
}

func normalizeCookie(cookie string) string {
	cookie = strings.TrimSpace(cookie)
	if len(cookie) >= 7 && strings.EqualFold(cookie[:7], "cookie:") {
		cookie = strings.TrimSpace(cookie[7:])
	}
	return cookie
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

func makeResult(inputURL, finalURL, source string, info roomInfo, streams []Stream) *Result {
	result := &Result{
		InputURL:  inputURL,
		FinalURL:  finalURL,
		Source:    source,
		FetchedAt: time.Now().UTC(),
		User: User{
			ID:            info.User.ID,
			UniqueID:      info.User.UniqueID,
			Nickname:      info.User.Nickname,
			RoomID:        info.User.RoomID,
			Status:        info.User.Status,
			FollowerCount: info.Stats.FollowerCount,
		},
		Live: Live{
			IsLive:      len(streams) > 0,
			Status:      info.LiveRoom.Status,
			Title:       info.LiveRoom.Title,
			StreamID:    info.LiveRoom.StreamID,
			ViewerCount: info.LiveRoom.Stats.UserCount,
			EnterCount:  info.LiveRoom.Stats.EnterCount,
		},
		Streams: streams,
	}
	if info.LiveRoom.StartTime > 0 {
		startTime := time.Unix(info.LiveRoom.StartTime, 0).UTC()
		result.Live.StartTime = &startTime
	}
	return result
}
