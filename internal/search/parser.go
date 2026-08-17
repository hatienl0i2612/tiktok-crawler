package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

var (
	hydrationPattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__UNIVERSAL_DATA_FOR_REHYDRATION__["'][^>]*>(.*?)</script\s*>`)
	videoLinkPattern = regexp.MustCompile(`(?i)(?:https://www\.tiktok\.com)?/(@[^/"'?#<>\s]+/video/[0-9]+)`)
)

type pageContext struct {
	Language         string `json:"language"`
	Region           string `json:"region"`
	DeviceID         string `json:"wid"`
	WebIDCreatedTime string `json:"webIdCreatedTime"`
}

func parsePageContext(body []byte) pageContext {
	match := hydrationPattern.FindSubmatch(body)
	if len(match) != 2 {
		return pageContext{}
	}
	var hydration struct {
		DefaultScope map[string]json.RawMessage `json:"__DEFAULT_SCOPE__"`
	}
	if json.Unmarshal(bytes.TrimSpace(match[1]), &hydration) != nil {
		return pageContext{}
	}
	var context pageContext
	_ = json.Unmarshal(hydration.DefaultScope["webapp.app-context"], &context)
	return context
}

func parseSearchResponse(body []byte) ([]string, error) {
	var response map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(body), &response); err != nil {
		return nil, fmt.Errorf("decode search API response: %w", err)
	}
	statusCode := decodeInt(response, "status_code", "statusCode")
	if statusCode != 0 {
		message := firstScalar(response, "status_msg", "statusMsg")
		if message == "" {
			message = "unknown error"
		}
		return nil, fmt.Errorf("TikTok returned status %d: %s", statusCode, message)
	}

	var entries []json.RawMessage
	foundList := false
	for _, key := range []string{"data", "itemList", "items"} {
		raw, exists := response[key]
		if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("decode search API field %s: %w", key, err)
		}
		foundList = true
		entries = append(entries, list...)
	}
	if !foundList {
		return nil, errors.New("TikTok response did not contain a video result list")
	}
	links := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		collectVideoLinks(entry, 0, &links, seen)
	}
	if links == nil {
		links = []string{}
	}
	return links, nil
}

func decodeInt(object map[string]json.RawMessage, keys ...string) int {
	for _, key := range keys {
		raw, exists := object[key]
		if !exists {
			continue
		}
		var value int
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return 0
}

func collectVideoLinks(raw json.RawMessage, depth int, links *[]string, seen map[string]struct{}) {
	if depth > 5 || len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return
	}

	id := firstScalar(object, "id", "itemId", "item_id", "aweme_id")
	author := authorName(object)
	if isNumericID(id) && author != "" {
		link := "https://www.tiktok.com/@" + url.PathEscape(strings.TrimPrefix(author, "@")) + "/video/" + id
		if _, exists := seen[link]; !exists {
			seen[link] = struct{}{}
			*links = append(*links, link)
		}
	}

	for _, key := range []string{"item", "itemInfo", "item_info", "itemStruct", "item_struct", "awemeInfo", "aweme_info"} {
		if nested, exists := object[key]; exists {
			collectVideoLinks(nested, depth+1, links, seen)
		}
	}
}

func authorName(object map[string]json.RawMessage) string {
	for _, key := range []string{"author", "authorMeta", "author_meta"} {
		raw, exists := object[key]
		if !exists {
			continue
		}
		var name string
		if json.Unmarshal(raw, &name) == nil && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
		var author map[string]json.RawMessage
		if json.Unmarshal(raw, &author) == nil {
			if name = firstScalar(author, "uniqueId", "unique_id", "name"); name != "" {
				return name
			}
		}
	}
	return firstScalar(object, "uniqueId", "unique_id")
}

func firstScalar(object map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		raw, exists := object[key]
		if !exists {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			return strings.TrimSpace(value)
		}
		var number json.Number
		if json.Unmarshal(raw, &number) == nil {
			return number.String()
		}
	}
	return ""
}

func isNumericID(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func parseVideoLinks(body []byte) []string {
	normalized := html.UnescapeString(string(body))
	normalized = strings.ReplaceAll(normalized, `\u002F`, "/")
	normalized = strings.ReplaceAll(normalized, `\u002f`, "/")
	normalized = strings.ReplaceAll(normalized, `\/`, "/")
	matches := videoLinkPattern.FindAllStringSubmatch(normalized, -1)
	links := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		link := "https://www.tiktok.com/" + match[1]
		if _, exists := seen[link]; exists {
			continue
		}
		seen[link] = struct{}{}
		links = append(links, link)
	}
	return links
}
