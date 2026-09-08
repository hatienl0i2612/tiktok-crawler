package profile

import "github.com/hatienl0i2612/tiktok-crawler/internal/jsonvalue"

type hydrationState struct {
	DefaultScope struct {
		UserDetail hydrationUserDetail `json:"webapp.user-detail"`
	} `json:"__DEFAULT_SCOPE__"`
}

type hydrationUserDetail struct {
	StatusCode int               `json:"statusCode"`
	StatusMsg  string            `json:"statusMsg"`
	UserInfo   hydrationUserInfo `json:"userInfo"`
}

type hydrationUserInfo struct {
	User    hydrationUser       `json:"user"`
	Stats   hydrationStatistics `json:"stats"`
	StatsV2 hydrationStatistics `json:"statsV2"`
}

type hydrationUser struct {
	ID             string `json:"id"`
	SecUID         string `json:"secUid"`
	UniqueID       string `json:"uniqueId"`
	Nickname       string `json:"nickname"`
	Signature      string `json:"signature"`
	Verified       bool   `json:"verified"`
	PrivateAccount bool   `json:"privateAccount"`
	AvatarThumb    string `json:"avatarThumb"`
	AvatarMedium   string `json:"avatarMedium"`
	AvatarLarger   string `json:"avatarLarger"`
	BioLink        struct {
		Link string `json:"link"`
	} `json:"bioLink"`
}

type hydrationStatistics struct {
	FollowingCount jsonvalue.Int64 `json:"followingCount"`
	FollowerCount  jsonvalue.Int64 `json:"followerCount"`
	HeartCount     jsonvalue.Int64 `json:"heartCount"`
	VideoCount     jsonvalue.Int64 `json:"videoCount"`
	DiggCount      jsonvalue.Int64 `json:"diggCount"`
	FriendCount    jsonvalue.Int64 `json:"friendCount"`
}

type embedState struct {
	Source struct {
		Data map[string]embedPage `json:"data"`
	} `json:"source"`
}

type embedPage struct {
	PageName     string       `json:"pageName"`
	PlaylistType string       `json:"playlistType"`
	PlaylistID   string       `json:"playlistId"`
	IsError      bool         `json:"isError"`
	UserInfo     embedUser    `json:"userInfo"`
	VideoList    []embedVideo `json:"videoList"`
}

type embedUser struct {
	ID             string          `json:"id"`
	UniqueID       string          `json:"uniqueId"`
	Nickname       string          `json:"nickname"`
	Signature      string          `json:"signature"`
	Verified       bool            `json:"verified"`
	PrivateAccount bool            `json:"privateAccount"`
	AvatarThumbURL string          `json:"avatarThumbUrl"`
	FollowingCount jsonvalue.Int64 `json:"followingCount"`
	FollowerCount  jsonvalue.Int64 `json:"followerCount"`
	HeartCount     jsonvalue.Int64 `json:"heartCount"`
	Code           int             `json:"code"`
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
