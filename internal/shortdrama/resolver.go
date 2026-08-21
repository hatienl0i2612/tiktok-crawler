package shortdrama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tiktok-crawler/internal/tiktok"
)

const shortDramaAPIRegion = ""

var digitsPattern = regexp.MustCompile(`^[0-9]+$`)

// Resolve fetches one TikTok Short Drama episode and all media variants.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	inputURL, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	dramaID, episode, ok := episodeFromURL(inputURL)
	if !ok {
		return nil, errors.New("URL must match https://www.tiktok.com/shortdrama/episode/<drama-id>/<episode>")
	}
	episodeItem, _, err := client.fetchShortDramaEpisode(ctx, inputURL.String(), dramaID, episode)
	if err != nil {
		return nil, err
	}
	result := buildShortDramaResult(inputURL.String(), inputURL.String(), dramaID, episode, episodeItem)
	if len(result.Media) > 0 {
		return result, nil
	}

	detailItem, err := client.fetchSignedShortDramaItem(ctx, inputURL.String(), episodeItem.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve Short Drama playback metadata: %w", err)
	}
	result = buildShortDramaResult(inputURL.String(), inputURL.String(), dramaID, episode, mergeShortDramaItems(episodeItem, detailItem))
	result.Sources = append(result.Sources, "api/item/detail")
	if len(result.Media) == 0 {
		return nil, errors.New("TikTok returned Short Drama metadata but no downloadable media profiles")
	}
	return result, nil
}

func (client *Client) fetchShortDramaEpisode(ctx context.Context, referer, dramaID string, episode int) (shortDramaItem, string, error) {
	endpoint, _ := url.Parse("https://www.tiktok.com/api/drama/episode/item_list/")
	query := endpoint.Query()
	query.Set("dramaID", dramaID)
	query.Set("cursor", strconv.Itoa(episode-1))
	query.Set("count", "1")
	query.Set("aid", "1988")
	query.Set("language", "en")
	query.Set("region", shortDramaAPIRegion)
	query.Set("storeRegion", shortDramaAPIRegion)
	endpoint.RawQuery = query.Encode()

	body, finalURL, err := client.session.Fetch(ctx, endpoint.String(), "application/json, text/plain, */*", referer, maxMetadataResponseSize)
	if err != nil {
		return shortDramaItem{}, finalURL, fmt.Errorf("fetch Short Drama episode: %w", err)
	}
	var response shortDramaEpisodeResponse
	if err := decodeJSON(body, &response); err != nil {
		return shortDramaItem{}, finalURL, fmt.Errorf("decode Short Drama episode: %w", err)
	}
	if response.statusCode() != 0 {
		return shortDramaItem{}, finalURL, fmt.Errorf("TikTok Short Drama returned status %d: %s", response.statusCode(), response.statusMessage())
	}
	if len(response.ItemList) == 0 || response.ItemList[0].ID == "" {
		return shortDramaItem{}, finalURL, errors.New("TikTok Short Drama returned no episode item")
	}
	return response.ItemList[0], finalURL, nil
}

func (client *Client) fetchSignedShortDramaItem(ctx context.Context, referer, itemID string) (shortDramaItem, error) {
	msToken := client.session.CookieValue("msToken")
	if msToken == "" {
		return shortDramaItem{}, errors.New("TikTok requires a valid msToken cookie for Short Drama playback; set TIKTOK_COOKIE from your own browser session")
	}
	now := time.Now()
	endpoint, _ := url.Parse("https://www.tiktok.com/api/item/detail/")
	query := endpoint.Query()
	parameters := map[string]string{
		"WebIdLastTime":    strconv.FormatInt(now.Unix(), 10),
		"aid":              "1988",
		"app_language":     "en",
		"app_name":         "tiktok_web",
		"browser_language": "en-US",
		"browser_name":     "Mozilla",
		"browser_online":   "true",
		"browser_platform": "Linux armv8l",
		"browser_version":  client.session.UserAgent(),
		"channel":          "tiktok_web",
		"cookie_enabled":   "true",
		"device_id":        itemID,
		"device_platform":  "web_pc",
		"focus_state":      "true",
		"from_page":        "video",
		"history_len":      "2",
		"is_fullscreen":    "false",
		"is_page_visible":  "true",
		"itemId":           itemID,
		"language":         "en",
		"msToken":          msToken,
		"os":               "linux",
		"priority_region":  shortDramaAPIRegion,
		"region":           shortDramaAPIRegion,
		"screen_height":    "1080",
		"screen_width":     "1920",
		"tz_name":          "Asia/Singapore",
		"webcast_language": "en",
	}
	for key, value := range parameters {
		query.Set(key, value)
	}
	endpoint.RawQuery = query.Encode()
	signedURL := signTikTokURL(endpoint, client.session.UserAgent(), now)
	body, _, err := client.session.Fetch(ctx, signedURL, "application/json, text/plain, */*", referer, maxMetadataResponseSize)
	if err != nil {
		return shortDramaItem{}, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return shortDramaItem{}, errors.New("TikTok returned an empty signed response; refresh TIKTOK_COOKIE and try again")
	}
	var response shortDramaItemDetailResponse
	if err := decodeJSON(body, &response); err != nil {
		return shortDramaItem{}, err
	}
	if response.statusCode() != 0 {
		return shortDramaItem{}, fmt.Errorf("TikTok item detail returned status %d: %s", response.statusCode(), response.statusMessage())
	}
	if response.ItemInfo.ItemStruct.ID == "" {
		return shortDramaItem{}, errors.New("TikTok item detail returned no episode item")
	}
	return response.ItemInfo.ItemStruct, nil
}

func decodeJSON(body []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	return decoder.Decode(destination)
}

func episodeFromURL(parsed *url.URL) (dramaID string, episode int, ok bool) {
	if parsed == nil {
		return "", 0, false
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 4 || !strings.EqualFold(parts[0], "shortdrama") || !strings.EqualFold(parts[1], "episode") {
		return "", 0, false
	}
	dramaID, dramaErr := url.PathUnescape(parts[2])
	episodeText, episodeErr := url.PathUnescape(parts[3])
	if dramaErr != nil || episodeErr != nil || !digitsPattern.MatchString(dramaID) {
		return "", 0, false
	}
	episode, parseErr := strconv.Atoi(episodeText)
	if parseErr != nil || episode <= 0 {
		return "", 0, false
	}
	return dramaID, episode, true
}
