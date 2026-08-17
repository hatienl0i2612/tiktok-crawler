English | [Tiếng Việt](README.vi.md) | [简体中文](README.zh-CN.md)

# TikTok Video and Livestream CLIs

This repository contains two independent Go command-line tools:

- `tiktok` crawls public TikTok video metadata, lists downloadable media profiles, and downloads a selected MP4 file.
- `tiktok_livestream` resolves signed playback URLs for a public TikTok LIVE room.

Both commands use only the Go standard library.

## Download a release

Prebuilt binaries are available on the [GitHub Releases page](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest). Download the two files that match your operating system:

| Operating system | Video CLI | Livestream CLI |
| --- | --- | --- |
| Linux (amd64) | [`tiktok-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-linux-amd64) | [`tiktok_livestream-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-darwin-arm64) | [`tiktok_livestream-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-darwin-arm64) |
| Windows (amd64) | [`tiktok-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-windows-amd64.exe) | [`tiktok_livestream-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-windows-amd64.exe) |

On Linux, make both files executable and run them from the download directory:

```bash
chmod +x tiktok-linux-amd64 tiktok_livestream-linux-amd64
./tiktok-linux-amd64 -help
./tiktok-linux-amd64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-linux-amd64 'https://www.tiktok.com/@example/live'
```

On macOS with Apple Silicon:

```bash
chmod +x tiktok-darwin-arm64 tiktok_livestream-darwin-arm64
./tiktok-darwin-arm64 -help
./tiktok-darwin-arm64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-darwin-arm64 'https://www.tiktok.com/@example/live'
```

On Windows, open PowerShell in the download directory:

```powershell
.\tiktok-windows-amd64.exe -help
.\tiktok-windows-amd64.exe "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_livestream-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

Replace the example URLs with the TikTok video or LIVE room you want to process. The remaining examples use `go run` for source builds; when using a release, replace `go run ./cmd/tiktok` or `go run ./cmd/tiktok_livestream` with the downloaded executable name shown above.

## Video CLI

Download the best available H.264 video without a watermark (the default behavior):

```bash
go run ./cmd/tiktok 'https://www.tiktok.com/@example/video/1234567890123456789'
```

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
go run ./cmd/tiktok -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Download an exact video height:

```bash
go run ./cmd/tiktok \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Print all normalized metadata and media profiles, including signed URLs, as JSON without downloading:

```bash
go run ./cmd/tiktok -json 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok's web player normally exposes playback profiles without a watermark. Some responses may also expose TikTok's official watermarked download address. Use `-watermark` to require that variant; the command returns an explicit error when TikTok does not provide it and never silently falls back to a no-watermark file.

```bash
go run ./cmd/tiktok -watermark 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Downloads are written to a temporary file and moved into place only after completion. Existing files are never overwritten; choose another `-output` path when necessary. Signed media URLs expire, so crawl the video again when an old URL stops working.

## Livestream CLI

Resolve a LIVE room and print every available stream as a table:

```bash
go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live'
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
go run ./cmd/tiktok_livestream -json 'https://www.tiktok.com/@example/live'
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
