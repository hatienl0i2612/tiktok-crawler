// Package portable downloads verified executable assets from GitHub releases.
package portable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hatienl0i2612/tiktok-crawler/tiktok"
)

const maxAssetSize = 256 << 20

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

func FetchRelease(ctx context.Context, client *http.Client, endpoint string) (Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, err
	}
	setHeaders(request)
	response, err := client.Do(request)
	if err != nil {
		return Release{}, fmt.Errorf("fetch release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("fetch release: HTTP %s", response.Status)
	}
	body, err := tiktok.ReadLimited(response.Body, 2<<20)
	if err != nil {
		return Release{}, err
	}
	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return Release{}, fmt.Errorf("decode release: %w", err)
	}
	if release.TagName == "" || len(release.Assets) == 0 {
		return Release{}, errors.New("release contains no downloadable assets")
	}
	return release, nil
}

func ValidateAsset(asset Asset) error {
	if asset.Size <= 0 || asset.Size > maxAssetSize {
		return fmt.Errorf("asset %s has invalid size %d", asset.Name, asset.Size)
	}
	_, err := expectedSHA256(asset.Digest)
	return err
}

func DownloadAsset(ctx context.Context, client *http.Client, asset Asset, destination string, enforceGitHub bool) error {
	if err := ValidateAsset(asset); err != nil {
		return err
	}
	parsed, err := url.Parse(asset.URL)
	if enforceGitHub && (err != nil || parsed.Scheme != "https" || !allowedHost(parsed.Hostname())) {
		return errors.New("asset has an invalid or untrusted GitHub HTTPS URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	setHeaders(request)
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download asset: HTTP %s", response.Status)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create asset: %w", err)
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(destination)
		}
	}()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, asset.Size+1))
	if err != nil {
		return fmt.Errorf("write asset: %w", err)
	}
	if written != asset.Size {
		return fmt.Errorf("asset size mismatch: received %d bytes, expected %d", written, asset.Size)
	}
	expected, _ := expectedSHA256(asset.Digest)
	if !bytes.Equal(hash.Sum(nil), expected) {
		return fmt.Errorf("asset SHA-256 mismatch: got %s", hex.EncodeToString(hash.Sum(nil)))
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync asset: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close asset: %w", err)
	}
	keep = true
	return nil
}

func expectedSHA256(digest string) ([]byte, error) {
	algorithm, value, ok := strings.Cut(strings.TrimSpace(digest), ":")
	if !ok || !strings.EqualFold(algorithm, "sha256") {
		return nil, errors.New("release metadata has no SHA-256 digest")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return nil, errors.New("release metadata has an invalid SHA-256 digest")
	}
	return decoded, nil
}

func HTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Minute,
		CheckRedirect: func(request *http.Request, previous []*http.Request) error {
			if len(previous) >= 10 {
				return errors.New("too many redirects while downloading executable")
			}
			if request.URL.Scheme != "https" || !allowedHost(request.URL.Hostname()) {
				return fmt.Errorf("executable download redirected to untrusted URL %s", request.URL.Redacted())
			}
			return nil
		},
	}
}

func allowedHost(host string) bool {
	return tiktok.DomainMatches(host, "github.com") || tiktok.DomainMatches(host, "githubusercontent.com")
}

func setHeaders(request *http.Request) {
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "tiktok-crawler-bootstrap")
}
