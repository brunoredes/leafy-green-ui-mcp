package tools

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/output"
	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
)

// result wraps Markdown in a truncated text result.
func (h *Handlers) result(md string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: output.Truncate(md, h.max)}}}
}

// lookup resolves name or returns a ready "not found" result listing the
// closest package names.
func (h *Handlers) lookup(name string) (*index.Package, *mcp.CallToolResult, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil, fmt.Errorf("name is required, e.g. \"button\" or \"@leafygreen-ui/combobox\"")
	}
	if p, ok := h.idx.Lookup(name); ok {
		return p, nil, nil
	}
	return nil, h.result(h.notFound(name)), nil
}

func (h *Handlers) notFound(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "No LeafyGreen package matches %q in the %s source (%d packages indexed).\n", name, h.idx.SourceName, len(h.idx.Packages))
	if s := h.idx.Suggest(name, 5); len(s) > 0 {
		b.WriteString("\nClosest matches:\n")
		for _, n := range s {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	b.WriteString("\nIf you are describing what the UI should do rather than naming a component, use consult_design_system.\n")
	return b.String()
}

// header is the first line of per-package responses.
func header(title string, p *index.Package) string {
	return fmt.Sprintf("# %s: %s %s\n\n", title, p.Name, p.Version)
}

// sourceNote tells the model how far to trust versions.
func (h *Handlers) sourceNote() string {
	if h.idx.SourceName == "repo" {
		return "Source: leafygreen-ui repository clone (reflects its checked-out branch, which may be ahead of installed versions)."
	}
	return "Source: project node_modules (installed versions)."
}

func (h *Handlers) designNote() string {
	if !h.idx.HasDesign() {
		return "Design snapshot: not loaded (run `lg-mcp sync --design`)."
	}
	return "Design snapshot: mongodb.design, fetched " + h.idx.DesignFetchedAt.Format("2006-01-02") + "."
}

// renderSections writes sections back as Markdown.
func renderSections(b *strings.Builder, secs []markdown.Section) {
	for _, s := range secs {
		if s.Heading != "" {
			fmt.Fprintf(b, "%s %s\n\n", strings.Repeat("#", max(s.Level, 1)), s.Heading)
		}
		if body := strings.TrimSpace(s.Body); body != "" {
			b.WriteString(body)
			b.WriteString("\n\n")
		}
	}
}

// cell escapes text for a Markdown table cell.
func cell(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.ReplaceAll(s, "|", `\|`)
}

func bullet(b *strings.Builder, format string, args ...any) {
	b.WriteString("- ")
	fmt.Fprintf(b, format, args...)
	b.WriteByte('\n')
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
