package search

const (
	// DefaultUserAgent matches the desktop browser profile used by TikTok's web search.
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
	DefaultPageSize  = 12
	MaxPageSize      = 30
)

// ClientOptions configures the TikTok search HTTP session.
type ClientOptions struct {
	Cookie    string
	UserAgent string
}

// Options selects one page of TikTok video search results.
type Options struct {
	Keyword   string
	Locale    string
	PageSize  int
	PageIndex int
}
