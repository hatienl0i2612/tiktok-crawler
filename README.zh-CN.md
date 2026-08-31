[English](README.md) | [Tiếng Việt](README.vi.md) | 简体中文

# TikTok Crawler

[![Go Reference](https://pkg.go.dev/badge/github.com/hatienl0i2612/tiktok-crawler.svg)](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)

本仓库包含一个 Go 库和一个命令行工具：

- **库**：可通过 `github.com/hatienl0i2612/tiktok-crawler` 导入的包，用于解析视频、创作者主页、图文帖、短剧剧集和 LIVE 直播间，并提供 cookie、media 和下载辅助函数。
- **CLI**：`tiktok_crawler` 自动识别 URL 类型，抓取公开创作者主页，下载公开 TikTok 视频、图文帖和短剧剧集，并使用 mpv 打开最佳公开 TikTok LIVE 播放流。

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

也可直接使用子包：`video`、`profile`、`photo`、`livestream`、`mpv`、`shortdrama`、`cookies`、`downloader`、`media`、`tiktok`。完整 API 见 [pkg.go.dev](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)。

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

TikTok 也会将 `/video/<id>` URL 用于视频 Story。当 player API 不返回 Story 时，解析器会读取 TikTok 的 embed 元数据以及网页播放器使用的 `playAddr`；JSON 输出会包含 `"is_story": true`。默认选择无 TikTok 水印的 `playAddr`，`-watermark` 则选择 `downloadAddr`。仍可公开访问的 Story 无需浏览器 Cookie。

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

## 创作者主页

输出公开用户元数据，以及 TikTok 创作者嵌入页公开的最近 10 个公开视频的规范链接：

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@forever0404_'
```

JSON 包含 `user`、`listing`、`video_urls` 和 `videos`。主要分页接口 `/api/post/item_list/` 受浏览器生成的 `X-Dynosaur`/`X-Gnarly` 签名保护，即使签名正确也可能返回交互式验证码。因此，抓取器使用公开的服务端渲染 `/embed/@username` 创作者卡片；它能稳定返回最近 10 个公开视频，但不提供用户的完整历史归档。

不使用 `-json` 时，抓取器会把每个规范 URL 交给现有视频解析器并下载所有返回的视频。`-quality`、`-watermark` 和 `-output` 会应用于整个集合：

```bash
go run ./cmd/tiktok_crawler \
  -quality 1080p \
  -output "$HOME/Downloads/forever0404-videos" \
  'https://www.tiktok.com/@forever0404_'
```

未指定 `-output` 时，文件保存在新的 `<username>_videos/` 目录中，每个文件名都会保留对应的 TikTok 视频 ID。

## 图文帖

按帖子中的原始顺序下载图文帖内的全部图片：

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/photo/1234567890123456789'
```

TikTok 也会将 `/photo/<id>` URL 用于图片 Story。当 player API 不返回 Story 时，解析器会自动改用 TikTok 的 embed 元数据；JSON 输出会包含 `"is_story": true`。仍可公开访问的 Story 无需浏览器 Cookie。

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

解析 LIVE 直播间、选择最佳播放流，并使用 mpv 打开：

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/live'
```

请将 `example` 替换为要观看的 LIVE 频道 TikTok 用户名。普通 LIVE 命令不再打印签名 URL 列表。工具优先选择 `origin`，然后选择其余最高画质；画质相同时依次优先 H.264、`main` CDN 线路和 HLS。

首次运行时，工具会先在 `PATH` 中查找 `mpv`。若未安装，CLI 会将便携版本下载并缓存到 `os.TempDir()/tiktok-crawler/mpv/<os>-<arch>`：

| 平台 | 便携版本来源 |
| --- | --- |
| Windows `amd64`、`arm64`、`386` | `mpv-player/mpv` 官方稳定版 ZIP |
| macOS `arm64`、`amd64` | `mpv-player/mpv` 官方稳定版 ZIP |
| Linux `amd64`、`arm64` | `pkgforge-dev/mpv-AppImage` 的 AppImage |

每个下载文件都有大小限制，并会在解压或执行前根据 GitHub Release 元数据验证 SHA-256。`os.TempDir` 会自动选择当前操作系统的临时目录；后续运行会复用系统 mpv 或缓存的便携版本。

正在直播的 LIVE 房间状态示例：

```text
Opening TikTok LIVE @example in mpv (origin, h264, hls)
```

若要在不启动 mpv 的情况下查看完整房间元数据和播放流 URL，请使用 JSON 模式：

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

## 贡献

欢迎任何形式的贡献。如果你希望改进抓取器、支持更多 TikTok 格式或完善文档，请提交 Pull Request。

如果遇到错误或异常的 TikTok 响应，请[创建 Issue](https://github.com/hatienl0i2612/tiktok-crawler/issues)，并尽可能提供相关 URL、执行命令、错误输出、操作系统和工具版本。请勿提交私人 Cookie 或身份验证数据。
