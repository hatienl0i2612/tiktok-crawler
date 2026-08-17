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
./tiktok-linux-amd64 -download 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-linux-amd64 'https://www.tiktok.com/@example/live'
```

On macOS with Apple Silicon:

```bash
chmod +x tiktok-darwin-arm64 tiktok_livestream-darwin-arm64
./tiktok-darwin-arm64 -help
./tiktok-darwin-arm64 -download 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-darwin-arm64 'https://www.tiktok.com/@example/live'
```

On Windows, open PowerShell in the download directory:

```powershell
.\tiktok-windows-amd64.exe -help
.\tiktok-windows-amd64.exe -download "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_livestream-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

Replace the example URLs with the TikTok video or LIVE room you want to process. The remaining examples use `go run` for source builds; when using a release, replace `go run ./cmd/tiktok` or `go run ./cmd/tiktok_livestream` with the downloaded executable name shown above.

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

Print the best available stream URL, preferring H.264 and HLS when that combination is available:

```bash
go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live'
```

Replace `example` with the TikTok username of the LIVE channel you want to resolve.

Use `-all` to print every available stream as a table:

```bash
go run ./cmd/tiktok_livestream -all 'https://www.tiktok.com/@example/live'
```

Sample output for an active LIVE room:

```text
CODEC  QUALITY  LINE  FORMAT  RESOLUTION  BITRATE  EXPIRES               URL
h264   origin   main  hls     1920x1080   9200000  2026-08-27T17:03:10Z  https://pull-hls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608.m3u8?expire=1787850190&sign=20758f803a9842b0ae05952d6c0777ed
h264   origin   main  flv     1920x1080   9200000  2026-08-27T17:03:10Z  https://pull-flv-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608.flv?expire=1787850190&sign=5c67c75a32a647024ec7653bb8499a85
h264   origin   main  lls     1920x1080   9200000  2026-08-27T17:03:10Z  https://pull-lls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608.sdp?expire=1787850190&sign=c761bdc232b17ee63db64e348b5a4695
h264   hd       main  hls     1280x720    1800000  2026-08-27T17:03:10Z  https://pull-hls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_hd.m3u8?expire=1787850190&sign=07f4c9daacbc59e003f43b9db09d9592
h264   hd       main  flv     1280x720    1800000  2026-08-27T17:03:10Z  https://pull-flv-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_hd.flv?expire=1787850190&sign=3526c3298b218eefa80fd106cfb56616
h264   hd       main  lls     1280x720    1800000  2026-08-27T17:03:10Z  https://pull-lls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_hd.sdp?expire=1787850190&sign=3d744a41b11ef635a08fcca5bd42a725
h265   hd       main  hls     1280x720    1350000  2026-08-27T17:03:10Z  https://pull-hls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_hd5.m3u8?expire=1787850190&sign=5ada1b09e0a210426677f35ffc457dbc
h265   hd       main  flv     1280x720    1350000  2026-08-27T17:03:10Z  https://pull-flv-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_hd5.flv?expire=1787850190&sign=606a4658c7049da9503b194506518766
h265   hd       main  lls     1280x720    1350000  2026-08-27T17:03:10Z  https://pull-lls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_hd5.sdp?expire=1787850190&sign=2fa5a816794b57abfebbfde08b7b5cfa
h265   sd       main  hls     960x540     1000000  2026-08-27T17:03:10Z  https://pull-hls-l11-sg01.tiktokcdn.com/game/stream-2137512010433429608_sd5.m3u8?expire=1787850190&sign=ff0572d0ead13dcbe7c64546f8eff072
```

Play the selected stream with [mpv](https://mpv.io/installation/), the recommended media player for these livestream URLs:

```bash
mpv "$(go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live')"
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
