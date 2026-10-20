package index

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/brunodonatelli/lg-mcp/internal/design"
	"github.com/brunodonatelli/lg-mcp/internal/parse/a11ysrc"
	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
	"github.com/brunodonatelli/lg-mcp/internal/parse/tstypes"
	"github.com/brunodonatelli/lg-mcp/internal/source"
)

// Build reads all packages from src, attaches design pages from snap (which
// may be nil) and builds the search corpora. Per-package problems become
// warnings; only a source failure is an error.
func Build(ctx context.Context, src source.Source, snap *design.Snapshot) (*Index, error) {
	raws, err := src.Packages(ctx)
	if err != nil {
		return nil, fmt.Errorf("index %s: %w", src.Name(), err)
	}
	ix := &Index{SourceName: src.Name(), Foundations: map[string]Foundation{}}
	for _, raw := range raws {
		p, warns, err := buildPackage(raw)
		if err != nil {
			ix.Warnings = append(ix.Warnings, fmt.Sprintf("%s: %v", raw.Dir, err))
			continue
		}
		for _, w := range warns {
			ix.Warnings = append(ix.Warnings, p.Name+": "+w)
		}
		ix.Packages = append(ix.Packages, p)
	}
	slices.SortFunc(ix.Packages, func(a, b *Package) int { return strings.Compare(a.Name, b.Name) })
	ix.attachDesign(snap)
	ix.buildLookups()
	ix.buildCorpora()
	return ix, nil
}

type packageJSON struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description"`
	Deprecated       any               `json:"deprecated"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

func buildPackage(raw source.RawPackage) (*Package, []string, error) {
	var pj packageJSON
	if err := json.Unmarshal(raw.PackageJSON, &pj); err != nil {
		return nil, nil, fmt.Errorf("parse package.json: %w", err)
	}
	scope, short, ok := splitName(pj.Name)
	if !ok {
		return nil, nil, fmt.Errorf("package name %q is not in a LeafyGreen scope", pj.Name)
	}
	p := &Package{
		Name:        pj.Name,
		Scope:       scope,
		Short:       short,
		Version:     pj.Version,
		Description: pj.Description,
		Peers:       slices.Sorted(maps.Keys(pj.PeerDependencies)),
		HasReadme:   strings.TrimSpace(raw.Readme) != "",
		Readme:      markdown.Sections(raw.Readme),
	}
	if s, ok := pj.Deprecated.(string); ok {
		p.Deprecated = s
	} else if b, ok := pj.Deprecated.(bool); ok && b {
		p.Deprecated = "deprecated"
	}
	p.Decls = buildDecls(raw.TypeFiles)
	p.Examples = append(readmeExamples(p.Readme), storyExamples(raw.Stories)...)
	p.Imports = importsFor(p.Name, p.Readme)
	p.A11y = append(readmeA11y(p.Readme), propA11y(p.Decls)...)
	for _, f := range a11ysrc.Facts(raw.SourceFiles) {
		p.A11y = append(p.A11y, A11yNote{Kind: A11ySource, Source: f.File, Text: f.Text})
	}
	return p, raw.Warnings, nil
}

// splitName splits "@scope/short" for the indexed scopes.
func splitName(name string) (scope, short string, ok bool) {
	for _, s := range source.Scopes {
		if rest, found := strings.CutPrefix(name, s+"/"); found && rest != "" {
			return strings.TrimPrefix(s, "@"), rest, true
		}
	}
	return "", "", false
}

// buildDecls extracts declarations from type files in path order, keeping
// the first declaration of each name (compiled packages often ship
// duplicate .d.ts trees for older TypeScript versions).
func buildDecls(files map[string]string) []Decl {
	seen := map[string]bool{}
	var out []Decl
	for _, path := range slices.Sorted(maps.Keys(files)) {
		for _, d := range tstypes.Extract(files[path]) {
			key := string(d.Kind) + ":" + d.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Decl{Decl: d, File: path})
		}
	}
	return out
}

var codeLangs = map[string]bool{"tsx": true, "jsx": true, "ts": true, "js": true, "typescript": true, "javascript": true, "": true}

func readmeExamples(sections []markdown.Section) []Example {
	var out []Example
	for _, s := range sections {
		for _, c := range s.Code {
			if !codeLangs[strings.ToLower(c.Lang)] || strings.TrimSpace(c.Text) == "" || isInstallSnippet(c.Text) {
				continue
			}
			out = append(out, Example{Source: "README › " + s.Title(), Lang: cmp.Or(c.Lang, "tsx"), Code: c.Text})
		}
	}
	return out
}

func isInstallSnippet(code string) bool {
	t := strings.TrimSpace(code)
	for _, p := range []string{"npm ", "yarn ", "pnpm "} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

var storyExportRe = regexp.MustCompile(`(?m)^export const ([A-Z]\w*)\b`)

// storyExamples returns each named story export as an example, from its
// declaration up to the next top-level export.
func storyExamples(stories map[string]string) []Example {
	var out []Example
	for _, path := range slices.Sorted(maps.Keys(stories)) {
		src := stories[path]
		locs := storyExportRe.FindAllStringSubmatchIndex(src, -1)
		for i, loc := range locs {
			end := len(src)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			if next := strings.Index(src[loc[1]:end], "\nexport "); next >= 0 {
				end = loc[1] + next
			}
			out = append(out, Example{
				Source: path + " › " + src[loc[2]:loc[3]],
				Lang:   "tsx",
				Code:   strings.TrimSpace(src[loc[0]:end]),
			})
		}
	}
	return out
}

var importLineRe = regexp.MustCompile(`(?m)^import\s[^;]*?\sfrom\s+['"]([^'"]+)['"];?`)

// importsFor collects distinct import statements for pkg from README code.
func importsFor(pkg string, sections []markdown.Section) []string {
	var out []string
	for _, s := range sections {
		for _, c := range s.Code {
			for _, m := range importLineRe.FindAllStringSubmatch(c.Text, -1) {
				line := collapseSpace(m[0])
				if m[1] == pkg && !slices.Contains(out, line) {
					out = append(out, line)
				}
			}
		}
	}
	return out
}

var spaceRe = regexp.MustCompile(`\s+`)

func collapseSpace(s string) string { return spaceRe.ReplaceAllString(strings.TrimSpace(s), " ") }

func readmeA11y(sections []markdown.Section) []A11yNote {
	var out []A11yNote
	for _, s := range markdown.A11ySections(sections) {
		if text := markdown.Prose(s.Body); text != "" {
			out = append(out, A11yNote{Kind: A11yReadme, Source: "README › " + s.Title(), Text: text})
		}
	}
	return out
}

// eitherRe matches LeafyGreen's Either<Props, 'a' | 'b'> helper, which makes
// exactly one of the listed keys required.
var (
	eitherRe   = regexp.MustCompile(`Either<[\s\S]*,\s*((?:'[^']+'\s*\|?\s*)+)>`)
	quotedRe   = regexp.MustCompile(`'([^']+)'`)
	labelKeyRe = regexp.MustCompile(`(?i)^(aria-|label$)`)
)

// eitherA11y reports type-level accessible-name requirements such as
// Either<BaseProps, 'label' | 'aria-label'>.
func eitherA11y(d Decl) (A11yNote, bool) {
	if d.Kind != tstypes.TypeAlias {
		return A11yNote{}, false
	}
	m := eitherRe.FindStringSubmatch(d.Source)
	if m == nil {
		return A11yNote{}, false
	}
	var keys []string
	for _, q := range quotedRe.FindAllStringSubmatch(m[1], -1) {
		if !labelKeyRe.MatchString(q[1]) {
			return A11yNote{}, false
		}
		keys = append(keys, "`"+q[1]+"`")
	}
	if len(keys) < 2 {
		return A11yNote{}, false
	}
	return A11yNote{
		Kind:   A11yProp,
		Source: d.File,
		Text:   fmt.Sprintf("`%s` requires one of %s (enforced by the type via `Either`).", d.Name, strings.Join(keys, ", ")),
	}, true
}

func propA11y(decls []Decl) []A11yNote {
	var out []A11yNote
	for _, d := range decls {
		if n, ok := eitherA11y(d); ok {
			out = append(out, n)
		}
		for _, m := range tstypes.AriaProps(d.Decl) {
			text := "`" + m.Name + "` (" + d.Name + ")"
			if m.JSDoc != "" {
				text += ": " + collapseSpace(m.JSDoc)
			}
			out = append(out, A11yNote{Kind: A11yProp, Source: d.File, Text: text})
		}
	}
	return out
}

// attachDesign links component design pages to packages by short name,
// preferring @leafygreen-ui when several scopes share a name, and records
// foundation pages.
func (ix *Index) attachDesign(snap *design.Snapshot) {
	if snap.Empty() {
		return
	}
	ix.DesignFetchedAt = snap.FetchedAt
	ix.DesignDir = snap.Dir
	for _, p := range ix.preferredOrder() {
		doc, ok := snap.Components[p.Short]
		if !ok || claimed(ix.Packages, p.Short) {
			continue
		}
		p.Design = &DesignDoc{URL: doc.URL, Sections: markdown.Sections(doc.Markdown)}
		var notes []A11yNote
		for _, s := range markdown.A11ySections(p.Design.Sections) {
			if text := markdown.Prose(s.Body); text != "" {
				notes = append(notes, A11yNote{Kind: A11yDesign, Source: "Design guidelines › " + s.Title(), Text: text})
			}
		}
		p.A11y = append(notes, p.A11y...)
	}
	for slug, doc := range snap.Foundations {
		ix.Foundations[slug] = Foundation{Slug: slug, URL: doc.URL, Sections: markdown.Sections(doc.Markdown)}
	}
}

// claimed reports whether a package with this short name already has a
// design page attached.
func claimed(pkgs []*Package, short string) bool {
	for _, p := range pkgs {
		if p.Short == short && p.Design != nil {
			return true
		}
	}
	return false
}

// preferredOrder returns packages with @leafygreen-ui first, then by name.
func (ix *Index) preferredOrder() []*Package {
	out := slices.Clone(ix.Packages)
	slices.SortStableFunc(out, func(a, b *Package) int {
		return cmp.Compare(scopeRank(a.Scope), scopeRank(b.Scope))
	})
	return out
}

func scopeRank(scope string) int {
	if scope == "leafygreen-ui" {
		return 0
	}
	return 1
}
