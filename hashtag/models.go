package hashtag

import "github.com/hatienl0i2612/tiktok-crawler/internal/jsonvalue"

type embedState struct {
	Source struct {
		Data map[string]embedPage `json:"data"`
	} `json:"source"`
}

type embedPage struct {
	PageName         string       `json:"pageName"`
	PlaylistType     string       `json:"playlistType"`
	PlaylistEndpoint string       `json:"playlistEndpoint"`
	IsError          bool         `json:"isError"`
	EmbedInfo        embedInfo    `json:"embedInfo"`
	VideoList        []embedVideo `json:"videoList"`
}

type embedInfo struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	CoverURL    string          `json:"coverUrl"`
	ViewCount   jsonvalue.Int64 `json:"viewCount"`
	VideoCount  jsonvalue.Int64 `json:"videoCount"`
	StatusCode  int             `json:"statusCode"`
	Code        int             `json:"code"`
}

type embedVideo struct {
	ID              string          `json:"id"`
	Description     string          `json:"desc"`
	Height          int             `json:"height"`
	Width           int             `json:"width"`
	Ratio           string          `json:"ratio"`
	CoverURL        string          `json:"coverUrl"`
	OriginCoverURL  string          `json:"originCoverUrl"`
	DynamicCoverURL string          `json:"dynamicCoverUrl"`
	PlayAddress     string          `json:"playAddr"`
	PlayCount       jsonvalue.Int64 `json:"playCount"`
	PrivateItem     bool            `json:"privateItem"`
	AuthorUniqueID  string          `json:"authorUniqueId"`
}
