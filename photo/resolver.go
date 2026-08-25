package photo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

var photoPathPattern = regexp.MustCompile(`^/@[^/]+/photo/([0-9]+)/?$`)

// Resolve crawls a TikTok Photo Post and returns its images in post order.
func (client *Client) Resolve(ctx context.Context, rawURL string) (*Result, error) {
	if client == nil || client.session == nil {
		return nil, errors.New("TikTok Photo Post client is not configured")
	}
	inputURL, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	photoID := photoIDFromURL(inputURL)
	if photoID == "" {
		return nil, errors.New("URL must contain a TikTok /photo/<id> path")
	}
	body, err := tiktok.FetchPlayerItem(ctx, client.session, photoID, inputURL.String(), maxMetadataResponseSize)
	if err != nil {
		return nil, err
	}
	var response playerResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode player metadata: %w", err)
	}
	if response.StatusCode != 0 {
		return nil, fmt.Errorf("TikTok player returned status %d: %s", response.StatusCode, response.StatusMsg)
	}
	if len(response.Items) == 0 {
		return nil, errors.New("TikTok player returned no Photo Post item")
	}
	item := response.Items[0]
	resolvedID := item.IDStr
	if resolvedID == "" {
		resolvedID = item.ID.String()
	}
	if resolvedID != "" && resolvedID != photoID {
		return nil, fmt.Errorf("TikTok player returned Photo Post %s instead of %s", resolvedID, photoID)
	}
	images := makeImages(item.ImagePostInfo.Images)
	if len(images) == 0 {
		return nil, errors.New("TikTok returned metadata but no downloadable Photo Post images")
	}
	result := &Result{
		InputURL:  inputURL.String(),
		FinalURL:  inputURL.String(),
		Sources:   []string{"player/api/v1/items"},
		FetchedAt: time.Now().UTC(),
		Post: video.Video{
			ID:          resolvedID,
			Description: item.Description,
			Region:      item.Region,
			Author: video.Author{
				SecUID:     item.AuthorInfo.SecretID,
				UniqueID:   item.AuthorInfo.UniqueID,
				Nickname:   item.AuthorInfo.Nickname,
				AvatarURLs: media.UniqueStrings(item.AuthorInfo.AvatarURLs),
			},
			Statistics: video.Statistics{
				LikeCount:    item.StatisticsInfo.DiggCount,
				CommentCount: item.StatisticsInfo.CommentCount,
				ShareCount:   item.StatisticsInfo.ShareCount,
			},
			Music: video.Music{ID: item.MusicInfo.IDStr, Title: item.MusicInfo.Title, Author: item.MusicInfo.Author},
		},
		Images: images,
	}
	if music, embedErr := video.FetchEmbedMusic(ctx, client.session, photoID, inputURL.String()); embedErr == nil {
		mergeMusic(&result.Post.Music, music)
		result.Sources = append(result.Sources, "embed/v2")
	} else {
		result.Warnings = append(result.Warnings, "embed music metadata unavailable: "+embedErr.Error())
	}
	return result, nil
}

func mergeMusic(target *video.Music, source video.Music) {
	if source.ID != "" {
		target.ID = source.ID
	}
	if source.Title != "" {
		target.Title = source.Title
	}
	if source.Author != "" {
		target.Author = source.Author
	}
	target.PlaybackURLs = media.UniqueStrings(append(target.PlaybackURLs, source.PlaybackURLs...))
	target.CoverURLs = media.UniqueStrings(append(target.CoverURLs, source.CoverURLs...))
}

func photoIDFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	matches := photoPathPattern.FindStringSubmatch(parsed.EscapedPath())
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

func makeImages(source []postImage) []Image {
	images := make([]Image, 0, len(source))
	for index, item := range source {
		variants := makeImageVariants(item)
		if len(variants) == 0 {
			continue
		}
		images = append(images, Image{Index: index + 1, Media: variants})
	}
	return images
}

func makeImageVariants(image postImage) []media.Variant {
	variants := appendImageVariant(nil, image.DisplayImage, false, "display")
	variants = appendImageVariant(variants, image.OwnerWatermarkImage, true, "owner_watermark")
	variants = appendImageVariant(variants, image.UserWatermarkImage, true, "user_watermark")
	variants = appendImageVariant(variants, image.WatermarkImage, true, "watermark")
	media.Sort(variants)
	return variants
}

func appendImageVariant(variants []media.Variant, address imageAddress, watermarked bool, kind string) []media.Variant {
	urls := media.UniqueStrings(address.URLList)
	if len(urls) == 0 {
		return variants
	}
	primaryURL := urls[0]
	for _, current := range variants {
		if current.Watermarked == watermarked && current.URL == primaryURL {
			return variants
		}
	}
	return append(variants, media.Variant{
		Type:        "image",
		Kind:        kind,
		Watermarked: watermarked,
		Codec:       "image",
		Format:      imageFormat(primaryURL),
		Quality:     media.QualityName(address.Height),
		Width:       address.Width,
		Height:      address.Height,
		Size:        address.DataSize,
		URI:         address.URI,
		URL:         primaryURL,
		BackupURLs:  urls[1:],
		ExpiresAt:   tiktok.ExpiryFromURL(primaryURL),
	})
}

func imageFormat(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if extension := strings.TrimPrefix(strings.ToLower(path.Ext(parsed.Path)), "."); extension != "" {
			return extension
		}
	}
	return "jpeg"
}
