package cliargs

import (
	"reflect"
	"testing"
)

func TestReorderInterspersedFlags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		value []string
		want  []string
	}{
		{
			name:  "flags after positional",
			args:  []string{"video-url", "-quality", "720p", "-watermark", "-output=video.mp4"},
			value: []string{"output", "quality"},
			want:  []string{"-quality", "720p", "-watermark", "-output=video.mp4", "video-url"},
		},
		{
			name:  "long flags with values",
			args:  []string{"live-url", "--json", "--timeout", "30s", "--user-agent=test-agent"},
			value: []string{"timeout", "user-agent"},
			want:  []string{"--json", "--timeout", "30s", "--user-agent=test-agent", "live-url"},
		},
		{
			name:  "double dash preserves remaining arguments",
			args:  []string{"-json", "--", "video-url", "-not-a-flag"},
			value: nil,
			want:  []string{"-json", "video-url", "-not-a-flag"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ReorderInterspersedFlags(test.args, test.value...)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("arguments = %#v, want %#v", got, test.want)
			}
		})
	}
}
