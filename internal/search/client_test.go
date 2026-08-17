package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSearchUsesKeywordPaginationLocaleAndSessionContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/search/video":
			if got := request.URL.Query().Get("q"); got != "go tutorial" {
				t.Errorf("page keyword = %q, want go tutorial", got)
			}
			fmt.Fprint(writer, `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{"__DEFAULT_SCOPE__":{"webapp.app-context":{"language":"en","region":"VN","wid":"device-123","webIdCreatedTime":"1700000000"}}}</script>`)
		case "/api/search/item/full/":
			query := request.URL.Query()
			assertQueryValue(t, query.Get("keyword"), "go tutorial", "keyword")
			assertQueryValue(t, query.Get("count"), "5", "count")
			assertQueryValue(t, query.Get("cursor"), "10", "cursor")
			assertQueryValue(t, query.Get("offset"), "10", "offset")
			assertQueryValue(t, query.Get("region"), "US", "region")
			assertQueryValue(t, query.Get("browser_language"), "en-US", "browser_language")
			assertQueryValue(t, query.Get("device_id"), "device-123", "device_id")
			assertQueryValue(t, query.Get("WebIdLastTime"), "1700000000", "WebIdLastTime")
			fmt.Fprint(writer, `{"data":[{"item":{"id":"7000000000000000001","author":{"uniqueId":"example"}}}]}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := testClient(t, server)
	links, err := client.Search(context.Background(), Options{
		Keyword:   "go tutorial",
		Locale:    "en-US",
		PageSize:  5,
		PageIndex: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://www.tiktok.com/@example/video/7000000000000000001"}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %#v, want %#v", links, want)
	}
}

func TestSearchWithoutKeywordUsesRecommendedList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/foryou":
			fmt.Fprint(writer, `<html></html>`)
		case "/api/recommend/item_list/":
			assertQueryValue(t, request.URL.Query().Get("from_page"), "fyp", "from_page")
			assertQueryValue(t, request.URL.Query().Get("cursor"), "12", "cursor")
			fmt.Fprint(writer, `{"itemList":[{"id":"7000000000000000002","author":"recommended"}]}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := testClient(t, server)
	links, err := client.Search(context.Background(), Options{PageSize: 12, PageIndex: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://www.tiktok.com/@recommended/video/7000000000000000002"}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %#v, want %#v", links, want)
	}
}

func TestSearchFallsBackToHydratedHTMLLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/search/video":
			fmt.Fprint(writer, `<html></html>`)
		case "/api/search/item/full/":
			return
		case "/discover/fallback":
			if got := request.Header.Get("User-Agent"); got != discoveryUserAgent {
				t.Errorf("discovery User-Agent = %q, want mobile User-Agent", got)
			}
			fmt.Fprint(writer, `<a href="/@fallback/video/7000000000000000003">video</a>`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := testClient(t, server)
	links, err := client.Search(context.Background(), Options{Keyword: "fallback", PageSize: 12})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://www.tiktok.com/@fallback/video/7000000000000000003"}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %#v, want %#v", links, want)
	}
}

func testClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	client.baseURL = server.URL
	client.httpClient = server.Client()
	return client
}

func assertQueryValue(t *testing.T, got, want, name string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}
