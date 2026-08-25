package shortdrama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

type shortDramaEpisodeResponse struct {
	StatusCode    int              `json:"statusCode"`
	StatusCodeAlt int              `json:"status_code"`
	StatusMessage string           `json:"statusMsg"`
	StatusMsgAlt  string           `json:"status_msg"`
	ItemList      []shortDramaItem `json:"itemList"`
}

func (response shortDramaEpisodeResponse) statusCode() int {
	if response.StatusCode != 0 {
		return response.StatusCode
	}
	return response.StatusCodeAlt
}

func (response shortDramaEpisodeResponse) statusMessage() string {
	return firstNonEmpty(response.StatusMessage, response.StatusMsgAlt)
}

type shortDramaItemDetailResponse struct {
	StatusCode    int    `json:"statusCode"`
	StatusCodeAlt int    `json:"status_code"`
	StatusMessage string `json:"statusMsg"`
	StatusMsgAlt  string `json:"status_msg"`
	ItemInfo      struct {
		ItemStruct shortDramaItem `json:"itemStruct"`
	} `json:"itemInfo"`
}

func (response shortDramaItemDetailResponse) statusCode() int {
	if response.StatusCode != 0 {
		return response.StatusCode
	}
	return response.StatusCodeAlt
}

func (response shortDramaItemDetailResponse) statusMessage() string {
	return firstNonEmpty(response.StatusMessage, response.StatusMsgAlt)
}

type shortDramaItem struct {
	ID          string           `json:"id"`
	Description string           `json:"desc"`
	Author      shortDramaAuthor `json:"author"`
	Stats       shortDramaStats  `json:"stats"`
	Video       shortDramaVideo  `json:"video"`
	DramaInfo   shortDramaInfo   `json:"dramaInfo"`
}

type shortDramaAuthor struct {
	ID           string `json:"id"`
	SecUID       string `json:"secUid"`
	UniqueID     string `json:"uniqueId"`
	Nickname     string `json:"nickname"`
	Signature    string `json:"signature"`
	Verified     bool   `json:"verified"`
	AvatarLarger string `json:"avatarLarger"`
	AvatarMedium string `json:"avatarMedium"`
	AvatarThumb  string `json:"avatarThumb"`
}

type shortDramaStats struct {
	PlayCount    int64 `json:"playCount"`
	DiggCount    int64 `json:"diggCount"`
	CommentCount int64 `json:"commentCount"`
	ShareCount   int64 `json:"shareCount"`
}

type shortDramaInfo struct {
	DramaID       string `json:"dramaID"`
	DramaName     string `json:"dramaName"`
	Description   string `json:"description"`
	NumVideos     int    `json:"numVideos"`
	IsLimitedFree bool   `json:"isLimitedFree"`
	VideoData     struct {
		EpisodeNumber         int    `json:"EpisodeNumber"`
		TranslatedDescription string `json:"TranslatedDescription"`
	} `json:"DramaVideoData"`
}

type shortDramaVideo struct {
	Cover        string                    `json:"cover"`
	OriginCover  string                    `json:"originCover"`
	DynamicCover string                    `json:"dynamicCover"`
	Duration     int                       `json:"duration"`
	Width        int                       `json:"width"`
	Height       int                       `json:"height"`
	Bitrate      int64                     `json:"bitrate"`
	PlayAddress  shortDramaPlaybackAddress `json:"playAddr"`
	BitrateInfo  []shortDramaBitrate       `json:"bitrateInfo"`
}

type shortDramaBitrate struct {
	GearName       string                    `json:"GearName"`
	GearNameAlt    string                    `json:"gear_name"`
	Bitrate        int64                     `json:"Bitrate"`
	BitrateAlt     int64                     `json:"bit_rate"`
	CodecType      string                    `json:"CodecType"`
	CodecTypeAlt   string                    `json:"codec_type"`
	PlayAddress    shortDramaPlaybackAddress `json:"PlayAddr"`
	PlayAddressAlt shortDramaPlaybackAddress `json:"playAddr"`
}

type shortDramaPlaybackAddress struct {
	URL          string   `json:"url"`
	URLList      []string `json:"urlList"`
	URLListAlt   []string `json:"url_list"`
	URLListUpper []string `json:"UrlList"`
	URI          string   `json:"uri"`
	URIAlt       string   `json:"Uri"`
	DataSize     int64    `json:"dataSize"`
	DataSizeAlt  int64    `json:"data_size"`
	Width        int      `json:"width"`
	Height       int      `json:"height"`
}

func (address *shortDramaPlaybackAddress) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	if data[0] == '"' {
		return json.Unmarshal(data, &address.URL)
	}
	type rawAddress shortDramaPlaybackAddress
	return json.Unmarshal(data, (*rawAddress)(address))
}

func buildShortDramaResult(inputURL, finalURL, dramaID string, episode int, item shortDramaItem) *Result {
	description := firstNonEmpty(item.DramaInfo.VideoData.TranslatedDescription, item.Description, item.DramaInfo.Description)
	if description == "" {
		description = fmt.Sprintf("Short Drama %s episode %d", dramaID, episode)
	}
	result := &Result{
		InputURL:  inputURL,
		FinalURL:  finalURL,
		FetchedAt: time.Now().UTC(),
		Sources:   []string{"api/drama/episode/item_list"},
		ShortDrama: &ShortDrama{
			ID:            firstNonEmpty(item.DramaInfo.DramaID, dramaID),
			Name:          item.DramaInfo.DramaName,
			Description:   item.DramaInfo.Description,
			Episode:       episode,
			EpisodeCount:  item.DramaInfo.NumVideos,
			IsLimitedFree: item.DramaInfo.IsLimitedFree,
		},
		Video: video.Video{
			ID:          item.ID,
			Description: description,
			Duration:    item.Video.Duration,
			Width:       item.Video.Width,
			Height:      item.Video.Height,
			Author:      video.Author{ID: item.Author.ID, SecUID: item.Author.SecUID, UniqueID: item.Author.UniqueID, Nickname: item.Author.Nickname, Signature: item.Author.Signature, Verified: item.Author.Verified, AvatarURLs: media.UniqueStrings([]string{item.Author.AvatarLarger, item.Author.AvatarMedium, item.Author.AvatarThumb})},
			Statistics:  video.Statistics{PlayCount: item.Stats.PlayCount, LikeCount: item.Stats.DiggCount, CommentCount: item.Stats.CommentCount, ShareCount: item.Stats.ShareCount},
			Covers:      video.Covers{Default: media.UniqueStrings([]string{item.Video.Cover}), Original: media.UniqueStrings([]string{item.Video.OriginCover}), Dynamic: media.UniqueStrings([]string{item.Video.DynamicCover})},
		},
	}
	result.Media = makeShortDramaMedia(item.Video)
	media.Sort(result.Media)
	return result
}

func makeShortDramaMedia(source shortDramaVideo) []media.Variant {
	variants := make([]media.Variant, 0, len(source.BitrateInfo)+1)
	for _, profile := range source.BitrateInfo {
		address := profile.playAddress()
		urls := address.urls()
		if len(urls) == 0 {
			continue
		}
		variants = append(variants, media.Variant{Type: "video", Kind: "short_drama_playback", Codec: tiktok.NormalizeCodec(firstNonEmpty(profile.CodecType, profile.CodecTypeAlt, "h264")), Format: "mp4", Quality: media.QualityName(address.Height), GearName: firstNonEmpty(profile.GearName, profile.GearNameAlt), Width: address.Width, Height: address.Height, Bitrate: firstNonZero(profile.Bitrate, profile.BitrateAlt), Size: firstNonZero(address.DataSize, address.DataSizeAlt), URI: firstNonEmpty(address.URI, address.URIAlt), URL: urls[0], BackupURLs: media.UniqueStrings(urls[1:]), ExpiresAt: tiktok.ExpiryFromURL(urls[0])})
	}
	if len(variants) > 0 {
		return variants
	}
	urls := source.PlayAddress.urls()
	if len(urls) == 0 {
		return nil
	}
	return []media.Variant{{Type: "video", Kind: "short_drama_playback", Codec: "h264", Format: "mp4", Quality: media.QualityName(source.Height), Width: source.Width, Height: source.Height, Bitrate: source.Bitrate, URL: urls[0], BackupURLs: media.UniqueStrings(urls[1:]), ExpiresAt: tiktok.ExpiryFromURL(urls[0])}}
}

func mergeShortDramaItems(base, detail shortDramaItem) shortDramaItem {
	detail.DramaInfo = base.DramaInfo
	if detail.Description == "" {
		detail.Description = base.Description
	}
	return detail
}

func (profile shortDramaBitrate) playAddress() shortDramaPlaybackAddress {
	if len(profile.PlayAddress.urls()) > 0 {
		return profile.PlayAddress
	}
	return profile.PlayAddressAlt
}

func (address shortDramaPlaybackAddress) urls() []string {
	values := append([]string{address.URL}, address.URLList...)
	values = append(values, address.URLListAlt...)
	values = append(values, address.URLListUpper...)
	return media.UniqueStrings(values)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
