package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/output"
)

// GetComponentExamplesInput is the input of get_component_examples.
type GetComponentExamplesInput struct {
	Name  string `json:"name" jsonschema:"component or package name, e.g. button or @leafygreen-ui/button"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of examples to return (default 3, max 10)"`
}

// GetComponentExamples returns README and story examples.
func (h *Handlers) GetComponentExamples(_ context.Context, _ *mcp.CallToolRequest, in GetComponentExamplesInput) (*mcp.CallToolResult, any, error) {
	p, nf, err := h.lookup(in.Name)
	if p == nil {
		return nf, nil, err
	}
	limit := clampLimit(in.Limit, 3, 10)
	var b strings.Builder
	b.WriteString(header("Examples", p))
	if len(p.Examples) == 0 {
		where := "its README"
		if h.idx.SourceName == "repo" {
			where = "its README or Storybook stories"
		}
		fmt.Fprintf(&b, "No code examples found for %s in %s.\n", p.Name, where)
		return h.result(b.String()), nil, nil
	}
	shown := min(limit, len(p.Examples))
	fmt.Fprintf(&b, "Showing %d of %d.\n\n", shown, len(p.Examples))
	for _, e := range p.Examples[:shown] {
		fmt.Fprintf(&b, "## %s\n\n", e.Source)
		b.WriteString(output.CodeBlock(e.Lang, e.Code))
		b.WriteString("\n")
	}
	return h.result(b.String()), nil, nil
}

// clampLimit applies a default to non-positive values and caps at hi.
func clampLimit(n, def, hi int) int {
	if n <= 0 {
		return def
	}
	return min(n, hi)
}
