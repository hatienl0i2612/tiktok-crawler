// Package downloader selects and downloads TikTok video media.
package downloader

import "github.com/hatienl0i2612/tiktok-crawler/media"

// DownloadProgress reports downloaded bytes and the expected total.
type DownloadProgress struct {
	DownloadedBytes int64
	TotalBytes      int64
}

// DownloadResult describes a completed media download.
type DownloadResult struct {
	Path        string `json:"path"`
	Bytes       int64  `json:"bytes"`
	ContentType string `json:"content_type"`
	SourceURL   string `json:"source_url"`
}

// FileInfo contains metadata used for the output filename and Referer header.
type FileInfo struct {
	Author  string
	VideoID string
	Referer string
}

// Options controls selecting and downloading one resolved video.
type Options struct {
	OutputPath  string
	Quality     string
	Watermarked bool
	Progress    func(DownloadProgress)
	OnStart     func(DownloadStart)
}

// DownloadStart describes the selected variant and destination.
type DownloadStart struct {
	Media      *media.Variant
	OutputPath string
}
