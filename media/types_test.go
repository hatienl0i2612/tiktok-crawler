package media

import "testing"

func TestHelpers(t *testing.T) {
	t.Parallel()
	if got := QualityName(1080); got != "1080p" {
		t.Fatalf("QualityName() = %q", got)
	}
	if got := UniqueStrings([]string{" one ", "", "one", "two"}); len(got) != 2 || got[1] != "two" {
		t.Fatalf("UniqueStrings() = %#v", got)
	}
	variants := []Variant{
		{Codec: "h264", Width: 540, Height: 960, Bitrate: 900},
		{Codec: "h265", Width: 1080, Height: 1920, Bitrate: 1_200},
	}
	Sort(variants)
	if variants[0].Codec != "h265" {
		t.Fatalf("Sort() = %#v", variants)
	}
}
