package video

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

// TestDownloadVideoPartialMD5Integration downloads only the first chunk of a
// real TikTok video and verifies it against a precomputed MD5 checksum, so the
// integration check stays cheap instead of pulling the whole file.
func TestDownloadVideoPartialMD5Integration(t *testing.T) {
	const (
		inputURL = "https://www.tiktok.com/@forever0404_/video/7671176369300327700"
		videoID  = "7671176369300327700"
		author   = "forever0404_"
		// Number of leading bytes to download and hash.
		chunkSize = 256 << 10
		// MD5 of the first chunkSize bytes of the best (highest quality,
		// no-watermark) variant, resolved with the same selection order as the
		// downloader (media.Sort).
		expectedMD5 = "56eece27760620f6a64a7d1e1b3ba3cf"
	)

	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := client.Resolve(ctx, inputURL)
	if err != nil {
		t.Fatalf("resolve %s: %v", inputURL, err)
	}
	if result.Video.ID != videoID {
		t.Fatalf("video id = %q, want %q", result.Video.ID, videoID)
	}
	if !strings.EqualFold(result.Video.Author.UniqueID, author) {
		t.Fatalf("author = %q, want %q", result.Video.Author.UniqueID, author)
	}
	if len(result.Media) == 0 {
		t.Fatal("resolved video has no downloadable media")
	}

	variants := append([]media.Variant(nil), result.Media...)
	media.Sort(variants)
	selected := variants[0]
	parsed, err := url.Parse(selected.URL)
	if err != nil || parsed.Scheme != "https" || !tiktok.IsAllowedHost(parsed.Hostname()) {
		t.Fatalf("invalid media URL %q: %v", selected.URL, err)
	}

	// Follow the downloader's backup order: TikTok can reject a CDN URL while
	// serving the same profile through its /aweme/v1/play/ backup. Request only
	// the leading chunk; limit the reader below if the server ignores Range.
	var response *http.Response
	var failures []string
	for _, mediaURL := range media.UniqueStrings(append([]string{selected.URL}, selected.BackupURLs...)) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
		if err != nil {
			t.Fatalf("build media request: %v", err)
		}
		client.Session().SetHeaders(request, "video/mp4,video/*;q=0.9,*/*;q=0.7", result.FinalURL)
		request.Header.Set("Range", fmt.Sprintf("bytes=0-%d", chunkSize-1))
		candidate, err := client.Session().HTTPClient().Do(request)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if candidate.StatusCode == http.StatusPartialContent || candidate.StatusCode == http.StatusOK {
			response = candidate
			break
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %s", request.URL.Hostname(), candidate.Status))
		candidate.Body.Close()
	}
	if response == nil {
		t.Fatalf("download first %d bytes: all profile URLs failed: %s", chunkSize, strings.Join(failures, "; "))
	}
	defer response.Body.Close()
	if finalHost := response.Request.URL.Hostname(); !tiktok.IsAllowedHost(finalHost) {
		t.Fatalf("download redirect landed on a disallowed host: %s", finalHost)
	}

	hash := md5.New()
	read, err := io.Copy(hash, io.LimitReader(response.Body, chunkSize))
	if err != nil {
		t.Fatalf("read first chunk: %v", err)
	}
	if read < chunkSize {
		t.Fatalf("expected at least %d bytes of data, got %d", chunkSize, read)
	}
	gotMD5 := hex.EncodeToString(hash.Sum(nil))
	if gotMD5 != expectedMD5 {
		t.Fatalf("first %d bytes MD5 = %s, want %s (quality=%s codec=%s watermarked=%v)",
			chunkSize, gotMD5, expectedMD5, selected.Quality, selected.Codec, selected.Watermarked)
	}
}
