English | [Tiếng Việt](README.vi.md) | [简体中文](README.zh-CN.md)

# TikTok Crawler

[![Go Reference](https://pkg.go.dev/badge/github.com/hatienl0i2612/tiktok-crawler.svg)](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)

This repository contains a Go library plus a command-line tool:

- **Library**: importable packages under `github.com/hatienl0i2612/tiktok-crawler` for resolving videos, creator profiles, Photo Posts, Short Drama episodes, and LIVE rooms, plus helpers for cookies, media, and downloads.
- **CLI**: `tiktok_crawler` detects the URL type, crawls public creator profiles, downloads public TikTok videos, Photo Posts, and Short Drama episodes, and resolves signed playback URLs for public TikTok LIVE rooms.

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

## Videos

Download the best available video without a watermark (the default behavior):

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok also uses `/video/<id>` URLs for video Stories. When the player API omits a Story, the resolver reads TikTok's embed metadata and the page `playAddr` used by the web player; JSON output includes `"is_story": true`. The no-watermark `playAddr` is selected by default, while `-watermark` selects `downloadAddr`. No browser cookie is required for a public Story that is still available.

Replace `example` and `1234567890123456789` with the username and video ID from the TikTok URL you want to crawl.

Sample output for a video without a watermark:

```text
Downloading TikTok video
  Author:  @example
  Video:   1234567890123456789
  Media:   No watermark | H.264 | 576p | 1024x576 | 23.74 MiB
  Output:  /path/to/example_1234567890123456789_no_watermark_576p_h264.mp4
  [=================>          ]  67.35%  15.99 MiB / 23.74 MiB  8.42 MiB/s  ETA 00:01
  [============================] 100.00%  23.74 MiB / 23.74 MiB  7.92 MiB/s  ETA 00:00
Completed: 23.74 MiB in 00:03 (7.91 MiB/s)
/path/to/example_1234567890123456789_no_watermark_576p_h264.mp4
```

The default filename includes the username, video ID, watermark type, quality, and codec. Use `-output` to choose a destination:

```bash
go run ./cmd/tiktok_crawler -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Download an exact video height:

```bash
go run ./cmd/tiktok_crawler \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Print all normalized metadata and media profiles, including signed URLs, as JSON without downloading:

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok's web player normally exposes playback profiles without a watermark. Some responses may also expose TikTok's official watermarked download address. Use `-watermark` to require that variant; the command returns an explicit error when TikTok does not provide it and never silently falls back to a no-watermark file.

```bash
go run ./cmd/tiktok_crawler -watermark 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Downloads are written to a temporary file and moved into place only after completion. Existing files are never overwritten; choose another `-output` path when necessary. Signed media URLs expire, so crawl the video again when an old URL stops working.

## Creator profiles

Print public user metadata and canonical links for the 10 most recent public videos exposed by TikTok's creator embed:

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@forever0404_'
```

The JSON result includes `user`, `listing`, `video_urls`, and detailed `videos`. TikTok's main `/api/post/item_list/` pagination is protected by browser-generated `X-Dynosaur` and `X-Gnarly` signatures and can return an interactive captcha even when correctly signed. The crawler therefore uses the public server-rendered `/embed/@username` creator card, which reliably exposes the latest 10 public videos but not the creator's complete archive.

Without `-json`, the crawler feeds each canonical URL back through the normal video resolver and downloads all returned videos. `-quality`, `-watermark`, and `-output` apply to the whole collection:

```bash
go run ./cmd/tiktok_crawler \
  -quality 1080p \
  -output "$HOME/Downloads/forever0404-videos" \
  'https://www.tiktok.com/@forever0404_'
```

When `-output` is omitted, files are written to a new `<username>_videos/` directory. Each filename retains its TikTok video ID.

## Photo Posts

Download every image in a Photo Post in its original post order:

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/photo/1234567890123456789'
```

TikTok also uses `/photo/<id>` URLs for photo Stories. When the player API omits a Story, the resolver automatically reads TikTok's embed metadata instead; JSON output includes `"is_story": true`. No browser cookie is required for a public Story that is still available.

Replace `example` and `1234567890123456789` with the username and Photo Post ID from the URL you want to crawl. By default, the images are stored in a new `<username>_<photo-id>_images/` directory in the current working directory. Use `-output` to choose the output directory; it is created when necessary:

```bash
go run ./cmd/tiktok_crawler \
  -output '~/Downloads/tiktok-photo-post' \
  'https://www.tiktok.com/@example/photo/1234567890123456789'
```

Use `-json` to print the post metadata, signed image URLs, and signed audio playback URLs when TikTok exposes them, without downloading. `-quality` is ignored for Photo Posts because TikTok exposes one display image per post image. Some Photo Posts may expose official watermark image sources; `-watermark` requires those sources and returns an error instead of silently downloading the no-watermark image when they are absent.

## Short Drama episodes

Short Drama episode URLs use the same video download options and download by default:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/shortdrama/episode/7665073849083368469/1'
```

Or load the cookies from a `.txt` file:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-file ./cookies.txt \
  'https://www.tiktok.com/shortdrama/episode/7665073849083368469/1'
```

TikTok exposes episode metadata publicly but currently requires a valid browser `msToken` for the signed playback-metadata request. Provide your own TikTok cookie with `-cookies-from-browser` or `-cookies-file`; the tool generates `X-Bogus` locally and never writes the cookie to disk. `-json`, `-output`, and `-quality` work the same way as for regular videos.

## Livestreams

Resolve a LIVE room and print every available stream as a table:

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/live'
```

Replace `example` with the TikTok username of the LIVE channel you want to resolve.

Sample output for an active LIVE room:

```text
CODEC  QUALITY  LINE  FORMAT  RESOLUTION  BITRATE  EXPIRES               URL
h265   origin   main  hls     1920x1080   2224000  2026-08-31T11:36:38Z  https://pull-hls-f16-sg01.tiktokcdn.com/game/stream-1561072868968628308/index.m3u8?expire=1788176198&sign=ab662553d3d43198f00eee6ff79263c0
h265   origin   main  flv     1920x1080   2224000  2026-08-31T11:36:38Z  https://pull-f5-sg01.tiktokcdn.com/game/stream-1561072868968628308.flv?expire=1788176198&sign=ce30a95c277094da2bb81d4d0e7662e0
h265   origin   main  cmaf    1920x1080   2224000  2026-08-31T11:36:38Z  https://pull-f5-sg01.tiktokcdn.com/game/stream-1561072868968628308/index.mpd?expire=1788176198&sign=f1e862ed8613e60ba0ad665fd0ab2cb4
h265   origin   main  lls     1920x1080   2224000  2026-08-31T11:36:38Z  https://pull-f5-sg01.tiktokcdn.com/game/stream-1561072868968628308.sdp?expire=1788176198&sign=4efcf378b009c553ec62a120b3b39e8c
h265   uhd_60   main  hls     1920x1080   4000000  2026-08-31T11:36:38Z  https://pull-hls-f16-sg01.tiktokcdn.com/game/stream-1561072868968628308_uhd560/index.m3u8?expire=1788176198&sign=0c33a22df71b89308ad48b8142e1606e
h265   uhd_60   main  flv     1920x1080   4000000  2026-08-31T11:36:38Z  https://pull-f5-sg01.tiktokcdn.com/game/stream-1561072868968628308_uhd560.flv?expire=1788176198&sign=f393522bdabef8d7ef0c25500b71ef2d
h265   uhd_60   main  cmaf    1920x1080   4000000  2026-08-31T11:36:38Z  https://pull-f5-sg01.tiktokcdn.com/game/stream-1561072868968628308_uhd560/index.mpd?expire=1788176198&sign=971ff4d9cfb652918867b2ae6443c923
h265   uhd_60   main  lls     1920x1080   4000000  2026-08-31T11:36:38Z  https://pull-f5-sg01.tiktokcdn.com/game/stream-1561072868968628308_uhd560.sdp?expire=1788176198&sign=6bd520bc35d399aa15e9cc7743f345b2
```

Print complete room metadata as JSON:

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@example/live'
```

## Authentication and regional restrictions

Public videos and live rooms usually work without authentication. If TikTok requires login, age verification, or a specific region, provide your own cookies from a logged-in browser or a cookie file:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Do not commit cookies to source control. Each client keeps cookies received from TikTok pages and reuses them for related requests.

### Cookies from a browser

Instead of pasting a cookie manually, read it from a browser that is already logged in to TikTok:

Instead of pasting a cookie manually, read it from a browser that is already logged in to TikTok:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Supported browser names follow the `browsercookie` library: brave, chrome, chromium, vivaldi, edge, edge-dev, arc, opera, opera-gx, firefox, librewolf, zen, safari. Only cookies belonging to TikTok hosts are read, and they are sent on every request. On macOS, the terminal may need Full Disk Access to read a browser's cookie store; Safari and some Chromium browsers also require the browser to be integrated with the macOS keychain.

### Cookies from a file

Pass a path to a cookies `.txt` file with `-cookies-file`. The file may be either a raw Cookie header value (`ttwid=...; sessionid=...`) or a Netscape cookie-jar export:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-file ./cookies.txt \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Cookies loaded from the file are sent on every request. Explicit `-cookies-file` takes precedence over `-cookies-from-browser`. Do not commit cookie files to source control.

### Custom headers

Add extra HTTP headers to every request with `-headers`. Repeat the flag, one `Key: Value` pair per occurrence:

```bash
go run ./cmd/tiktok_crawler \
  -headers 'User-Agent: Mozilla/5.0 ...' \
  -headers 'X-Forwarded-For: 1.2.3.4' \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Custom headers override the tool's defaults (`User-Agent`, `Accept-Language`, `Cache-Control`, `Accept`). To send a specific browser signature, set `User-Agent` through `-headers` (the tool's default is otherwise used). A `Referer` or `Cookie` key is superseded by the request-specific values the crawler computes.

## Build

```bash
go build -o tiktok_crawler ./cmd/tiktok_crawler
```

The generated root binaries are ignored by Git. You can also place builds under `bin/`.
