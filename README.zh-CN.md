[English](README.md) | [Tiếng Việt](README.vi.md) | 简体中文

# TikTok Crawler

[![Go Reference](https://pkg.go.dev/badge/github.com/hatienl0i2612/tiktok-crawler.svg)](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)

本仓库包含一个 Go 库和一个命令行工具：

- **库**：可通过 `github.com/hatienl0i2612/tiktok-crawler` 导入的包，用于解析视频、创作者主页、话题标签页、图文帖、短剧剧集和 LIVE 直播间，并提供 cookie、media 和下载辅助函数。
- **CLI**：`tiktok_crawler` 自动识别 URL 类型，抓取公开创作者主页和话题标签，下载对应的公开视频、图文帖和短剧剧集，并使用 mpv 打开最佳公开 TikTok LIVE 播放流。

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

也可直接使用子包：`video`、`profile`、`hashtag`、`photo`、`livestream`、`mpv`、`shortdrama`、`cookies`、`downloader`、`media`、`tiktok`。完整 API 见 [pkg.go.dev](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)。

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

## 支持的 URL 类型

| 类型 | URL 格式 | 默认操作 |
| --- | --- | --- |
| 视频 | `https://www.tiktok.com/@username/video/<id>` | 下载视频 |
| 图文帖 / 图片 Story | `https://www.tiktok.com/@username/photo/<id>` | 下载全部图片，并在可用时下载帖子音乐 |
| 创作者主页 | `https://www.tiktok.com/@username` | 下载 TikTok 提供的近期公开视频 |
| 话题标签 | `https://www.tiktok.com/tag/<name>` | 下载 TikTok 标签嵌入页提供的公开视频 |
| 短剧剧集 | `https://www.tiktok.com/shortdrama/episode/<id>/<episode>` | 下载该剧集 |
| LIVE 直播 | `https://www.tiktok.com/@username/live` | 使用 mpv 打开最佳可用直播流 |

## CLI 选项

用法：

```bash
tiktok_crawler [options] <TikTok URL>
```

选项可以放在 URL 之前或之后。

不带参数运行可执行文件，或在 Windows 上双击它，即可进入交互模式。根据提示粘贴 TikTok URL；发生错误时程序会返回 URL 输入界面，处理成功后则会等待你按 Enter 键再关闭。

工具启动时会检查 GitHub 上是否有更新版本，如有则向 stderr 输出更新提示。检查超时时间为 5 秒；网络错误不会影响正常使用。

| 选项 | 说明 |
| --- | --- |
| `-json` | 以 JSON 格式输出解析后的元数据，而不下载或播放 |
| `-output <path>` | 设置输出文件或目录；对 LIVE URL 无效 |
| `-quality <height>` | 选择视频画质，例如 `720p` 或 `1080p`；默认为 `best` |
| `-watermark` | 下载 TikTok 官方带水印的视频或图片版本 |
| `-verbose` | 输出 LIVE 直播间摘要 |
| `-timeout <duration>` | 设置 LIVE 请求超时时间；默认为 `20s` |
| `-cookies-file <path>` | 从 Netscape cookie 文件或原始 Cookie header 中读取 TikTok cookie |
| `-cookies-from-browser <browser>` | 从已安装的浏览器中读取 TikTok cookie |
| `-headers 'Key: Value'` | 添加 HTTP header；可重复使用此选项添加多个 header |
| `-version` | 输出 CLI 版本 |
| `-help` | 输出命令帮助 |

## 从源代码构建

```bash
go build -o tiktok_crawler ./cmd/tiktok_crawler
```

在项目根目录生成的可执行文件已被 Git 忽略。你也可以将构建结果放到 `bin/` 目录中。

## 贡献

欢迎任何形式的贡献。如果你希望改进抓取器、支持更多 TikTok 格式或完善文档，请提交 Pull Request。

如果遇到错误或异常的 TikTok 响应，请[创建 Issue](https://github.com/hatienl0i2612/tiktok-crawler/issues)，并尽可能提供相关 URL、执行命令、错误输出、操作系统和工具版本。请勿提交私人 Cookie 或身份验证数据。
