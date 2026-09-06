[English](README.md) | Tiếng Việt | [简体中文](README.zh-CN.md)

# TikTok Crawler

[![Go Reference](https://pkg.go.dev/badge/github.com/hatienl0i2612/tiktok-crawler.svg)](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler)

Repository này cung cấp một thư viện Go cùng công cụ dòng lệnh:

- **Thư viện**: các package có thể import được từ `github.com/hatienl0i2612/tiktok-crawler` để resolve video, profile creator, Photo Post, tập Short Drama và phòng LIVE, kèm helper cho cookie, media và download.
- **CLI**: `tiktok_crawler` tự nhận diện loại URL, crawl profile creator, tải video TikTok, Photo Post và tập Short Drama công khai, đồng thời mở stream TikTok LIVE tốt nhất bằng mpv.

Công cụ sử dụng thư viện chuẩn của Go, cùng package `browsercookie` để tùy chọn nhập cookie từ trình duyệt đã cài đặt.

## Sử dụng như thư viện Go

```bash
go get github.com/hatienl0i2612/tiktok-crawler@latest
```

```go
import "github.com/hatienl0i2612/tiktok-crawler"

config := tiktokcrawler.ClientOptions{Cookie: "ttwid=...; sessionid=..."}
client, _ := tiktokcrawler.NewClient(config)
result, err := client.Resolve(ctx, "https://www.tiktok.com/@example/video/1234567890123456789")
```

Hoặc dùng trực tiếp các subpackage: `video`, `profile`, `photo`, `livestream`, `mpv`, `shortdrama`, `cookies`, `downloader`, `media`, `tiktok`. API đầy đủ trên [pkg.go.dev](https://pkg.go.dev/github.com/hatienl0i2612/tiktok-crawler).

## Tải bản phát hành

Các binary được build sẵn có trên [trang GitHub Releases](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest). Hãy tải file phù hợp với hệ điều hành của bạn:

| Hệ điều hành | Binary |
| --- | --- |
| Linux (amd64) | [`tiktok_crawler-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok_crawler-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-darwin-arm64) |
| Windows (amd64) | [`tiktok_crawler-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_crawler-windows-amd64.exe) |

Trên Linux, cấp quyền thực thi rồi chạy file từ thư mục tải xuống:

```bash
chmod +x tiktok_crawler-linux-amd64
./tiktok_crawler-linux-amd64 -help
./tiktok_crawler-linux-amd64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_crawler-linux-amd64 'https://www.tiktok.com/@example/live'
```

Trên macOS sử dụng Apple Silicon:

```bash
chmod +x tiktok_crawler-darwin-arm64
./tiktok_crawler-darwin-arm64 -help
./tiktok_crawler-darwin-arm64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_crawler-darwin-arm64 'https://www.tiktok.com/@example/live'
```

Trên Windows, mở PowerShell tại thư mục tải xuống:

```powershell
.\tiktok_crawler-windows-amd64.exe -help
.\tiktok_crawler-windows-amd64.exe "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_crawler-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

Thay các URL ví dụ bằng video TikTok hoặc phòng LIVE mà bạn muốn xử lý. Các ví dụ còn lại sử dụng `go run` để chạy từ source; khi sử dụng bản phát hành, hãy thay `go run ./cmd/tiktok_crawler` bằng executable ở trên.

## Các loại URL được hỗ trợ

| Loại | Định dạng URL | Hành động mặc định |
| --- | --- | --- |
| Video | `https://www.tiktok.com/@username/video/<id>` | Tải video |
| Photo Post / Story ảnh | `https://www.tiktok.com/@username/photo/<id>` | Tải toàn bộ ảnh và nhạc của bài đăng nếu có |
| Profile creator | `https://www.tiktok.com/@username` | Tải các video public gần đây do TikTok cung cấp |
| Tập Short Drama | `https://www.tiktok.com/shortdrama/episode/<id>/<episode>` | Tải tập phim |
| LIVE | `https://www.tiktok.com/@username/live` | Mở stream tốt nhất bằng mpv |

## Tùy chọn CLI

Cách dùng:

```bash
tiktok_crawler [options] <TikTok URL>
```

Các option có thể đặt trước hoặc sau URL.

Chạy executable không có argument—hoặc double-click trên Windows—để vào interactive mode. Dán TikTok URL khi được hỏi; nếu có lỗi, chương trình sẽ quay lại màn hình nhập URL, còn sau khi xử lý thành công chương trình sẽ chờ bạn nhấn Enter rồi mới đóng.

| Option | Mô tả |
| --- | --- |
| `-json` | In metadata đã resolve dưới dạng JSON thay vì tải hoặc phát |
| `-output <path>` | Chọn file hoặc thư mục đầu ra; không áp dụng cho URL LIVE |
| `-quality <height>` | Chọn chất lượng video, ví dụ `720p` hoặc `1080p`; mặc định là `best` |
| `-watermark` | Tải phiên bản video hoặc ảnh có watermark chính thức của TikTok |
| `-verbose` | In thông tin tóm tắt của phòng LIVE |
| `-timeout <duration>` | Đặt timeout cho request LIVE; mặc định là `20s` |
| `-cookies-file <path>` | Đọc TikTok cookie từ file Netscape cookie hoặc raw Cookie header |
| `-cookies-from-browser <browser>` | Đọc TikTok cookie từ browser đã cài đặt |
| `-headers 'Key: Value'` | Thêm HTTP header; lặp lại option để thêm nhiều header |
| `-version` | In phiên bản CLI |
| `-help` | In hướng dẫn sử dụng |

## Build từ source

```bash
go build -o tiktok_crawler ./cmd/tiktok_crawler
```

Các binary được tạo tại thư mục gốc đã được Git bỏ qua. Bạn cũng có thể đặt chúng trong thư mục `bin/`.

## Đóng góp

Mọi đóng góp đều được chào đón. Nếu bạn muốn cải thiện crawler, hỗ trợ thêm định dạng TikTok hoặc hoàn thiện tài liệu, hãy gửi pull request.

Nếu gặp lỗi hoặc response TikTok bất thường, hãy [mở issue](https://github.com/hatienl0i2612/tiktok-crawler/issues) và cung cấp URL bị ảnh hưởng, command đã chạy, error output, hệ điều hành và version của tool nếu có thể. Không đăng cookie riêng tư hoặc dữ liệu xác thực.
