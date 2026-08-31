package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/livestream"
	"github.com/hatienl0i2612/tiktok-crawler/mpv"
)

type failingWriter struct{ err error }

func (writer failingWriter) Write([]byte) (int, error) { return 0, writer.err }

type fakeLivePlayer struct {
	executable string
	ensured    bool
	played     bool
	streamURL  string
	options    mpv.PlayOptions
}

func (player *fakeLivePlayer) Ensure(context.Context) (string, error) {
	player.ensured = true
	return player.executable, nil
}

func (player *fakeLivePlayer) Play(_ context.Context, executable, streamURL string, options mpv.PlayOptions) error {
	if executable != player.executable {
		return errors.New("unexpected executable")
	}
	player.played = true
	player.streamURL = streamURL
	player.options = options
	return nil
}

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

func TestRunPrintsVersionWithoutURL(t *testing.T) {
	previous := buildVersion
	previousFetcher := latestReleaseFetcher
	buildVersion = "v1.2.3"
	latestReleaseFetcher = func(context.Context) (string, error) { return "v1.2.3", nil }
	t.Cleanup(func() {
		buildVersion = previous
		latestReleaseFetcher = previousFetcher
	})

	for _, argument := range []string{"-version", "--version"} {
		t.Run(argument, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run([]string{argument}, &stdout, &stderr); err != nil {
				t.Fatalf("run(%q): %v", argument, err)
			}
			if got := stdout.String(); got != "tiktok_crawler v1.2.3\n"+cliDescription+"\n" {
				t.Fatalf("stdout = %q", got)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestPrintVersionWarnsWhenUpdateIsAvailable(t *testing.T) {
	previous := buildVersion
	previousFetcher := latestReleaseFetcher
	buildVersion = "v1.1.5"
	latestReleaseFetcher = func(context.Context) (string, error) { return "v1.1.6", nil }
	t.Cleanup(func() {
		buildVersion = previous
		latestReleaseFetcher = previousFetcher
	})

	var stdout, stderr bytes.Buffer
	if err := printVersion(&stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "v1.1.6 (current: v1.1.5)") || !strings.Contains(stderr.String(), projectLatestReleaseURL) {
		t.Fatalf("update warning = %q", stderr.String())
	}
}

func TestPrintVersionIgnoresFailedUpdateCheck(t *testing.T) {
	previous := buildVersion
	previousFetcher := latestReleaseFetcher
	buildVersion = "v1.1.5"
	latestReleaseFetcher = func(context.Context) (string, error) { return "", errors.New("offline") }
	t.Cleanup(func() {
		buildVersion = previous
		latestReleaseFetcher = previousFetcher
	})

	var stdout, stderr bytes.Buffer
	if err := printVersion(&stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "v1.1.5") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestNewerVersionAvailable(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v1.1.5", "v1.1.6", true},
		{"1.1.5", "v1.1.6", true},
		{"v1.1.6", "v1.1.6", false},
		{"v1.2.0", "v1.1.6", false},
		{"dev", "v1.1.6", false},
	}
	for _, test := range tests {
		if got := newerVersionAvailable(test.current, test.latest); got != test.want {
			t.Errorf("newerVersionAvailable(%q, %q) = %v, want %v", test.current, test.latest, got, test.want)
		}
	}
}

func TestFetchLatestReleaseFromGitHubResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept") != "application/vnd.github+json" || request.Header.Get("User-Agent") == "" {
			t.Errorf("GitHub headers = %v", request.Header)
		}
		_, _ = writer.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	t.Cleanup(server.Close)

	got, err := fetchLatestReleaseFrom(context.Background(), server.Client(), server.URL)
	if err != nil || got != "v1.2.3" {
		t.Fatalf("fetchLatestReleaseFrom() = %q, %v", got, err)
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

func TestOpenLiveWithMPVSelectsBestStream(t *testing.T) {
	result := &livestream.Result{
		FinalURL: "https://www.tiktok.com/@example/live",
		User:     livestream.User{UniqueID: "example"},
		Streams: []livestream.Stream{
			{Codec: "h264", Quality: "hd", Protocol: "hls", URL: "https://pull.tiktokcdn.com/hd.m3u8"},
			{Codec: "h264", Quality: "origin", Protocol: "hls", URL: "https://pull.tiktokcdn.com/best.m3u8"},
		},
	}
	player := &fakeLivePlayer{executable: "/tmp/mpv"}
	var stdout, stderr bytes.Buffer
	if err := openLiveWithMPV(result, player, &stdout, &stderr); err != nil {
		t.Fatalf("openLiveWithMPV(): %v", err)
	}
	if !player.ensured || !player.played || player.streamURL != "https://pull.tiktokcdn.com/best.m3u8" {
		t.Fatalf("player = %+v", player)
	}
	if player.options.Referer != result.FinalURL || player.options.Title != "TikTok LIVE @example" {
		t.Fatalf("play options = %+v", player.options)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "origin, h264, hls") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
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
