package livestream

type roomInfo struct {
	User     wireUser     `json:"user"`
	Stats    wireStats    `json:"stats"`
	LiveRoom wireLiveRoom `json:"liveRoom"`
}

type wireUser struct {
	ID           string `json:"id"`
	UniqueID     string `json:"uniqueId"`
	Nickname     string `json:"nickname"`
	RoomID       string `json:"roomId"`
	Status       int    `json:"status"`
	AvatarThumb  string `json:"avatarThumb"`
	AvatarMedium string `json:"avatarMedium"`
	AvatarLarger string `json:"avatarLarger"`
}

type wireStats struct {
	FollowerCount int64 `json:"followerCount"`
}

type wireLiveRoom struct {
	Title          string          `json:"title"`
	Status         int             `json:"status"`
	StartTime      int64           `json:"startTime"`
	StreamID       string          `json:"streamId"`
	CoverURL       string          `json:"coverUrl"`
	SquareCoverURL string          `json:"squareCoverImg"`
	StreamData     streamContainer `json:"streamData"`
	HEVCStreamData streamContainer `json:"hevcStreamData"`
	Stats          wireLiveStats   `json:"liveRoomStats"`
}

type wireLiveStats struct {
	EnterCount int64 `json:"enterCount"`
	UserCount  int64 `json:"userCount"`
}

type streamContainer struct {
	PullData pullData `json:"pull_data"`
}

type pullData struct {
	StreamData string `json:"stream_data"`
}

type apiResponse struct {
	Data            roomInfo `json:"data"`
	Message         string   `json:"message"`
	Prompts         string   `json:"prompts"`
	StatusCode      int      `json:"statusCode"`
	StatusCodeSnake int      `json:"status_code"`
}

type sigiState struct {
	LiveRoom struct {
		LiveRoomUserInfo roomInfo `json:"liveRoomUserInfo"`
	} `json:"LiveRoom"`
}
