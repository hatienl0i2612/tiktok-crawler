package downloader

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

func TestDownloadDASHRejectsMissingAudio(t *testing.T) {
	t.Parallel()
	session, err := tiktok.NewSession(tiktok.SessionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		t.Fatal("incomplete DASH metadata must be rejected before downloading")
		return nil, errors.New("unexpected request")
	})
	_, err = Download(context.Background(), session, []media.Variant{{
		URL: "https://v16.tiktokcdn.com/video", Audio: &media.Variant{Type: "audio"},
	}}, FileInfo{}, Options{OutputPath: filepath.Join(t.TempDir(), "video.mp4")})
	if err == nil || !strings.Contains(err.Error(), "no matching audio URL") {
		t.Fatalf("Download() error = %v", err)
	}
}

func TestDownloadDASHMuxesAudioAndUsesFallback(t *testing.T) {
	t.Parallel()
	executable, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg is required for the local mux integration check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	fixtures := t.TempDir()
	videoPath, audioPath := filepath.Join(fixtures, "video.mp4"), filepath.Join(fixtures, "audio.m4a")
	for _, arguments := range [][]string{
		{"-f", "lavfi", "-i", "color=c=black:s=16x16:r=10", "-t", "0.2", "-an", "-c:v", "mpeg4", videoPath},
		{"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", "0.2", "-c:a", "aac", audioPath},
	} {
		if output, err := exec.CommandContext(ctx, executable, append([]string{"-v", "error", "-nostdin"}, arguments...)...).CombinedOutput(); err != nil {
			t.Fatalf("generate fixture: %v: %s", err, output)
		}
	}
	video, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range []bool{false, true} {
		name := "single"
		if batch {
			name = "batch"
		}
		t.Run(name, func(t *testing.T) {
			session, err := tiktok.NewSession(tiktok.SessionOptions{Cookie: "session=test"})
			if err != nil {
				t.Fatal(err)
			}
			var paths []string
			session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				paths = append(paths, request.URL.Path)
				if request.Header.Get("Cookie") != "session=test" || request.Header.Get("Referer") != "https://www.tiktok.com/@creator/video/123" {
					t.Fatal("track request lost session headers")
				}
				status, payload := http.StatusOK, audio
				switch request.URL.Path {
				case "/video":
					payload = video
				case "/audio-main", "/audio-backup":
					status, payload = http.StatusForbidden, []byte("forbidden")
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/octet-stream"}}, Body: io.NopCloser(bytes.NewReader(payload)), ContentLength: int64(len(payload)), Request: request}, nil
			})
			variants := []media.Variant{{Type: "video", Format: "mp4", URL: "https://v16.tiktokcdn.com/video", Size: int64(len(video)), Audio: &media.Variant{
				Type: "audio", URL: "https://v16.tiktokcdn.com/audio-main", Size: int64(len(audio)),
				BackupURLs: []string{"https://v19.tiktokcdn.com/audio-backup", "https://www.tiktok.com/audio-fallback"},
			}}}
			var status bytes.Buffer
			var progress []DownloadProgress
			file := FileInfo{VideoID: "123", Referer: "https://www.tiktok.com/@creator/video/123"}
			directory := t.TempDir()
			var result *DownloadResult
			if batch {
				collection, err := DownloadAll(ctx, session, []BatchItem{{Variants: variants}}, file, BatchOptions{
					OutputDir: directory, Status: &status,
					Progress: func(update BatchProgress) { progress = append(progress, update.DownloadProgress) },
				})
				if err != nil {
					t.Fatal(err)
				}
				result = &collection.Downloads[0]
			} else {
				result, err = Download(ctx, session, variants, file, Options{
					OutputPath: filepath.Join(directory, "merged.mp4"), Status: &status,
					Progress: func(update DownloadProgress) { progress = append(progress, update) },
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			if strings.Join(paths, ",") != "/video,/audio-main,/audio-backup,/audio-fallback" {
				t.Fatalf("request order = %v", paths)
			}
			if result.Bytes <= 0 || result.ContentType != "video/mp4" || !strings.Contains(status.String(), "Merging video and audio") {
				t.Fatalf("result=%+v status=%q", result, status.String())
			}
			for index, update := range progress {
				if index > 0 && update.DownloadedBytes < progress[index-1].DownloadedBytes {
					t.Fatalf("progress moved backwards: %v", progress)
				}
			}
			last := progress[len(progress)-1]
			if want := int64(len(video) + len(audio)); last.DownloadedBytes != want || last.TotalBytes != want {
				t.Fatalf("final progress = %+v, want %d", last, want)
			}
			// Explicit maps fail if either stream is absent; decoding also verifies the output is playable.
			if output, err := exec.CommandContext(ctx, executable, "-v", "error", "-nostdin", "-i", result.Path, "-map", "0:v:0", "-map", "0:a:0", "-f", "null", "-").CombinedOutput(); err != nil {
				t.Fatalf("verify muxed streams: %v: %s", err, output)
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 {
				t.Fatalf("temporary tracks were retained: %v, %v", entries, err)
			}
		})
	}
}

func TestDownloadDASHCleansUpFailures(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"audio", "mux", "cancelled"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			session, err := tiktok.NewSession(tiktok.SessionOptions{})
			if err != nil {
				t.Fatal(err)
			}
			payload := []byte{0, 0, 0, 20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 1, 2, 3, 4}
			session.HTTPClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				status := http.StatusOK
				if failure == "audio" && request.URL.Path == "/audio" {
					status = http.StatusForbidden
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"video/mp4"}}, Body: io.NopCloser(bytes.NewReader(payload)), ContentLength: int64(len(payload)), Request: request}, nil
			})
			directory := t.TempDir()
			_, err = downloadWithAudio(ctx, session, media.Variant{
				URL: "https://v16.tiktokcdn.com/video", Audio: &media.Variant{Type: "audio", URL: "https://v16.tiktokcdn.com/audio"},
			}, downloadOptions{
				OutputPath: filepath.Join(directory, "output.mp4"),
				Progress: func(update DownloadProgress) {
					if failure == "cancelled" && update.DownloadedBytes == int64(2*len(payload)) {
						cancel()
					}
				},
			}, filepath.Join(directory, "missing-ffmpeg"))
			if err == nil || !strings.Contains(err.Error(), map[string]string{"audio": "download audio track", "mux": "mux video and audio", "cancelled": "context canceled"}[failure]) {
				t.Fatalf("error = %v", err)
			}
			if failure == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation was not preserved: %v", err)
			}
			entries, readErr := os.ReadDir(directory)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("failed download left output or staging files: %v, %v", entries, readErr)
			}
		})
	}
}

func TestPublishDownloadPreservesExistingOutput(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	staged, output := filepath.Join(directory, "staged.mp4"), filepath.Join(directory, "output.mp4")
	for path, content := range map[string]string{staged: "new muxed file", output: "existing file"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := publishDownload(staged, output); err == nil {
		t.Fatal("publishing should reject an existing output file")
	}
	contents, err := os.ReadFile(output)
	if err != nil || string(contents) != "existing file" {
		t.Fatalf("existing output was modified: %q, %v", contents, err)
	}
}
