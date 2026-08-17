[English](README.md) | Tiếng Việt | [简体中文](README.zh-CN.md)

# Công cụ dòng lệnh TikTok Video, Livestream và Search

Repository này cung cấp ba công cụ dòng lệnh Go độc lập:

- `tiktok` thu thập metadata của video TikTok công khai, liệt kê các phiên bản media có thể tải xuống và tải file MP4 đã chọn.
- `tiktok_livestream` lấy các URL phát có chữ ký cho một phòng TikTok LIVE công khai.
- `tiktok_search` tìm kiếm video TikTok công khai và in các URL chuẩn dưới dạng JSON.

Cả ba công cụ chỉ sử dụng thư viện chuẩn của Go.

## Tải bản phát hành

Các binary được build sẵn có trên [trang GitHub Releases](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest). Hãy tải các file phù hợp với hệ điều hành của bạn:

| Hệ điều hành | Video CLI | Livestream CLI | Search CLI |
| --- | --- | --- | --- |
| Linux (amd64) | [`tiktok-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-linux-amd64) | [`tiktok_livestream-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-linux-amd64) | [`tiktok_search-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_search-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-darwin-arm64) | [`tiktok_livestream-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-darwin-arm64) | [`tiktok_search-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_search-darwin-arm64) |
| Windows (amd64) | [`tiktok-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-windows-amd64.exe) | [`tiktok_livestream-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-windows-amd64.exe) | [`tiktok_search-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_search-windows-amd64.exe) |

Trên Linux, cấp quyền thực thi cho các file rồi chạy chúng từ thư mục tải xuống:

```bash
chmod +x tiktok-linux-amd64 tiktok_livestream-linux-amd64 tiktok_search-linux-amd64
./tiktok-linux-amd64 -help
./tiktok-linux-amd64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-linux-amd64 'https://www.tiktok.com/@example/live'
./tiktok_search-linux-amd64 'example keyword'
```

Trên macOS sử dụng Apple Silicon:

```bash
chmod +x tiktok-darwin-arm64 tiktok_livestream-darwin-arm64 tiktok_search-darwin-arm64
./tiktok-darwin-arm64 -help
./tiktok-darwin-arm64 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-darwin-arm64 'https://www.tiktok.com/@example/live'
./tiktok_search-darwin-arm64 'example keyword'
```

Trên Windows, mở PowerShell tại thư mục tải xuống:

```powershell
.\tiktok-windows-amd64.exe -help
.\tiktok-windows-amd64.exe "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_livestream-windows-amd64.exe "https://www.tiktok.com/@example/live"
.\tiktok_search-windows-amd64.exe "example keyword"
```

Thay các URL và từ khóa ví dụ bằng giá trị của bạn. Các ví dụ còn lại sử dụng `go run` để chạy từ source; khi sử dụng bản phát hành, hãy thay phần `go run ./cmd/...` bằng tên executable tương ứng ở trên.

## Video CLI

Tải video H.264 tốt nhất hiện có không kèm watermark (hành vi mặc định):

```bash
go run ./cmd/tiktok 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Thay `example` và `1234567890123456789` bằng username và video ID trong URL TikTok mà bạn muốn thu thập.

Output mẫu khi tải video không có watermark:

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

Tên file mặc định gồm username, video ID, loại watermark, chất lượng và codec. Sử dụng `-output` để chọn đường dẫn đích:

```bash
go run ./cmd/tiktok -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Tải video với một chiều cao cụ thể:

```bash
go run ./cmd/tiktok \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

In toàn bộ metadata và các phiên bản media đã chuẩn hóa, bao gồm URL có chữ ký, dưới dạng JSON mà không tải video:

```bash
go run ./cmd/tiktok -json 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Trình phát web của TikTok thường cung cấp các phiên bản phát không có watermark. Một số response cũng có thể cung cấp địa chỉ tải xuống chính thức có watermark của TikTok. Sử dụng `-watermark` để yêu cầu đúng phiên bản đó; công cụ sẽ trả về lỗi rõ ràng nếu TikTok không cung cấp và không tự động chuyển sang file không có watermark.

```bash
go run ./cmd/tiktok -watermark 'https://www.tiktok.com/@example/video/1234567890123456789'
```

File tải xuống được ghi vào một file tạm và chỉ được chuyển đến đường dẫn đích sau khi hoàn tất. File đã tồn tại sẽ không bao giờ bị ghi đè; hãy chọn đường dẫn `-output` khác khi cần. URL media có chữ ký sẽ hết hạn, vì vậy hãy thu thập lại video nếu URL cũ không còn hoạt động.

## Livestream CLI

Lấy thông tin một phòng LIVE và in tất cả stream hiện có dưới dạng bảng:

```bash
go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live'
```

Thay `example` bằng username TikTok của kênh LIVE mà bạn muốn lấy stream.

Output mẫu cho một phòng LIVE đang hoạt động:

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

In toàn bộ metadata của phòng dưới dạng JSON:

```bash
go run ./cmd/tiktok_livestream -json 'https://www.tiktok.com/@example/live'
```

## Search CLI

Tìm kiếm video TikTok công khai theo từ khóa. Output luôn là một JSON array chứa các URL video chuẩn:

```bash
go run ./cmd/tiktok_search 'golang tutorial'
```

```json
[
  "https://www.tiktok.com/@example/video/7000000000000000001",
  "https://www.tiktok.com/@another_example/video/7000000000000000002"
]
```

Không truyền từ khóa để yêu cầu danh sách video đề xuất mặc định của TikTok:

```bash
go run ./cmd/tiktok_search
```

Sử dụng `-locale` với mã khu vực gồm hai chữ cái hoặc thẻ ngôn ngữ-khu vực để tác động đến thứ tự kết quả. Đây chỉ là gợi ý xếp hạng, không phải proxy; TikTok vẫn có thể áp dụng khu vực của mạng và session.

```bash
go run ./cmd/tiktok_search -locale VN 'billiards'
go run ./cmd/tiktok_search -locale vi-VN 'billiards'
```

Request web hiện tại của TikTok sử dụng `count`, `cursor` và `offset`. CLI cung cấp chúng dưới dạng kích thước trang và chỉ số trang bắt đầu từ 0:

```bash
go run ./cmd/tiktok_search -page-size 20 -page-index 1 'billiards'
```

Kích thước trang phải nằm trong khoảng từ 1 đến 30. Nếu TikTok yêu cầu xác minh đối với API phân trang, CLI sẽ fallback sang trang discovery được TikTok render phía server cho page index 0; các trang sau vẫn cần truy cập được API. Sử dụng `-help` với bất kỳ công cụ nào để xem tất cả tùy chọn.

## Xác thực và giới hạn khu vực

Video, phòng LIVE và kết quả tìm kiếm công khai thường hoạt động mà không cần xác thực. Nếu TikTok yêu cầu đăng nhập, xác minh độ tuổi hoặc một bước xác minh tương tác, hãy cung cấp cookie của riêng bạn qua biến môi trường:

```bash
TIKTOK_COOKIE='ttwid=...; sessionid=...' \
  go run ./cmd/tiktok \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Không commit cookie vào source control. Mỗi client lưu các cookie nhận được từ trang TikTok và tái sử dụng chúng cho các request liên quan.

## Build từ source

```bash
go build -o tiktok ./cmd/tiktok
go build -o tiktok_livestream ./cmd/tiktok_livestream
go build -o tiktok_search ./cmd/tiktok_search
```

Các binary được tạo tại thư mục gốc đã được Git bỏ qua. Bạn cũng có thể đặt chúng trong thư mục `bin/`.
