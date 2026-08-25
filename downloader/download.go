package downloader

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

type downloadOptions struct {
	OutputPath string
	Referer    string
	Progress   func(DownloadProgress)
}

// Download selects the requested variant and saves one media file to disk.
func Download(
	ctx context.Context,
	session *tiktok.Session,
	variants []media.Variant,
	file FileInfo,
	options Options,
) (*DownloadResult, error) {
	selected, err := selectVariant(variants, options)
	if err != nil {
		return nil, err
	}
	outputPath := options.OutputPath
	if strings.TrimSpace(outputPath) == "" {
		outputPath = defaultFilename(file, selected)
	}
	outputPath, err = resolveOutputPath(outputPath)
	if err != nil {
		return nil, err
	}
	if options.OnStart != nil {
		options.OnStart(DownloadStart{Media: selected, OutputPath: outputPath})
	}
	return downloadVariant(ctx, session, *selected, downloadOptions{
		OutputPath: outputPath,
		Referer:    file.Referer,
		Progress:   options.Progress,
	})
}

// DownloadAll selects one variant for every ordered item and saves the media
// files in an output directory. It reuses the same validation, retry, atomic
// write, and progress behavior as Download.
func DownloadAll(
	ctx context.Context,
	session *tiktok.Session,
	items []BatchItem,
	file FileInfo,
	options BatchOptions,
) (*BatchResult, error) {
	if len(items) == 0 {
		return nil, errors.New("media collection has no downloadable items")
	}
	outputDirectory := strings.TrimSpace(options.OutputDir)
	if outputDirectory == "" {
		outputDirectory = defaultCollectionDirectory(file)
	}
	outputDirectory, err := resolveOutputPath(outputDirectory)
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Stat(outputDirectory); statErr == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("output directory is not a directory: %s", outputDirectory)
		}
	} else if errors.Is(statErr, os.ErrNotExist) {
		if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
			return nil, fmt.Errorf("create output directory: %w", err)
		}
	} else {
		return nil, fmt.Errorf("inspect output directory: %w", statErr)
	}

	result := &BatchResult{Downloads: make([]DownloadResult, 0, len(items))}
	for index, item := range items {
		selected, err := selectVariant(item.Variants, Options{Quality: "best", Watermarked: options.Watermarked})
		if err != nil {
			return nil, fmt.Errorf("select media %d/%d: %w", index+1, len(items), err)
		}
		outputPath := filepath.Join(outputDirectory, collectionFilename(file, index+1, len(items), selected))
		if options.OnStart != nil {
			options.OnStart(BatchStart{
				Index:         index + 1,
				Total:         len(items),
				DownloadStart: DownloadStart{Media: selected, OutputPath: outputPath},
			})
		}
		download, err := downloadVariant(ctx, session, *selected, downloadOptions{
			OutputPath: outputPath,
			Referer:    file.Referer,
			Progress: func(progress DownloadProgress) {
				if options.Progress != nil {
					options.Progress(BatchProgress{Index: index + 1, Total: len(items), DownloadProgress: progress})
				}
			},
		})
		if err != nil {
			return nil, fmt.Errorf("download media %d/%d: %w", index+1, len(items), err)
		}
		result.Downloads = append(result.Downloads, *download)
	}
	return result, nil
}

func downloadVariant(
	ctx context.Context,
	session *tiktok.Session,
	variant media.Variant,
	options downloadOptions,
) (*DownloadResult, error) {
	if session == nil {
		return nil, errors.New("TikTok download session is not configured")
	}
	absolutePath, err := resolveOutputPath(options.OutputPath)
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Lstat(absolutePath); statErr == nil {
		if info.IsDir() {
			return nil, fmt.Errorf("output path is a directory: %s", absolutePath)
		}
		return nil, fmt.Errorf("output file already exists: %s (choose another -output path)", absolutePath)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect output path: %w", statErr)
	}
	directory := filepath.Dir(absolutePath)
	if info, statErr := os.Stat(directory); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return nil, fmt.Errorf("output directory does not exist: %s", directory)
		}
		return nil, fmt.Errorf("inspect output directory: %w", statErr)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("output directory is not a directory: %s", directory)
	}

	urls := media.UniqueStrings(append([]string{variant.URL}, variant.BackupURLs...))
	if len(urls) == 0 {
		return nil, errors.New("selected media has no download URL")
	}
	var attemptErrors []string
	for _, mediaURL := range urls {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		result, err := downloadURL(ctx, session, mediaURL, absolutePath, variant, options)
		if err == nil {
			return result, nil
		}
		attemptErrors = append(attemptErrors, err.Error())
	}
	return nil, errors.New("all TikTok media URLs failed: " + strings.Join(attemptErrors, "; "))
}

func downloadURL(
	ctx context.Context,
	session *tiktok.Session,
	mediaURL string,
	outputPath string,
	variant media.Variant,
	options downloadOptions,
) (*DownloadResult, error) {
	parsed, err := url.Parse(mediaURL)
	if err != nil || parsed.Scheme != "https" || !tiktok.IsAllowedHost(parsed.Hostname()) {
		return nil, fmt.Errorf("refuse disallowed media URL: %s", redactURL(mediaURL))
	}
	response, err := session.Get(ctx, parsed.String(), "video/mp4,video/*;q=0.9,image/avif,image/webp,image/apng,image/*;q=0.8,*/*;q=0.7", options.Referer)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", parsed.Hostname(), err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download %s: HTTP %s", parsed.Hostname(), response.Status)
	}
	if response.StatusCode == http.StatusPartialContent {
		return nil, fmt.Errorf("download %s returned an unexpected partial response", parsed.Hostname())
	}
	if !tiktok.IsAllowedHost(response.Request.URL.Hostname()) {
		return nil, fmt.Errorf("download ended at a disallowed host: %s", response.Request.URL.Redacted())
	}

	reader := bufio.NewReader(response.Body)
	header, peekErr := reader.Peek(12)
	if peekErr != nil && !errors.Is(peekErr, io.EOF) {
		return nil, fmt.Errorf("inspect media response: %w", peekErr)
	}
	contentType := response.Header.Get("Content-Type")
	if !looksLikeMedia(variant.Type, contentType, header) {
		return nil, fmt.Errorf("download %s returned %q instead of %s data", parsed.Hostname(), contentType, expectedMediaType(variant.Type))
	}

	temporary, err := os.CreateTemp(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+"-*.part")
	if err != nil {
		return nil, fmt.Errorf("create temporary download: %w", err)
	}
	temporaryPath := temporary.Name()
	keepTemporary := false
	defer func() {
		_ = temporary.Close()
		if !keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	totalBytes := response.ContentLength
	if totalBytes <= 0 {
		totalBytes = variant.Size
	}
	progress := progressWriter{writer: temporary, total: totalBytes, callback: options.Progress}
	progress.report()
	written, copyErr := io.Copy(&progress, reader)
	if copyErr != nil {
		return nil, fmt.Errorf("write temporary download: %w", copyErr)
	}
	if response.ContentLength >= 0 && written != response.ContentLength {
		return nil, fmt.Errorf("incomplete download: received %d bytes, expected %d", written, response.ContentLength)
	}
	if err := temporary.Sync(); err != nil {
		return nil, fmt.Errorf("sync temporary download: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close temporary download: %w", err)
	}
	if _, err := os.Lstat(outputPath); err == nil {
		return nil, fmt.Errorf("output file appeared during download: %s", outputPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect output path: %w", err)
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return nil, fmt.Errorf("publish download: %w", err)
	}
	keepTemporary = true
	return &DownloadResult{Path: outputPath, Bytes: written, ContentType: contentType, SourceURL: response.Request.URL.String()}, nil
}

type progressWriter struct {
	writer     io.Writer
	downloaded int64
	total      int64
	callback   func(DownloadProgress)
}

func (writer *progressWriter) Write(data []byte) (int, error) {
	written, err := writer.writer.Write(data)
	writer.downloaded += int64(written)
	writer.report()
	return written, err
}

func (writer *progressWriter) report() {
	if writer.callback != nil {
		writer.callback(DownloadProgress{DownloadedBytes: writer.downloaded, TotalBytes: writer.total})
	}
}

func defaultFilename(file FileInfo, variant *media.Variant) string {
	username := sanitizeFilenamePart(file.Author)
	if username == "" {
		username = "video"
	}
	videoID := sanitizeFilenamePart(file.VideoID)
	if videoID == "" {
		videoID = "unknown"
	}
	watermark, quality, codec, format := "no_watermark", "unknown", "unknown", "mp4"
	if variant != nil {
		if variant.Watermarked {
			watermark = "watermark"
		}
		if value := sanitizeFilenamePart(variant.Quality); value != "" {
			quality = value
		}
		if value := sanitizeFilenamePart(variant.Codec); value != "" {
			codec = value
		}
		if value := sanitizeFilenamePart(variant.Format); value != "" {
			format = value
		}
	}
	return strings.Join([]string{username, videoID, watermark, quality, codec}, "_") + "." + format
}

func defaultCollectionDirectory(file FileInfo) string {
	username := sanitizeFilenamePart(file.Author)
	if username == "" {
		username = "tiktok"
	}
	contentID := sanitizeFilenamePart(file.VideoID)
	if contentID == "" {
		contentID = "unknown"
	}
	return strings.Join([]string{username, contentID, "images"}, "_")
}

func collectionFilename(file FileInfo, index, total int, variant *media.Variant) string {
	username := sanitizeFilenamePart(file.Author)
	if username == "" {
		username = "tiktok"
	}
	contentID := sanitizeFilenamePart(file.VideoID)
	if contentID == "" {
		contentID = "unknown"
	}
	watermark := "no_watermark"
	if variant != nil && variant.Watermarked {
		watermark = "watermark"
	}
	quality := "unknown"
	format := "jpg"
	if variant != nil {
		if value := sanitizeFilenamePart(variant.Quality); value != "" {
			quality = value
		}
		if value := sanitizeFilenamePart(variant.Format); value != "" {
			format = value
		}
	}
	width := len(strconv.Itoa(total))
	if width < 3 {
		width = 3
	}
	return fmt.Sprintf("%s_%s_%0*d_%s_%s.%s", username, contentID, width, index, watermark, quality, format)
}

func resolveOutputPath(outputPath string) (string, error) {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return "", errors.New("download output path must not be empty")
	}
	if outputPath == "~" || strings.HasPrefix(outputPath, "~/") || strings.HasPrefix(outputPath, `~\`) {
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if outputPath == "~" {
			outputPath = homeDirectory
		} else {
			relativePath := strings.TrimLeft(outputPath[1:], `/\`)
			relativePath = filepath.FromSlash(strings.ReplaceAll(relativePath, `\`, "/"))
			outputPath = filepath.Join(homeDirectory, relativePath)
		}
	}
	absolutePath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	return absolutePath, nil
}

func looksLikeMedia(variantType, contentType string, header []byte) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.EqualFold(strings.TrimSpace(variantType), "image") {
		return strings.HasPrefix(mediaType, "image/") || looksLikeImage(header)
	}
	return strings.HasPrefix(mediaType, "video/") || len(header) >= 8 && bytes.Equal(header[4:8], []byte("ftyp"))
}

func expectedMediaType(variantType string) string {
	if strings.EqualFold(strings.TrimSpace(variantType), "image") {
		return "image"
	}
	return "video"
}

func looksLikeImage(header []byte) bool {
	return len(header) >= 3 && bytes.Equal(header[:3], []byte{0xff, 0xd8, 0xff}) ||
		len(header) >= 8 && bytes.Equal(header[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) ||
		len(header) >= 6 && (bytes.Equal(header[:6], []byte("GIF87a")) || bytes.Equal(header[:6], []byte("GIF89a"))) ||
		len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WEBP")) ||
		len(header) >= 12 && bytes.Equal(header[4:8], []byte("ftyp")) && (bytes.Equal(header[8:12], []byte("avif")) || bytes.Equal(header[8:12], []byte("avis")))
}

func redactURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "invalid URL"
	}
	return parsed.Redacted()
}

func sanitizeFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	lastUnderscore := false
	for _, character := range value {
		allowed := unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '.'
		if allowed {
			builder.WriteRune(character)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "._-")
}
