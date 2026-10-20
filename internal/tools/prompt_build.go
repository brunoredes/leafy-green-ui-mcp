package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var buildPrompt = &mcp.Prompt{
	Name:        "build_with_leafygreen",
	Title:       "Build UI with LeafyGreen",
	Description: "Plan and write UI for an intent using LeafyGreen components, grounded in the leafygreen tools.",
	Arguments: []*mcp.PromptArgument{{
		Name:        "intent",
		Description: "What the UI should do, e.g. \"confirm before deleting a cluster\"",
		Required:    true,
	}},
}

// BuildWithLeafyGreen renders the build_with_leafygreen prompt.
func (h *Handlers) BuildWithLeafyGreen(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	intent := strings.TrimSpace(req.Params.Arguments["intent"])
	if intent == "" {
		return nil, fmt.Errorf("argument \"intent\" is required")
	}
	text := fmt.Sprintf(`I want to build this UI with LeafyGreen: %s

Work from the leafygreen MCP tools, not from memory:
1. Call consult_design_system with the intent above and pick the component(s) whose design guidance fits.
2. For each chosen component, call get_component_props and use only props, imports and values the tools confirm.
3. Call get_accessibility for each chosen component and apply its notes (labels, focus, keyboard behaviour).
4. If anything you need is listed under "Gaps" or is missing from the tool output, say so explicitly instead of guessing.
5. Then write the code, and briefly cite which guideline or prop each design decision relies on.`, intent)
	return &mcp.GetPromptResult{
		Description: "Build UI with LeafyGreen for: " + intent,
		Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
	}, nil
}
