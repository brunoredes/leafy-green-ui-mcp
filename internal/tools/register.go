// Package tools implements the MCP tools and prompts served by lg-mcp. Each
// tool lives in its own file; this file wires them to a server.
package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/output"
)

// Handlers holds the read-only state shared by all tools.
type Handlers struct {
	idx *index.Index
	max int
}

// NewHandlers returns handlers over idx whose responses are truncated to
// maxOutput characters.
func NewHandlers(idx *index.Index, maxOutput int) *Handlers {
	if maxOutput <= 0 {
		maxOutput = output.DefaultMax
	}
	return &Handlers{idx: idx, max: maxOutput}
}

func readOnly(title string) *mcp.ToolAnnotations {
	closed := false
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &closed}
}

// Register adds every tool and prompt to s.
func Register(s *mcp.Server, h *Handlers) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "consult_design_system",
		Description: "Call this first when deciding which LeafyGreen UI component(s) to use for a UI intent " +
			"(e.g. \"let users pick several tags from a long list\"). Returns ranked candidate components, each with " +
			"why it matched, install/import, design guidance from mongodb.design, key props, a code example and " +
			"accessibility notes, plus an explicit list of gaps where the docs are silent.",
		Annotations: readOnly("Consult LeafyGreen for a UI intent"),
	}, h.ConsultDesignSystem)
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_component",
		Description: "Overview of one LeafyGreen package: version, install command, import statements, peer " +
			"dependencies, deprecation and README overview. Use when you already know the component name.",
		Annotations: readOnly("Get LeafyGreen component"),
	}, h.GetComponent)
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_component_props",
		Description: "Exported prop interfaces and types of a LeafyGreen package, as declared in source, with JSDoc. " +
			"Use before writing JSX so you only pass props that exist. Extends clauses are listed but not expanded.",
		Annotations: readOnly("Get LeafyGreen component props"),
	}, h.GetComponentProps)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_component_examples",
		Description: "Code examples for a LeafyGreen package from its README and, with a repo source, its Storybook stories.",
		Annotations: readOnly("Get LeafyGreen component examples"),
	}, h.GetComponentExamples)
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_accessibility",
		Description: "All accessibility facts for one LeafyGreen package: design-guideline a11y sections, README " +
			"keyboard/focus/ARIA notes, accessibility-related props, and a11y behaviour observed in source " +
			"(e.g. required aria-label). Use when implementing or reviewing a component's accessibility.",
		Annotations: readOnly("Get LeafyGreen accessibility notes"),
	}, h.GetAccessibility)
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_design_guidelines",
		Description: "Full mongodb.design guideline page for a component (usage, when to use, spacing, sizing) or a " +
			"foundation such as \"accessibility\", from the local snapshot.",
		Annotations: readOnly("Get LeafyGreen design guidelines"),
	}, h.GetDesignGuidelines)
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_docs",
		Description: "Ranked README and design-guideline sections across all LeafyGreen packages and foundations. " +
			"Use for cross-cutting questions (e.g. \"keyboard navigation\", \"dark mode\") rather than a single component.",
		Annotations: readOnly("Search LeafyGreen docs"),
	}, h.SearchDocs)
	s.AddPrompt(buildPrompt, h.BuildWithLeafyGreen)
}
