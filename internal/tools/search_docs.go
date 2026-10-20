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

// SearchDocsInput is the input of search_docs.
type SearchDocsInput struct {
	Query string `json:"query" jsonschema:"words to search for, e.g. keyboard navigation or dark mode"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of sections to return (default 5, max 20)"`
}

// SearchDocs ranks README, design and foundation sections.
func (h *Handlers) SearchDocs(_ context.Context, _ *mcp.CallToolRequest, in SearchDocsInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Query) == "" {
		return nil, nil, fmt.Errorf("query is required")
	}
	hits := h.idx.SearchSections(in.Query, clampLimit(in.Limit, 5, 20), nil)
	var b strings.Builder
	fmt.Fprintf(&b, "# Search: %q\n\n", in.Query)
	if len(hits) == 0 {
		b.WriteString("No README or design-guideline section matched. Search is lexical: try the words the docs would use (e.g. \"modal\" rather than \"popup\").\n")
		if !h.idx.HasDesign() {
			b.WriteString("\nOnly READMEs were searched: " + h.designNote() + "\n")
		}
		return h.result(b.String()), nil, nil
	}
	fmt.Fprintf(&b, "%d %s.\n\n", len(hits), plural(len(hits), "result", "results"))
	per := max(300, (h.max-200)/len(hits)-120)
	for i, hit := range hits {
		fmt.Fprintf(&b, "## %d. %s\n\n", i+1, refTitle(hit.Ref))
		b.WriteString(output.Clip(markdown.Prose(hit.Ref.Section.Body), per))
		b.WriteString("\n\n")
	}
	return h.result(b.String()), nil, nil
}

// refTitle names where a section lives, e.g.
// "@leafygreen-ui/modal › README › Additional notes > Initial focus behavior".
func refTitle(r index.SectionRef) string {
	switch {
	case r.Package != nil && r.Origin == "design":
		return r.Package.Name + " › Design guidelines › " + r.Section.Title()
	case r.Package != nil:
		return r.Package.Name + " › README › " + r.Section.Title()
	}
	return "Foundation: " + strings.TrimPrefix(r.Origin, "foundation:") + " › " + r.Section.Title()
}
