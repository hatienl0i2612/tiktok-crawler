package search

import (
	"reflect"
	"testing"
)

func TestParsePageContext(t *testing.T) {
	body := []byte(`<html><script type="application/json" id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{"__DEFAULT_SCOPE__":{"webapp.app-context":{"language":"en","region":"VN","wid":"12345","webIdCreatedTime":"1700000000"}}}</script></html>`)
	context := parsePageContext(body)
	want := pageContext{
		Language:         "en",
		Region:           "VN",
		DeviceID:         "12345",
		WebIDCreatedTime: "1700000000",
	}
	if !reflect.DeepEqual(context, want) {
		t.Fatalf("context = %#v, want %#v", context, want)
	}
}

func TestParseSearchResponseNormalizesAndDeduplicatesLinks(t *testing.T) {
	body := []byte(`{
  "status_code": 0,
  "data": [
    {"type": 1, "item": {"id": "7670393762513718549", "author": {"uniqueId": "gokuedit777"}}},
    {"item_info": {"item_struct": {"id": 7673406565210230036, "author": "thanhgoat209"}}},
    {"item": {"id": "7670393762513718549", "author": {"uniqueId": "gokuedit777"}}}
  ]
}`)
	links, err := parseSearchResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://www.tiktok.com/@gokuedit777/video/7670393762513718549",
		"https://www.tiktok.com/@thanhgoat209/video/7673406565210230036",
	}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %#v, want %#v", links, want)
	}
}

func TestParseSearchResponseSupportsRecommendList(t *testing.T) {
	body := []byte(`{"itemList":[{"id":"7000000000000000001","authorMeta":{"name":"example"}}]}`)
	links, err := parseSearchResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://www.tiktok.com/@example/video/7000000000000000001"}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %#v, want %#v", links, want)
	}
}

func TestParseSearchResponseReportsTikTokStatus(t *testing.T) {
	_, err := parseSearchResponse([]byte(`{"status_code":10216,"status_msg":"verification required"}`))
	if err == nil {
		t.Fatal("parseSearchResponse succeeded, want an error")
	}
}

func TestParseSearchResponseRejectsMissingResultList(t *testing.T) {
	if _, err := parseSearchResponse([]byte(`{"statusCode":0}`)); err == nil {
		t.Fatal("parseSearchResponse succeeded, want an error")
	}
}

func TestParseVideoLinksSupportsEscapedAndRelativeURLs(t *testing.T) {
	body := []byte(`<a href="/@first/video/7000000000000000001"></a>{"url":"https:\u002F\u002Fwww.tiktok.com\u002F@second\u002Fvideo\u002F7000000000000000002"}<a href="/@first/video/7000000000000000001"></a>`)
	links := parseVideoLinks(body)
	want := []string{
		"https://www.tiktok.com/@first/video/7000000000000000001",
		"https://www.tiktok.com/@second/video/7000000000000000002",
	}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %#v, want %#v", links, want)
	}
}
