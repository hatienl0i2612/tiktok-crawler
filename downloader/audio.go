package downloader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hatienl0i2612/tiktok-crawler/media"
	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

func downloadWithAudio(ctx context.Context, session *tiktok.Session, variant media.Variant, options downloadOptions, executable string) (*DownloadResult, error) {
	staging, err := os.MkdirTemp(filepath.Dir(options.OutputPath), ".tiktok-mux-*")
	if err != nil {
		return nil, fmt.Errorf("create video/audio staging directory: %w", err)
	}
	defer os.RemoveAll(staging)
	videoPath, audioPath := filepath.Join(staging, "video.mp4"), filepath.Join(staging, "audio.m4a")
	video, audio := variant, *variant.Audio
	video.Audio, audio.Audio = nil, nil
	videoDownload, err := downloadVariant(ctx, session, video, downloadOptions{
		OutputPath: videoPath, Referer: options.Referer,
		Progress: func(progress DownloadProgress) {
			if options.Progress != nil {
				options.Progress(DownloadProgress{DownloadedBytes: progress.DownloadedBytes, TotalBytes: progress.TotalBytes + audio.Size})
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("download video track: %w", err)
	}
	_, err = downloadVariant(ctx, session, audio, downloadOptions{
		OutputPath: audioPath, Referer: options.Referer,
		Progress: func(progress DownloadProgress) {
			if options.Progress != nil {
				options.Progress(DownloadProgress{DownloadedBytes: videoDownload.Bytes + progress.DownloadedBytes, TotalBytes: videoDownload.Bytes + progress.TotalBytes})
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("download audio track: %w", err)
	}
	if options.Status != nil {
		_, _ = fmt.Fprintln(options.Status, "\nMerging video and audio with FFmpeg")
	}
	mergedPath := filepath.Join(staging, "merged.mp4")
	command := exec.CommandContext(ctx, executable,
		"-hide_banner", "-loglevel", "error", "-nostdin", "-n",
		"-i", videoPath, "-i", audioPath,
		"-map", "0:v:0", "-map", "1:a:0", "-c", "copy",
		"-movflags", "+faststart", "-f", "mp4", mergedPath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("mux video and audio with FFmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(mergedPath)
	if err != nil {
		return nil, fmt.Errorf("inspect muxed video: %w", err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("FFmpeg produced an empty video")
	}
	if err := publishDownload(mergedPath, options.OutputPath); err != nil {
		return nil, err
	}
	return &DownloadResult{
		Path: options.OutputPath, Bytes: info.Size(), ContentType: "video/mp4", SourceURL: videoDownload.SourceURL,
	}, nil
}
