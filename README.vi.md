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

## Video

Tải video chất lượng tốt nhất hiện có không kèm watermark (hành vi mặc định):

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/video/1234567890123456789'
```

TikTok cũng dùng URL `/video/<id>` cho Story dạng video. Khi player API không trả Story, resolver sẽ đọc metadata embed và `playAddr` mà trang web TikTok dùng để phát video; JSON output có thêm `"is_story": true`. `playAddr` là source mặc định không có TikTok watermark, còn `-watermark` chọn `downloadAddr`. Story công khai còn hiệu lực không cần cookie trình duyệt.

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
go run ./cmd/tiktok_crawler -output './video.mp4' 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Tải video với một chiều cao cụ thể:

```bash
go run ./cmd/tiktok_crawler \
  -quality 1080p \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

In toàn bộ metadata và các phiên bản media đã chuẩn hóa, bao gồm URL có chữ ký, dưới dạng JSON mà không tải video:

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@example/video/1234567890123456789'
```

Trình phát web của TikTok thường cung cấp các phiên bản phát không có watermark. Một số response cũng có thể cung cấp địa chỉ tải xuống chính thức có watermark của TikTok. Sử dụng `-watermark` để yêu cầu đúng phiên bản đó; công cụ sẽ trả về lỗi rõ ràng nếu TikTok không cung cấp và không tự động chuyển sang file không có watermark.

```bash
go run ./cmd/tiktok_crawler -watermark 'https://www.tiktok.com/@example/video/1234567890123456789'
```

File tải xuống được ghi vào một file tạm và chỉ được chuyển đến đường dẫn đích sau khi hoàn tất. File đã tồn tại sẽ không bao giờ bị ghi đè; hãy chọn đường dẫn `-output` khác khi cần. URL media có chữ ký sẽ hết hạn, vì vậy hãy thu thập lại video nếu URL cũ không còn hoạt động.

## Profile creator

In metadata user công khai và canonical URL của 10 video public mới nhất mà creator embed của TikTok cung cấp:

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@forever0404_'
```

JSON gồm `user`, `listing`, `video_urls` và `videos`. API phân trang chính `/api/post/item_list/` hiện được bảo vệ bằng chữ ký `X-Dynosaur`/`X-Gnarly` do browser tạo và vẫn có thể trả interactive captcha dù request được ký đúng. Vì vậy crawler dùng creator card SSR công khai `/embed/@username`: nhánh này ổn định và trả 10 video public mới nhất, nhưng không cung cấp toàn bộ archive của user.

Khi không dùng `-json`, crawler đưa từng canonical URL trở lại video resolver hiện tại rồi tải toàn bộ video nhận được. `-quality`, `-watermark` và `-output` áp dụng cho cả collection:

```bash
go run ./cmd/tiktok_crawler \
  -quality 1080p \
  -output "$HOME/Downloads/forever0404-videos" \
  'https://www.tiktok.com/@forever0404_'
```

Nếu không truyền `-output`, các file được lưu trong thư mục mới `<username>_videos/`. Mỗi tên file vẫn giữ TikTok video ID tương ứng.

## Photo Post

Tải tất cả ảnh trong Photo Post theo đúng thứ tự của bài đăng:

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/photo/1234567890123456789'
```

TikTok cũng dùng URL `/photo/<id>` cho Story dạng ảnh. Khi player API không trả Story, resolver sẽ tự động đọc embed metadata của TikTok; JSON output có thêm `"is_story": true`. Story công khai còn hiệu lực không cần cookie trình duyệt.

Hãy thay `example` và `1234567890123456789` bằng username và Photo Post ID trong URL TikTok cần thu thập. Mặc định, ảnh được lưu vào thư mục mới `<username>_<photo-id>_images/` trong thư mục hiện tại. Dùng `-output` để chọn thư mục đích; tool sẽ tạo thư mục nếu cần:

```bash
go run ./cmd/tiktok_crawler \
  -output '~/Downloads/tiktok-photo-post' \
  'https://www.tiktok.com/@example/photo/1234567890123456789'
```

Dùng `-json` để in metadata bài đăng, signed image URL và signed audio playback URL khi TikTok cung cấp, mà không tải. `-quality` bị bỏ qua cho Photo Post vì TikTok chỉ cung cấp một display image cho mỗi ảnh trong bài. Một số Photo Post có thể công bố image source watermark chính thức; `-watermark` yêu cầu đúng source đó và trả lỗi thay vì âm thầm tải ảnh không watermark khi source không tồn tại.

## Tập Short Drama

URL tập Short Drama sử dụng cùng các tùy chọn video và mặc định sẽ download:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/shortdrama/episode/7665073849083368469/1'
```

Hoặc nạp cookie từ file `.txt`:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-file ./cookies.txt \
  'https://www.tiktok.com/shortdrama/episode/7665073849083368469/1'
```

TikTok cho phép lấy metadata tập công khai nhưng hiện yêu cầu `msToken` hợp lệ từ browser cho request playback metadata có chữ ký. Hãy cung cấp cookie TikTok của chính bạn bằng `-cookies-from-browser` hoặc `-cookies-file`; tool tạo `X-Bogus` hoàn toàn ở local và không ghi cookie xuống ổ đĩa. Các option `-json`, `-output` và `-quality` hoạt động giống video thường.

## Livestream

Resolve phòng LIVE, chọn stream tốt nhất và mở bằng mpv:

```bash
go run ./cmd/tiktok_crawler 'https://www.tiktok.com/@example/live'
```

Thay `example` bằng username TikTok của kênh LIVE mà bạn muốn xem. Lệnh LIVE thông thường không còn in danh sách signed URL. Tool ưu tiên quality `origin`, sau đó tới quality tốt nhất còn lại; nếu cùng quality thì ưu tiên H.264, CDN line `main`, rồi HLS.

Ở lần chạy đầu tiên, tool tìm `mpv` trong `PATH`. Nếu chưa có, CLI tải và cache bản portable trong `os.TempDir()/tiktok-crawler/mpv/<os>-<arch>`:

| Nền tảng | Nguồn bản portable |
| --- | --- |
| Windows `amd64`, `arm64`, `386` | ZIP stable first-party từ `mpv-player/mpv` |
| macOS `arm64`, `amd64` | ZIP stable first-party từ `mpv-player/mpv` |
| Linux `amd64`, `arm64` | AppImage từ `pkgforge-dev/mpv-AppImage` |

Mỗi artifact bị giới hạn kích thước và được kiểm SHA-256 theo metadata GitHub Release trước khi giải nén hoặc chạy. `os.TempDir` tự chọn thư mục temp đúng với hệ điều hành. Những lần sau sẽ tái sử dụng mpv trong hệ thống hoặc binary portable đã cache.

Status mẫu cho một phòng LIVE đang hoạt động:

```text
Opening TikTok LIVE @example in mpv (origin, h264, hls)
```

Muốn xem toàn bộ metadata và URL stream mà không mở mpv, dùng JSON mode:

```bash
go run ./cmd/tiktok_crawler -json 'https://www.tiktok.com/@example/live'
```

## Xác thực và giới hạn khu vực

Video và phòng LIVE công khai thường hoạt động mà không cần xác thực. Nếu TikTok yêu cầu đăng nhập, xác minh độ tuổi hoặc một khu vực cụ thể, hãy cung cấp cookie từ trình duyệt đã đăng nhập hoặc từ file cookie:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Không commit cookie vào source control. Mỗi client lưu các cookie nhận được từ trang TikTok và tái sử dụng chúng cho các request liên quan.

### Cookie từ trình duyệt

Thay vì sao chép cookie thủ công, hãy đọc cookie từ trình duyệt đã đăng nhập TikTok:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-from-browser chrome \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Các trình duyệt được hỗ trợ theo package `browsercookie`: brave, chrome, chromium, vivaldi, edge, edge-dev, arc, opera, opera-gx, firefox, librewolf, zen, safari. Công cụ chỉ đọc cookie thuộc tiktok.com và gửi chúng trong mọi request. Trên macOS, Terminal có thể cần quyền Full Disk Access để đọc kho cookie của trình duyệt; Safari và một số trình duyệt Chromium cũng yêu cầu trình duyệt phải được liên kết với Keychain của macOS.

### Cookie từ file

Truyền đường dẫn tới file cookie `.txt` qua option `-cookies-file`. File có thể chứa giá trị Cookie header thô (`ttwid=...; sessionid=...`) hoặc là file xuất Netscape cookie-jar:

```bash
go run ./cmd/tiktok_crawler \
  -cookies-file ./cookies.txt \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Cookie tải từ file sẽ được gửi trong mọi request. `-cookies-file` được ưu tiên hơn `-cookies-from-browser`. Không commit file cookie vào source control.

### Header tùy chỉnh

Thêm HTTP header vào mọi request bằng `-headers`. Có thể lặp lại nhiều lần, mỗi lần một cặp `Key: Value`:

```bash
go run ./cmd/tiktok_crawler \
  -headers 'User-Agent: Mozilla/5.0 ...' \
  -headers 'X-Forwarded-For: 1.2.3.4' \
  'https://www.tiktok.com/@example/video/1234567890123456789'
```

Các header tùy chỉnh ghi đè header mặc định của tool (`User-Agent`, `Accept-Language`, `Cache-Control`, `Accept`). Để gửi dấu vân tay trình duyệt khác, hãy đặt `User-Agent` qua `-headers` (mặc định tool sẽ tự dùng). Khóa `Referer` hoặc `Cookie` sẽ bị thay bởi giá trị mà crawler tự tính theo từng request.

## Build từ source

```bash
go build -o tiktok_crawler ./cmd/tiktok_crawler
```

Các binary được tạo tại thư mục gốc đã được Git bỏ qua. Bạn cũng có thể đặt chúng trong thư mục `bin/`.

## Đóng góp

Mọi đóng góp đều được chào đón. Nếu bạn muốn cải thiện crawler, hỗ trợ thêm định dạng TikTok hoặc hoàn thiện tài liệu, hãy gửi pull request.

Nếu gặp lỗi hoặc response TikTok bất thường, hãy [mở issue](https://github.com/hatienl0i2612/tiktok-crawler/issues) và cung cấp URL bị ảnh hưởng, command đã chạy, error output, hệ điều hành và version của tool nếu có thể. Không đăng cookie riêng tư hoặc dữ liệu xác thực.
