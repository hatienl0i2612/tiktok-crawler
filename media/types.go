// Package media defines normalized TikTok video media variants.
package media

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Variant is one downloadable encoding of a TikTok video.
type Variant struct {
	Kind        string     `json:"kind"`
	Watermarked bool       `json:"watermarked"`
	Codec       string     `json:"codec"`
	Format      string     `json:"format"`
	Quality     string     `json:"quality"`
	GearName    string     `json:"gear_name,omitempty"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	FPS         int        `json:"fps,omitempty"`
	Bitrate     int64      `json:"bitrate,omitempty"`
	Size        int64      `json:"size,omitempty"`
	URI         string     `json:"uri,omitempty"`
	URL         string     `json:"url"`
	BackupURLs  []string   `json:"backup_urls,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Sort orders variants by watermark, pixel count, bitrate, codec, and URL.
func Sort(variants []Variant) {
	sort.SliceStable(variants, func(left, right int) bool {
		if variants[left].Watermarked != variants[right].Watermarked {
			return !variants[left].Watermarked
		}
		leftPixels := variants[left].Width * variants[left].Height
		rightPixels := variants[right].Width * variants[right].Height
		if leftPixels != rightPixels {
			return leftPixels > rightPixels
		}
		if variants[left].Bitrate != variants[right].Bitrate {
			return variants[left].Bitrate > variants[right].Bitrate
		}
		if variants[left].Codec != variants[right].Codec {
			return variants[left].Codec < variants[right].Codec
		}
		return variants[left].URL < variants[right].URL
	})
}

// QualityName formats a height as a stable quality label.
func QualityName(height int) string {
	if height <= 0 {
		return "unknown"
	}
	return strconv.Itoa(height) + "p"
}

// UniqueStrings trims, removes empty values, and preserves first occurrence order.
func UniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
