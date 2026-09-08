package jsonvalue

import (
	"encoding/json"
	"testing"
)

func TestInt64UnmarshalJSON(t *testing.T) {
	t.Parallel()

	for _, input := range []string{`123`, `"123"`} {
		var value Int64
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}
		if value != 123 {
			t.Fatalf("value = %d, want 123", value)
		}
	}
	var nullValue Int64 = 9
	if err := json.Unmarshal([]byte(`null`), &nullValue); err != nil || nullValue != 0 {
		t.Fatalf("unmarshal null = %d, %v", nullValue, err)
	}
	var invalid Int64
	if err := json.Unmarshal([]byte(`"abc"`), &invalid); err == nil {
		t.Fatal("invalid flexible integer unexpectedly succeeded")
	}
}
