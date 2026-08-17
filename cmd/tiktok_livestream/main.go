package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"tiktok-crawler/internal/livestream"
)

type options struct {
	liveURL   string
	json      bool
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

	if options.verbose {
		printSummary(stderr, result)
	}
	if options.json {
		return printJSON(stdout, result)
	}
	return printStreams(stdout, result.Streams)
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var options options
	flags := flag.NewFlagSet("tiktok_livestream", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.liveURL, "url", "", "TikTok LIVE URL (may also be supplied as the final argument)")
	flags.BoolVar(&options.json, "json", false, "print metadata and all streams as JSON")
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
	if options.liveURL == "" {
		if flags.NArg() != 1 {
			flags.Usage()
			return options, errors.New("exactly one TikTok LIVE URL is required")
		}
		options.liveURL = flags.Arg(0)
	} else if flags.NArg() != 0 {
		return options, errors.New("do not pass the URL through both -url and the final argument")
	}

	if options.cookie == "" {
		options.cookie = os.Getenv("TIKTOK_COOKIE")
	}
	if options.timeout <= 0 {
		return options, errors.New("timeout must be greater than zero")
	}
	return options, nil
}

func printSummary(writer io.Writer, result *livestream.Result) {
	fmt.Fprintf(
		writer,
		"@%s | %s | room=%s | viewers=%d | source=%s\n",
		result.User.UniqueID,
		result.User.Nickname,
		result.User.RoomID,
		result.Live.ViewerCount,
		result.Source,
	)
}

func printJSON(writer io.Writer, result *livestream.Result) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printStreams(writer io.Writer, streams []livestream.Stream) error {
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
	return table.Flush()
}
