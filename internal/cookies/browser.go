// Package cookies imports valued cookies from a local browser store and renders
// them as the Cookie header value the TikTok crawlers send on every request.
package cookies

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Code-Hex/browsercookie"
)

// tiktokDomains filters the browser store down to cookies belonging to TikTok
// page hosts. Browsercookie matches the domain exactly or by subdomain suffix,
// so both tiktok.com and www.tiktok.com are kept.
var tiktokDomains = []string{"tiktok.com"}

// SupportedBrowsers lists every browser name accepted by
// LoadTikTokCookieHeader; it is surfaced in error messages and help text.
const SupportedBrowsers = "brave, chrome, chromium, vivaldi, edge, edge-dev, arc, opera, opera-gx, firefox, librewolf, zen, safari"

// LoadTikTokCookieHeader reads TikTok cookies from the named browser store and
// renders them as a Cookie header value. It returns an error when the browser
// is not supported, its store cannot be read, or it holds no TikTok cookies.
func LoadTikTokCookieHeader(browser string) (string, error) {
	loader := loaderFor(browser)
	if loader == nil {
		return "", fmt.Errorf("unsupported browser %q; supported browsers: %s", browser, SupportedBrowsers)
	}
	cookies, err := loader(browsercookie.WithDomains(tiktokDomains...))
	if err != nil {
		return "", fmt.Errorf("read cookies from the %s browser: %w", browser, err)
	}
	header := FormatCookieHeader(cookies)
	if header == "" {
		return "", fmt.Errorf("no TikTok cookies found in the %s browser", browser)
	}
	return header, nil
}

// browserLoader loads every cookie stored by one browser.
type browserLoader func(...browsercookie.Option) ([]*http.Cookie, error)

// loaderFor maps a user-supplied browser name to the matching browsercookie
// loader, or nil when the name is not supported.
func loaderFor(name string) browserLoader {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "brave":
		return browsercookie.Brave
	case "chrome":
		return browsercookie.Chrome
	case "chromium":
		return browsercookie.Chromium
	case "vivaldi":
		return browsercookie.Vivaldi
	case "edge":
		return browsercookie.Edge
	case "edge-dev", "edgedev", "edge dev":
		return browsercookie.EdgeDev
	case "arc":
		return browsercookie.Arc
	case "opera":
		return browsercookie.Opera
	case "opera-gx", "operagx", "opera gx":
		return browsercookie.OperaGX
	case "firefox":
		return browsercookie.Firefox
	case "librewolf":
		return browsercookie.LibreWolf
	case "zen":
		return browsercookie.Zen
	case "safari":
		return browsercookie.Safari
	}
	return nil
}

// FormatCookieHeader renders cookies as a Cookie header value in the order the
// loaders returned them, skipping cookies without a name.
func FormatCookieHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name == "" {
			continue
		}
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}
