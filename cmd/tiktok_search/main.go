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
	"time"

	"tiktok-crawler/internal/search"
)

const requestTimeout = 30 * time.Second

type options struct {
	keyword   string
	locale    string
	pageSize  int
	pageIndex int
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
	client, err := search.NewClient(search.ClientOptions{Cookie: os.Getenv("TIKTOK_COOKIE")})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	links, err := client.Search(ctx, search.Options{
		Keyword:   options.keyword,
		Locale:    options.locale,
		PageSize:  options.pageSize,
		PageIndex: options.pageIndex,
	})
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(links)
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var options options
	flags := flag.NewFlagSet("tiktok_search", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.locale, "locale", "", "ranking region, such as VN, US, or vi-VN")
	flags.IntVar(&options.pageSize, "page-size", search.DefaultPageSize, "number of videos to request per page")
	flags.IntVar(&options.pageIndex, "page-index", 0, "zero-based result page index")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] [keyword]\n\n", flags.Name())
		fmt.Fprintln(flags.Output(), "When keyword is omitted, TikTok's default recommended video list is returned.")
		fmt.Fprintln(flags.Output())
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() > 1 {
		flags.Usage()
		return options, errors.New("keyword must be supplied as one quoted argument")
	}
	if flags.NArg() == 1 {
		options.keyword = strings.TrimSpace(flags.Arg(0))
	}
	if options.pageSize < 1 || options.pageSize > search.MaxPageSize {
		return options, fmt.Errorf("page size must be between 1 and %d", search.MaxPageSize)
	}
	if options.pageIndex < 0 {
		return options, errors.New("page index must be zero or greater")
	}
	return options, nil
}
