package photo

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

	"github.com/hatienl0i2612/tiktok-crawler/internal/jsonvalue"
)

var universalStatePattern = regexp.MustCompile(`(?is)<script\b[^>]*\bid\s*=\s*["']__UNIVERSAL_DATA_FOR_REHYDRATION__["'][^>]*>(.*?)</script\s*>`)

type pagePhotoItem struct {
	ID              string          `json:"id"`
	Description     string          `json:"desc"`
	CreateTime      jsonvalue.Int64 `json:"createTime"`
	LocationCreated string          `json:"locationCreated"`
	Author          struct {
		ID           string `json:"id"`
		SecUID       string `json:"secUid"`
		UniqueID     string `json:"uniqueId"`
		Nickname     string `json:"nickname"`
		Signature    string `json:"signature"`
		Verified     bool   `json:"verified"`
		AvatarThumb  string `json:"avatarThumb"`
		AvatarMedium string `json:"avatarMedium"`
		AvatarLarger string `json:"avatarLarger"`
	} `json:"author"`
	AuthorStats embedAuthorStatistics `json:"authorStats"`
	Stats       struct {
		PlayCount    jsonvalue.Int64 `json:"playCount"`
		DiggCount    jsonvalue.Int64 `json:"diggCount"`
		CommentCount jsonvalue.Int64 `json:"commentCount"`
		ShareCount   jsonvalue.Int64 `json:"shareCount"`
	} `json:"stats"`
	Music struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		AuthorName  string `json:"authorName"`
		PlayURL     string `json:"playUrl"`
		CoverThumb  string `json:"coverThumb"`
		CoverMedium string `json:"coverMedium"`
		CoverLarge  string `json:"coverLarge"`
	} `json:"music"`
	ImagePost struct {
		Images []pageImage `json:"images"`
		Cover  pageImage   `json:"cover"`
	} `json:"imagePost"`
}

type pageImage struct {
	Width   int `json:"imageWidth"`
	Height  int `json:"imageHeight"`
	Address struct {
		URLList []string `json:"urlList"`
	} `json:"imageURL"`
}

func (client *Client) fetchPagePhoto(ctx context.Context, inputURL *url.URL, photoID string) (*Result, error) {
	// The /photo page is client-rendered; /video/<same ID> exposes its photo hydration data.
	target := *inputURL
	target.Path = strings.Replace(target.Path, "/photo/", "/video/", 1)
	target.RawPath = ""
	body, _, err := client.session.Fetch(ctx, target.String(), "text/html,application/xhtml+xml", inputURL.String(), maxMetadataResponseSize)
	if err != nil {
		return nil, fmt.Errorf("fetch Photo Post page: %w", err)
	}
	match := universalStatePattern.FindSubmatch(body)
	if len(match) != 2 {
		return nil, errors.New("Photo Post page hydration state was not found")
	}
	var state struct {
		Scope struct {
			Detail struct {
				StatusCode int `json:"statusCode"`
				ItemInfo   struct {
					Item *pagePhotoItem `json:"itemStruct"`
				} `json:"itemInfo"`
			} `json:"webapp.video-detail"`
		} `json:"__DEFAULT_SCOPE__"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(match[1]), &state); err != nil {
		return nil, fmt.Errorf("decode Photo Post page metadata: %w", err)
	}
	detail := state.Scope.Detail
	if detail.StatusCode != 0 {
		return nil, fmt.Errorf("Photo Post page returned status %d", detail.StatusCode)
	}
	item := detail.ItemInfo.Item
	if item == nil || item.ID != photoID {
		return nil, errors.New("Photo Post page did not contain the requested photo")
	}
	source := embedPhoto{
		Item: embedItem{
			ID: item.ID, Text: item.Description, CreateTime: strconv.FormatInt(int64(item.CreateTime), 10), LocationCreated: item.LocationCreated,
			AuthorID: item.Author.ID, PlayCount: int64(item.Stats.PlayCount), DiggCount: int64(item.Stats.DiggCount),
			CommentCount: int64(item.Stats.CommentCount), ShareCount: int64(item.Stats.ShareCount), Covers: item.ImagePost.Cover.Address.URLList,
		},
		Author: embedAuthor{
			UserID: item.Author.ID, SecUID: item.Author.SecUID, UniqueID: item.Author.UniqueID, Nickname: item.Author.Nickname,
			Signature: item.Author.Signature, Verified: item.Author.Verified,
			Covers: []string{item.Author.AvatarThumb}, CoversMedium: []string{item.Author.AvatarMedium}, CoversLarger: []string{item.Author.AvatarLarger},
		},
		AuthorStats: item.AuthorStats,
		Music: embedMusic{
			MusicID: item.Music.ID, MusicName: item.Music.Title, AuthorName: item.Music.AuthorName, PlayURL: []string{item.Music.PlayURL},
			Covers: []string{item.Music.CoverThumb}, CoversMedium: []string{item.Music.CoverMedium}, CoversLarger: []string{item.Music.CoverLarge},
		},
	}
	for _, image := range item.ImagePost.Images {
		source.ImagePostInfo.DisplayImages = append(source.ImagePostInfo.DisplayImages, embedImage{
			Width: image.Width, Height: image.Height, URLList: image.Address.URLList,
		})
	}
	result := buildEmbedResult(inputURL.String(), source, false)
	result.Sources = []string{"__UNIVERSAL_DATA_FOR_REHYDRATION__"}
	if len(result.Images) == 0 {
		return nil, errors.New("Photo Post page contained no downloadable images")
	}
	return result, nil
}
