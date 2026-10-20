package tools

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
)

// GetDesignGuidelinesInput is the input of get_design_guidelines.
type GetDesignGuidelinesInput struct {
	Name string `json:"name" jsonschema:"component name (e.g. modal) or foundation slug (e.g. accessibility)"`
}

// GetDesignGuidelines returns a component's or foundation's design page
// from the snapshot.
func (h *Handlers) GetDesignGuidelines(_ context.Context, _ *mcp.CallToolRequest, in GetDesignGuidelinesInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, nil, fmt.Errorf("name is required, e.g. \"modal\" or \"accessibility\"")
	}
	if !h.idx.HasDesign() {
		return h.result("No design snapshot is loaded, so no mongodb.design guidelines are available.\n\n" +
			"Run `lg-mcp sync --design` once to download a snapshot, then restart the server.\n"), nil, nil
	}
	slug := strings.TrimPrefix(index.NormalizeName(strings.TrimPrefix(in.Name, "foundation:")), "foundation-")
	if f, ok := h.idx.Foundations[slug]; ok && !h.isComponentName(in.Name) {
		return h.result(h.renderDesign("Foundation: "+slug, f.URL, f.Sections)), nil, nil
	}
	p, ok := h.idx.Lookup(in.Name)
	if !ok {
		return h.result(h.notFound(in.Name) + "\nFoundation pages: " + strings.Join(slices.Sorted(maps.Keys(h.idx.Foundations)), ", ") + "\n"), nil, nil
	}
	if p.Design == nil {
		return h.result(fmt.Sprintf("No design guidelines page for %s in the snapshot (mongodb.design may not document it under %q).\n", p.Name, p.Short)), nil, nil
	}
	return h.result(h.renderDesign(p.Name, p.Design.URL, p.Design.Sections)), nil, nil
}

// isComponentName reports whether name resolves to a package, so that a
// component sharing a slug with a foundation page wins when named directly.
func (h *Handlers) isComponentName(name string) bool {
	if strings.HasPrefix(name, "foundation:") {
		return false
	}
	_, ok := h.idx.Lookup(name)
	return ok
}

func (h *Handlers) renderDesign(title, url string, secs []markdown.Section) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Design guidelines: %s\n\n", title)
	fmt.Fprintf(&b, "%s\nPage: %s\n\n", h.designNote(), url)
	body := slices.Clone(secs)
	for i := range body {
		if body[i].Level == 1 {
			body[i].Heading = "" // the page title duplicates ours; keep its intro text
		}
	}
	renderSections(&b, body)
	return b.String()
}
