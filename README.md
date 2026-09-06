English | [Tiếng Việt](README.vi.md) | [简体中文](README.zh-CN.md)

# TikTok Crawler

[![Go Reference](https://pkg.go.dev/badge/github.com/hatienl0i2612/tiktok-crawler.svg)](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)

This repository contains a Go library plus a command-line tool:

- **Library**: importable packages under `github.com/hatienl0i2612/tiktok-crawler` for resolving videos, creator profiles, Photo Posts, Short Drama episodes, and LIVE rooms, plus helpers for cookies, media, and downloads.
- **CLI**: `tiktok_crawler` detects the URL type, crawls public creator profiles, downloads public TikTok videos, Photo Posts, and Short Drama episodes, and opens the best public TikTok LIVE stream in mpv.

The code uses the Go standard library, plus the `browsercookie` package to optionally import cookies from an installed browser.

## Use as a Go library

Add the module to your project:

```bash
go get github.com/hatienl0i2612/tiktok-crawler@latest
```

Then resolve any supported TikTok URL with the high-level client:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hatienl0i2612/tiktok-crawler"
)

func main() {
	config := tiktokcrawler.ClientOptions{
		Cookie:  "ttwid=...; sessionid=...",                                  // optional
		Headers: map[string]string{"User-Agent": "my-custom-agent"},          // optional
	}
	client, err := tiktokcrawler.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Resolve(context.Background(), "https://www.tiktok.com/@example/video/1234567890123456789")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("kind: %s, final_url: %s\n", result.Kind, result.Video.FinalURL)
}
```

For more focused control, use the subpackages directly:

```go
import (
	"github.com/hatienl0i2612/tiktok-crawler/video"       // video.Result, video.Video, ...
	"github.com/hatienl0i2612/tiktok-crawler/profile"     // profile.Result, profile.User, ...
	"github.com/hatienl0i2612/tiktok-crawler/photo"       // photo.Result, photo.Image, ...
	"github.com/hatienl0i2612/tiktok-crawler/livestream"  // livestream.Result, livestream.Stream, ...
	"github.com/hatienl0i2612/tiktok-crawler/mpv"         // portable mpv discovery, install, and playback
	"github.com/hatienl0i2612/tiktok-crawler/shortdrama"  // shortdrama.Result, ...
	"github.com/hatienl0i2612/tiktok-crawler/cookies"     // cookies.LoadTikTokCookieHeader, ...
	"github.com/hatienl0i2612/tiktok-crawler/downloader"  // downloader.Download, ...
)
```

Published module versions follow semantic versioning via Git tags (for example `v0.1.0`); the API is documented on [pkg.go.dev](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler).

## Download a release

Prebuilt binaries are available on the [GitHub Releases page](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest). Download the file that matches your operating system:

| Operating system | Binary |
| --- | --- |
| Linux (amd64) | [`tiktok_crawler-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok_crawler-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-darwin-arm64) |
| Windows (amd64) | [`tiktok_crawler-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-windows-amd64.exe) |

On Linux, make both files executable and run them from the download directory:

```bash
chmod +x tiktok_crawler-linux-amd64
./tiktok_crawler-linux-amd64 -help
./tiktok_crawler-linux-amd64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_crawler-linux-amd64 'https://www.tiktok.com/@example/live'
```

On macOS with Apple Silicon:

```bash
chmod +x tiktok_crawler-darwin-arm64
./tiktok_crawler-darwin-arm64 -help
./tiktok_crawler-darwin-arm64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_crawler-darwin-arm64 'https://www.tiktok.com/@example/live'
```

On Windows, open PowerShell in the download directory:

```powershell
.\tiktok_crawler-windows-amd64.exe -help
.\tiktok_crawler-windows-amd64.exe "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_crawler-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

Replace the example URLs with the TikTok video or LIVE room you want to process. The remaining examples use `go run` for source builds; when using a release, replace `go run ./cmd/tiktok_crawler` with the downloaded executable name shown above.

## Supported URL types

| Type | URL format | Default action |
| --- | --- | --- |
| Video | `https://www.tiktok.com/@username/video/<id>` | Download the video |
| Photo Post / photo Story | `https://www.tiktok.com/@username/photo/<id>` | Download all photos and the post music when available |
| Creator profile | `https://www.tiktok.com/@username` | Download the recent public videos exposed by TikTok |
| Short Drama episode | `https://www.tiktok.com/shortdrama/episode/<id>/<episode>` | Download the episode |
| LIVE | `https://www.tiktok.com/@username/live` | Open the best available stream in mpv |

## CLI options

Usage:

```bash
tiktok_crawler [options] <TikTok URL>
```

Options may appear before or after the URL.

Run the executable without arguments—or double-click it on Windows—to enter interactive mode. Paste a TikTok URL when prompted; errors return to the prompt, and a successful run waits for Enter before closing.

| Option | Description |
| --- | --- |
| `-json` | Print resolved metadata as JSON instead of downloading or playing |
| `-output <path>` | Set the output file or directory; ignored for LIVE URLs |
| `-quality <height>` | Select video quality, such as `720p` or `1080p`; default is `best` |
| `-watermark` | Download TikTok's official watermarked video or image variant |
| `-verbose` | Print a summary for LIVE rooms |
| `-timeout <duration>` | Set the LIVE request timeout; default is `20s` |
| `-cookies-file <path>` | Load TikTok cookies from a Netscape cookie file or raw Cookie header |
| `-cookies-from-browser <browser>` | Load TikTok cookies from an installed browser |
| `-headers 'Key: Value'` | Add an HTTP header; repeat the option to add multiple headers |
| `-version` | Print the CLI version |
| `-help` | Print command help |

## Build

```bash
go build -o tiktok_crawler ./cmd/tiktok_crawler
```

The generated root binaries are ignored by Git. You can also place builds under `bin/`.

## Contributing

Contributions are welcome. If you would like to improve the crawler, add support for another TikTok format, or refine the documentation, please open a pull request.

If you encounter a bug or unexpected TikTok response, please [open an issue](https://github.com/hatienl0i2612/tiktok-crawler/issues) with the affected URL, command, error output, operating system, and tool version when possible. Do not include private cookies or authentication data.
