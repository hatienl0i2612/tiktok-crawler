[English](README.md) | Tiếng Việt | [简体中文](README.zh-CN.md)

# Công cụ dòng lệnh TikTok Video và Livestream

Repository này cung cấp hai công cụ dòng lệnh Go độc lập:

- `tiktok` thu thập metadata của video TikTok công khai, liệt kê các phiên bản media có thể tải xuống và tải file MP4 đã chọn.
- `tiktok_livestream` lấy các URL phát có chữ ký cho một phòng TikTok LIVE công khai.

Cả hai công cụ chỉ sử dụng thư viện chuẩn của Go.

## Tải bản phát hành

Các binary được build sẵn có trên [trang GitHub Releases](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest). Hãy tải hai file phù hợp với hệ điều hành của bạn:

| Hệ điều hành | Video CLI | Livestream CLI |
| --- | --- | --- |
| Linux (amd64) | [`tiktok-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-linux-amd64) | [`tiktok_livestream-linux-amd64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-linux-amd64) |
| macOS Apple Silicon (arm64) | [`tiktok-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-darwin-arm64) | [`tiktok_livestream-darwin-arm64`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-darwin-arm64) |
| Windows (amd64) | [`tiktok-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok-windows-amd64.exe) | [`tiktok_livestream-windows-amd64.exe`](https://github.com/hatienl0i2612/tiktok-crawler/releases/latest/download/tiktok_livestream-windows-amd64.exe) |

Trên Linux, cấp quyền thực thi cho cả hai file rồi chạy chúng từ thư mục tải xuống:

```bash
chmod +x tiktok-linux-amd64 tiktok_livestream-linux-amd64
./tiktok-linux-amd64 -help
./tiktok-linux-amd64 -download 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-linux-amd64 'https://www.tiktok.com/@example/live'
```

Trên macOS sử dụng Apple Silicon:

```bash
chmod +x tiktok-darwin-arm64 tiktok_livestream-darwin-arm64
./tiktok-darwin-arm64 -help
./tiktok-darwin-arm64 -download 'https://www.tiktok.com/@example/video/1234567890123456789'
./tiktok_livestream-darwin-arm64 'https://www.tiktok.com/@example/live'
```

Trên Windows, mở PowerShell tại thư mục tải xuống:

```powershell
.\tiktok-windows-amd64.exe -help
.\tiktok-windows-amd64.exe -download "https://www.tiktok.com/@example/video/1234567890123456789"
.\tiktok_livestream-windows-amd64.exe "https://www.tiktok.com/@example/live"
```

Thay các URL ví dụ bằng video TikTok hoặc phòng LIVE mà bạn muốn xử lý. Các ví dụ còn lại sử dụng `go run` để chạy từ source; khi sử dụng bản phát hành, hãy thay `go run ./cmd/tiktok` hoặc `go run ./cmd/tiktok_livestream` bằng tên executable tương ứng ở trên.

## Video CLI

In metadata đã chuẩn hóa cùng tất cả phiên bản media có sẵn dưới dạng JSON:

```bash
go run ./cmd/tiktok 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Thay `example` và `1234567890123456789` bằng username và video ID trong URL TikTok mà bạn muốn thu thập.

Tải phiên bản H.264 mặc định không có watermark:

```bash
go run ./cmd/tiktok -download 'https://www.tiktok.com/@example/video/1234567890123456789'
```

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

Tên file mặc định gồm username, video ID, loại watermark, chất lượng và codec. Sử dụng `-output` để chọn đường dẫn đích; tùy chọn này cũng tự động bật chế độ tải xuống:

```bash
go run ./cmd/tiktok  -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Chọn phiên bản H.265 tốt nhất hoặc một chiều cao cụ thể:

```bash
go run ./cmd/tiktok \
  -download \
  -codec h265 \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Chỉ in URL có chữ ký đã được chọn:

```bash
go run ./cmd/tiktok -url-only 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Trình phát web của TikTok thường cung cấp các phiên bản phát không có watermark. Một số response cũng có thể cung cấp địa chỉ tải xuống chính thức có watermark của TikTok. Sử dụng `-watermark` để yêu cầu đúng phiên bản đó; công cụ sẽ trả về lỗi rõ ràng nếu TikTok không cung cấp và không tự động chuyển sang file không có watermark.

File tải xuống được ghi vào một file tạm và chỉ được chuyển đến đường dẫn đích sau khi hoàn tất. File đã tồn tại sẽ được giữ nguyên trừ khi sử dụng `-force`. URL media có chữ ký sẽ hết hạn, vì vậy hãy thu thập lại video nếu URL cũ không còn hoạt động.

## Livestream CLI

In URL stream tốt nhất hiện có, ưu tiên kết hợp H.264 và HLS khi TikTok cung cấp:

```bash
go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live'
```

Thay `example` bằng username TikTok của kênh LIVE mà bạn muốn lấy stream.

Sử dụng `-all` để in tất cả stream hiện có dưới dạng bảng:

```bash
go run ./cmd/tiktok_livestream -all 'https://www.tiktok.com/@example/live'
```

Output mẫu cho một phòng LIVE đang hoạt động:

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

Phát stream đã chọn bằng [mpv](https://mpv.io/installation/), trình phát media được khuyến nghị cho các URL livestream này:

```bash
mpv "$(go run ./cmd/tiktok_livestream 'https://www.tiktok.com/@example/live')"
```

In toàn bộ metadata của phòng dưới dạng JSON:

```bash
go run ./cmd/tiktok_livestream -json 'https://www.tiktok.com/@example/live'
```

Chọn stream H.265 HLS chất lượng gốc:

```bash
go run ./cmd/tiktok_livestream \
  -codec h265 \
  -quality origin \
  -format hls \
  'https://www.tiktok.com/@example/live'
```

Sử dụng `-help` với một trong hai công cụ để xem tất cả tùy chọn.

## Xác thực và giới hạn khu vực

Video và phòng LIVE công khai thường hoạt động mà không cần xác thực. Nếu TikTok yêu cầu đăng nhập, xác minh độ tuổi hoặc một khu vực cụ thể, hãy cung cấp cookie của riêng bạn qua biến môi trường:

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
```

Các binary được tạo tại thư mục gốc đã được Git bỏ qua. Bạn cũng có thể đặt chúng trong thư mục `bin/`.
