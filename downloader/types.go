// Package downloader selects and downloads TikTok media files.
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

// FileInfo contains metadata used for output names and the Referer header.
type FileInfo struct {
	Author  string
	VideoID string
	Referer string
}

// Options controls selecting and downloading one resolved media file.
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

// BatchItem contains every source variant for one file in an ordered media
// collection, such as one image in a TikTok Photo Post.
type BatchItem struct {
	Variants []media.Variant
	// File optionally overrides the collection-level author, content ID, and
	// Referer. A non-empty VideoID also selects the normal per-video filename.
	File FileInfo
}

// BatchOptions controls downloading an ordered media collection. OutputDir is
// a directory, unlike Options.OutputPath which is a single file path.
type BatchOptions struct {
	OutputDir   string
	Quality     string
	Watermarked bool
	Progress    func(BatchProgress)
	OnStart     func(BatchStart)
}

// BatchProgress reports the progress of one file in a media collection.
type BatchProgress struct {
	Index int
	Total int
	DownloadProgress
}

// BatchStart describes the next file selected from a media collection.
type BatchStart struct {
	Index int
	Total int
	File  FileInfo
	DownloadStart
}

// BatchResult contains the completed files in their source order.
type BatchResult struct {
	Downloads []DownloadResult `json:"downloads"`
}
