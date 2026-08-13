package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	frontityStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__FRONTITY_CONNECT_STATE__["'][^>]*>(.*?)</script\s*>`)
	jsonScriptPattern    = regexp.MustCompile(`(?is)<script\b[^>]*>(.*?)</script\s*>`)
	digitsPattern        = regexp.MustCompile(`^[0-9]+$`)
)

const maxVideoPageAttempts = 3

// Resolve crawls a TikTok video page and returns normalized metadata and media variants.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	inputURL, err := parseTikTokURL(rawURL)
	if err != nil {
		return nil, err
	}

	videoID := videoIDFromURL(inputURL)
	pageBody, fetchedURL, pageErr := client.fetchMetadata(
		ctx,
		inputURL.String(),
		"text/html,application/xhtml+xml",
		"",
	)
	finalURL := inputURL.String()
	if fetchedURL != "" {
		if parsedFinal, parseErr := url.Parse(fetchedURL); parseErr == nil {
			if redirectedID := videoIDFromURL(parsedFinal); redirectedID != "" {
				finalURL = fetchedURL
				if videoID == "" {
					videoID = redirectedID
				}
			}
		}
	}
	if videoID == "" {
		if pageErr != nil {
			return nil, fmt.Errorf("resolve video URL: %w", pageErr)
		}
		return nil, errors.New("URL must contain a TikTok /video/<id> path or redirect to one")
	}
	pageDownloadURLs := extractDownloadURLsFromHTML(pageBody)
	for attempt := 1; len(pageDownloadURLs) == 0 && attempt < maxVideoPageAttempts; attempt++ {
		retryBody, retryURL, retryErr := client.fetchMetadata(
			ctx,
			finalURL,
			"text/html,application/xhtml+xml",
			"",
		)
		if retryErr != nil {
			continue
		}
		pageErr = nil
		pageBody = retryBody
		if parsedRetry, parseErr := url.Parse(retryURL); parseErr == nil && videoIDFromURL(parsedRetry) == videoID {
			finalURL = retryURL
		}
		pageDownloadURLs = extractDownloadURLsFromHTML(pageBody)
	}

	player, playerBody, err := client.fetchPlayer(ctx, videoID, finalURL)
	if err != nil {
		return nil, err
	}
	result := buildPlayerResult(inputURL.String(), finalURL, player.Items[0])
	result.Sources = append(result.Sources, "player/api/v1/items")

	embed, embedErr := client.fetchEmbed(ctx, videoID, finalURL)
	if embedErr == nil {
		mergeEmbedMetadata(&result.Video, embed)
		result.Sources = append(result.Sources, "embed/v2")
	} else {
		result.Warnings = append(result.Warnings, "embed metadata unavailable: "+embedErr.Error())
	}
	if pageErr != nil {
		result.Warnings = append(result.Warnings, "video page unavailable: "+pageErr.Error())
	}

	downloadURLs := extractDownloadURLs(playerBody)
	if len(pageDownloadURLs) > 0 {
		result.Sources = append(result.Sources, "video page")
		downloadURLs = append(downloadURLs, pageDownloadURLs...)
	}
	appendWatermarkedMedia(result, uniqueStrings(downloadURLs))
	sortMedia(result.Media)
	if len(result.Media) == 0 {
		return nil, errors.New("TikTok returned metadata but no downloadable media profiles")
	}
	return result, nil
}

func (client *Client) fetchPlayer(
	ctx context.Context,
	videoID string,
	referer string,
) (playerResponse, []byte, error) {
	endpoint, _ := url.Parse("https://www.tiktok.com/player/api/v1/items")
	query := endpoint.Query()
	query.Set("item_ids", videoID)
	query.Set("language", "en")
	query.Set("aid", "1459")
	query.Set("data_source", "web_core")
	endpoint.RawQuery = query.Encode()

	body, _, err := client.fetchMetadata(
		ctx,
		endpoint.String(),
		"application/json, text/plain, */*",
		referer,
	)
	if err != nil {
		return playerResponse{}, nil, fmt.Errorf("fetch player metadata: %w", err)
	}
	var response playerResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return playerResponse{}, body, fmt.Errorf("decode player metadata: %w", err)
	}
	if response.StatusCode != 0 {
		return playerResponse{}, body, fmt.Errorf(
			"TikTok player returned status %d: %s",
			response.StatusCode,
			response.StatusMsg,
		)
	}
	if len(response.Items) == 0 {
		return playerResponse{}, body, errors.New("TikTok player returned no video item")
	}
	resolvedID := response.Items[0].IDStr
	if resolvedID == "" {
		resolvedID = response.Items[0].ID.String()
	}
	if resolvedID != "" && resolvedID != videoID {
		return playerResponse{}, body, fmt.Errorf(
			"TikTok player returned video %s instead of %s",
			resolvedID,
			videoID,
		)
	}
	return response, body, nil
}

func (client *Client) fetchEmbed(
	ctx context.Context,
	videoID string,
	referer string,
) (embedVideoData, error) {
	target := "https://www.tiktok.com/embed/v2/" + url.PathEscape(videoID)
	body, _, err := client.fetchMetadata(ctx, target, "text/html,application/xhtml+xml", referer)
	if err != nil {
		return embedVideoData{}, fmt.Errorf("fetch embed page: %w", err)
	}
	match := frontityStatePattern.FindSubmatch(body)
	if len(match) != 2 {
		return embedVideoData{}, errors.New("Frontity state was not found in the embed page")
	}
	var state frontityState
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return embedVideoData{}, fmt.Errorf("decode embed metadata: %w", err)
	}
	for _, page := range state.Source.Data {
		if page.VideoData.ItemInfos.ID != videoID {
			continue
		}
		if page.IsError || (page.Code != 0 && page.Code != http.StatusOK) {
			return embedVideoData{}, fmt.Errorf("embed page returned code %d", page.Code)
		}
		return page.VideoData, nil
	}
	return embedVideoData{}, errors.New("embed page did not contain the requested video")
}

func videoIDFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for index := 0; index+1 < len(parts); index++ {
		if !strings.EqualFold(parts[index], "video") {
			continue
		}
		videoID, err := url.PathUnescape(parts[index+1])
		if err == nil && digitsPattern.MatchString(videoID) {
			return videoID
		}
	}
	return ""
}

func buildPlayerResult(inputURL, finalURL string, item playerItem) *Result {
	videoID := item.IDStr
	if videoID == "" {
		videoID = item.ID.String()
	}
	result := &Result{
		InputURL:  inputURL,
		FinalURL:  finalURL,
		FetchedAt: time.Now().UTC(),
		Video: Video{
			ID:          videoID,
			Description: item.Description,
			Region:      item.Region,
			Duration:    item.VideoInfo.Meta.Duration,
			Width:       item.VideoInfo.Meta.Width,
			Height:      item.VideoInfo.Meta.Height,
			Author: Author{
				SecUID:     item.AuthorInfo.SecretID,
				UniqueID:   item.AuthorInfo.UniqueID,
				Nickname:   item.AuthorInfo.Nickname,
				AvatarURLs: uniqueStrings(item.AuthorInfo.AvatarURLs),
			},
			Statistics: Statistics{
				LikeCount:    item.StatisticsInfo.DiggCount,
				CommentCount: item.StatisticsInfo.CommentCount,
				ShareCount:   item.StatisticsInfo.ShareCount,
			},
			Music: Music{
				ID:     item.MusicInfo.IDStr,
				Title:  item.MusicInfo.Title,
				Author: item.MusicInfo.Author,
			},
			Covers: Covers{
				Default:  uniqueStrings(item.VideoInfo.Cover.URLList),
				Original: uniqueStrings(item.VideoInfo.OriginCover.URLList),
			},
			Captions: makeCaptions(item.VideoInfo.CLAInfo.CaptionInfos),
		},
	}
	result.Media = makePlaybackMedia(item.VideoInfo)
	return result
}

func makePlaybackMedia(video playerVideoInfo) []Media {
	media := make([]Media, 0, len(video.Profiles))
	for _, profile := range video.Profiles {
		if len(profile.PlayAddr.URLList) == 0 {
			continue
		}
		format := video.Meta.Format
		if format == "" {
			format = "mp4"
		}
		media = append(media, Media{
			Kind:        "playback",
			Watermarked: false,
			Codec:       normalizeCodec(profile.CodecType),
			Format:      strings.ToLower(format),
			Quality:     qualityName(profile.PlayAddr.Height),
			GearName:    profile.GearName,
			Width:       profile.PlayAddr.Width,
			Height:      profile.PlayAddr.Height,
			FPS:         profile.FPS,
			Bitrate:     profile.Bitrate,
			Size:        profile.PlayAddr.DataSize,
			URI:         profile.PlayAddr.URI,
			URL:         profile.PlayAddr.URLList[0],
			BackupURLs:  uniqueStrings(profile.PlayAddr.URLList[1:]),
			ExpiresAt:   expiryFromURL(profile.PlayAddr.URLList[0]),
		})
	}
	if len(media) > 0 || len(video.URLList) == 0 {
		return media
	}
	format := video.Meta.Format
	if format == "" {
		format = "mp4"
	}
	return []Media{{
		Kind:        "playback",
		Watermarked: false,
		Codec:       "h264",
		Format:      strings.ToLower(format),
		Quality:     qualityName(video.Meta.Height),
		Width:       video.Meta.Width,
		Height:      video.Meta.Height,
		Bitrate:     video.Meta.Bitrate,
		URI:         video.URI,
		URL:         video.URLList[0],
		BackupURLs:  uniqueStrings(video.URLList[1:]),
		ExpiresAt:   expiryFromURL(video.URLList[0]),
	}}
}

func makeCaptions(captions []playerCaption) []Caption {
	result := make([]Caption, 0, len(captions))
	for _, caption := range captions {
		captionURL := strings.TrimSpace(caption.URL)
		if captionURL == "" && len(caption.URLList) > 0 {
			captionURL = caption.URLList[0]
		}
		if captionURL == "" {
			continue
		}
		var expiresAt *time.Time
		if caption.Expire > 0 {
			value := time.Unix(caption.Expire, 0).UTC()
			expiresAt = &value
		}
		result = append(result, Caption{
			Language:      caption.Language,
			Format:        caption.CaptionFormat,
			URL:           captionURL,
			AutoGenerated: caption.IsAutoGenerated,
			Original:      caption.IsOriginalCaption,
			ExpiresAt:     expiresAt,
		})
	}
	return result
}

func mergeEmbedMetadata(video *Video, embed embedVideoData) {
	item := embed.ItemInfos
	if item.Text != "" {
		video.Description = item.Text
	}
	if item.LocationCreated != "" {
		video.Region = item.LocationCreated
	}
	if item.Video.VideoMeta.Duration > 0 {
		video.Duration = item.Video.VideoMeta.Duration
	}
	if item.Video.VideoMeta.Width > 0 {
		video.Width = item.Video.VideoMeta.Width
	}
	if item.Video.VideoMeta.Height > 0 {
		video.Height = item.Video.VideoMeta.Height
	}
	if unixTime, err := strconv.ParseInt(item.CreateTime, 10, 64); err == nil && unixTime > 0 {
		createdAt := time.Unix(unixTime, 0).UTC()
		video.CreatedAt = &createdAt
	}
	video.Statistics.PlayCount = item.PlayCount
	video.Statistics.LikeCount = maxInt64(video.Statistics.LikeCount, item.DiggCount)
	video.Statistics.CommentCount = maxInt64(video.Statistics.CommentCount, item.CommentCount)
	video.Statistics.ShareCount = maxInt64(video.Statistics.ShareCount, item.ShareCount)

	author := embed.AuthorInfos
	if item.AuthorID != "" {
		video.Author.ID = item.AuthorID
	} else if author.UserID != "" {
		video.Author.ID = author.UserID
	}
	if author.SecUID != "" {
		video.Author.SecUID = author.SecUID
	}
	if author.UniqueID != "" {
		video.Author.UniqueID = author.UniqueID
	}
	if author.Nickname != "" {
		video.Author.Nickname = author.Nickname
	}
	video.Author.Signature = author.Signature
	video.Author.Verified = author.Verified
	video.Author.AvatarURLs = uniqueStrings(append(
		append(append([]string{}, author.Covers...), author.CoversMedium...),
		author.CoversLarger...,
	))
	stats := embed.AuthorStats
	video.Author.Statistics = AuthorStatistics{
		FollowingCount: int64(stats.FollowingCount),
		FollowerCount:  int64(stats.FollowerCount),
		HeartCount:     int64(stats.HeartCount),
		VideoCount:     int64(stats.VideoCount),
		DiggCount:      int64(stats.DiggCount),
	}

	music := embed.MusicInfos
	if music.MusicID != "" {
		video.Music.ID = music.MusicID
	}
	if music.MusicName != "" {
		video.Music.Title = music.MusicName
	}
	if music.AuthorName != "" {
		video.Music.Author = music.AuthorName
	}
	video.Music.PlaybackURLs = uniqueStrings(music.PlayURL)
	video.Music.CoverURLs = uniqueStrings(append(
		append(append([]string{}, music.Covers...), music.CoversMedium...),
		music.CoversLarger...,
	))
	video.Covers.Default = uniqueStrings(append(video.Covers.Default, item.Covers...))
	video.Covers.Original = uniqueStrings(append(video.Covers.Original, item.CoversOrigin...))
	video.Covers.Dynamic = uniqueStrings(item.CoversDynamic)

	video.Hashtags = make([]Hashtag, 0, len(embed.ChallengeInfoList))
	for _, challenge := range embed.ChallengeInfoList {
		if challenge.ChallengeName == "" {
			continue
		}
		video.Hashtags = append(video.Hashtags, Hashtag{
			ID:         challenge.ChallengeID,
			Name:       challenge.ChallengeName,
			Text:       challenge.Text,
			IsCommerce: challenge.IsCommerce,
		})
	}
}

func appendWatermarkedMedia(result *Result, urls []string) {
	if len(urls) == 0 {
		return
	}
	result.Media = append(result.Media, Media{
		Kind:        "download",
		Watermarked: true,
		Codec:       "unknown",
		Format:      "mp4",
		Quality:     qualityName(result.Video.Height),
		Width:       result.Video.Width,
		Height:      result.Video.Height,
		URL:         urls[0],
		BackupURLs:  urls[1:],
		ExpiresAt:   expiryFromURL(urls[0]),
	})
}

func extractDownloadURLsFromHTML(body []byte) []string {
	var result []string
	for _, match := range jsonScriptPattern.FindAllSubmatch(body, -1) {
		if len(match) != 2 {
			continue
		}
		content := bytes.TrimSpace(match[1])
		if len(content) == 0 || (content[0] != '{' && content[0] != '[') {
			continue
		}
		result = append(result, extractDownloadURLs(content)...)
	}
	return uniqueStrings(result)
}

func extractDownloadURLs(body []byte) []string {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	var result []string
	walkDownloadFields(value, false, &result)
	return uniqueStrings(result)
}

func walkDownloadFields(value any, collect bool, result *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
			isDownloadField := normalized == "downloadaddr" || normalized == "downloadurl"
			walkDownloadFields(child, collect || isDownloadField, result)
		}
	case []any:
		for _, child := range typed {
			walkDownloadFields(child, collect, result)
		}
	case string:
		if !collect {
			return
		}
		parsed, err := url.Parse(strings.TrimSpace(typed))
		if err == nil && parsed.Scheme == "https" && isAllowedTikTokHost(parsed.Hostname()) {
			*result = append(*result, parsed.String())
		}
	}
}

func qualityName(height int) string {
	if height <= 0 {
		return "unknown"
	}
	return strconv.Itoa(height) + "p"
}

func normalizeCodec(codec string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))
	switch {
	case strings.Contains(codec, "265"), strings.Contains(codec, "hevc"), strings.Contains(codec, "bytevc1"):
		return "h265"
	case strings.Contains(codec, "264"), strings.Contains(codec, "avc"):
		return "h264"
	case codec == "":
		return "unknown"
	default:
		return codec
	}
}

func expiryFromURL(rawURL string) *time.Time {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	for _, key := range []string{"expire", "expires", "x-expires", "x-m-expire"} {
		value := parsed.Query().Get(key)
		if value == "" {
			continue
		}
		unixTime, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			continue
		}
		if unixTime > 1_000_000_000_000 {
			unixTime /= 1000
		}
		expiresAt := time.Unix(unixTime, 0).UTC()
		return &expiresAt
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func maxInt64(left, right int64) int64 {
	if right > left {
		return right
	}
	return left
}

func sortMedia(media []Media) {
	sort.SliceStable(media, func(left, right int) bool {
		if media[left].Watermarked != media[right].Watermarked {
			return !media[left].Watermarked
		}
		leftPixels := media[left].Width * media[left].Height
		rightPixels := media[right].Width * media[right].Height
		if leftPixels != rightPixels {
			return leftPixels > rightPixels
		}
		if media[left].Bitrate != media[right].Bitrate {
			return media[left].Bitrate > media[right].Bitrate
		}
		if media[left].Codec != media[right].Codec {
			return media[left].Codec < media[right].Codec
		}
		return media[left].URL < media[right].URL
	})
}
