[English](README.md) | [Tiếng Việt](README.vi.md) | 简体中文

# TikTok Crawler

[![Go Reference](https://pkg.go.dev/badge/github.com/hatienl0i2612/tiktok-crawler.svg)](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)

本仓库包含一个 Go 库和一个命令行工具：

- **库**：可通过 `github.com/hatienl0i2612/tiktok-crawler` 导入的包，用于解析视频、图文帖、短剧剧集和 LIVE 直播间，并提供 cookie、media 和下载辅助函数。
- **CLI**：`tiktok_crawler` 自动识别 URL 类型，下载公开 TikTok 视频、图文帖和短剧剧集，或获取公开 TikTok LIVE 直播间的带签名播放地址。

该工具使用 Go 标准库，并通过 `browsercookie` 包可选地从已安装的浏览器中导入 cookie。

## 作为 Go 库使用

```bash
go get github.com/hatienl0i2612/tiktok-crawler@latest
```

```go
import "github.com/hatienl0i2612/tiktok-crawler"

config := tiktokcrawler.ClientOptions{Cookie: "ttwid=...; sessionid=..."}
client, err := tiktokcrawler.NewClient(config)
result, err := client.Resolve(ctx, "https://www.tiktok.com/@example/video/1234567890123456789")
```

也可直接使用子包：`video`、`photo`、`livestream`、`shortdrama`、`cookies`、`downloader`、`media`、`tiktok`。完整 API 见 [pkg.go.dev](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)。

## 下载发行版

预编译的可执行文件可从 [GitHub Releases 页面](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest)下载。请根据你的操作系统下载对应文件：

| 操作系统 | 可执行文件 |
| --- | --- |
| Linux (amd64) | [`tiktok_crawler-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok_crawler-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-darwin-arm64) |
| Windows (amd64) | [`tiktok_crawler-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-windows-amd64.exe) |

在 Linux 上，为文件添加执行权限，然后从下载目录运行：

```bash
chmod +x tiktok_crawler-linux-amd64
./tiktok_crawler-linux-amd64 -help
./tiktok_crawler-linux-amd64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_crawler-linux-amd64 'https://www.tiktok.com/@example/live'
```

在使用 Apple Silicon 的 macOS 上：

```bash
chmod +x tiktok_crawler-darwin-arm64
./tiktok_crawler-darwin-arm64 -help
./tiktok_crawler-darwin-arm64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_crawler-darwin-arm64 'https://www.tiktok.com/@example/live'
```

在 Windows 上，在下载目录中打开 PowerShell：

```powershell
.\tiktok_crawler-windows-amd64.exe -help
.\tiktok_crawler-windows-amd64.exe "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_crawler-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

请将示例 URL 替换为你要处理的 TikTok 视频或 LIVE 直播间。本文后续示例使用 `go run` 从源代码运行；使用发行版时，请将 `go run ./cmd/tiktok_crawler` 替换为上表中对应的可执行文件名。

## 视频

下载当前质量最佳的无水印视频（默认行为）：

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/video/1234567890123456789'
```

请将 `example` 和 `1234567890123456789` 替换为目标 TikTok URL 中的用户名和视频 ID。

下载无水印视频时的输出示例：

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

默认文件名包含用户名、视频 ID、水印类型、画质和编解码器。使用 `-output` 指定保存路径：

```bash
go run ./cmd/tiktok_crawler -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

下载指定高度的视频：

```bash
go run ./cmd/tiktok_crawler \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

以 JSON 格式输出所有标准化元数据和媒体版本（包括带签名 URL），且不下载视频：

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok 网页播放器通常提供无水印播放版本。部分响应也可能包含 TikTok 官方的有水印下载地址。使用 `-watermark` 可要求该版本；如果 TikTok 未提供，工具会返回明确错误，并且不会静默回退到无水印文件。

```bash
go run ./cmd/tiktok_crawler -watermark 'https://www.tiktok.com/@example/video/1234567890123456789'
```

下载内容会先写入临时文件，仅在完成后移动到目标路径。已有文件永远不会被覆盖；必要时请选择其他 `-output` 路径。带签名媒体 URL 会过期，因此旧 URL 失效后请重新抓取视频。

## 图文帖

按帖子中的原始顺序下载图文帖内的全部图片：

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/photo/1234567890123456789'
```

请将 `example` 和 `1234567890123456789` 替换为目标 TikTok URL 中的用户名和图文帖 ID。默认情况下，图片会保存到当前工作目录中新建的 `<username>_<photo-id>_images/` 目录。使用 `-output` 指定输出目录；如有需要，工具会自动创建该目录：

```bash
go run ./cmd/tiktok_crawler \
  -output '~/Downloads/tiktok-photo-post' \
  'https://www.tiktok.com/@example/photo/1234567890123456789'
```

使用 `-json` 输出帖子元数据、带签名图片 URL，以及 TikTok 提供时的带签名音频播放 URL，而不下载文件。Photo Post 会为每张图片提供一个 display image，因此 `-quality` 会被忽略。部分图文帖可能提供官方水印图片源；`-watermark` 会要求该图片源，如不存在则返回错误，而不会静默下载无水印图片。

## 短剧剧集

短剧剧集 URL 使用与普通视频相同的选项，并默认开始下载：

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/shortdrama/episode/7665073849083368469/1'
```

或从 `.txt` 文件加载 cookie：

```bash
go run ./cmd/tiktok_crawler \
  -cookies-file ./cookies.txt \
  'https://www.tiktok.com/shortdrama/episode/7665073849083368469/1'
```

TikTok 允许公开获取剧集元数据，但目前要求使用浏览器会话中的有效 `msToken` 请求带签名的播放元数据。请通过 `-cookies-from-browser` 或 `-cookies-file` 提供你自己的 TikTok cookie；工具会在本地生成 `X-Bogus`，不会将 cookie 写入磁盘。`-json`、`-output` 和 `-quality` 与普通视频的行为相同。

## 直播

解析 LIVE 直播间，并以表格形式输出所有可用播放流：

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/live'
```

请将 `example` 替换为要解析的 LIVE 频道 TikTok 用户名。

正在直播的 LIVE 房间输出示例：

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

以 JSON 格式输出完整的直播间元数据：

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@example/live'
```

## 身份验证与地区限制

公开视频和 LIVE 直播间通常无需身份验证即可使用。如果 TikTok 要求登录、年龄验证或特定地区，请使用已登录的浏览器或 cookie 文件提供你自己的 cookie：

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

请勿将 cookie 提交到版本控制中。每个客户端都会保存从 TikTok 页面收到的 cookie，并在相关请求中重复使用。

### 从浏览器导入 cookie

无需手动复制 cookie，可直接从已登录 TikTok 的浏览器中读取：

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

支持的浏览器与 `browsercookie` 库一致：brave、chrome、chromium、vivaldi、edge、edge-dev、arc、opera、opera-gx、firefox、librewolf、zen、safari。该工具仅读取属于 TikTok 域的 cookie，并在后续每个请求中自动携带。在 macOS 上，终端可能需要“完全磁盘访问权限”才能读取浏览器的 cookie 存储；Safari 和部分 Chromium 浏览器还要求浏览器已接入 macOS 钥匙串。

### 从文件导入 cookie

使用 `-cookies-file` 传入 cookie `.txt` 文件的路径。文件内容可以是原始 Cookie header 值（`ttwid=...; sessionid=...`），也可以是 Netscape cookie-jar 导出格式：

```bash
go run ./cmd/tiktok_crawler \
  -cookies-file ./cookies.txt \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

从文件加载的 cookie 会在后续每个请求中自动携带。`-cookies-file` 优先于 `-cookies-from-browser`。请勿将 cookie 文件提交到版本控制。

### 自定义请求头

使用 `-headers` 为每个请求添加额外的 HTTP 头，可重复使用，每次一个 `Key: Value` 对：

```bash
go run ./cmd/tiktok_crawler \
  -headers 'User-Agent: Mozilla/5.0 ...' \
  -headers 'X-Forwarded-For: 1.2.3.4' \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

自定义请求头会覆盖工具的默认值（`User-Agent`、`Accept-Language`、`Cache-Control`、`Accept`）。如需发送不同的浏览器特征，请通过 `-headers` 设置 `User-Agent`（否则使用工具默认值）。`Referer` 或 `Cookie` 键会被爬虫按请求计算出的值所取代。

## 从源代码构建

```bash
go build -o tiktok_crawler ./cmd/tiktok_crawler
```

在项目根目录生成的可执行文件已被 Git 忽略。你也可以将构建结果放到 `bin/` 目录中。
