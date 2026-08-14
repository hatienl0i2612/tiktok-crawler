[English](README.md) | [Tiếng Việt](README.vi.md) | 简体中文

# TikTok 视频与直播命令行工具

本仓库包含两个相互独立的 Go 命令行工具：

- `tiktok` 抓取公开 TikTok 视频的元数据、列出可下载的媒体版本，并下载选定的 MP4 文件。
- `tiktok_livestream` 获取公开 TikTok LIVE 直播间的带签名播放地址。

两个工具均仅使用 Go 标准库。

## 下载发行版

预编译的可执行文件可从 [GitHub Releases 页面](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest)下载。请根据你的操作系统下载对应的两个文件：

| 操作系统 | 视频工具 | 直播工具 |
| --- | --- | --- |
| Linux (amd64) | [`tiktok-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-linux-amd64) | [`tiktok_livestream-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-darwin-arm64) | [`tiktok_livestream-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-darwin-arm64) |
| Windows (amd64) | [`tiktok-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-windows-amd64.exe) | [`tiktok_livestream-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-windows-amd64.exe) |

在 Linux 上，为两个文件添加执行权限，然后从下载目录运行：

```bash
chmod +x tiktok-linux-amd64 tiktok_livestream-linux-amd64
./tiktok-linux-amd64 -help
./tiktok-linux-amd64 -download 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-linux-amd64 'https://www.tiktok.com/@example/live'
```

在使用 Apple Silicon 的 macOS 上：

```bash
chmod +x tiktok-darwin-arm64 tiktok_livestream-darwin-arm64
./tiktok-darwin-arm64 -help
./tiktok-darwin-arm64 -download 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-darwin-arm64 'https://www.tiktok.com/@example/live'
```

在 Windows 上，在下载目录中打开 PowerShell：

```powershell
.\tiktok-windows-amd64.exe -help
.\tiktok-windows-amd64.exe -download "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_livestream-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

请将示例 URL 替换为你要处理的 TikTok 视频或 LIVE 直播间。本文后续示例使用 `go run` 从源代码运行；使用发行版时，请将 `go run ./cmd/tiktok` 或 `go run ./cmd/tiktok_livestream` 替换为上表中对应的可执行文件名。

## 项目结构

```text
.
├── cmd/tiktok/                  视频 CLI 入口
├── cmd/tiktok_livestream/       直播 CLI 入口
├── internal/video/              视频协议、模型、选择与下载逻辑
├── internal/livestream/         直播协议、模型与流解析逻辑
├── go.mod
├── README.md                    英文文档
├── README.vi.md                 越南语文档
└── README.zh-CN.md              简体中文文档
```

每个工具都在各自的 `main.go` 中保留入口、CLI 参数解析和输出格式化逻辑。`internal` 包仅包含 TikTok 协议请求、响应模型、媒体选择、数据解析和文件下载逻辑。

## 视频 CLI

以 JSON 格式输出标准化元数据和所有可用媒体版本：

```bash
go run ./cmd/tiktok 'https://www.tiktok.com/@example/video/1234567890123456789'
```

请将 `example` 和 `1234567890123456789` 替换为目标 TikTok URL 中的用户名和视频 ID。

下载默认的无水印 H.264 版本：

```bash
go run ./cmd/tiktok -download 'https://www.tiktok.com/@example/video/1234567890123456789'
```

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

默认文件名包含用户名、视频 ID、水印类型、画质和编解码器。使用 `-output` 指定保存路径；该参数也会自动启用下载模式：

```bash
go run ./cmd/tiktok  -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

选择最佳 H.265 版本或指定视频高度：

```bash
go run ./cmd/tiktok \
  -download \
  -codec h265 \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

仅输出选中的带签名 URL：

```bash
go run ./cmd/tiktok -url-only 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok 网页播放器通常提供无水印播放版本。部分响应也可能包含 TikTok 官方的有水印下载地址。使用 `-watermark` 可要求该版本；如果 TikTok 未提供，工具会返回明确错误，并且不会静默回退到无水印文件。

下载内容会先写入临时文件，仅在完成后移动到目标路径。除非指定 `-force`，否则已有文件不会被覆盖。带签名媒体 URL 会过期，因此旧 URL 失效后请重新抓取视频。

## 直播 CLI

输出当前最佳播放流 URL；如果 TikTok 提供 H.264 与 HLS 的组合，将优先选择该组合：

```bash
go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live'
```

请将 `example` 替换为要解析的 LIVE 频道 TikTok 用户名。

使用 `-all` 以表格形式输出所有可用播放流：

```bash
go run ./cmd/tiktok_livestream -all 'https://www.tiktok.com/@example/live'
```

正在直播的 LIVE 房间输出示例：

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

使用 [mpv](https://mpv.io/installation/) 播放选中的流；对于这些直播 URL，推荐使用该媒体播放器：

```bash
mpv "$(go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live')"
```

以 JSON 格式输出完整的直播间元数据：

```bash
go run ./cmd/tiktok_livestream -json 'https://www.tiktok.com/@example/live'
```

选择原始画质的 H.265 HLS 播放流：

```bash
go run ./cmd/tiktok_livestream \
  -codec h265 \
  -quality origin \
  -format hls \
  'https://www.tiktok.com/@example/live'
```

对任一工具使用 `-help` 可查看所有选项。

## 身份验证与地区限制

公开视频和 LIVE 直播间通常无需身份验证即可使用。如果 TikTok 要求登录、年龄验证或特定地区，请通过环境变量提供你自己的 cookie：

```bash
TIKTOK_COOKIE='ttwid=...; sessionid=...' \
  go run ./cmd/tiktok \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

请勿将 cookie 提交到版本控制中。每个客户端都会保存从 TikTok 页面收到的 cookie，并在相关请求中重复使用。

## 从源代码构建

```bash
go build -o tiktok ./cmd/tiktok
go build -o tiktok_livestream ./cmd/tiktok_livestream
```

在项目根目录生成的可执行文件已被 Git 忽略。你也可以将构建结果放到 `bin/` 目录中。
