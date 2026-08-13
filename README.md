# TikTok Video and Livestream CLIs

This repository contains two independent Go command-line tools:

- `tiktok` crawls public TikTok video metadata, lists downloadable media profiles, and downloads a selected MP4 file.
- `tiktok_livestream` resolves signed playback URLs for a public TikTok LIVE room.

Both commands use only the Go standard library.

## Project layout

```text
.
├── cmd/tiktok/                  Video CLI entry point
├── cmd/tiktok_livestream/       Livestream CLI entry point
├── internal/video/              Video protocol, models, selection, and downloader
├── internal/livestream/         Livestream protocol, models, and stream parsing
├── go.mod
└── README.md
```

Each command keeps its entry point, CLI parsing, and output formatting in its own `main.go`. The `internal` packages contain only TikTok protocol requests, response models, media selection, parsing, and file downloads.

## Video CLI

Print normalized metadata and all available media profiles as JSON:

```bash
go run ./cmd/tiktok 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Replace `example` and `1234567890123456789` with the username and video ID from the TikTok URL you want to crawl.

Download the default H.264 profile without a watermark:

```bash
go run ./cmd/tiktok -download 'https://www.tiktok.com/@example/video/1234567890123456789'
```

The default filename includes the username, video ID, watermark type, quality, and codec. Use `-output` to choose a destination; it also enables download mode:

```bash
go run ./cmd/tiktok  -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Select the best H.265 profile or an exact height:

```bash
go run ./cmd/tiktok \
  -download \
  -codec h265 \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Print only the selected signed URL:

```bash
go run ./cmd/tiktok -url-only 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok's web player normally exposes playback profiles without a watermark. Some responses may also expose TikTok's official watermarked download address. Use `-watermark` to require that variant; the command returns an explicit error when TikTok does not provide it and never silently falls back to a no-watermark file.

Downloads are written to a temporary file and moved into place only after completion. Existing files are preserved unless `-force` is supplied. Signed media URLs expire, so crawl the video again when an old URL stops working.

## Livestream CLI

Print one best-quality HLS/H.264 URL:

```bash
go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live'
```

Replace `example` with the TikTok username of the LIVE channel you want to resolve.

Pass the selected stream directly to a player:

```bash
ffplay "$(go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live')"
```

List every stream:

```bash
go run ./cmd/tiktok_livestream -all 'https://www.tiktok.com/@example/live'
```

Print complete room metadata as JSON:

```bash
go run ./cmd/tiktok_livestream -json 'https://www.tiktok.com/@example/live'
```

Select an original H.265 HLS stream:

```bash
go run ./cmd/tiktok_livestream \
  -codec h265 \
  -quality origin \
  -format hls \
  'https://www.tiktok.com/@example/live'
```

Use `-help` on either command to view all options.

## Authentication and regional restrictions

Public videos and live rooms usually work without authentication. If TikTok requires login, age verification, or a specific region, provide your own cookie through the environment:

```bash
TIKTOK_COOKIE='ttwid=...; sessionid=...' \
  go run ./cmd/tiktok \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Do not commit cookies to source control. Each client keeps cookies received from TikTok pages and reuses them for related requests.

## Build

```bash
go build -o tiktok ./cmd/tiktok
go build -o tiktok_livestream ./cmd/tiktok_livestream
```

The generated root binaries are ignored by Git. You can also place builds under `bin/`.
