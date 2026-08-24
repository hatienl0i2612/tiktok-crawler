package downloader

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hatienl0i2612/tiktok-crawler/media"
)

func selectVariant(variants []media.Variant, options Options) (*media.Variant, error) {
	quality, bestQuality, err := normalizeQualityFilter(options.Quality)
	if err != nil {
		return nil, err
	}

	filtered := make([]media.Variant, 0, len(variants))
	for _, candidate := range variants {
		if candidate.Watermarked != options.Watermarked {
			continue
		}
		if !bestQuality && candidate.Height != quality {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if len(filtered) == 0 {
		variant := "media without a watermark"
		if options.Watermarked {
			variant = "media with a TikTok watermark"
		}
		return nil, fmt.Errorf(
			"no %s matches quality=%s; available variants: %s",
			variant,
			options.Quality,
			availableSummary(variants),
		)
	}
	media.Sort(filtered)
	selected := filtered[0]
	return &selected, nil
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

func availableSummary(variants []media.Variant) string {
	if len(variants) == 0 {
		return "none"
	}
	values := make([]string, 0, len(variants))
	seen := make(map[string]bool)
	for _, candidate := range variants {
		watermark := "no-watermark"
		if candidate.Watermarked {
			watermark = "watermark"
		}
		value := strings.Join([]string{watermark, candidate.Codec, candidate.Quality}, "/")
		if !seen[value] {
			seen[value] = true
			values = append(values, value)
		}
	}
	sort.Strings(values)
	return strings.Join(values, ", ")
}
