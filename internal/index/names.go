package index

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
)

var (
	camelBoundaryRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	nonSlugRe       = regexp.MustCompile(`[^a-z0-9]+`)
)

// NormalizeName turns any user spelling of a package or component name into
// the package short-name form: "Button", "@leafygreen-ui/button",
// "leafygreen-ui/button" and " button " all become "button";
// "ConfirmationModal" and "confirmation modal" become "confirmation-modal".
func NormalizeName(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}
	s = camelBoundaryRe.ReplaceAllString(s, "$1-$2")
	s = strings.ToLower(s)
	return strings.Trim(nonSlugRe.ReplaceAllString(s, "-"), "-")
}

// buildLookups indexes packages by full name, short name and exported
// interface name. Short-name collisions resolve to @leafygreen-ui.
func (ix *Index) buildLookups() {
	ix.byName = map[string]*Package{}
	ix.byDecl = map[string]*Package{}
	for _, p := range ix.preferredOrder() {
		ix.byName[strings.ToLower(p.Name)] = p
		ix.byName[p.Scope+"/"+p.Short] = p
		if _, ok := ix.byName[p.Short]; !ok {
			ix.byName[p.Short] = p
		}
		for _, d := range p.Decls {
			if d.Kind != "interface" {
				continue
			}
			for _, k := range []string{NormalizeName(d.Name), NormalizeName(strings.TrimSuffix(d.Name, "Props"))} {
				if _, ok := ix.byDecl[k]; !ok {
					ix.byDecl[k] = p
				}
			}
		}
	}
}

// Lookup resolves a lenient package or component name. Exported component
// names such as "ComboboxOption" resolve to the package declaring
// ComboboxOptionProps.
func (ix *Index) Lookup(name string) (*Package, bool) {
	raw := strings.ToLower(strings.TrimSpace(name))
	if p, ok := ix.byName[raw]; ok {
		return p, true
	}
	if p, ok := ix.byName[strings.TrimPrefix(raw, "@")]; ok {
		return p, true
	}
	n := NormalizeName(name)
	if p, ok := ix.byName[n]; ok {
		return p, true
	}
	p, ok := ix.byDecl[n]
	return p, ok
}

// Suggest returns up to limit package names closest to name, for "not
// found" replies.
func (ix *Index) Suggest(name string, limit int) []string {
	n := NormalizeName(name)
	type scored struct {
		name string
		d    int
	}
	var all []scored
	for _, p := range ix.Packages {
		d := levenshtein(n, p.Short)
		if n != "" && (strings.Contains(p.Short, n) || strings.Contains(n, p.Short)) {
			d = min(d, 1)
		}
		all = append(all, scored{p.Name, d})
	}
	slices.SortStableFunc(all, func(a, b scored) int { return cmp.Compare(a.d, b.d) })
	var out []string
	for _, s := range all {
		if len(out) == limit || s.d > max(3, len(n)/2) {
			break
		}
		out = append(out, s.name)
	}
	return out
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
