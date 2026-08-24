package tiktok

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NormalizeCodec converts TikTok codec labels to stable h264 or h265 values.
func NormalizeCodec(codec string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))
	switch {
	case strings.Contains(codec, "265"), strings.Contains(codec, "hevc"), strings.Contains(codec, "bytevc1"):
		return "h265"
	case strings.Contains(codec, "264"), strings.Contains(codec, "avc"):
		return "h264"
	case codec == "":
		return "unknown"
	default:
		return codec
	}
}

// ExpiryFromURL extracts a common TikTok URL expiry query parameter.
func ExpiryFromURL(rawURL string) *time.Time {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	for _, key := range []string{"expire", "expires", "x-expires", "x-m-expire"} {
		value := parsed.Query().Get(key)
		if value == "" {
			continue
		}
		unixTime, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			continue
		}
		if unixTime > 1_000_000_000_000 {
			unixTime /= 1000
		}
		expiresAt := time.Unix(unixTime, 0).UTC()
		return &expiresAt
	}
	return nil
}
