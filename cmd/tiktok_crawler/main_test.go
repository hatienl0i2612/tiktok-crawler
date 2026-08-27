package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/livestream"
)

type failingWriter struct{ err error }

func (writer failingWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestParseOptionsDetectsVideoAndLiveURLs(t *testing.T) {
	const videoURL = "https://www.tiktok.com/@forever0404_/video/7671176369300327700"
	const photoURL = "https://www.tiktok.com/@5gvietteldv/photo/7666697358540852500"
	const liveURL = "https://www.tiktok.com/@weathernewslive/live"
	const shortDramaURL = "https://www.tiktok.com/shortdrama/episode/7665073849083368469/1"
	const profileURL = "https://www.tiktok.com/@forever0404_"
	tests := []struct {
		name string
		args []string
		want options
	}{
		{
			name: "video with flags after URL",
			args: []string{videoURL, "-watermark", "-quality", "720p", "-output", "result.mp4"},
			want: options{inputURL: videoURL, content: contentTypeVideo, output: "result.mp4", watermark: true, quality: "720p", timeout: 20 * time.Second},
		},
		{
			name: "live ignores video-only options",
			args: []string{liveURL, "-quality", "720p", "-watermark", "-output", "unused.mp4", "-json"},
			want: options{inputURL: liveURL, content: contentTypeLive, output: "unused.mp4", watermark: true, quality: "720p", json: true, timeout: 20 * time.Second},
		},
		{
			name: "live flags before URL",
			args: []string{"-verbose", "-timeout", "30s", "-headers", "User-Agent: test-agent", liveURL},
			want: options{inputURL: liveURL, content: contentTypeLive, quality: "best", verbose: true, headers: map[string]string{"User-Agent": "test-agent"}, timeout: 30 * time.Second},
		},
		{
			name: "cookies from browser flag",
			args: []string{liveURL, "-cookies-from-browser", "chrome"},
			want: options{inputURL: liveURL, content: contentTypeLive, quality: "best", cookiesBrowser: "chrome", timeout: 20 * time.Second},
		},
		{
			name: "cookies file flag",
			args: []string{videoURL, "-cookies-file", "cookies.txt"},
			want: options{inputURL: videoURL, content: contentTypeVideo, quality: "best", cookiesFile: "cookies.txt", timeout: 20 * time.Second},
		},
		{
			name: "Short Drama uses video options",
			args: []string{shortDramaURL, "-quality", "720p", "-output", "episode.mp4"},
			want: options{inputURL: shortDramaURL, content: contentTypeShortDrama, output: "episode.mp4", quality: "720p", timeout: 20 * time.Second},
		},
		{
			name: "Photo Post uses an output directory and ignores quality",
			args: []string{photoURL, "-quality", "720p", "-output", "images", "-watermark"},
			want: options{inputURL: photoURL, content: contentTypePhoto, output: "images", quality: "best", watermark: true, timeout: 20 * time.Second},
		},
		{
			name: "profile uses video options and an output directory",
			args: []string{profileURL, "-quality", "1080p", "-output", "videos", "-watermark"},
			want: options{inputURL: profileURL, content: contentTypeProfile, output: "videos", quality: "1080p", watermark: true, timeout: 20 * time.Second},
		},
		{
			name: "repeated headers flag",
			args: []string{liveURL, "-headers", "X-Custom: abc", "-headers", "user-agent: custom-ua"},
			want: options{inputURL: liveURL, content: contentTypeLive, quality: "best", headers: map[string]string{"X-Custom": "abc", "User-Agent": "custom-ua"}, timeout: 20 * time.Second},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got, err := parseOptions(test.args, &stderr)
			if err != nil {
				t.Fatalf("parseOptions(%q): %v\nstderr: %s", test.args, err, stderr.String())
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("options = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseOptionsRejectsInvalidHeaders(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseOptions([]string{"https://www.tiktok.com/@example/video/1234567890123456789", "-headers", "MissingColon"}, &stderr)
	if err == nil {
		t.Fatal("parseOptions() accepted a -headers value without a colon")
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		url  string
		want contentType
		fail bool
	}{
		{"https://www.tiktok.com/@example/live", contentTypeLive, false},
		{"https://www.tiktok.com/@example/video/1234567890123456789?lang=en", contentTypeVideo, false},
		{"https://www.tiktok.com/@example/photo/1234567890123456789", contentTypePhoto, false},
		{"https://www.tiktok.com/shortdrama/episode/7665073849083368469/1", contentTypeShortDrama, false},
		{"https://www.tiktok.com/@example", contentTypeProfile, false},
		{"https://www.tiktok.com/@example/", contentTypeProfile, false},
		{"https://example.com/@example/live", "", true},
		{"https://www.tiktok.com/explore", "", true},
	}
	for _, test := range tests {
		got, err := detectContentType(test.url)
		if test.fail {
			if err == nil {
				t.Errorf("detectContentType(%q) succeeded with %q", test.url, got)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Errorf("detectContentType(%q) = %q, %v; want %q", test.url, got, err, test.want)
		}
	}
}

func TestDefaultProfileOutputDirectory(t *testing.T) {
	if got := defaultProfileOutputDirectory("forever0404_"); got != "forever0404_videos" {
		t.Fatalf("defaultProfileOutputDirectory() = %q", got)
	}
}

func TestPrintSummaryReturnsWriterError(t *testing.T) {
	want := errors.New("write failed")
	err := printSummary(failingWriter{err: want}, &livestream.Result{})
	if !errors.Is(err, want) {
		t.Fatalf("printSummary() error = %v, want %v", err, want)
	}
}

func TestResolveCookieSourcesEmptyWithoutSource(t *testing.T) {
	cookie, err := resolveCookieSources(options{})
	if err != nil || cookie != "" {
		t.Fatalf("resolveCookieSources() = %q, %v; want empty", cookie, err)
	}
}

func TestResolveCookieSourcesRejectsUnsupportedBrowser(t *testing.T) {
	_, err := resolveCookieSources(options{cookiesBrowser: "netscape"})
	if err == nil {
		t.Fatal("resolveCookieSources() succeeded for an unsupported browser")
	}
}

func TestResolveCookieSourcesLoadsFile(t *testing.T) {
	path := writeTempCookieFile(t, "sessionid=abc; ttwid=xyz")
	cookie, err := resolveCookieSources(options{cookiesFile: path, cookiesBrowser: "chrome"})
	if err != nil {
		t.Fatalf("resolveCookieSources(): %v", err)
	}
	if cookie != "sessionid=abc; ttwid=xyz" {
		t.Fatalf("cookie = %q, want cookies loaded from the file", cookie)
	}
}

func TestResolveCookieSourcesFileTakesPrecedence(t *testing.T) {
	path := writeTempCookieFile(t, "sessionid=from-file")
	cookie, err := resolveCookieSources(options{cookiesFile: path, cookiesBrowser: "chrome"})
	if err != nil {
		t.Fatalf("resolveCookieSources(): %v", err)
	}
	if cookie != "sessionid=from-file" {
		t.Fatalf("cookie = %q, want the file to take precedence", cookie)
	}
}

func TestResolveCookieSourcesMissingFile(t *testing.T) {
	_, err := resolveCookieSources(options{cookiesFile: "does-not-exist.txt"})
	if err == nil {
		t.Fatal("resolveCookieSources() succeeded for a missing cookie file")
	}
}

func writeTempCookieFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/cookies.txt"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write cookie file: %v", err)
	}
	return path
}
