package video

import "context"

// ResolvedDownloadOptions controls downloading a media variant from resolved video metadata.
type ResolvedDownloadOptions struct {
	OutputPath  string
	Quality     string
	Watermarked bool
	Progress    func(DownloadProgress)
	OnStart     func(DownloadStart)
}

// DownloadStart describes the media variant and destination selected for a download.
type DownloadStart struct {
	Result     *Result
	Media      *Media
	OutputPath string
}

// DownloadResolved selects media from result and saves it to disk.
func (client *Client) DownloadResolved(
	ctx context.Context,
	result *Result,
	options ResolvedDownloadOptions,
) (*DownloadResult, error) {
	selected, err := SelectMedia(result.Media, SelectOptions{
		Quality:     options.Quality,
		Watermarked: options.Watermarked,
	})
	if err != nil {
		return nil, err
	}

	outputPath := options.OutputPath
	if outputPath == "" {
		outputPath = DefaultFilename(result, selected)
	}
	if options.OnStart != nil {
		options.OnStart(DownloadStart{
			Result:     result,
			Media:      selected,
			OutputPath: outputPath,
		})
	}
	return client.Download(ctx, *selected, DownloadOptions{
		OutputPath: outputPath,
		Referer:    result.FinalURL,
		Progress:   options.Progress,
	})
}
