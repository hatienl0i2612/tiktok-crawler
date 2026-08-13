package livestream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	protocolOrder = []string{"hls", "flv", "cmaf", "dash", "lls", "tsl", "rtc"}
	qualityOrder  = []string{"origin", "uhd60", "uhd", "hd60", "hd", "sd", "ld", "auto", "ao"}
)

func collectStreams(info roomInfo) ([]Stream, []string) {
	candidates := []struct {
		codec string
		data  string
	}{
		{codec: "h264", data: info.LiveRoom.StreamData.PullData.StreamData},
		{codec: "h265", data: info.LiveRoom.HEVCStreamData.PullData.StreamData},
	}

	var streams []Stream
	var parseErrors []string
	for _, candidate := range candidates {
		if candidate.data == "" {
			continue
		}
		parsed, err := parseStreamData(candidate.data, candidate.codec)
		if err != nil {
			parseErrors = append(parseErrors, candidate.codec+": "+err.Error())
			continue
		}
		streams = append(streams, parsed...)
	}

	seen := make(map[string]bool, len(streams))
	unique := streams[:0]
	for _, stream := range streams {
		key := strings.Join(
			[]string{stream.Codec, stream.Quality, stream.Line, stream.Protocol, stream.URL},
			"\x00",
		)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, stream)
	}
	sortStreams(unique)
	return unique, parseErrors
}

func parseStreamData(encoded, defaultCodec string) ([]Stream, error) {
	var envelope struct {
		Data map[string]map[string]map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(encoded), &envelope); err != nil {
		return nil, fmt.Errorf("decode pull_data.stream_data: %w", err)
	}
	if len(envelope.Data) == 0 {
		return nil, errors.New("stream_data contains no data")
	}

	var streams []Stream
	for quality, lines := range envelope.Data {
		for line, endpoint := range lines {
			metadata := parseSDKParams(endpoint["sdk_params"])
			codec := normalizeCodec(metadata.stringValue("VCodec", "v_codec", "codec"))
			if codec == "" {
				codec = defaultCodec
			}
			for _, protocol := range protocolOrder {
				streamURL := rawString(endpoint[protocol])
				if streamURL == "" {
					continue
				}
				streams = append(streams, Stream{
					Codec:      codec,
					Quality:    strings.ToLower(quality),
					Line:       strings.ToLower(line),
					Protocol:   protocol,
					URL:        streamURL,
					Resolution: metadata.stringValue("resolution", "Resolution"),
					Bitrate:    metadata.intValue("vbitrate", "bitrate", "v_rtbitrate"),
					CDN:        metadata.stringValue("cdn_name", "cdn"),
					ExpiresAt:  expiryFromURL(streamURL),
				})
			}
		}
	}
	return streams, nil
}

// SelectStream returns the highest-ranked stream matching the requested filters.
func SelectStream(streams []Stream, options SelectOptions) (*Stream, error) {
	codec := strings.ToLower(strings.TrimSpace(options.Codec))
	quality := strings.ToLower(strings.TrimSpace(options.Quality))
	format := strings.ToLower(strings.TrimSpace(options.Format))

	filtered := make([]Stream, 0, len(streams))
	for _, stream := range streams {
		if codec != "auto" && stream.Codec != codec {
			continue
		}
		if format != "auto" && stream.Protocol != format {
			continue
		}
		if quality != "best" && stream.Quality != quality {
			continue
		}
		if quality != "ao" && stream.Quality == "ao" {
			continue
		}
		filtered = append(filtered, stream)
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf(
			"no stream matches codec=%s quality=%s format=%s; available streams: %s",
			codec,
			quality,
			format,
			availableSummary(streams),
		)
	}
	sortStreams(filtered)
	selected := filtered[0]
	return &selected, nil
}

func sortStreams(streams []Stream) {
	sort.SliceStable(streams, func(i, j int) bool {
		left, right := streams[i], streams[j]
		if qualityRank(left.Quality) != qualityRank(right.Quality) {
			return qualityRank(left.Quality) < qualityRank(right.Quality)
		}
		if codecRank(left.Codec) != codecRank(right.Codec) {
			return codecRank(left.Codec) < codecRank(right.Codec)
		}
		if lineRank(left.Line) != lineRank(right.Line) {
			return lineRank(left.Line) < lineRank(right.Line)
		}
		if protocolRank(left.Protocol) != protocolRank(right.Protocol) {
			return protocolRank(left.Protocol) < protocolRank(right.Protocol)
		}
		return left.URL < right.URL
	})
}

func qualityRank(quality string) int {
	return rank(qualityOrder, quality)
}

func protocolRank(protocol string) int {
	return rank(protocolOrder, protocol)
}

func rank(order []string, value string) int {
	for index, candidate := range order {
		if value == candidate {
			return index
		}
	}
	return len(order)
}

func codecRank(codec string) int {
	switch codec {
	case "h264":
		return 0
	case "h265":
		return 1
	default:
		return 2
	}
}

func lineRank(line string) int {
	switch line {
	case "main":
		return 0
	case "backup":
		return 1
	default:
		return 2
	}
}

func availableSummary(streams []Stream) string {
	seen := make(map[string]bool)
	values := make([]string, 0, len(streams))
	for _, stream := range streams {
		value := stream.Codec + "/" + stream.Quality + "/" + stream.Protocol
		if seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	if len(values) > 12 {
		values = append(values[:12], "...")
	}
	return strings.Join(values, ", ")
}

type sdkMetadata map[string]any

func parseSDKParams(raw json.RawMessage) sdkMetadata {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return sdkMetadata{}
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		var metadata sdkMetadata
		if json.Unmarshal([]byte(encoded), &metadata) == nil {
			return metadata
		}
		return sdkMetadata{}
	}
	var metadata sdkMetadata
	if json.Unmarshal(raw, &metadata) == nil {
		return metadata
	}
	return sdkMetadata{}
}

func (metadata sdkMetadata) stringValue(keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case json.Number:
				return typed.String()
			}
		}
	}
	return ""
}

func (metadata sdkMetadata) intValue(keys ...string) int64 {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			switch typed := value.(type) {
			case float64:
				return int64(typed)
			case json.Number:
				number, _ := typed.Int64()
				return number
			case string:
				number, _ := strconv.ParseFloat(typed, 64)
				return int64(number)
			}
		}
	}
	return 0
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func normalizeCodec(codec string) string {
	codec = strings.ToLower(codec)
	switch {
	case strings.Contains(codec, "265"), strings.Contains(codec, "hevc"), strings.Contains(codec, "bytevc1"):
		return "h265"
	case strings.Contains(codec, "264"), strings.Contains(codec, "avc"):
		return "h264"
	default:
		return codec
	}
}

func expiryFromURL(rawURL string) *time.Time {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	for _, key := range []string{"expire", "expires", "x-expires"} {
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
