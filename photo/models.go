package photo

import (
	"bytes"
	"encoding/json"
	"strconv"
)

type playerResponse struct {
	StatusCode int          `json:"status_code"`
	StatusMsg  string       `json:"status_msg"`
	Items      []playerItem `json:"items"`
}

type playerItem struct {
	ID             json.Number      `json:"id"`
	IDStr          string           `json:"id_str"`
	AwemeType      int              `json:"aweme_type"`
	Description    string           `json:"desc"`
	Region         string           `json:"region"`
	AuthorInfo     playerAuthor     `json:"author_info"`
	StatisticsInfo playerStatistics `json:"statistics_info"`
	MusicInfo      playerMusic      `json:"music_info"`
	ImagePostInfo  imagePostInfo    `json:"image_post_info"`
}

type playerAuthor struct {
	AvatarURLs []string `json:"avatar_url_list"`
	Nickname   string   `json:"nickname"`
	SecretID   string   `json:"secret_id"`
	UniqueID   string   `json:"unique_id"`
}

type playerStatistics struct {
	CommentCount int64 `json:"comment_count"`
	DiggCount    int64 `json:"digg_count"`
	ShareCount   int64 `json:"share_count"`
}

type playerMusic struct {
	IDStr  string `json:"id_str"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type imagePostInfo struct {
	Images []postImage `json:"images"`
}

type postImage struct {
	DisplayImage        imageAddress `json:"display_image"`
	OwnerWatermarkImage imageAddress `json:"owner_watermark_image"`
	UserWatermarkImage  imageAddress `json:"user_watermark_image"`
	WatermarkImage      imageAddress `json:"watermark_image"`
}

type imageAddress struct {
	DataSize int64    `json:"data_size"`
	Height   int      `json:"height"`
	URI      string   `json:"uri"`
	URLList  []string `json:"url_list"`
	Width    int      `json:"width"`
}

type embedState struct {
	Source struct {
		Data map[string]embedPage `json:"data"`
	} `json:"source"`
}

type embedPage struct {
	VideoData embedPhoto `json:"videoData"`
	Code      int        `json:"code"`
	IsError   bool       `json:"isError"`
}

type embedPhoto struct {
	Item          embedItem             `json:"itemInfos"`
	Author        embedAuthor           `json:"authorInfos"`
	AuthorStats   embedAuthorStatistics `json:"authorStats"`
	Music         embedMusic            `json:"musicInfos"`
	ImagePostInfo embedImagePostInfo    `json:"imagePostInfo"`
}

type embedItem struct {
	ID              string   `json:"id"`
	Text            string   `json:"text"`
	CreateTime      string   `json:"createTime"`
	AuthorID        string   `json:"authorId"`
	Covers          []string `json:"covers"`
	CoversOrigin    []string `json:"coversOrigin"`
	CoversDynamic   []string `json:"coversDynamic"`
	DiggCount       int64    `json:"diggCount"`
	ShareCount      int64    `json:"shareCount"`
	PlayCount       int64    `json:"playCount"`
	CommentCount    int64    `json:"commentCount"`
	LocationCreated string   `json:"locationCreated"`
}

type embedAuthor struct {
	SecUID       string   `json:"secUid"`
	UserID       string   `json:"userId"`
	UniqueID     string   `json:"uniqueId"`
	Nickname     string   `json:"nickName"`
	Signature    string   `json:"signature"`
	Verified     bool     `json:"verified"`
	Covers       []string `json:"covers"`
	CoversMedium []string `json:"coversMedium"`
	CoversLarger []string `json:"coversLarger"`
}

type embedAuthorStatistics struct {
	FollowingCount flexibleInt64 `json:"followingCount"`
	FollowerCount  flexibleInt64 `json:"followerCount"`
	HeartCount     flexibleInt64 `json:"heartCount"`
	VideoCount     flexibleInt64 `json:"videoCount"`
	DiggCount      flexibleInt64 `json:"diggCount"`
}

type embedMusic struct {
	MusicID      string   `json:"musicId"`
	MusicName    string   `json:"musicName"`
	AuthorName   string   `json:"authorName"`
	PlayURL      []string `json:"playUrl"`
	Covers       []string `json:"covers"`
	CoversMedium []string `json:"coversMedium"`
	CoversLarger []string `json:"coversLarger"`
}

type embedImagePostInfo struct {
	DisplayImages []embedImage `json:"displayImages"`
}

type embedImage struct {
	Height  int      `json:"height"`
	Width   int      `json:"width"`
	URLList []string `json:"urlList"`
}

type flexibleInt64 int64

func (value *flexibleInt64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*value = 0
		return nil
	}
	if data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return err
		}
		*value = flexibleInt64(parsed)
		return nil
	}
	parsed, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*value = flexibleInt64(parsed)
	return nil
}
