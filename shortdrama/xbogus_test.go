package shortdrama

import "testing"

func TestGenerateXBogus(t *testing.T) {
	got := generateXBogus(
		"aid=1988&itemId=7665074054730091796",
		"Mozilla/5.0",
		1_700_000_000,
	)
	const want = "DFSzswVOvWTANeXOtmWx-GlUrnMj"
	if got != want {
		t.Fatalf("generateXBogus() = %q, want %q", got, want)
	}
}
