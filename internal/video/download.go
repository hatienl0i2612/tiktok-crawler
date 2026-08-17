package video

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
	"strings"
	"unicode"
)

// Download saves a selected media variant to an atomic temporary file before publishing it.
func (client *Client) Download(
	ctx context.Context,
	media Media,
	options DownloadOptions,
) (*DownloadResult, error) {
	outputPath := strings.TrimSpace(options.OutputPath)
	if outputPath == "" {
		return nil, errors.New("download output path must not be empty")
	}
	absolutePath, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}
	if info, statErr := os.Lstat(absolutePath); statErr == nil {
		if info.IsDir() {
			return nil, fmt.Errorf("output path is a directory: %s", absolutePath)
		}
		return nil, fmt.Errorf("output file already exists: %s (choose another -output path)", absolutePath)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect output path: %w", statErr)
	}

	urls := uniqueStrings(append([]string{media.URL}, media.BackupURLs...))
	if len(urls) == 0 {
		return nil, errors.New("selected media has no download URL")
	}
	var attemptErrors []string
	for _, mediaURL := range urls {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		result, err := client.downloadURL(ctx, mediaURL, absolutePath, media.Size, options)
		if err == nil {
			return result, nil
		}
		attemptErrors = append(attemptErrors, err.Error())
	}
	return nil, errors.New("all TikTok media URLs failed: " + strings.Join(attemptErrors, "; "))
}

func (client *Client) downloadURL(
	ctx context.Context,
	mediaURL string,
	outputPath string,
	expectedSize int64,
	options DownloadOptions,
) (*DownloadResult, error) {
	parsed, err := url.Parse(mediaURL)
	if err != nil || parsed.Scheme != "https" || !isAllowedTikTokHost(parsed.Hostname()) {
		return nil, fmt.Errorf("refuse disallowed media URL: %s", redactURL(mediaURL))
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	client.setHeaders(request, "video/mp4,video/*;q=0.9,*/*;q=0.8", options.Referer)

	response, err := client.httpClient.Do(request)
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
	if !isAllowedTikTokHost(response.Request.URL.Hostname()) {
		return nil, fmt.Errorf("download ended at a disallowed host: %s", response.Request.URL.Redacted())
	}

	reader := bufio.NewReader(response.Body)
	header, peekErr := reader.Peek(12)
	if peekErr != nil && !errors.Is(peekErr, io.EOF) {
		return nil, fmt.Errorf("inspect media response: %w", peekErr)
	}
	contentType := response.Header.Get("Content-Type")
	if !looksLikeVideo(contentType, header) {
		return nil, fmt.Errorf(
			"download %s returned %q instead of video data",
			parsed.Hostname(),
			contentType,
		)
	}

	directory := filepath.Dir(outputPath)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(outputPath)+"-*.part")
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
		totalBytes = expectedSize
	}
	progress := progressWriter{
		writer:   temporary,
		total:    totalBytes,
		callback: options.Progress,
	}
	progress.report()
	written, copyErr := io.Copy(&progress, reader)
	if copyErr != nil {
		return nil, fmt.Errorf("write temporary download: %w", copyErr)
	}
	if response.ContentLength >= 0 && written != response.ContentLength {
		return nil, fmt.Errorf(
			"incomplete download: received %d bytes, expected %d",
			written,
			response.ContentLength,
		)
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
	return &DownloadResult{
		Path:        outputPath,
		Bytes:       written,
		ContentType: contentType,
		SourceURL:   response.Request.URL.String(),
	}, nil
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
	if writer.callback == nil {
		return
	}
	writer.callback(DownloadProgress{
		DownloadedBytes: writer.downloaded,
		TotalBytes:      writer.total,
	})
}

// DefaultFilename returns a filesystem-safe filename for a selected media variant.
func DefaultFilename(result *Result, media *Media) string {
	username := "video"
	videoID := "unknown"
	if result != nil {
		if sanitized := sanitizeFilenamePart(result.Video.Author.UniqueID); sanitized != "" {
			username = sanitized
		}
		if sanitized := sanitizeFilenamePart(result.Video.ID); sanitized != "" {
			videoID = sanitized
		}
	}
	variant := "no_watermark"
	quality := "unknown"
	codec := "unknown"
	format := "mp4"
	if media != nil {
		if media.Watermarked {
			variant = "watermark"
		}
		if sanitized := sanitizeFilenamePart(media.Quality); sanitized != "" {
			quality = sanitized
		}
		if sanitized := sanitizeFilenamePart(media.Codec); sanitized != "" {
			codec = sanitized
		}
		if sanitized := sanitizeFilenamePart(media.Format); sanitized != "" {
			format = sanitized
		}
	}
	return strings.Join([]string{username, videoID, variant, quality, codec}, "_") + "." + format
}

func looksLikeVideo(contentType string, header []byte) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(mediaType, "video/") {
		return true
	}
	return len(header) >= 8 && bytes.Equal(header[4:8], []byte("ftyp"))
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
