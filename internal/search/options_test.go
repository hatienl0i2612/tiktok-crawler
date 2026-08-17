package search

import "testing"

func TestNormalizeOptionsUsesDefaultPageSize(t *testing.T) {
	options, locale, err := normalizeOptions(Options{Keyword: "  golang  ", Locale: " vi_VN "})
	if err != nil {
		t.Fatal(err)
	}
	if options.Keyword != "golang" {
		t.Fatalf("keyword = %q, want golang", options.Keyword)
	}
	if options.PageSize != DefaultPageSize {
		t.Fatalf("page size = %d, want %d", options.PageSize, DefaultPageSize)
	}
	if locale.language != "vi" || locale.region != "VN" || locale.browserLanguage != "vi-VN" {
		t.Fatalf("unexpected locale: %#v", locale)
	}
}

func TestParseLocaleAcceptsRegion(t *testing.T) {
	locale, err := parseLocale("us")
	if err != nil {
		t.Fatal(err)
	}
	if locale.language != "" || locale.region != "US" || locale.browserLanguage != "en-US" {
		t.Fatalf("unexpected locale: %#v", locale)
	}
}

func TestNormalizeOptionsRejectsInvalidPaginationAndLocale(t *testing.T) {
	tests := []Options{
		{PageSize: MaxPageSize + 1},
		{PageSize: 1, PageIndex: -1},
		{PageSize: 1, Locale: "Vietnam"},
		{PageSize: 1, Locale: "vi-419"},
	}
	for _, test := range tests {
		if _, _, err := normalizeOptions(test); err == nil {
			t.Fatalf("normalizeOptions(%#v) succeeded, want an error", test)
		}
	}
}
