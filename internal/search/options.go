package search

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type localeOptions struct {
	language        string
	region          string
	browserLanguage string
}

func normalizeOptions(options Options) (Options, localeOptions, error) {
	options.Keyword = strings.TrimSpace(options.Keyword)
	options.Locale = strings.TrimSpace(options.Locale)
	if options.PageSize == 0 {
		options.PageSize = DefaultPageSize
	}
	if options.PageSize < 1 || options.PageSize > MaxPageSize {
		return Options{}, localeOptions{}, fmt.Errorf("page size must be between 1 and %d", MaxPageSize)
	}
	if options.PageIndex < 0 {
		return Options{}, localeOptions{}, errors.New("page index must be zero or greater")
	}
	maxInt := int(^uint(0) >> 1)
	if options.PageIndex > maxInt/options.PageSize {
		return Options{}, localeOptions{}, errors.New("page index is too large")
	}
	locale, err := parseLocale(options.Locale)
	if err != nil {
		return Options{}, localeOptions{}, err
	}
	return options, locale, nil
}

func parseLocale(raw string) (localeOptions, error) {
	if raw == "" {
		return localeOptions{}, nil
	}
	normalized := strings.ReplaceAll(raw, "_", "-")
	parts := strings.Split(normalized, "-")
	for _, part := range parts {
		if part == "" || !isASCIIAlpha(part) {
			return localeOptions{}, errors.New("locale must be a region such as VN or a language-region tag such as vi-VN")
		}
	}

	if len(parts) == 1 && len(parts[0]) == 2 {
		region := strings.ToUpper(parts[0])
		return localeOptions{region: region, browserLanguage: "en-" + region}, nil
	}
	if len(parts) != 2 || len(parts[0]) < 2 || len(parts[0]) > 3 || len(parts[1]) != 2 {
		return localeOptions{}, errors.New("locale must be a region such as VN or a language-region tag such as vi-VN")
	}
	language := strings.ToLower(parts[0])
	region := strings.ToUpper(parts[1])
	return localeOptions{
		language:        language,
		region:          region,
		browserLanguage: language + "-" + region,
	}, nil
}

func isASCIIAlpha(value string) bool {
	for _, character := range value {
		if character > unicode.MaxASCII || character < 'A' || character > 'z' || (character > 'Z' && character < 'a') {
			return false
		}
	}
	return true
}

func resolvedBrowserLanguage(locale localeOptions, page pageContext) string {
	if locale.browserLanguage != "" {
		return locale.browserLanguage
	}
	if page.Language != "" && page.Region != "" {
		return strings.ToLower(page.Language) + "-" + strings.ToUpper(page.Region)
	}
	if page.Language != "" {
		return strings.ToLower(page.Language)
	}
	return "en-US"
}
