package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"tiktok-crawler/internal/livestream"
)

type options struct {
	liveURL   string
	format    string
	quality   string
	codec     string
	codecSet  bool
	formatSet bool
	json      bool
	all       bool
	verbose   bool
	cookie    string
	userAgent string
	timeout   time.Duration
}

type output struct {
	InputURL       string              `json:"input_url"`
	FinalURL       string              `json:"final_url"`
	Source         string              `json:"source"`
	FetchedAt      time.Time           `json:"fetched_at"`
	User           livestream.User     `json:"user"`
	Live           livestream.Live     `json:"live"`
	SelectedStream *livestream.Stream  `json:"selected_stream,omitempty"`
	Streams        []livestream.Stream `json:"streams"`
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

	client, err := livestream.NewClient(livestream.ClientOptions{
		Cookie:    options.cookie,
		UserAgent: options.userAgent,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()

	result, err := client.Resolve(ctx, options.liveURL)
	if err != nil {
		return err
	}
	selected, err := selectPreferredStream(result.Streams, options)
	if err != nil {
		return err
	}

	if options.verbose {
		printSummary(stderr, result, selected)
	}
	if options.json {
		return printJSON(stdout, result, selected)
	}
	if options.all {
		printAllStreams(stdout, result.Streams)
		return nil
	}
	_, err = fmt.Fprintln(stdout, selected.URL)
	return err
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var options options
	flags := flag.NewFlagSet("tiktok_livestream", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.liveURL, "url", "", "TikTok LIVE URL (may also be supplied as the final argument)")
	flags.StringVar(&options.format, "format", "hls", "preferred stream format: hls, flv, cmaf, dash, lls, or auto")
	flags.StringVar(&options.quality, "quality", "best", "quality: best, origin, uhd60, uhd, hd60, hd, sd, ld, or ao")
	flags.StringVar(&options.codec, "codec", "h264", "preferred video codec: h264, h265, or auto")
	flags.BoolVar(&options.json, "json", false, "print metadata and all streams as JSON")
	flags.BoolVar(&options.all, "all", false, "print every stream instead of only the selected URL")
	flags.BoolVar(&options.verbose, "verbose", false, "print a short room summary to stderr")
	flags.StringVar(&options.cookie, "cookie", "", "TikTok Cookie header value (or use TIKTOK_COOKIE)")
	flags.StringVar(&options.userAgent, "user-agent", livestream.DefaultUserAgent, "User-Agent sent to TikTok")
	flags.DurationVar(&options.timeout, "timeout", 20*time.Second, "overall request timeout")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] <TikTok LIVE URL>\n\n", flags.Name())
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return options, err
	}
	flags.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "codec":
			options.codecSet = true
		case "format":
			options.formatSet = true
		}
	})
	if options.liveURL == "" {
		if flags.NArg() != 1 {
			flags.Usage()
			return options, errors.New("exactly one TikTok LIVE URL is required")
		}
		options.liveURL = flags.Arg(0)
	} else if flags.NArg() != 0 {
		return options, errors.New("do not pass the URL through both -url and the final argument")
	}

	options.codec = strings.ToLower(strings.TrimSpace(options.codec))
	options.quality = strings.ToLower(strings.TrimSpace(options.quality))
	options.format = strings.ToLower(strings.TrimSpace(options.format))
	if options.cookie == "" {
		options.cookie = os.Getenv("TIKTOK_COOKIE")
	}
	if err := validateOptions(options); err != nil {
		return options, err
	}
	return options, nil
}

func selectPreferredStream(streams []livestream.Stream, cliOptions options) (*livestream.Stream, error) {
	candidates := []livestream.SelectOptions{{
		Codec:   cliOptions.codec,
		Quality: cliOptions.quality,
		Format:  cliOptions.format,
	}}
	if !cliOptions.codecSet {
		candidates = append(candidates, livestream.SelectOptions{
			Codec:   "auto",
			Quality: cliOptions.quality,
			Format:  cliOptions.format,
		})
	}
	if !cliOptions.formatSet {
		candidates = append(candidates, livestream.SelectOptions{
			Codec:   cliOptions.codec,
			Quality: cliOptions.quality,
			Format:  "auto",
		})
	}
	if !cliOptions.codecSet && !cliOptions.formatSet {
		candidates = append(candidates, livestream.SelectOptions{
			Codec:   "auto",
			Quality: cliOptions.quality,
			Format:  "auto",
		})
	}

	seen := make(map[string]bool, len(candidates))
	var selectionErr error
	for _, candidate := range candidates {
		key := candidate.Codec + "\x00" + candidate.Quality + "\x00" + candidate.Format
		if seen[key] {
			continue
		}
		seen[key] = true
		selected, err := livestream.SelectStream(streams, candidate)
		if err == nil {
			return selected, nil
		}
		selectionErr = err
	}
	return nil, selectionErr
}

func validateOptions(options options) error {
	formats := map[string]bool{
		"auto": true,
		"hls":  true,
		"flv":  true,
		"cmaf": true,
		"dash": true,
		"lls":  true,
	}
	if !formats[options.format] {
		return fmt.Errorf("invalid format %q", options.format)
	}
	codecs := map[string]bool{"auto": true, "h264": true, "h265": true}
	if !codecs[options.codec] {
		return fmt.Errorf("invalid codec %q", options.codec)
	}
	if options.quality == "" {
		return errors.New("quality must not be empty")
	}
	if options.timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	return nil
}

func printSummary(writer io.Writer, result *livestream.Result, selected *livestream.Stream) {
	fmt.Fprintf(
		writer,
		"@%s | %s | room=%s | viewers=%d | source=%s\n",
		result.User.UniqueID,
		result.User.Nickname,
		result.User.RoomID,
		result.Live.ViewerCount,
		result.Source,
	)
	fmt.Fprintf(
		writer,
		"%s %s %s %s\n",
		selected.Codec,
		selected.Quality,
		selected.Protocol,
		selected.Resolution,
	)
}

func printJSON(writer io.Writer, result *livestream.Result, selected *livestream.Stream) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output{
		InputURL:       result.InputURL,
		FinalURL:       result.FinalURL,
		Source:         result.Source,
		FetchedAt:      result.FetchedAt,
		User:           result.User,
		Live:           result.Live,
		SelectedStream: selected,
		Streams:        result.Streams,
	})
}

func printAllStreams(writer io.Writer, streams []livestream.Stream) {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "CODEC\tQUALITY\tLINE\tFORMAT\tRESOLUTION\tBITRATE\tEXPIRES\tURL")
	for _, stream := range streams {
		expires := ""
		if stream.ExpiresAt != nil {
			expires = stream.ExpiresAt.Format(time.RFC3339)
		}
		fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			stream.Codec,
			stream.Quality,
			stream.Line,
			stream.Protocol,
			stream.Resolution,
			stream.Bitrate,
			expires,
			stream.URL,
		)
	}
	_ = table.Flush()
}
