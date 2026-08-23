package cookies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRawCookieHeaderFromText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "single line",
			text: "ttwid=1%7Cv1; sessionid=abc123",
			want: "ttwid=1%7Cv1; sessionid=abc123",
		},
		{
			name: "cookie prefix",
			text: "  Cookie: ttwid=abc; msToken=def ",
			want: "ttwid=abc; msToken=def",
		},
		{
			name: "line wrapped",
			text: "ttwid=abc;\nmsToken=def;  sessionid=xyz",
			want: "ttwid=abc; msToken=def; sessionid=xyz",
		},
		{
			name: "empty",
			text: "   \n  ",
			want: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeRawCookieHeader(test.text)
			if got != test.want {
				t.Errorf("normalizeRawCookieHeader() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNetscapeCookieFileFromText(t *testing.T) {
	t.Parallel()
	text := "# Netscape HTTP Cookie File\n" +
		"# a generated comment\n" +
		".tiktok.com\tTRUE\t/\tFALSE\t1826095200\tttwid\t1%7Cv1\n" +
		"#HttpOnly_.tiktok.com\tTRUE\t/\tTRUE\t0\tsessionid\tabc123\n" +
		"not-a-cookie-line\n"
	got := CookieHeaderFromText([]byte(text))
	if want := "ttwid=1%7Cv1; sessionid=abc123"; got != want {
		t.Fatalf("CookieHeaderFromText() = %q, want %q", got, want)
	}
}

func TestEmptyCookieFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := CookieHeaderFromText([]byte("# only comments\n\n"))
	if got != "" {
		t.Fatalf("CookieHeaderFromText() = %q, want empty", got)
	}
}

func TestLoadCookieFileHeaderRaw(t *testing.T) {
	path := writeCookiesFile(t, "sessionid=abc; ttwid=xyz")
	header, err := LoadCookieFileHeader(path)
	if err != nil {
		t.Fatalf("LoadCookieFileHeader(): %v", err)
	}
	if want := "sessionid=abc; ttwid=xyz"; header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
}

func TestLoadCookieFileHeaderNetscape(t *testing.T) {
	content := "# Netscape HTTP Cookie File\n" +
		"#HttpOnly_.tiktok.com\tTRUE\t/\tTRUE\t0\tsessionid\tabc123\n"
	path := writeCookiesFile(t, content)
	header, err := LoadCookieFileHeader(path)
	if err != nil {
		t.Fatalf("LoadCookieFileHeader(): %v", err)
	}
	if want := "sessionid=abc123"; header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
}

func TestLoadCookieFileHeaderMissingFile(t *testing.T) {
	_, err := LoadCookieFileHeader("does-not-exist.txt")
	if err == nil {
		t.Fatal("LoadCookieFileHeader() succeeded for a missing file")
	}
}

func TestLoadCookieFileHeaderEmptyFile(t *testing.T) {
	path := writeCookiesFile(t, "")
	_, err := LoadCookieFileHeader(path)
	if err == nil {
		t.Fatal("LoadCookieFileHeader() succeeded for an empty file")
	}
	if !strings.Contains(err.Error(), "no cookies") {
		t.Fatalf("error %q should mention the missing cookies", err)
	}
}

func writeCookiesFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write cookies file: %v", err)
	}
	return path
}
