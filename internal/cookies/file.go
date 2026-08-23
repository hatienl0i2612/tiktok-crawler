// Package cookies loads cookie values from browser stores and cookie files and
// renders them as the Cookie header value the TikTok crawlers send.
package cookies

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadCookieFileHeader reads cookies from path and returns them as a Cookie
// header value. Two text formats are accepted:
//
//   - a Netscape cookie-jar export (the tab-separated file produced by browsers
//     and tools such as curl -c), and
//   - a raw Cookie header value such as "ttwid=...; sessionid=...".
func LoadCookieFileHeader(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read cookie file %q: %w", path, err)
	}
	header := CookieHeaderFromText(data)
	if header == "" {
		return "", fmt.Errorf("no cookies found in cookie file %q", path)
	}
	return header, nil
}

// CookieHeaderFromText converts cookie-file content into a Cookie header value.
func CookieHeaderFromText(text []byte) string {
	if strings.Contains(string(text), "\t") {
		return FormatCookieHeader(parseNetscapeCookieFile(text))
	}
	return normalizeRawCookieHeader(string(text))
}

// parseNetscapeCookieFile parses the tab-separated Netscape cookie-jar format:
//
//	domain\tincludeSubdomains\tpath\tsecure\texpiry\tname\tvalue
func parseNetscapeCookieFile(text []byte) []*http.Cookie {
	var cookies []*http.Cookie
	for _, rawLine := range strings.Split(string(text), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		// #HttpOnly_ prefixes a cookie domain, while every other # line is a
		// comment in the Netscape format.
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#HttpOnly_") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}
		domain := strings.TrimSpace(fields[0])
		httpOnly := strings.HasPrefix(domain, "#HttpOnly_")
		if httpOnly {
			domain = strings.TrimPrefix(domain, "#HttpOnly_")
		}
		cookie := &http.Cookie{
			Domain:   domain,
			Path:     strings.TrimSpace(fields[2]),
			Secure:   strings.EqualFold(strings.TrimSpace(fields[3]), "TRUE"),
			HttpOnly: httpOnly,
			Name:     strings.TrimSpace(fields[5]),
			Value:    strings.Join(fields[6:], "\t"),
		}
		if expiry, parseErr := strconv.ParseInt(strings.TrimSpace(fields[4]), 10, 64); parseErr == nil && expiry > 0 {
			cookie.Expires = time.Unix(expiry, 0).UTC()
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

// normalizeRawCookieHeader turns a raw cookies.txt value into a single-line
// Cookie header: it drops comment and blank lines, trims whitespace, removes a
// leading "Cookie:" prefix, and collapses line breaks and repeated spaces.
func normalizeRawCookieHeader(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	text = strings.Join(lines, " ")
	if lowered := strings.ToLower(text); strings.HasPrefix(lowered, "cookie:") {
		text = strings.TrimSpace(text[len("cookie:"):])
	}
	return strings.Join(strings.Fields(text), " ")
}
