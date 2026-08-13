package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tiktok-crawler/internal/video"
)

type options struct {
	videoURL  string
	download  bool
	output    string
	force     bool
	watermark bool
	codec     string
	quality   string
	urlOnly   bool
	verbose   bool
	cookie    string
	userAgent string
	timeout   time.Duration
}

type output struct {
	InputURL      string        `json:"input_url"`
	FinalURL      string        `json:"final_url"`
	Sources       []string      `json:"sources"`
	FetchedAt     time.Time     `json:"fetched_at"`
	Warnings      []string      `json:"warnings,omitempty"`
	Video         video.Video   `json:"video"`
	SelectedMedia *video.Media  `json:"selected_media"`
	Media         []video.Media `json:"media"`
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

	client, err := video.NewClient(video.ClientOptions{
		Cookie:    options.cookie,
		UserAgent: options.userAgent,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.videoURL)
	if err != nil {
		return err
	}
	selected, err := video.SelectMedia(result.Media, video.SelectOptions{
		Codec:       options.codec,
		Quality:     options.quality,
		Watermarked: options.watermark,
	})
	if err != nil {
		return err
	}

	if options.verbose {
		printSummary(stderr, result, selected)
	}
	if options.download {
		outputPath := options.output
		if outputPath == "" {
			outputPath = video.DefaultFilename(result, selected)
		}
		progress := newProgressDisplay(stderr)
		progress.start(result, selected, outputPath)
		downloadOptions := video.DownloadOptions{
			OutputPath: outputPath,
			Referer:    result.FinalURL,
			Overwrite:  options.force,
			Progress:   progress.update,
		}
		download, err := client.Download(ctx, *selected, downloadOptions)
		if err != nil {
			progress.stop()
			return err
		}
		progress.complete(download)
		_, err = fmt.Fprintln(stdout, download.Path)
		return err
	}
	if options.urlOnly {
		_, err = fmt.Fprintln(stdout, selected.URL)
		return err
	}
	return printJSON(stdout, result, selected)
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var options options
	flags := flag.NewFlagSet("tiktok", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.videoURL, "url", "", "TikTok video URL (may also be supplied as the final argument)")
	flags.BoolVar(&options.download, "download", false, "download the selected video instead of printing metadata")
	flags.StringVar(&options.output, "output", "", "download destination; setting it also enables -download")
	flags.BoolVar(&options.force, "force", false, "replace an existing download destination")
	flags.BoolVar(&options.watermark, "watermark", false, "require TikTok's official watermarked download variant")
	flags.StringVar(&options.codec, "codec", "h264", "video codec: h264, h265, or auto")
	flags.StringVar(&options.quality, "quality", "best", "video height: best, 576, 720, 1080p, and so on")
	flags.BoolVar(&options.urlOnly, "url-only", false, "print only the selected signed media URL")
	flags.BoolVar(&options.verbose, "verbose", false, "print a short video summary to stderr")
	flags.StringVar(&options.cookie, "cookie", "", "TikTok Cookie header value (or use TIKTOK_COOKIE)")
	flags.StringVar(&options.userAgent, "user-agent", video.DefaultUserAgent, "User-Agent sent to TikTok")
	flags.DurationVar(&options.timeout, "timeout", 5*time.Minute, "overall metadata and download timeout")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] <TikTok video URL>\n\n", flags.Name())
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return options, err
	}
	codecExplicit := false
	flags.Visit(func(current *flag.Flag) {
		if current.Name == "codec" {
			codecExplicit = true
		}
	})
	if options.videoURL == "" {
		if flags.NArg() != 1 {
			flags.Usage()
			return options, errors.New("exactly one TikTok video URL is required")
		}
		options.videoURL = flags.Arg(0)
	} else if flags.NArg() != 0 {
		return options, errors.New("do not pass the URL through both -url and the final argument")
	}

	options.codec = strings.ToLower(strings.TrimSpace(options.codec))
	options.quality = strings.ToLower(strings.TrimSpace(options.quality))
	options.output = strings.TrimSpace(options.output)
	if options.watermark && !codecExplicit {
		options.codec = "auto"
	}
	if options.output != "" {
		options.download = true
	}
	if options.cookie == "" {
		options.cookie = os.Getenv("TIKTOK_COOKIE")
	}
	if options.timeout <= 0 {
		return options, errors.New("timeout must be greater than zero")
	}
	if options.force && !options.download {
		return options, errors.New("-force requires -download or -output")
	}
	if options.urlOnly && options.download {
		return options, errors.New("-url-only cannot be combined with -download or -output")
	}
	if options.codec != "h264" && options.codec != "h265" && options.codec != "auto" {
		return options, errors.New("codec must be h264, h265, or auto")
	}
	if options.watermark && codecExplicit && options.codec != "auto" {
		return options, errors.New("-codec cannot filter a watermarked download because TikTok does not label its codec; omit it or use -codec auto")
	}
	quality := strings.TrimSuffix(options.quality, "p")
	if options.quality != "best" {
		height, err := strconv.Atoi(quality)
		if err != nil || height <= 0 {
			return options, errors.New("quality must be best or a positive height such as 720 or 1080p")
		}
	}
	return options, nil
}

func printSummary(writer io.Writer, result *video.Result, selected *video.Media) {
	variant := "no watermark"
	if selected.Watermarked {
		variant = "TikTok watermark"
	}
	fmt.Fprintf(
		writer,
		"@%s | video=%s | %dx%d | %ds | likes=%d | plays=%d\n",
		result.Video.Author.UniqueID,
		result.Video.ID,
		result.Video.Width,
		result.Video.Height,
		result.Video.Duration,
		result.Video.Statistics.LikeCount,
		result.Video.Statistics.PlayCount,
	)
	fmt.Fprintf(
		writer,
		"selected: %s %s %s %dx%d\n",
		variant,
		selected.Codec,
		selected.Quality,
		selected.Width,
		selected.Height,
	)
}

func printJSON(writer io.Writer, result *video.Result, selected *video.Media) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output{
		InputURL:      result.InputURL,
		FinalURL:      result.FinalURL,
		Sources:       result.Sources,
		FetchedAt:     result.FetchedAt,
		Warnings:      result.Warnings,
		Video:         result.Video,
		SelectedMedia: selected,
		Media:         result.Media,
	})
}

type progressDisplay struct {
	writer       io.Writer
	startedAt    time.Time
	lastRendered time.Time
	lastBytes    int64
	lastWidth    int
	lineVisible  bool
}

func newProgressDisplay(writer io.Writer) *progressDisplay {
	return &progressDisplay{writer: writer}
}

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
	fmt.Fprintf(display.writer, "  Author:  @%s\n", username)
	fmt.Fprintf(display.writer, "  Video:   %s\n", result.Video.ID)
	fmt.Fprintf(display.writer, "  Media:   %s\n", strings.Join(details, " | "))
	fmt.Fprintf(display.writer, "  Output:  %s\n", absolutePath)
}

func (display *progressDisplay) update(progress video.DownloadProgress) {
	now := time.Now()
	if display.startedAt.IsZero() || progress.DownloadedBytes < display.lastBytes {
		display.startedAt = now
		display.lastRendered = time.Time{}
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
			remaining := float64(progress.TotalBytes-progress.DownloadedBytes) / bytesPerSecond
			eta = formatDuration(time.Duration(remaining * float64(time.Second)))
		} else if progress.DownloadedBytes >= progress.TotalBytes {
			eta = "00:00"
		}
		line = fmt.Sprintf(
			"  [%s] %6.2f%%  %s / %s  %s/s  ETA %s",
			progressBar(ratio, 28),
			ratio*100,
			formatBytes(progress.DownloadedBytes),
			formatBytes(progress.TotalBytes),
			formatBytes(int64(bytesPerSecond)),
			eta,
		)
	} else {
		line = fmt.Sprintf(
			"  Downloaded %s  %s/s  Elapsed %s",
			formatBytes(progress.DownloadedBytes),
			formatBytes(int64(bytesPerSecond)),
			formatDuration(elapsed),
		)
	}

	padding := ""
	if display.lastWidth > len(line) {
		padding = strings.Repeat(" ", display.lastWidth-len(line))
	}
	fmt.Fprintf(display.writer, "\r%s%s", line, padding)
	display.lastWidth = len(line)
	display.lineVisible = true
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
	fmt.Fprintf(
		display.writer,
		"Completed: %s in %s (%s/s)\n",
		formatBytes(result.Bytes),
		formatElapsed(elapsed),
		formatBytes(int64(bytesPerSecond)),
	)
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
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
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
