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

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

var (
	frontityStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__FRONTITY_CONNECT_STATE__["'][^>]*>(.*?)</script\s*>`)
	jsonScriptPattern    = regexp.MustCompile(`(?is)<script\b[^>]*>(.*?)</script\s*>`)
	digitsPattern        = regexp.MustCompile(`^[0-9]+$`)
)

const maxVideoPageAttempts = 3

// Resolve crawls a TikTok video page and returns normalized metadata and media variants.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	inputURL, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	videoID := videoIDFromURL(inputURL)
	pageMediaURLs, fetchedURL, pageErr := client.fetchPageMedia(ctx, inputURL.String())
	pageAttempts := 1
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
	for len(pageMediaURLs.Download) == 0 && pageAttempts < maxVideoPageAttempts {
		pageAttempts++
		retryMediaURLs, retryURL, retryErr := client.fetchPageMedia(ctx, finalURL)
		if retryErr != nil {
			continue
		}
		pageErr = nil
		if parsedRetry, parseErr := url.Parse(retryURL); parseErr == nil && videoIDFromURL(parsedRetry) == videoID {
			finalURL = retryURL
		}
		pageMediaURLs = mergeExtractedMediaURLs(retryMediaURLs, pageMediaURLs)
	}

	player, playerBody, err := client.fetchPlayer(ctx, videoID, finalURL)
	if err != nil {
		return nil, err
	}
	if len(player.Items) == 0 {
		// TikTok occasionally emits a signed Story URL that always returns 403.
		// Collect independently signed page URLs so the downloader can fall back
		// without silently switching to the watermarked embed object.
		for pageAttempts < maxVideoPageAttempts {
			pageAttempts++
			retryMediaURLs, retryURL, retryErr := client.fetchPageMedia(ctx, finalURL)
			if retryErr != nil {
				continue
			}
			pageErr = nil
			if parsedRetry, parseErr := url.Parse(retryURL); parseErr == nil && videoIDFromURL(parsedRetry) == videoID {
				finalURL = retryURL
			}
			pageMediaURLs = mergeExtractedMediaURLs(retryMediaURLs, pageMediaURLs)
		}
		embed, embedErr := client.fetchEmbed(ctx, videoID, finalURL)
		if embedErr != nil {
			return nil, fmt.Errorf("TikTok player returned no video item and the Story embed fallback failed: %w", embedErr)
		}
		result := buildEmbedResult(inputURL.String(), finalURL, embed, true)
		if pageErr != nil {
			result.Warnings = append(result.Warnings, "video page unavailable: "+pageErr.Error())
		}
		if len(pageMediaURLs.Playback) > 0 || len(pageMediaURLs.Download) > 0 {
			result.Sources = append(result.Sources, "video page")
		}
		appendPlaybackMedia(result, pageMediaURLs.Playback)
		watermarkedURLs := append(append([]string{}, pageMediaURLs.Download...), embed.ItemInfos.Video.URLs...)
		appendWatermarkedMedia(result, uniqueStrings(watermarkedURLs))
		sortMedia(result.Media)
		if len(result.Media) == 0 {
			return nil, errors.New("TikTok returned Story metadata but no downloadable video")
		}
		return result, nil
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
	if len(pageMediaURLs.Download) > 0 {
		result.Sources = append(result.Sources, "video page")
		downloadURLs = append(downloadURLs, pageMediaURLs.Download...)
	}
	appendWatermarkedMedia(result, uniqueStrings(downloadURLs))
	sortMedia(result.Media)
	if len(result.Media) == 0 {
		return nil, errors.New("TikTok returned metadata but no downloadable media profiles")
	}
	return result, nil
}

func (client *Client) fetchPageMedia(ctx context.Context, target string) (extractedMediaURLs, string, error) {
	body, finalURL, err := client.fetchMetadata(
		ctx,
		target,
		"text/html,application/xhtml+xml",
		"",
	)
	return extractMediaURLsFromHTML(body), finalURL, err
}

func (client *Client) fetchPlayer(
	ctx context.Context,
	videoID string,
	referer string,
) (playerResponse, []byte, error) {
	body, err := tiktok.FetchPlayerItem(ctx, client.session, videoID, referer, maxMetadataResponseSize)
	if err != nil {
		return playerResponse{}, nil, err
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
		return response, body, nil
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
	return fetchEmbedMetadata(ctx, client.session, videoID, referer)
}

// FetchEmbedMusic resolves the audio metadata exposed by TikTok's embed page.
// It is useful for content types such as Photo Posts whose Player API response
// includes music labels but omits the signed audio URL.
func FetchEmbedMusic(
	ctx context.Context,
	session *tiktok.Session,
	itemID string,
	referer string,
) (Music, error) {
	embed, err := fetchEmbedMetadata(ctx, session, itemID, referer)
	if err != nil {
		return Music{}, err
	}
	music := embed.MusicInfos
	return Music{
		ID:           music.MusicID,
		Title:        music.MusicName,
		Author:       music.AuthorName,
		PlaybackURLs: uniqueStrings(music.PlayURL),
		CoverURLs: uniqueStrings(append(
			append(append([]string{}, music.Covers...), music.CoversMedium...),
			music.CoversLarger...,
		)),
	}, nil
}

func fetchEmbedMetadata(
	ctx context.Context,
	session *tiktok.Session,
	videoID string,
	referer string,
) (embedVideoData, error) {
	if session == nil {
		return embedVideoData{}, errors.New("TikTok session is not configured")
	}
	target := "https://www.tiktok.com/embed/v2/" + url.PathEscape(videoID)
	body, _, err := session.Fetch(ctx, target, "text/html,application/xhtml+xml", referer, maxMetadataResponseSize)
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

func buildEmbedResult(inputURL, finalURL string, source embedVideoData, isStory bool) *Result {
	result := &Result{
		InputURL:  inputURL,
		FinalURL:  finalURL,
		Sources:   []string{"embed/v2"},
		FetchedAt: time.Now().UTC(),
		IsStory:   isStory,
		Video:     Video{ID: source.ItemInfos.ID},
	}
	mergeEmbedMetadata(&result.Video, source)
	return result
}

func appendPlaybackMedia(result *Result, urls []string) {
	urls = uniqueStrings(urls)
	if len(urls) == 0 {
		return
	}
	result.Media = append(result.Media, Media{
		Type:        "video",
		Kind:        "playback",
		Watermarked: false,
		Codec:       "h264",
		Format:      "mp4",
		Quality:     qualityName(result.Video.Height),
		Width:       result.Video.Width,
		Height:      result.Video.Height,
		URL:         urls[0],
		BackupURLs:  urls[1:],
		ExpiresAt:   expiryFromURL(urls[0]),
	})
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
			Type:        "video",
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
		Type:        "video",
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
		Type:        "video",
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

type extractedMediaURLs struct {
	Playback []string
	Download []string
}

func mergeExtractedMediaURLs(preferred, fallback extractedMediaURLs) extractedMediaURLs {
	return extractedMediaURLs{
		Playback: uniqueStrings(append(preferred.Playback, fallback.Playback...)),
		Download: uniqueStrings(append(preferred.Download, fallback.Download...)),
	}
}

type mediaURLKind uint8

const (
	mediaURLNone mediaURLKind = iota
	mediaURLPlayback
	mediaURLDownload
)

func extractMediaURLsFromHTML(body []byte) extractedMediaURLs {
	var result extractedMediaURLs
	for _, match := range jsonScriptPattern.FindAllSubmatch(body, -1) {
		if len(match) != 2 {
			continue
		}
		content := bytes.TrimSpace(match[1])
		if len(content) == 0 || (content[0] != '{' && content[0] != '[') {
			continue
		}
		extracted := extractMediaURLs(content)
		result.Playback = append(result.Playback, extracted.Playback...)
		result.Download = append(result.Download, extracted.Download...)
	}
	result.Playback = uniqueStrings(result.Playback)
	result.Download = uniqueStrings(result.Download)
	return result
}

func extractDownloadURLsFromHTML(body []byte) []string {
	return extractMediaURLsFromHTML(body).Download
}

func extractDownloadURLs(body []byte) []string {
	return extractMediaURLs(body).Download
}

func extractMediaURLs(body []byte) extractedMediaURLs {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return extractedMediaURLs{}
	}
	var result extractedMediaURLs
	walkMediaFields(value, mediaURLNone, &result)
	result.Playback = uniqueStrings(result.Playback)
	result.Download = uniqueStrings(result.Download)
	sort.Strings(result.Playback)
	sort.Strings(result.Download)
	return result
}

func walkMediaFields(value any, kind mediaURLKind, result *extractedMediaURLs) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
			childKind := kind
			switch normalized {
			case "playaddr":
				childKind = mediaURLPlayback
			case "downloadaddr", "downloadurl":
				childKind = mediaURLDownload
			}
			walkMediaFields(child, childKind, result)
		}
	case []any:
		for _, child := range typed {
			walkMediaFields(child, kind, result)
		}
	case string:
		if kind == mediaURLNone {
			return
		}
		parsed, err := url.Parse(strings.TrimSpace(typed))
		if err == nil && parsed.Scheme == "https" && tiktok.IsAllowedHost(parsed.Hostname()) {
			if kind == mediaURLPlayback {
				result.Playback = append(result.Playback, parsed.String())
			} else {
				result.Download = append(result.Download, parsed.String())
			}
		}
	}
}

func qualityName(height int) string {
	return media.QualityName(height)
}

func normalizeCodec(codec string) string {
	return tiktok.NormalizeCodec(codec)
}

func expiryFromURL(rawURL string) *time.Time {
	return tiktok.ExpiryFromURL(rawURL)
}

func uniqueStrings(values []string) []string {
	return media.UniqueStrings(values)
}

func maxInt64(left, right int64) int64 {
	if right > left {
		return right
	}
	return left
}

func sortMedia(variants []Media) {
	media.Sort(variants)
}
