package hashtag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

var embedStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__FRONTITY_CONNECT_STATE__["'][^>]*>(.*?)</script\s*>`)

// Resolve fetches the public videos TikTok publishes through its tag embed.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	inputURL, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	tagName := tagNameFromURL(inputURL)
	if tagName == "" {
		return nil, errors.New("URL must match https://www.tiktok.com/tag/<name>")
	}

	page, err := client.fetchTagEmbed(ctx, tagName, inputURL.String())
	if err != nil {
		return nil, err
	}
	return buildResult(inputURL.String(), tagName, page), nil
}

func (client *Client) fetchTagEmbed(ctx context.Context, tagName, referer string) (embedPage, error) {
	target := "https://www.tiktok.com/embed/tag/" + url.PathEscape(tagName)
	body, _, err := client.session.Fetch(ctx, target, "text/html,application/xhtml+xml", referer, maxHashtagResponseSize)
	if err != nil {
		return embedPage{}, fmt.Errorf("fetch tag embed: %w", err)
	}
	match := embedStatePattern.FindSubmatch(body)
	if len(match) != 2 {
		return embedPage{}, errors.New("tag embed Frontity state was not found")
	}
	var state embedState
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return embedPage{}, fmt.Errorf("decode tag embed: %w", err)
	}
	for _, page := range state.Source.Data {
		if page.IsError || !strings.EqualFold(page.PageName, "tag") || !strings.EqualFold(page.PlaylistType, "hashtag") {
			continue
		}
		if page.PlaylistEndpoint != "" && !strings.EqualFold(page.PlaylistEndpoint, tagName) {
			continue
		}
		if page.EmbedInfo.StatusCode != 0 {
			return embedPage{}, fmt.Errorf("tag embed returned status %d", page.EmbedInfo.StatusCode)
		}
		if page.EmbedInfo.Code != 0 && page.EmbedInfo.Code != 200 {
			return embedPage{}, fmt.Errorf("tag embed returned code %d", page.EmbedInfo.Code)
		}
		return page, nil
	}
	return embedPage{}, errors.New("tag embed did not contain the requested hashtag")
}

func buildResult(inputURL, requestedTag string, page embedPage) *Result {
	tagName := page.PlaylistEndpoint
	if tagName == "" {
		tagName = requestedTag
	}
	result := &Result{
		InputURL:  inputURL,
		FinalURL:  inputURL,
		Sources:   []string{"tag embed"},
		FetchedAt: time.Now().UTC(),
		Hashtag: Hashtag{
			ID:          page.EmbedInfo.ID,
			Name:        tagName,
			Description: page.EmbedInfo.Description,
			CoverURL:    page.EmbedInfo.CoverURL,
			ViewCount:   int64(page.EmbedInfo.ViewCount),
			VideoCount:  int64(page.EmbedInfo.VideoCount),
		},
		Listing:   Listing{TotalCount: int64(page.EmbedInfo.VideoCount), EmbedOnly: true},
		Videos:    make([]Video, 0, len(page.VideoList)),
		VideoURLs: make([]string, 0, len(page.VideoList)),
	}
	seen := make(map[string]struct{}, len(page.VideoList))
	for _, item := range page.VideoList {
		if item.ID == "" || item.AuthorUniqueID == "" || item.PrivateItem {
			continue
		}
		// Tag embeds mix normal videos and Photo Posts. Only normal videos have
		// playAddr, and only those can be handed safely to video.Client.
		if strings.TrimSpace(item.PlayAddress) == "" {
			result.Listing.SkippedNonVideoCount++
			continue
		}
		canonicalURL := "https://www.tiktok.com/@" + url.PathEscape(item.AuthorUniqueID) + "/video/" + url.PathEscape(item.ID)
		if _, exists := seen[canonicalURL]; exists {
			continue
		}
		seen[canonicalURL] = struct{}{}
		result.VideoURLs = append(result.VideoURLs, canonicalURL)
		result.Videos = append(result.Videos, Video{
			ID:             item.ID,
			URL:            canonicalURL,
			Description:    item.Description,
			Width:          item.Width,
			Height:         item.Height,
			Quality:        item.Ratio,
			PlayCount:      int64(item.PlayCount),
			CoverURLs:      media.UniqueStrings([]string{item.CoverURL, item.OriginCoverURL, item.DynamicCoverURL}),
			AuthorUniqueID: item.AuthorUniqueID,
		})
	}
	result.Listing.ReturnedCount = len(result.Videos)
	return result
}

func tagNameFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "tag") {
		return ""
	}
	tagName, err := url.PathUnescape(parts[1])
	if err != nil {
		return ""
	}
	tagName = strings.TrimSpace(tagName)
	if tagName == "" || tagName == "." || tagName == ".." || strings.ContainsAny(tagName, "/\\\x00") {
		return ""
	}
	return tagName
}
