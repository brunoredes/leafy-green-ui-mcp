package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestServerEndToEnd runs an in-memory MCP client against a server with
// every tool and prompt registered, and calls each one over the protocol.
func TestServerEndToEnd(t *testing.T) {
	ctx := context.Background()
	srv := mcp.NewServer(&mcp.Implementation{Name: "lg-mcp", Version: "test"}, nil)
	Register(srv, NewHandlers(fixtureIndex(t, true), 12000))

	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ss.Close() }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()

	calls := map[string]map[string]any{
		"consult_design_system":  {"intent": "pick several tags from a long list"},
		"get_component":          {"name": "button"},
		"get_component_props":    {"name": "combobox", "component": "ComboboxOption"},
		"get_component_examples": {"name": "button", "limit": 1},
		"get_accessibility":      {"name": "combobox"},
		"get_design_guidelines":  {"name": "accessibility"},
		"search_docs":            {"query": "keyboard"},
	}

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != len(calls) {
		t.Fatalf("listed %d tools, want %d", len(tools.Tools), len(calls))
	}
	for _, tool := range tools.Tools {
		args, ok := calls[tool.Name]
		if !ok {
			t.Errorf("tool %s has no end-to-end call", tool.Name)
			continue
		}
		t.Run(tool.Name, func(t *testing.T) {
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Errorf("tool is not annotated read-only")
			}
			if len(tool.Description) < 40 {
				t.Errorf("description too thin: %q", tool.Description)
			}
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: tool.Name, Arguments: args})
			if err != nil {
				t.Fatal(err)
			}
			if res.IsError {
				t.Fatalf("IsError: %+v", res.Content)
			}
			if text := textOf(t, res); strings.TrimSpace(text) == "" {
				t.Fatalf("empty response")
			}
		})
	}

	t.Run("schema rejects missing required input", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_component", Arguments: map[string]any{}})
		if err != nil {
			t.Fatal(err)
		}
		if !res.IsError {
			t.Fatalf("expected IsError for missing name")
		}
	})

	t.Run("prompt", func(t *testing.T) {
		prompts, err := cs.ListPrompts(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(prompts.Prompts) != 1 || prompts.Prompts[0].Name != "build_with_leafygreen" {
			t.Fatalf("prompts = %+v", prompts.Prompts)
		}
		got, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{Name: "build_with_leafygreen", Arguments: map[string]string{"intent": "delete a cluster"}})
		if err != nil {
			t.Fatal(err)
		}
		msg, ok := got.Messages[0].Content.(*mcp.TextContent)
		if !ok || !strings.Contains(msg.Text, "delete a cluster") || !strings.Contains(msg.Text, "consult_design_system") {
			t.Fatalf("prompt = %+v", got.Messages[0].Content)
		}
	})
}
