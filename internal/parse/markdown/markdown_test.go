package markdown

import (
	"strings"
	"testing"
)

const doc = "# Modal\n\nintro text\n\n## Example\n\n```tsx\nimport Modal from '@leafygreen-ui/modal';\n# not a heading\n```\n\n## Additional notes\n\n### Initial focus behavior\n\nFocus moves to the first `autoFocus` child.\n\n### Backdrop styling\n\nUse ::backdrop.\n\n## [Linked](http://x) heading ##\n"

func TestSections(t *testing.T) {
	got := Sections(doc)
	want := []struct {
		level int
		title string
	}{
		{1, "Modal"},
		{2, "Example"},
		{2, "Additional notes"},
		{3, "Additional notes > Initial focus behavior"},
		{3, "Additional notes > Backdrop styling"},
		{2, "Linked heading"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sections: %+v", len(got), got)
	}
	for i, w := range want {
		t.Run(w.title, func(t *testing.T) {
			if got[i].Level != w.level || got[i].Title() != w.title {
				t.Fatalf("section %d = level %d %q, want level %d %q", i, got[i].Level, got[i].Title(), w.level, w.title)
			}
		})
	}
	if len(got[1].Code) != 1 || got[1].Code[0].Lang != "tsx" || !strings.Contains(got[1].Code[0].Text, "# not a heading") {
		t.Fatalf("example code = %+v", got[1].Code)
	}
}

func TestSectionsPreamble(t *testing.T) {
	got := Sections("before\n\n## A\nbody")
	if len(got) != 2 || got[0].Level != 0 || got[0].Body != "before" {
		t.Fatalf("got %+v", got)
	}
}

func TestCodeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		langs []string
	}{
		{name: "two blocks", in: "```js\na\n```\ntext\n~~~shell\nb\n~~~", langs: []string{"js", "shell"}},
		{name: "info string attrs", in: "```tsx {1,3}\na\n```", langs: []string{"tsx"}},
		{name: "longer fence wraps shorter", in: "````md\n```js\nx\n```\n````", langs: []string{"md"}},
		{name: "unterminated", in: "```js\na", langs: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CodeBlocks(tt.in)
			if len(got) != len(tt.langs) {
				t.Fatalf("got %+v", got)
			}
			for i, l := range tt.langs {
				if got[i].Lang != l {
					t.Fatalf("block %d lang = %q, want %q", i, got[i].Lang, l)
				}
			}
		})
	}
}

func TestProse(t *testing.T) {
	in := "![badge](x.svg) Hello <b>world</b>\n\n| a | b |\n| --- | --- |\n| 1 | 2 |\n\n```js\ncode\n```\n\n\n\nend"
	got := Prose(in)
	for _, bad := range []string{"badge", "<b>", "---", "code"} {
		if strings.Contains(got, bad) {
			t.Errorf("Prose kept %q: %q", bad, got)
		}
	}
	if !strings.Contains(got, "Hello world") || !strings.Contains(got, "| 1 | 2 |") || strings.Contains(got, "\n\n\n") {
		t.Errorf("Prose = %q", got)
	}
}

func TestA11ySections(t *testing.T) {
	secs := Sections("# X\n## Keyboard interaction\na\n## Props\nb\n### Initial focus behavior\nc\n## Accessibility\nd\n## Variants\ne")
	var got []string
	for _, s := range A11ySections(secs) {
		got = append(got, s.Heading)
	}
	want := "Keyboard interaction|Initial focus behavior|Accessibility"
	if strings.Join(got, "|") != want {
		t.Fatalf("A11ySections = %v, want %s", got, want)
	}
}
