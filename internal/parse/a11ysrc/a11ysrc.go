// Package a11ysrc finds accessibility facts in component source files:
// runtime validators, a11y utility usage, roles and aria-* attributes the
// component sets itself. It is a text-level pass and reports only what the
// source literally contains.
package a11ysrc

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Fact is one accessibility observation with the file it came from.
type Fact struct {
	File string
	Text string
}

type rule struct {
	re   *regexp.Regexp
	text string
}

var rules = []rule{
	{regexp.MustCompile(`\bvalidateAriaLabelProps\(`), "Requires `aria-label` or `aria-labelledby`; validated at runtime by `validateAriaLabelProps`."},
	{regexp.MustCompile(`\buseAccessibleFormField\(`), "Links its label and input with `useAccessibleFormField`."},
	{regexp.MustCompile(`\bprefersReducedMotion\b`), "Reduces motion when the user prefers reduced motion (`prefersReducedMotion`)."},
	{regexp.MustCompile(`<VisuallyHidden\b`), "Renders screen-reader-only text with `VisuallyHidden`."},
	{regexp.MustCompile(`\buseIdAllocator\(`), "Generates stable element ids with `useIdAllocator` (used for ARIA relationships)."},
	{regexp.MustCompile(`\bfocus-trap-react\b|<FocusTrap\b`), "Traps focus while open (`FocusTrap`)."},
	{regexp.MustCompile(`\buseBackdropClick\(|\buseEscapeKey\(`), "Closes on Escape or backdrop click."},
}

var (
	roleRe = regexp.MustCompile(`\brole=(?:"([a-z]+)"|\{['"]([a-z]+)['"]\})`)
	ariaRe = regexp.MustCompile(`\b(aria-[a-z]+)=`)
)

// Facts scans files (path → content). Behaviour facts (validators, focus
// traps, utilities) come first, deduplicated and sorted by text; then one
// line per file summarising the roles and aria-* attributes it sets.
func Facts(files map[string]string) []Fact {
	seen := map[string]bool{}
	var behaviour, attrs []Fact
	for _, path := range slices.Sorted(maps.Keys(files)) {
		src := files[path]
		for _, r := range rules {
			if r.re.MatchString(src) && !seen[r.text] {
				seen[r.text] = true
				behaviour = append(behaviour, Fact{File: path, Text: r.text})
			}
		}
		if line := attrLine(src); line != "" {
			attrs = append(attrs, Fact{File: path, Text: line})
		}
	}
	slices.SortStableFunc(behaviour, func(a, b Fact) int { return strings.Compare(a.Text, b.Text) })
	return append(behaviour, attrs...)
}

// attrLine summarises the roles and aria-* attributes set in one file, e.g.
// "Role `listbox`; sets `aria-expanded`, `aria-labelledby`.".
func attrLine(src string) string {
	var parts []string
	if roles := uniqueMatches(roleRe, src); len(roles) > 0 {
		parts = append(parts, fmt.Sprintf("%s %s", plural(len(roles), "Role", "Roles"), codeList(roles)))
	}
	if attrs := uniqueMatches(ariaRe, src); len(attrs) > 0 {
		verb := "sets"
		if len(parts) == 0 {
			verb = "Sets"
		}
		parts = append(parts, verb+" "+codeList(attrs))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "; ") + "."
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func uniqueMatches(re *regexp.Regexp, src string) []string {
	set := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		for _, g := range m[1:] {
			if g != "" {
				set[g] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(set))
}

func codeList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "`" + s + "`"
	}
	return strings.Join(q, ", ")
}
