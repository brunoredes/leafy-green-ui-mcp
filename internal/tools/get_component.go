package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/output"
	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
)

// GetComponentInput is the input of get_component.
type GetComponentInput struct {
	Name string `json:"name" jsonschema:"component or package name, e.g. button, Combobox or @leafygreen-ui/combobox"`
}

// GetComponent returns the package overview.
func (h *Handlers) GetComponent(_ context.Context, _ *mcp.CallToolRequest, in GetComponentInput) (*mcp.CallToolResult, any, error) {
	p, nf, err := h.lookup(in.Name)
	if p == nil {
		return nf, nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n", p.Name, p.Version)
	if p.Deprecated != "" {
		fmt.Fprintf(&b, "**Deprecated:** %s\n\n", p.Deprecated)
	}
	if p.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Description)
	}
	fmt.Fprintf(&b, "Install: `npm install %s`\n", p.Name)
	writeImports(&b, p)
	if len(p.Peers) > 0 {
		fmt.Fprintf(&b, "Peers: %s\n", strings.Join(p.Peers, ", "))
	}
	fmt.Fprintf(&b, "%s\n\n", h.sourceNote())

	b.WriteString("## Overview\n\n")
	if ov := readmeOverview(p); ov != "" {
		b.WriteString(output.Clip(ov, 2500))
		b.WriteString("\n\n")
	} else if !p.HasReadme {
		b.WriteString("No README found for this package.\n\n")
	} else {
		b.WriteString("The README has no overview text.\n\n")
	}
	if headings := readmeHeadings(p); len(headings) > 0 {
		fmt.Fprintf(&b, "README sections: %s\n\n", strings.Join(headings, " · "))
	}

	b.WriteString("## More detail\n\n")
	bullet(&b, "%d exported type %s → get_component_props", len(p.Decls), plural(len(p.Decls), "declaration", "declarations"))
	bullet(&b, "%d code %s → get_component_examples", len(p.Examples), plural(len(p.Examples), "example", "examples"))
	bullet(&b, "%d accessibility %s → get_accessibility", len(p.A11y), plural(len(p.A11y), "note", "notes"))
	if p.Design != nil {
		bullet(&b, "Design guidelines available → get_design_guidelines")
	} else {
		bullet(&b, "No design guidelines page in the snapshot")
	}
	return h.result(b.String()), nil, nil
}

func writeImports(b *strings.Builder, p *index.Package) {
	switch len(p.Imports) {
	case 0:
		b.WriteString("Import: no import statement found in the README examples\n")
	case 1:
		fmt.Fprintf(b, "Import: `%s`\n", p.Imports[0])
	default:
		b.WriteString("Imports:\n")
		for _, imp := range p.Imports {
			fmt.Fprintf(b, "  - `%s`\n", imp)
		}
	}
}

// readmeOverview returns the prose before the first second-level heading:
// the preamble and the document-title section.
func readmeOverview(p *index.Package) string {
	var parts []string
	for _, s := range p.Readme {
		if s.Level > 1 {
			break
		}
		if t := markdown.Prose(s.Body); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n\n")
}

func readmeHeadings(p *index.Package) []string {
	var out []string
	for _, s := range p.Readme {
		if s.Level == 2 {
			out = append(out, s.Heading)
		}
	}
	return out
}
