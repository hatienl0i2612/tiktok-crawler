package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"tiktok-crawler/internal/cliargs"
	"tiktok-crawler/internal/livestream"
	"tiktok-crawler/internal/video"
)

const videoRequestTimeout = 5 * time.Minute

var (
	livePathPattern  = regexp.MustCompile(`^/@[^/]+/live/?$`)
	videoPathPattern = regexp.MustCompile(`^/@[^/]+/video/[0-9]+/?$`)
)

type contentType string

const (
	contentTypeLive  contentType = "live"
	contentTypeVideo contentType = "video"
)

type options struct {
	inputURL  string
	content   contentType
	json      bool
	output    string
	quality   string
	watermark bool
	verbose   bool
	cookie    string
	userAgent string
	timeout   time.Duration
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

	switch options.content {
	case contentTypeVideo:
		return runVideo(options, stdout, stderr)
	case contentTypeLive:
		return runLive(options, stdout, stderr)
	default:
		return fmt.Errorf("unsupported TikTok URL type %q", options.content)
	}
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var options options
	flags := flag.NewFlagSet("tiktok_crawler", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.inputURL, "url", "", "TikTok video or LIVE URL (may also be supplied as the final argument)")
	flags.BoolVar(&options.json, "json", false, "print resolved metadata as JSON")
	flags.StringVar(&options.output, "output", "", "video download destination (ignored for LIVE URLs)")
	flags.StringVar(&options.quality, "quality", "best", "video height: best, 576, 720, 1080p, and so on (ignored for LIVE URLs)")
	flags.BoolVar(&options.watermark, "watermark", false, "download TikTok's official watermarked video variant (ignored for LIVE URLs)")
	flags.BoolVar(&options.verbose, "verbose", false, "print a LIVE room summary to stderr (ignored for video URLs)")
	flags.StringVar(&options.cookie, "cookie", "", "TikTok Cookie header value (or use TIKTOK_COOKIE)")
	flags.StringVar(&options.userAgent, "user-agent", livestream.DefaultUserAgent, "User-Agent sent to TikTok LIVE (ignored for video URLs)")
	flags.DurationVar(&options.timeout, "timeout", 20*time.Second, "LIVE request timeout (ignored for video URLs)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] <TikTok video or LIVE URL>\nOptions may appear before or after the URL.\n\n", flags.Name())
		flags.PrintDefaults()
	}

	if err := flags.Parse(cliargs.ReorderInterspersedFlags(args, "url", "output", "quality", "cookie", "user-agent", "timeout")); err != nil {
		return options, err
	}
	if options.inputURL == "" {
		if flags.NArg() != 1 {
			flags.Usage()
			return options, errors.New("exactly one TikTok video or LIVE URL is required")
		}
		options.inputURL = flags.Arg(0)
	} else if flags.NArg() != 0 {
		return options, errors.New("do not pass the URL through both -url and the final argument")
	}

	content, err := detectContentType(options.inputURL)
	if err != nil {
		return options, err
	}
	options.content = content
	if options.cookie == "" {
		options.cookie = os.Getenv("TIKTOK_COOKIE")
	}

	if options.content == contentTypeVideo {
		options.quality = strings.ToLower(strings.TrimSpace(options.quality))
		options.output = strings.TrimSpace(options.output)
		if options.json && (options.output != "" || options.watermark || options.quality != "best") {
			return options, errors.New("-json cannot be combined with -output, -watermark, or a custom -quality for video URLs")
		}
		if options.quality != "best" {
			height, parseErr := strconv.Atoi(strings.TrimSuffix(options.quality, "p"))
			if parseErr != nil || height <= 0 {
				return options, errors.New("quality must be best or a positive height such as 720 or 1080p")
			}
		}
	} else if options.timeout <= 0 {
		return options, errors.New("timeout must be greater than zero")
	}
	return options, nil
}

func detectContentType(rawURL string) (contentType, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("invalid TikTok URL %q", rawURL)
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "tiktok.com" && !strings.HasSuffix(host, ".tiktok.com") {
		return "", fmt.Errorf("URL must use a tiktok.com host: %q", rawURL)
	}
	switch {
	case livePathPattern.MatchString(parsed.EscapedPath()):
		return contentTypeLive, nil
	case videoPathPattern.MatchString(parsed.EscapedPath()):
		return contentTypeVideo, nil
	default:
		return "", fmt.Errorf("URL must be a TikTok video or LIVE URL: %q", rawURL)
	}
}

func runVideo(options options, stdout, stderr io.Writer) error {
	client, err := video.NewClient(video.ClientOptions{Cookie: options.cookie})
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
		return printVideoJSON(stdout, result)
	}
	progress := newProgressDisplay(stderr)
	download, err := client.DownloadResolved(ctx, result, video.ResolvedDownloadOptions{
		OutputPath:  options.output,
		Quality:     options.quality,
		Watermarked: options.watermark,
		Progress:    progress.update,
		OnStart: func(start video.DownloadStart) {
			progress.start(start.Result, start.Media, start.OutputPath)
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

func runLive(options options, stdout, stderr io.Writer) error {
	client, err := livestream.NewClient(livestream.ClientOptions{Cookie: options.cookie, UserAgent: options.userAgent})
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
		printSummary(stderr, result)
	}
	if options.json {
		return printLiveJSON(stdout, result)
	}
	return printStreams(stdout, result.Streams)
}

func printVideoJSON(writer io.Writer, result *video.Result) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printLiveJSON(writer io.Writer, result *livestream.Result) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printSummary(writer io.Writer, result *livestream.Result) {
	fmt.Fprintf(writer, "@%s | %s | room=%s | viewers=%d | source=%s\n", result.User.UniqueID, result.User.Nickname, result.User.RoomID, result.Live.ViewerCount, result.Source)
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

func (display *progressDisplay) start(result *video.Result, media *video.Media, outputPath string) {
	absolutePath, err := filepath.Abs(outputPath)
	if err != nil {
		absolutePath = outputPath
	}
	username := result.Video.Author.UniqueID
	if username == "" {
		username = "unknown"
	}
	variant := "No watermark"
	if media.Watermarked {
		variant = "TikTok watermark"
	}
	details := []string{variant, displayCodec(media.Codec), media.Quality}
	if media.Width > 0 && media.Height > 0 {
		details = append(details, fmt.Sprintf("%dx%d", media.Width, media.Height))
	}
	if media.Size > 0 {
		details = append(details, formatBytes(media.Size))
	}
	fmt.Fprintln(display.writer, "Downloading TikTok video")
	fmt.Fprintf(display.writer, "  Author:  @%s\n  Video:   %s\n  Media:   %s\n  Output:  %s\n", username, result.Video.ID, strings.Join(details, " | "), absolutePath)
}

func (display *progressDisplay) update(progress video.DownloadProgress) {
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

func (display *progressDisplay) render(progress video.DownloadProgress, now time.Time) {
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

func (display *progressDisplay) complete(result *video.DownloadResult) {
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
