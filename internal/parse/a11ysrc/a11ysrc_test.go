package a11ysrc

import (
	"strings"
	"testing"
)

func TestFacts(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{name: "none", files: map[string]string{"a.tsx": "export const A = () => <div />;"}},
		{
			name:  "validator",
			files: map[string]string{"src/C.tsx": "validateAriaLabelProps(props, 'C');"},
			want:  []string{"Requires `aria-label` or `aria-labelledby`; validated at runtime by `validateAriaLabelProps`."},
		},
		{
			name: "one attribute line per file, after behaviour facts",
			files: map[string]string{
				"b.tsx": `<li role={'option'} aria-selected={s}>`,
				"a.tsx": `validateAriaLabelProps(p); <ul role="listbox" aria-multiselectable={m} aria-label={l}>`,
				"c.tsx": `<span aria-hidden="true" />`,
			},
			want: []string{
				"Requires `aria-label` or `aria-labelledby`; validated at runtime by `validateAriaLabelProps`.",
				"Role `listbox`; sets `aria-label`, `aria-multiselectable`.",
				"Role `option`; sets `aria-selected`.",
				"Sets `aria-hidden`.",
			},
		},
		{
			name:  "utilities",
			files: map[string]string{"x.tsx": "import { VisuallyHidden, prefersReducedMotion } from '@leafygreen-ui/a11y';\n<VisuallyHidden>x</VisuallyHidden>"},
			want: []string{
				"Reduces motion when the user prefers reduced motion (`prefersReducedMotion`).",
				"Renders screen-reader-only text with `VisuallyHidden`.",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, f := range Facts(tt.files) {
				got = append(got, f.Text)
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Fatalf("Facts =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}
