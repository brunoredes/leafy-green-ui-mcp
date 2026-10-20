package output

import (
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		max        int
		wantSame   bool
		wantMarker string
	}{
		{name: "under budget", in: "hello", max: 10, wantSame: true},
		{name: "disabled", in: strings.Repeat("x", 50), max: 0, wantSame: true},
		{name: "cuts plain", in: strings.Repeat("x", 50), max: 10, wantMarker: "[truncated: 40 more characters"},
		{name: "cuts at newline", in: "line one\nline two\nline three", max: 15, wantMarker: "[truncated: 20 more characters"},
		{name: "multibyte", in: strings.Repeat("é", 20), max: 5, wantMarker: "[truncated: 15 more characters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.in, tt.max)
			if tt.wantSame {
				if got != tt.in {
					t.Fatalf("Truncate changed input: %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantMarker) {
				t.Fatalf("Truncate = %q, want marker %q", got, tt.wantMarker)
			}
		})
	}
}

func TestClip(t *testing.T) {
	if got := Clip("short", 10); got != "short" {
		t.Fatalf("Clip = %q", got)
	}
	got := Clip("one two three four five six", 12)
	if !strings.HasSuffix(got, " …") || len([]rune(got)) > 14 {
		t.Fatalf("Clip = %q", got)
	}
}

func TestCodeBlock(t *testing.T) {
	got := CodeBlock("md", "a\n```\nb")
	if !strings.HasPrefix(got, "````md\n") || !strings.HasSuffix(got, "````\n") {
		t.Fatalf("CodeBlock did not widen fence: %q", got)
	}
}
