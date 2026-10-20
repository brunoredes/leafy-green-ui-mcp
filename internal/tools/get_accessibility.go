package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/output"
)

// GetAccessibilityInput is the input of get_accessibility.
type GetAccessibilityInput struct {
	Name string `json:"name" jsonschema:"component or package name, e.g. modal or @leafygreen-ui/combobox"`
}

var a11yGroups = []struct {
	kind  string
	title string
}{
	{index.A11yDesign, "Design guidelines"},
	{index.A11yReadme, "README"},
	{index.A11yProp, "Accessibility-related props"},
	{index.A11ySource, "Observed in source"},
}

// GetAccessibility returns every accessibility fact for a package, grouped
// by where it came from.
func (h *Handlers) GetAccessibility(_ context.Context, _ *mcp.CallToolRequest, in GetAccessibilityInput) (*mcp.CallToolResult, any, error) {
	p, nf, err := h.lookup(in.Name)
	if p == nil {
		return nf, nil, err
	}
	var b strings.Builder
	b.WriteString(header("Accessibility", p))
	if len(p.A11y) == 0 {
		fmt.Fprintf(&b, "No accessibility notes found for %s in its README, type declarations, source or design snapshot.\n\n", p.Name)
	}
	for _, g := range a11yGroups {
		writeA11yGroup(&b, p.A11y, g.kind, g.title)
	}
	h.writeA11yPointers(&b)
	return h.result(b.String()), nil, nil
}

func writeA11yGroup(b *strings.Builder, notes []index.A11yNote, kind, title string) {
	var group []index.A11yNote
	for _, n := range notes {
		if n.Kind == kind {
			group = append(group, n)
		}
	}
	if len(group) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n\n", title)
	for _, n := range group {
		switch kind {
		case index.A11yDesign, index.A11yReadme:
			heading := strings.TrimPrefix(strings.TrimPrefix(n.Source, "README › "), "Design guidelines › ")
			fmt.Fprintf(b, "### %s\n\n%s\n\n", heading, output.Clip(n.Text, 1500))
		default:
			bullet(b, "%s — `%s`", n.Text, n.Source)
		}
	}
	if kind == index.A11yProp || kind == index.A11ySource {
		b.WriteString("\n")
	}
}

// writeA11yPointers lists general resources that exist in the index: the
// accessibility foundation page and the @leafygreen-ui/a11y utilities.
func (h *Handlers) writeA11yPointers(b *strings.Builder) {
	var lines []string
	if f, ok := h.idx.Foundations["accessibility"]; ok {
		var heads []string
		for _, s := range f.Sections {
			if s.Level == 2 {
				heads = append(heads, s.Heading)
			}
		}
		lines = append(lines, fmt.Sprintf("Accessibility foundation (%s) → get_design_guidelines name=\"accessibility\"", strings.Join(heads, ", ")))
	}
	if a, ok := h.idx.Lookup("@leafygreen-ui/a11y"); ok {
		var utils []string
		for _, s := range a.Readme {
			if s.Level == 2 && !strings.EqualFold(s.Heading, "Installation") && !strings.EqualFold(s.Heading, "Example") {
				utils = append(utils, "`"+s.Heading+"`")
			}
		}
		if len(utils) > 0 {
			lines = append(lines, fmt.Sprintf("%s documents %s → get_component name=\"a11y\"", a.Name, strings.Join(utils, ", ")))
		}
	}
	if len(lines) == 0 {
		return
	}
	b.WriteString("## General resources\n\n")
	for _, l := range lines {
		bullet(b, "%s", l)
	}
}
