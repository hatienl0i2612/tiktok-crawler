package photo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

var embedStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__FRONTITY_CONNECT_STATE__["'][^>]*>(.*?)</script\s*>`)

func (client *Client) fetchEmbedPhoto(ctx context.Context, itemID, referer string) (embedPhoto, error) {
	target := "https://www.tiktok.com/embed/v2/" + url.PathEscape(itemID)
	body, _, err := client.session.Fetch(ctx, target, "text/html,application/xhtml+xml", referer, maxMetadataResponseSize)
	if err != nil {
		return embedPhoto{}, fmt.Errorf("fetch embed page: %w", err)
	}
	match := embedStatePattern.FindSubmatch(body)
	if len(match) != 2 {
		return embedPhoto{}, errors.New("Frontity state was not found in the embed page")
	}
	var state embedState
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return embedPhoto{}, fmt.Errorf("decode embed metadata: %w", err)
	}
	for _, page := range state.Source.Data {
		if page.VideoData.Item.ID != itemID {
			continue
		}
		if page.IsError || (page.Code != 0 && page.Code != http.StatusOK) {
			return embedPhoto{}, fmt.Errorf("embed page returned code %d", page.Code)
		}
		return page.VideoData, nil
	}
	return embedPhoto{}, errors.New("embed page did not contain the requested photo")
}

func buildEmbedResult(inputURL string, source embedPhoto, isStory bool) *Result {
	result := &Result{
		InputURL:  inputURL,
		FinalURL:  inputURL,
		Sources:   []string{"embed/v2"},
		FetchedAt: time.Now().UTC(),
		IsStory:   isStory,
		Post:      video.Video{ID: source.Item.ID},
		Images:    makeEmbedImages(source.ImagePostInfo.DisplayImages),
	}
	mergeEmbedMetadata(&result.Post, source)
	return result
}

func mergeEmbedMetadata(post *video.Video, source embedPhoto) {
	item := source.Item
	if item.Text != "" {
		post.Description = item.Text
	}
	if item.LocationCreated != "" {
		post.Region = item.LocationCreated
	}
	if unixTime, err := strconv.ParseInt(item.CreateTime, 10, 64); err == nil && unixTime > 0 {
		createdAt := time.Unix(unixTime, 0).UTC()
		post.CreatedAt = &createdAt
	}
	post.Statistics.PlayCount = item.PlayCount
	post.Statistics.LikeCount = maxInt64(post.Statistics.LikeCount, item.DiggCount)
	post.Statistics.CommentCount = maxInt64(post.Statistics.CommentCount, item.CommentCount)
	post.Statistics.ShareCount = maxInt64(post.Statistics.ShareCount, item.ShareCount)

	author := source.Author
	if item.AuthorID != "" {
		post.Author.ID = item.AuthorID
	} else if author.UserID != "" {
		post.Author.ID = author.UserID
	}
	if author.SecUID != "" {
		post.Author.SecUID = author.SecUID
	}
	if author.UniqueID != "" {
		post.Author.UniqueID = author.UniqueID
	}
	if author.Nickname != "" {
		post.Author.Nickname = author.Nickname
	}
	if author.Signature != "" {
		post.Author.Signature = author.Signature
	}
	post.Author.Verified = author.Verified
	post.Author.AvatarURLs = media.UniqueStrings(append(
		append(append(append([]string{}, post.Author.AvatarURLs...), author.Covers...), author.CoversMedium...),
		author.CoversLarger...,
	))
	stats := source.AuthorStats
	post.Author.Statistics = video.AuthorStatistics{
		FollowingCount: maxInt64(post.Author.Statistics.FollowingCount, int64(stats.FollowingCount)),
		FollowerCount:  maxInt64(post.Author.Statistics.FollowerCount, int64(stats.FollowerCount)),
		HeartCount:     maxInt64(post.Author.Statistics.HeartCount, int64(stats.HeartCount)),
		VideoCount:     maxInt64(post.Author.Statistics.VideoCount, int64(stats.VideoCount)),
		DiggCount:      maxInt64(post.Author.Statistics.DiggCount, int64(stats.DiggCount)),
	}

	music := source.Music
	mergeMusic(&post.Music, video.Music{
		ID:           music.MusicID,
		Title:        music.MusicName,
		Author:       music.AuthorName,
		PlaybackURLs: music.PlayURL,
		CoverURLs: media.UniqueStrings(append(
			append(append([]string{}, music.Covers...), music.CoversMedium...),
			music.CoversLarger...,
		)),
	})
	post.Covers.Default = media.UniqueStrings(append(post.Covers.Default, item.Covers...))
	post.Covers.Original = media.UniqueStrings(append(post.Covers.Original, item.CoversOrigin...))
	post.Covers.Dynamic = media.UniqueStrings(append(post.Covers.Dynamic, item.CoversDynamic...))
}

func makeEmbedImages(source []embedImage) []Image {
	images := make([]Image, 0, len(source))
	for index, image := range source {
		variants := appendImageVariant(nil, imageAddress{
			Height:  image.Height,
			Width:   image.Width,
			URLList: image.URLList,
		}, false, "display")
		if len(variants) == 0 {
			continue
		}
		media.Sort(variants)
		images = append(images, Image{Index: index + 1, Media: variants})
	}
	return images
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
