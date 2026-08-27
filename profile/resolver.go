package profile

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

var (
	hydrationPattern  = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__UNIVERSAL_DATA_FOR_REHYDRATION__["'][^>]*>(.*?)</script\s*>`)
	embedStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__FRONTITY_CONNECT_STATE__["'][^>]*>(.*?)</script\s*>`)
)

// Resolve fetches public profile metadata and the recent video links TikTok
// publishes through its creator embed.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	inputURL, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	username := usernameFromURL(inputURL)
	if username == "" {
		return nil, errors.New("URL must match https://www.tiktok.com/@username")
	}

	pageBody, fetchedURL, pageErr := client.session.Fetch(
		ctx, inputURL.String(), "text/html,application/xhtml+xml", "", maxProfileResponseSize,
	)
	finalURL := inputURL.String()
	if fetchedURL != "" {
		finalURL = fetchedURL
	}
	hydration, hydrationErr := parseHydration(pageBody, username)
	if pageErr != nil {
		hydrationErr = pageErr
	}

	embed, embedErr := client.fetchCreatorEmbed(ctx, username, finalURL)
	if embedErr != nil {
		if hydrationErr != nil {
			return nil, fmt.Errorf("resolve profile page: %v; resolve creator embed: %w", hydrationErr, embedErr)
		}
		return nil, embedErr
	}

	result := buildResult(inputURL.String(), finalURL, username, hydration, embed)
	if hydrationErr == nil {
		result.Sources = append(result.Sources, "profile hydration")
	} else {
		result.Warnings = append(result.Warnings, "profile hydration unavailable: "+hydrationErr.Error())
	}
	result.Sources = append(result.Sources, "creator embed")
	return result, nil
}

func (client *Client) fetchCreatorEmbed(ctx context.Context, username, referer string) (embedPage, error) {
	target := "https://www.tiktok.com/embed/@" + url.PathEscape(username)
	body, _, err := client.session.Fetch(ctx, target, "text/html,application/xhtml+xml", referer, maxProfileResponseSize)
	if err != nil {
		return embedPage{}, fmt.Errorf("fetch creator embed: %w", err)
	}
	match := embedStatePattern.FindSubmatch(body)
	if len(match) != 2 {
		return embedPage{}, errors.New("creator embed Frontity state was not found")
	}
	var state embedState
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return embedPage{}, fmt.Errorf("decode creator embed: %w", err)
	}
	for _, page := range state.Source.Data {
		if page.IsError || !strings.EqualFold(page.PlaylistType, "creator") {
			continue
		}
		if !strings.EqualFold(page.UserInfo.UniqueID, username) && !strings.EqualFold(page.PlaylistID, username) {
			continue
		}
		if page.UserInfo.Code != 0 && page.UserInfo.Code != 200 {
			return embedPage{}, fmt.Errorf("creator embed returned user code %d", page.UserInfo.Code)
		}
		return page, nil
	}
	return embedPage{}, errors.New("creator embed did not contain the requested user")
}

func parseHydration(body []byte, username string) (hydrationUserInfo, error) {
	match := hydrationPattern.FindSubmatch(body)
	if len(match) != 2 {
		return hydrationUserInfo{}, errors.New("profile hydration state was not found")
	}
	var state hydrationState
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return hydrationUserInfo{}, fmt.Errorf("decode profile hydration: %w", err)
	}
	detail := state.DefaultScope.UserDetail
	if detail.StatusCode != 0 {
		return hydrationUserInfo{}, fmt.Errorf("profile hydration returned status %d: %s", detail.StatusCode, detail.StatusMsg)
	}
	if !strings.EqualFold(detail.UserInfo.User.UniqueID, username) {
		return hydrationUserInfo{}, errors.New("profile hydration did not contain the requested user")
	}
	return detail.UserInfo, nil
}

func buildResult(inputURL, finalURL, requestedUsername string, hydration hydrationUserInfo, embed embedPage) *Result {
	embedUser := embed.UserInfo
	user := User{
		ID:             embedUser.ID,
		UniqueID:       embedUser.UniqueID,
		Nickname:       embedUser.Nickname,
		Signature:      embedUser.Signature,
		Verified:       embedUser.Verified,
		PrivateAccount: embedUser.PrivateAccount,
		AvatarURLs:     media.UniqueStrings([]string{embedUser.AvatarThumbURL}),
		Statistics: UserStatistics{
			FollowingCount: int64(embedUser.FollowingCount),
			FollowerCount:  int64(embedUser.FollowerCount),
			HeartCount:     int64(embedUser.HeartCount),
		},
	}
	mergeHydrationUser(&user, hydration)
	if user.UniqueID == "" {
		user.UniqueID = requestedUsername
	}

	videos := make([]Video, 0, len(embed.VideoList))
	videoURLs := make([]string, 0, len(embed.VideoList))
	for _, item := range embed.VideoList {
		if item.ID == "" || item.PrivateItem {
			continue
		}
		author := item.AuthorUniqueID
		if author == "" {
			author = user.UniqueID
		}
		canonicalURL := "https://www.tiktok.com/@" + url.PathEscape(author) + "/video/" + url.PathEscape(item.ID)
		videos = append(videos, Video{
			ID:             item.ID,
			URL:            canonicalURL,
			Description:    item.Description,
			Width:          item.Width,
			Height:         item.Height,
			Quality:        item.Ratio,
			PlayCount:      int64(item.PlayCount),
			Private:        item.PrivateItem,
			CoverURLs:      media.UniqueStrings([]string{item.CoverURL, item.OriginCoverURL, item.DynamicCoverURL}),
			AuthorUniqueID: author,
		})
		videoURLs = append(videoURLs, canonicalURL)
	}

	return &Result{
		InputURL:  inputURL,
		FinalURL:  finalURL,
		FetchedAt: time.Now().UTC(),
		User:      user,
		Listing: Listing{
			ReturnedCount: len(videos),
			TotalCount:    user.Statistics.VideoCount,
			LatestOnly:    true,
		},
		VideoURLs: videoURLs,
		Videos:    videos,
	}
}

func mergeHydrationUser(target *User, source hydrationUserInfo) {
	user := source.User
	if user.ID != "" {
		target.ID = user.ID
	}
	if user.SecUID != "" {
		target.SecUID = user.SecUID
	}
	if user.UniqueID != "" {
		target.UniqueID = user.UniqueID
	}
	if user.Nickname != "" {
		target.Nickname = user.Nickname
	}
	if user.Signature != "" {
		target.Signature = user.Signature
	}
	target.Verified = user.Verified
	target.PrivateAccount = user.PrivateAccount
	target.AvatarURLs = media.UniqueStrings(append(
		[]string{user.AvatarThumb, user.AvatarMedium, user.AvatarLarger}, target.AvatarURLs...,
	))
	target.BioLink = user.BioLink.Link
	stats := source.StatsV2
	if allStatisticsZero(stats) {
		stats = source.Stats
	}
	target.Statistics = UserStatistics{
		FollowingCount: int64(stats.FollowingCount),
		FollowerCount:  int64(stats.FollowerCount),
		HeartCount:     int64(stats.HeartCount),
		VideoCount:     int64(stats.VideoCount),
		DiggCount:      int64(stats.DiggCount),
		FriendCount:    int64(stats.FriendCount),
	}
}

func allStatisticsZero(stats hydrationStatistics) bool {
	return stats.FollowingCount == 0 && stats.FollowerCount == 0 && stats.HeartCount == 0 &&
		stats.VideoCount == 0 && stats.DiggCount == 0 && stats.FriendCount == 0
}

func usernameFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	path := strings.Trim(parsed.EscapedPath(), "/")
	if !strings.HasPrefix(path, "@") || strings.Contains(strings.TrimPrefix(path, "@"), "/") {
		return ""
	}
	username, err := url.PathUnescape(strings.TrimPrefix(path, "@"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(username)
}
