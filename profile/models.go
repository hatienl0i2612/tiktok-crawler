package profile

import (
	"bytes"
	"encoding/json"
	"strconv"
)

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
	FollowingCount flexibleInt64 `json:"followingCount"`
	FollowerCount  flexibleInt64 `json:"followerCount"`
	HeartCount     flexibleInt64 `json:"heartCount"`
	VideoCount     flexibleInt64 `json:"videoCount"`
	DiggCount      flexibleInt64 `json:"diggCount"`
	FriendCount    flexibleInt64 `json:"friendCount"`
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
	ID             string        `json:"id"`
	UniqueID       string        `json:"uniqueId"`
	Nickname       string        `json:"nickname"`
	Signature      string        `json:"signature"`
	Verified       bool          `json:"verified"`
	PrivateAccount bool          `json:"privateAccount"`
	AvatarThumbURL string        `json:"avatarThumbUrl"`
	FollowingCount flexibleInt64 `json:"followingCount"`
	FollowerCount  flexibleInt64 `json:"followerCount"`
	HeartCount     flexibleInt64 `json:"heartCount"`
	Code           int           `json:"code"`
}

type embedVideo struct {
	ID              string        `json:"id"`
	Description     string        `json:"desc"`
	Height          int           `json:"height"`
	Width           int           `json:"width"`
	Ratio           string        `json:"ratio"`
	CoverURL        string        `json:"coverUrl"`
	OriginCoverURL  string        `json:"originCoverUrl"`
	DynamicCoverURL string        `json:"dynamicCoverUrl"`
	PlayAddress     string        `json:"playAddr"`
	PlayCount       flexibleInt64 `json:"playCount"`
	PrivateItem     bool          `json:"privateItem"`
	AuthorUniqueID  string        `json:"authorUniqueId"`
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
