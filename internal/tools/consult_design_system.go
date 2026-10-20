package tools

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/output"
	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
	"github.com/brunodonatelli/lg-mcp/internal/parse/tstypes"
)

// ConsultDesignSystemInput is the input of consult_design_system.
type ConsultDesignSystemInput struct {
	Intent string `json:"intent" jsonschema:"what the UI should do, in plain words, e.g. let users pick several tags from a long list"`
	Limit  int    `json:"limit,omitempty" jsonschema:"number of candidate components to return (default 3, max 5)"`
	Scope  string `json:"scope,omitempty" jsonschema:"restrict to one npm scope: leafygreen-ui, lg-chat, lg-charts or all (default all)"`
}

var validScopes = []string{"", "all", "leafygreen-ui", "lg-chat", "lg-charts"}

// Share of each candidate's character budget per dimension.
const (
	shareGuidance = 0.25
	shareProps    = 0.20
	shareExample  = 0.30
	shareA11y     = 0.25
	maxKeyProps   = 8
	maxA11yNotes  = 6
	// minRelativeScore drops candidates far weaker than the best match, which
	// usually matched only incidental words.
	minRelativeScore = 0.25
)

// ConsultDesignSystem crosses design guidance, code docs, examples and
// accessibility for the components that best match a UI intent.
func (h *Handlers) ConsultDesignSystem(_ context.Context, _ *mcp.CallToolRequest, in ConsultDesignSystemInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Intent) == "" {
		return nil, nil, fmt.Errorf("intent is required: describe what the UI should do")
	}
	scope := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(in.Scope)), "@")
	if !slices.Contains(validScopes, scope) {
		return nil, nil, fmt.Errorf("scope %q: want one of leafygreen-ui, lg-chat, lg-charts, all", in.Scope)
	}
	hits := h.idx.Search(in.Intent, index.SearchOptions{Limit: clampLimit(in.Limit, 3, 5), Scope: scope, MinRelative: minRelativeScore})

	var b strings.Builder
	fmt.Fprintf(&b, "# LeafyGreen consult: %q\n\n%s\n%s\n\n", in.Intent, h.sourceNote(), h.designNote())
	if len(hits) == 0 {
		b.WriteString("No LeafyGreen package matched this intent. Matching is lexical: rephrase with the kind of " +
			"control you have in mind (e.g. \"select\", \"modal\", \"table\", \"banner\"), or use search_docs.\n")
		return h.result(b.String()), nil, nil
	}
	names := make([]string, len(hits))
	for i, hit := range hits {
		names[i] = hit.Package.Name
	}
	fmt.Fprintf(&b, "Candidates, best first: %s\n\n", strings.Join(names, ", "))

	terms := index.Terms(in.Intent)
	budget := max(1200, (h.max-1800)/len(hits))
	var gaps []string
	for i, hit := range hits {
		block, g := h.consultCandidate(i+1, hit, terms, budget)
		b.WriteString(block)
		gaps = append(gaps, g...)
	}
	h.writeCrossCutting(&b, in.Intent, hits)
	writeGaps(&b, h, gaps)
	writeNextSteps(&b, hits[0].Package)
	return h.result(b.String()), nil, nil
}

func (h *Handlers) consultCandidate(rank int, hit index.Hit, terms map[string]bool, budget int) (string, []string) {
	p := hit.Package
	var b strings.Builder
	var gaps []string
	fmt.Fprintf(&b, "## %d. %s %s\n\n", rank, p.Name, p.Version)
	if p.Deprecated != "" {
		fmt.Fprintf(&b, "**Deprecated:** %s\n\n", p.Deprecated)
	}
	if p.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Description)
	}
	fmt.Fprintf(&b, "Why it matched: %s\n", explainMatch(hit.Matched))
	writeImports(&b, p)
	if len(p.Peers) > 0 {
		fmt.Fprintf(&b, "Peers: %s\n", strings.Join(p.Peers, ", "))
	}
	b.WriteString("\n")

	b.WriteString("### Design guidance\n\n")
	switch {
	case !h.idx.HasDesign():
		b.WriteString("_Design snapshot not loaded._\n\n")
	case p.Design == nil:
		b.WriteString("_No design guidelines page in the snapshot._\n\n")
		gaps = append(gaps, fmt.Sprintf("No mongodb.design page for %s in the snapshot.", p.Name))
	default:
		b.WriteString(designExcerpt(p.Design.Sections, terms, int(float64(budget)*shareGuidance)))
		fmt.Fprintf(&b, "Source: %s\n\n", p.Design.URL)
	}

	b.WriteString(keyProps(p, terms, int(float64(budget)*shareProps)))
	if len(p.Decls) == 0 {
		gaps = append(gaps, fmt.Sprintf("No props declaration found for %s.", p.Name))
	}

	if ex, ok := bestExample(p.Examples, terms); ok {
		fmt.Fprintf(&b, "### Example (%s)\n\n", ex.Source)
		b.WriteString(output.CodeBlock(ex.Lang, clipCode(ex.Code, int(float64(budget)*shareExample))))
		b.WriteString("\n")
	} else {
		b.WriteString("### Example\n\n_No code example found._\n\n")
		gaps = append(gaps, fmt.Sprintf("No code example found for %s.", p.Name))
	}

	b.WriteString("### Accessibility\n\n")
	if len(p.A11y) == 0 {
		b.WriteString("_No accessibility notes found._\n\n")
		gaps = append(gaps, fmt.Sprintf("No accessibility notes found for %s; check the accessibility foundation guidance.", p.Name))
	} else {
		b.WriteString(a11ySummary(p.A11y, int(float64(budget)*shareA11y)))
	}
	return b.String(), gaps
}

func explainMatch(matches []index.TermMatch) string {
	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		parts = append(parts, fmt.Sprintf("`%s` (%s)", m.Term, strings.Join(m.Fields, ", ")))
	}
	return strings.Join(parts, "; ")
}

var guidanceRe = regexp.MustCompile(`(?i)overview|when to use|how to use|usage|best practice`)

// designExcerpt shows usage-oriented sections first, then the sections that
// share the most terms with the intent, within budget.
func designExcerpt(secs []markdown.Section, terms map[string]bool, budget int) string {
	type cand struct {
		sec   markdown.Section
		score int
	}
	var cands []cand
	for _, s := range secs {
		text := markdown.Prose(s.Body)
		// Accessibility sections are shown under Accessibility instead.
		if s.Heading == "" || text == "" || s.Level == 1 || markdown.IsA11yText(s.Heading) {
			continue
		}
		score := index.Overlap(terms, s.Heading+" "+text)
		if guidanceRe.MatchString(s.Heading) {
			score += 100
		}
		cands = append(cands, cand{s, score})
	}
	slices.SortStableFunc(cands, func(a, b cand) int { return cmp.Compare(b.score, a.score) })
	var b strings.Builder
	for _, c := range cands {
		if c.score == 0 || b.Len() >= budget {
			break
		}
		room := max(150, budget-b.Len())
		fmt.Fprintf(&b, "**%s:** %s\n\n", c.sec.Title(), output.Clip(markdown.Prose(c.sec.Body), room))
	}
	if b.Len() == 0 {
		return "_The design page has no usage text matching this intent; see get_design_guidelines._\n\n"
	}
	return b.String()
}

func keyProps(p *index.Package, terms map[string]bool, budget int) string {
	main, parts, ok := p.MainProps()
	if !ok {
		return "### Key props\n\n_No props declaration found._\n\n"
	}
	var b strings.Builder
	b.WriteString(keyPropsHeader(main))
	members := main.Members
	for _, d := range parts {
		members = append(slices.Clone(members), d.Members...)
	}
	if len(members) == 0 {
		b.WriteString("_No members declared in this package; see the declaration above._\n\n")
		return b.String()
	}
	shown := 0
	for _, m := range rankMembers(members, terms) {
		if shown == maxKeyProps || b.Len() >= budget {
			break
		}
		writeKeyProp(&b, m)
		shown++
	}
	if rest := len(members) - shown; rest > 0 {
		fmt.Fprintf(&b, "- … %d more → get_component_props\n", rest)
	}
	b.WriteString("\n")
	return b.String()
}

// keyPropsHeader names the main declaration; a composed type alias is shown
// verbatim because it carries constraints such as Either<..., 'label' | 'aria-label'>.
func keyPropsHeader(main index.Decl) string {
	if main.Kind == tstypes.TypeAlias {
		return fmt.Sprintf("### Key props\n\n`%s`\n\n", output.Clip(strings.Join(strings.Fields(main.Source), " "), 300))
	}
	h := fmt.Sprintf("### Key props (%s%s", main.Name, main.TypeParams)
	if main.Extends != "" {
		h += fmt.Sprintf(", extends `%s`", main.Extends)
	}
	return h + ")\n\n"
}

// rankMembers orders members by overlap with the intent, then required
// before optional; deprecated members always go last.
func rankMembers(members []tstypes.Member, terms map[string]bool) []tstypes.Member {
	rank := func(m tstypes.Member) int {
		if index.IsDeprecated(m) {
			return -1
		}
		r := 10 * index.Overlap(terms, m.Name+" "+m.JSDoc)
		if !m.Optional {
			r += 5
		}
		return r
	}
	out := slices.Clone(members)
	slices.SortStableFunc(out, func(a, b tstypes.Member) int { return cmp.Compare(rank(b), rank(a)) })
	return out
}

func writeKeyProp(b *strings.Builder, m tstypes.Member) {
	opt := ""
	if m.Optional {
		opt = "?"
	}
	fmt.Fprintf(b, "- `%s%s: %s`", m.Name, opt, output.Clip(m.Type, 80))
	if index.IsDeprecated(m) {
		b.WriteString(" **(deprecated)**")
	}
	if m.JSDoc != "" {
		fmt.Fprintf(b, " — %s", output.Clip(strings.Join(strings.Fields(m.JSDoc), " "), 140))
	}
	b.WriteString("\n")
}

// bestExample picks the example sharing the most terms with the intent.
// README examples are canonical usage, so they are preferred whenever one
// exists; Storybook stories (often full of test scaffolding) are the
// fallback. Ties keep source order.
func bestExample(examples []index.Example, terms map[string]bool) (index.Example, bool) {
	var readme []index.Example
	for _, e := range examples {
		if strings.HasPrefix(e.Source, "README") {
			readme = append(readme, e)
		}
	}
	if len(readme) > 0 {
		examples = readme
	}
	if len(examples) == 0 {
		return index.Example{}, false
	}
	best, bestScore := 0, -1
	for i, e := range examples {
		if s := index.Overlap(terms, e.Code); s > bestScore {
			best, bestScore = i, s
		}
	}
	return examples[best], true
}

// clipCode cuts code at a line boundary within limit bytes.
func clipCode(code string, limit int) string {
	if len(code) <= limit {
		return code
	}
	cut := code[:limit]
	if i := strings.LastIndexByte(cut, '\n'); i > 0 {
		cut = cut[:i]
	}
	return cut + "\n// … truncated; see get_component_examples"
}

func a11ySummary(notes []index.A11yNote, budget int) string {
	var b strings.Builder
	shown := 0
	for _, n := range notes {
		if shown == maxA11yNotes || b.Len() >= budget {
			break
		}
		fmt.Fprintf(&b, "- %s %s\n", a11yLabel(n), output.Clip(strings.Join(strings.Fields(n.Text), " "), 260))
		shown++
	}
	if rest := len(notes) - shown; rest > 0 {
		fmt.Fprintf(&b, "- … %d more → get_accessibility\n", rest)
	}
	b.WriteString("\n")
	return b.String()
}

func a11yLabel(n index.A11yNote) string {
	switch n.Kind {
	case index.A11yDesign, index.A11yReadme:
		return "[" + n.Source + "]"
	case index.A11yProp:
		return "[prop]"
	}
	return "[source]"
}

// writeCrossCutting adds foundation guidance relevant to the intent and
// provider requirements shared by the candidates.
func (h *Handlers) writeCrossCutting(b *strings.Builder, intent string, hits []index.Hit) {
	var lines []string
	for _, sh := range h.idx.SearchSections(intent+" accessibility", 30, nil) {
		if !strings.HasPrefix(sh.Ref.Origin, "foundation:") {
			continue
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", refTitle(sh.Ref), output.Clip(markdown.Prose(sh.Ref.Section.Body), 260)))
		if len(lines) == 2 {
			break
		}
	}
	var needProvider []string
	for _, hit := range hits {
		if slices.Contains(hit.Package.Peers, "@leafygreen-ui/leafygreen-provider") {
			needProvider = append(needProvider, hit.Package.Short)
		}
	}
	if len(needProvider) > 0 {
		lines = append(lines, fmt.Sprintf("`@leafygreen-ui/leafygreen-provider` is a peer dependency of %s; render inside `LeafyGreenProvider`.", strings.Join(needProvider, ", ")))
	}
	if len(lines) == 0 {
		return
	}
	b.WriteString("## Cross-cutting guidance\n\n")
	for _, l := range lines {
		bullet(b, "%s", l)
	}
	b.WriteString("\n")
}

func writeGaps(b *strings.Builder, h *Handlers, gaps []string) {
	if !h.idx.HasDesign() {
		gaps = append([]string{"No design snapshot loaded, so design guidance is missing for every candidate. Run `lg-mcp sync --design`."}, gaps...)
	}
	if len(gaps) == 0 {
		return
	}
	b.WriteString("## Gaps (do not fill these from memory)\n\n")
	for _, g := range gaps {
		bullet(b, "%s", g)
	}
	b.WriteString("\n")
}

func writeNextSteps(b *strings.Builder, p *index.Package) {
	fmt.Fprintf(b, "## Next steps\n\n- get_component_props name=%q for every prop before writing JSX\n- get_accessibility name=%q for the full accessibility notes\n- get_component_examples name=%q for more examples\n", p.Short, p.Short, p.Short)
}
