package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/cliargs"
	"github.com/hatienl0i2612/tiktok-crawler/cookies"
	"github.com/hatienl0i2612/tiktok-crawler/downloader"
	"github.com/hatienl0i2612/tiktok-crawler/livestream"
	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/photo"
	"github.com/hatienl0i2612/tiktok-crawler/profile"
	"github.com/hatienl0i2612/tiktok-crawler/shortdrama"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
	"github.com/hatienl0i2612/tiktok-crawler/video"
)

const videoRequestTimeout = 5 * time.Minute

var (
	livePathPattern       = regexp.MustCompile(`^/@[^/]+/live/?$`)
	videoPathPattern      = regexp.MustCompile(`^/@[^/]+/video/[0-9]+/?$`)
	photoPathPattern      = regexp.MustCompile(`^/@[^/]+/photo/[0-9]+/?$`)
	shortDramaPathPattern = regexp.MustCompile(`^/shortdrama/episode/[0-9]+/[1-9][0-9]*/?$`)
	profilePathPattern    = regexp.MustCompile(`^/@[^/]+/?$`)
)

type contentType string

const (
	contentTypeLive       contentType = "live"
	contentTypeVideo      contentType = "video"
	contentTypePhoto      contentType = "photo"
	contentTypeShortDrama contentType = "short_drama"
	contentTypeProfile    contentType = "profile"
)

type options struct {
	inputURL       string
	content        contentType
	json           bool
	output         string
	quality        string
	watermark      bool
	verbose        bool
	cookiesFile    string
	cookiesBrowser string
	headers        map[string]string
	timeout        time.Duration
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	options, err := parseOptions(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	cookie, err := resolveCookieSources(options)
	if err != nil {
		return err
	}

	switch options.content {
	case contentTypeVideo:
		return runVideo(options, cookie, stdout, stderr)
	case contentTypePhoto:
		return runPhoto(options, cookie, stdout, stderr)
	case contentTypeShortDrama:
		return runShortDrama(options, cookie, stdout, stderr)
	case contentTypeProfile:
		return runProfile(options, cookie, stdout, stderr)
	case contentTypeLive:
		return runLive(options, cookie, stdout, stderr)
	default:
		return fmt.Errorf("unsupported TikTok URL type %q", options.content)
	}
}

// resolveCookieSources returns a Cookie header value from -cookies-file or
// -cookies-from-browser. A -cookies-file path takes precedence over
// -cookies-from-browser. The returned header is sent on every subsequent
// request because each content-type runner passes it into the shared session.
func resolveCookieSources(options options) (string, error) {
	if options.cookiesFile != "" {
		return cookies.LoadCookieFileHeader(options.cookiesFile)
	}
	if options.cookiesBrowser != "" {
		return cookies.LoadTikTokCookieHeader(options.cookiesBrowser)
	}
	return "", nil
}

// stringListFlag collects repeated flag values such as -headers.
type stringListFlag []string

// String implements flag.Value.
func (list *stringListFlag) String() string {
	return strings.Join(*list, ", ")
}

// Set appends one repeated flag value.
func (list *stringListFlag) Set(value string) error {
	*list = append(*list, value)
	return nil
}

// parseHeaderPairs turns repeated -headers "Key: Value" occurrences into a
// canonical header map. Each occurrence must be one pair so header values that
// contain colons or semicolons are preserved untouched.
func parseHeaderPairs(values []string) (map[string]string, error) {
	headers := make(map[string]string, len(values))
	for _, raw := range values {
		key, value, ok := strings.Cut(raw, ":")
		if !ok {
			return nil, fmt.Errorf("invalid -headers value %q, want \"Key: Value\"", raw)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid -headers value %q: empty header name", raw)
		}
		headers[http.CanonicalHeaderKey(key)] = value
	}
	return headers, nil
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var options options
	var headerValues stringListFlag
	flags := flag.NewFlagSet("tiktok_crawler", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.BoolVar(&options.json, "json", false, "print resolved metadata as JSON")
	flags.StringVar(&options.output, "output", "", "download destination; an output directory for profiles and Photo Posts (ignored for LIVE URLs)")
	flags.StringVar(&options.quality, "quality", "best", "video height: best, 576, 720, 1080p, and so on (ignored for Photo Posts and LIVE URLs)")
	flags.BoolVar(&options.watermark, "watermark", false, "download TikTok's official watermarked video or image variant (ignored for LIVE URLs)")
	flags.BoolVar(&options.verbose, "verbose", false, "print a LIVE room summary to stderr (ignored for video URLs)")
	flags.StringVar(&options.cookiesFile, "cookies-file", "", "path to a TikTok cookies .txt file (Netscape cookie-jar export or raw Cookie header value)")
	flags.StringVar(&options.cookiesBrowser, "cookies-from-browser", "", "read a TikTok Cookie header from an installed browser (brave, chrome, edge, firefox, safari, ...)")
	flags.Var(&headerValues, "headers", "additional HTTP header to send on every request, as 'Key: Value'; User-Agent can be set here; may be repeated")
	flags.DurationVar(&options.timeout, "timeout", 20*time.Second, "LIVE request timeout (ignored for video URLs)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] <TikTok video, Photo Post, profile, Short Drama episode, or LIVE URL>\nOptions may appear before or after the URL.\n\n", flags.Name())
		flags.PrintDefaults()
	}

	if err := flags.Parse(cliargs.ReorderInterspersedFlags(args, "output", "quality", "cookies-file", "cookies-from-browser", "headers", "timeout")); err != nil {
		return options, err
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return options, errors.New("exactly one TikTok video, Photo Post, profile, Short Drama episode, or LIVE URL is required")
	}
	options.inputURL = flags.Arg(0)

	content, err := detectContentType(options.inputURL)
	if err != nil {
		return options, err
	}
	options.content = content
	headers, err := parseHeaderPairs(headerValues)
	if err != nil {
		return options, err
	}
	if len(headers) > 0 {
		options.headers = headers
	}

	switch options.content {
	case contentTypeVideo, contentTypeShortDrama, contentTypeProfile:
		options.quality = strings.ToLower(strings.TrimSpace(options.quality))
		options.output = strings.TrimSpace(options.output)
		if options.json && (options.output != "" || options.watermark || options.quality != "best") {
			return options, errors.New("-json cannot be combined with -output, -watermark, or a custom -quality for downloadable URLs")
		}
		if options.quality != "best" {
			height, parseErr := strconv.Atoi(strings.TrimSuffix(options.quality, "p"))
			if parseErr != nil || height <= 0 {
				return options, errors.New("quality must be best or a positive height such as 720 or 1080p")
			}
		}
	case contentTypePhoto:
		options.quality = "best"
		options.output = strings.TrimSpace(options.output)
		if options.json && (options.output != "" || options.watermark) {
			return options, errors.New("-json cannot be combined with -output or -watermark for Photo Posts")
		}
	default:
		if options.timeout <= 0 {
			return options, errors.New("timeout must be greater than zero")
		}
	}
	return options, nil
}

func detectContentType(rawURL string) (contentType, error) {
	parsed, err := tiktok.ParseURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid TikTok URL %q", rawURL)
	}
	switch {
	case livePathPattern.MatchString(parsed.EscapedPath()):
		return contentTypeLive, nil
	case videoPathPattern.MatchString(parsed.EscapedPath()):
		return contentTypeVideo, nil
	case photoPathPattern.MatchString(parsed.EscapedPath()):
		return contentTypePhoto, nil
	case shortDramaPathPattern.MatchString(parsed.EscapedPath()):
		return contentTypeShortDrama, nil
	case profilePathPattern.MatchString(parsed.EscapedPath()):
		return contentTypeProfile, nil
	default:
		return "", fmt.Errorf("URL must be a TikTok video, Photo Post, profile, Short Drama episode, or LIVE URL: %q", rawURL)
	}
}

func runVideo(options options, cookie string, stdout, stderr io.Writer) error {
	client, err := video.NewClient(video.ClientOptions{Cookie: cookie, Headers: options.headers})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), videoRequestTimeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.inputURL)
	if err != nil {
		return err
	}
	if options.json {
		return printJSON(stdout, result)
	}
	return downloadVideo(ctx, client.Session(), downloader.FileInfo{
		Author:  result.Video.Author.UniqueID,
		VideoID: result.Video.ID,
		Referer: result.FinalURL,
	}, result.Media, options, stdout, stderr)
}

func runShortDrama(options options, cookie string, stdout, stderr io.Writer) error {
	client, err := shortdrama.NewClient(shortdrama.ClientOptions{Cookie: cookie, Headers: options.headers})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), videoRequestTimeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.inputURL)
	if err != nil {
		return err
	}
	if options.json {
		return printJSON(stdout, result)
	}
	return downloadVideo(ctx, client.Session(), downloader.FileInfo{
		Author:  result.Video.Author.UniqueID,
		VideoID: result.Video.ID,
		Referer: result.FinalURL,
	}, result.Media, options, stdout, stderr)
}

func runPhoto(options options, cookie string, stdout, stderr io.Writer) error {
	client, err := photo.NewClient(photo.ClientOptions{Cookie: cookie, Headers: options.headers})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), videoRequestTimeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.inputURL)
	if err != nil {
		return err
	}
	if options.json {
		return printJSON(stdout, result)
	}
	return downloadPhoto(ctx, client.Session(), downloader.FileInfo{
		Author:  result.Post.Author.UniqueID,
		VideoID: result.Post.ID,
		Referer: result.FinalURL,
	}, result.Images, options, stdout, stderr)
}

func runProfile(options options, cookie string, stdout, stderr io.Writer) error {
	client, err := profile.NewClient(profile.ClientOptions{Cookie: cookie, Headers: options.headers})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), videoRequestTimeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.inputURL)
	if err != nil {
		return err
	}
	if options.json {
		return printJSON(stdout, result)
	}
	if len(result.Videos) == 0 {
		return errors.New("TikTok creator embed returned no public videos to download")
	}

	items := make([]downloader.BatchItem, 0, len(result.Videos))
	for index, item := range result.Videos {
		resolved, resolveErr := client.Video().Resolve(ctx, item.URL)
		if resolveErr != nil {
			return fmt.Errorf("resolve profile video %d/%d (%s): %w", index+1, len(result.Videos), item.ID, resolveErr)
		}
		items = append(items, downloader.BatchItem{
			Variants: resolved.Media,
			File: downloader.FileInfo{
				Author:  resolved.Video.Author.UniqueID,
				VideoID: resolved.Video.ID,
				Referer: resolved.FinalURL,
			},
		})
	}

	outputDirectory := options.output
	if outputDirectory == "" {
		outputDirectory = defaultProfileOutputDirectory(result.User.UniqueID)
	}
	progress := newProgressDisplay(stderr)
	downloads, err := downloader.DownloadAll(ctx, client.Session(), items, downloader.FileInfo{
		Author: result.User.UniqueID, VideoID: "videos", Referer: result.FinalURL,
	}, downloader.BatchOptions{
		OutputDir: outputDirectory, Quality: options.quality, Watermarked: options.watermark,
		Progress: func(update downloader.BatchProgress) {
			progress.update(update.DownloadProgress)
		},
		OnStart: func(start downloader.BatchStart) {
			progress.startProfileVideo(start.Index, start.Total, start.File.Author, start.File.VideoID, start.Media, start.OutputPath)
		},
	})
	if err != nil {
		progress.stop()
		return err
	}
	progress.completeVideoBatch(downloads.Downloads)
	for _, download := range downloads.Downloads {
		if _, err := fmt.Fprintln(stdout, download.Path); err != nil {
			return err
		}
	}
	return nil
}

func defaultProfileOutputDirectory(username string) string {
	username = strings.Trim(strings.TrimSpace(username), "._")
	if username == "" {
		username = "tiktok"
	}
	return username + "_videos"
}

func downloadVideo(
	ctx context.Context,
	session *tiktok.Session,
	file downloader.FileInfo,
	variants []media.Variant,
	options options,
	stdout io.Writer,
	stderr io.Writer,
) error {
	progress := newProgressDisplay(stderr)
	download, err := downloader.Download(ctx, session, variants, file, downloader.Options{
		OutputPath:  options.output,
		Quality:     options.quality,
		Watermarked: options.watermark,
		Progress:    progress.update,
		OnStart: func(start downloader.DownloadStart) {
			progress.start(file.Author, file.VideoID, start.Media, start.OutputPath)
		},
	})
	if err != nil {
		progress.stop()
		return err
	}
	progress.complete(download)
	_, err = fmt.Fprintln(stdout, download.Path)
	return err
}

func downloadPhoto(
	ctx context.Context,
	session *tiktok.Session,
	file downloader.FileInfo,
	images []photo.Image,
	options options,
	stdout io.Writer,
	stderr io.Writer,
) error {
	items := make([]downloader.BatchItem, 0, len(images))
	for _, image := range images {
		items = append(items, downloader.BatchItem{Variants: image.Media})
	}
	progress := newProgressDisplay(stderr)
	downloads, err := downloader.DownloadAll(ctx, session, items, file, downloader.BatchOptions{
		OutputDir:   options.output,
		Watermarked: options.watermark,
		Progress: func(update downloader.BatchProgress) {
			progress.update(update.DownloadProgress)
		},
		OnStart: func(start downloader.BatchStart) {
			progress.startImage(start.Index, start.Total, file.Author, file.VideoID, start.Media, start.OutputPath)
		},
	})
	if err != nil {
		progress.stop()
		return err
	}
	progress.completeBatch(downloads.Downloads)
	for _, download := range downloads.Downloads {
		if _, err := fmt.Fprintln(stdout, download.Path); err != nil {
			return err
		}
	}
	return nil
}

func runLive(options options, cookie string, stdout, stderr io.Writer) error {
	client, err := livestream.NewClient(livestream.ClientOptions{Cookie: cookie, Headers: options.headers})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.inputURL)
	if err != nil {
		return err
	}
	if options.verbose {
		if err := printSummary(stderr, result); err != nil {
			return err
		}
	}
	if options.json {
		return printJSON(stdout, result)
	}
	return printStreams(stdout, result.Streams)
}

func printJSON(writer io.Writer, result any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printSummary(writer io.Writer, result *livestream.Result) error {
	_, err := fmt.Fprintf(writer, "@%s | %s | room=%s | viewers=%d | source=%s\n", result.User.UniqueID, result.User.Nickname, result.User.RoomID, result.Live.ViewerCount, result.Source)
	return err
}

func printStreams(writer io.Writer, streams []livestream.Stream) error {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "CODEC\tQUALITY\tLINE\tFORMAT\tRESOLUTION\tBITRATE\tEXPIRES\tURL")
	for _, stream := range streams {
		expires := ""
		if stream.ExpiresAt != nil {
			expires = stream.ExpiresAt.Format(time.RFC3339)
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", stream.Codec, stream.Quality, stream.Line, stream.Protocol, stream.Resolution, stream.Bitrate, expires, stream.URL)
	}
	return table.Flush()
}

type progressDisplay struct {
	writer       io.Writer
	startedAt    time.Time
	lastRendered time.Time
	lastBytes    int64
	lastWidth    int
	lineVisible  bool
}

func newProgressDisplay(writer io.Writer) *progressDisplay { return &progressDisplay{writer: writer} }

func (display *progressDisplay) start(username, videoID string, variant *media.Variant, outputPath string) {
	display.startAsset("Downloading TikTok video", "Video", "Media", username, videoID, variant, outputPath)
}

func (display *progressDisplay) startImage(index, total int, username, photoID string, variant *media.Variant, outputPath string) {
	display.startAsset(fmt.Sprintf("Downloading TikTok image %d/%d", index, total), "Photo", "Image", username, photoID, variant, outputPath)
}

func (display *progressDisplay) startProfileVideo(index, total int, username, videoID string, variant *media.Variant, outputPath string) {
	display.startAsset(fmt.Sprintf("Downloading TikTok profile video %d/%d", index, total), "Video", "Media", username, videoID, variant, outputPath)
}

func (display *progressDisplay) startAsset(title, idLabel, mediaLabel, username, contentID string, variant *media.Variant, outputPath string) {
	display.stop()
	if username == "" {
		username = "unknown"
	}
	watermark := "No watermark"
	if variant.Watermarked {
		watermark = "TikTok watermark"
	}
	codec := displayCodec(variant.Codec)
	if strings.EqualFold(variant.Type, "image") && variant.Format != "" {
		codec = strings.ToUpper(variant.Format)
	}
	details := []string{watermark, codec, variant.Quality}
	if variant.Width > 0 && variant.Height > 0 {
		details = append(details, fmt.Sprintf("%dx%d", variant.Width, variant.Height))
	}
	if variant.Size > 0 {
		details = append(details, formatBytes(variant.Size))
	}
	fmt.Fprintln(display.writer, title)
	fmt.Fprintf(display.writer, "  Author:  @%s\n  %s:   %s\n  %s:   %s\n  Output:  %s\n", username, idLabel, contentID, mediaLabel, strings.Join(details, " | "), outputPath)
}

func (display *progressDisplay) update(progress downloader.DownloadProgress) {
	now := time.Now()
	if display.startedAt.IsZero() || progress.DownloadedBytes < display.lastBytes {
		display.startedAt, display.lastRendered = now, time.Time{}
	}
	display.lastBytes = progress.DownloadedBytes
	complete := progress.TotalBytes > 0 && progress.DownloadedBytes >= progress.TotalBytes
	if !complete && !display.lastRendered.IsZero() && now.Sub(display.lastRendered) < 100*time.Millisecond {
		return
	}
	display.lastRendered = now
	display.render(progress, now)
}

func (display *progressDisplay) render(progress downloader.DownloadProgress, now time.Time) {
	elapsed := now.Sub(display.startedAt)
	bytesPerSecond := float64(0)
	if elapsed > 0 {
		bytesPerSecond = float64(progress.DownloadedBytes) / elapsed.Seconds()
	}
	var line string
	if progress.TotalBytes > 0 {
		ratio := float64(progress.DownloadedBytes) / float64(progress.TotalBytes)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		eta := "--:--"
		if bytesPerSecond > 0 && progress.DownloadedBytes < progress.TotalBytes {
			eta = formatDuration(time.Duration(float64(progress.TotalBytes-progress.DownloadedBytes) / bytesPerSecond * float64(time.Second)))
		} else if progress.DownloadedBytes >= progress.TotalBytes {
			eta = "00:00"
		}
		line = fmt.Sprintf("  [%s] %6.2f%%  %s / %s  %s/s  ETA %s", progressBar(ratio, 28), ratio*100, formatBytes(progress.DownloadedBytes), formatBytes(progress.TotalBytes), formatBytes(int64(bytesPerSecond)), eta)
	} else {
		line = fmt.Sprintf("  Downloaded %s  %s/s  Elapsed %s", formatBytes(progress.DownloadedBytes), formatBytes(int64(bytesPerSecond)), formatDuration(elapsed))
	}
	padding := ""
	if display.lastWidth > len(line) {
		padding = strings.Repeat(" ", display.lastWidth-len(line))
	}
	fmt.Fprintf(display.writer, "\r%s%s", line, padding)
	display.lastWidth, display.lineVisible = len(line), true
}

func (display *progressDisplay) complete(result *downloader.DownloadResult) {
	display.stop()
	elapsed := time.Since(display.startedAt)
	if display.startedAt.IsZero() {
		elapsed = 0
	}
	bytesPerSecond := float64(0)
	if elapsed > 0 {
		bytesPerSecond = float64(result.Bytes) / elapsed.Seconds()
	}
	fmt.Fprintf(display.writer, "Completed: %s in %s (%s/s)\n", formatBytes(result.Bytes), formatElapsed(elapsed), formatBytes(int64(bytesPerSecond)))
}

func (display *progressDisplay) completeBatch(downloads []downloader.DownloadResult) {
	display.stop()
	var totalBytes int64
	for _, download := range downloads {
		totalBytes += download.Bytes
	}
	fmt.Fprintf(display.writer, "Completed: %d images (%s)\n", len(downloads), formatBytes(totalBytes))
}

func (display *progressDisplay) completeVideoBatch(downloads []downloader.DownloadResult) {
	display.stop()
	var totalBytes int64
	for _, download := range downloads {
		totalBytes += download.Bytes
	}
	fmt.Fprintf(display.writer, "Completed: %d videos (%s)\n", len(downloads), formatBytes(totalBytes))
}

func (display *progressDisplay) stop() {
	if display.lineVisible {
		fmt.Fprintln(display.writer)
		display.lineVisible = false
	}
}

func progressBar(ratio float64, width int) string {
	filled := int(ratio * float64(width))
	if filled >= width {
		return strings.Repeat("=", width)
	}
	if filled <= 0 {
		return ">" + strings.Repeat(" ", width-1)
	}
	return strings.Repeat("=", filled-1) + ">" + strings.Repeat(" ", width-filled)
}

func displayCodec(codec string) string {
	switch strings.ToLower(codec) {
	case "h264":
		return "H.264"
	case "h265":
		return "H.265"
	case "unknown", "":
		return "TikTok default codec"
	default:
		return strings.ToUpper(codec)
	}
}

func formatBytes(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	for _, label := range units {
		value /= unit
		if value < unit || label == units[len(units)-1] {
			return fmt.Sprintf("%.2f %s", value, label)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	totalSeconds := int64(duration.Round(time.Second) / time.Second)
	hours, minutes, seconds := totalSeconds/3600, (totalSeconds%3600)/60, totalSeconds%60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatElapsed(duration time.Duration) string {
	if duration > 0 && duration < time.Second {
		return fmt.Sprintf("%.2fs", duration.Seconds())
	}
	return formatDuration(duration)
}
