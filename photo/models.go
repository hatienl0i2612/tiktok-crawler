package photo

import "encoding/json"

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
