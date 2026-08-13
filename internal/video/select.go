package video

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SelectMedia returns the highest-resolution media variant matching the requested filters.
func SelectMedia(media []Media, options SelectOptions) (*Media, error) {
	codec, err := normalizeCodecFilter(options.Codec)
	if err != nil {
		return nil, err
	}
	quality, bestQuality, err := normalizeQualityFilter(options.Quality)
	if err != nil {
		return nil, err
	}

	filtered := make([]Media, 0, len(media))
	for _, candidate := range media {
		if candidate.Watermarked != options.Watermarked {
			continue
		}
		if codec != "auto" && candidate.Codec != codec {
			continue
		}
		if !bestQuality && candidate.Height != quality {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if len(filtered) == 0 {
		variant := "without a watermark"
		if options.Watermarked {
			variant = "with a TikTok watermark"
		}
		return nil, fmt.Errorf(
			"no media %s matches codec=%s quality=%s; available variants: %s",
			variant,
			codec,
			options.Quality,
			availableMediaSummary(media),
		)
	}
	sortMedia(filtered)
	selected := filtered[0]
	return &selected, nil
}

func normalizeCodecFilter(codec string) (string, error) {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "" {
		return "h264", nil
	}
	codec = normalizeCodec(codec)
	if codec != "auto" && codec != "h264" && codec != "h265" {
		return "", errors.New("codec must be h264, h265, or auto")
	}
	return codec, nil
}

func normalizeQualityFilter(quality string) (int, bool, error) {
	quality = strings.ToLower(strings.TrimSpace(quality))
	if quality == "" || quality == "best" {
		return 0, true, nil
	}
	quality = strings.TrimSuffix(quality, "p")
	height, err := strconv.Atoi(quality)
	if err != nil || height <= 0 {
		return 0, false, errors.New("quality must be best or a positive height such as 720 or 1080p")
	}
	return height, false, nil
}

func availableMediaSummary(media []Media) string {
	if len(media) == 0 {
		return "none"
	}
	values := make([]string, 0, len(media))
	seen := make(map[string]bool)
	for _, candidate := range media {
		watermark := "no-watermark"
		if candidate.Watermarked {
			watermark = "watermark"
		}
		value := strings.Join([]string{watermark, candidate.Codec, candidate.Quality}, "/")
		if seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	sort.Strings(values)
	return strings.Join(values, ", ")
}
