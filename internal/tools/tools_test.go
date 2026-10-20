package tools

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/design"
	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/source"
)

var update = flag.Bool("update", false, "rewrite golden files")

func fixtureIndex(t *testing.T, withDesign bool) *index.Index {
	t.Helper()
	ctx := context.Background()
	var snap *design.Snapshot
	if withDesign {
		var err error
		if snap, _, err = design.Load(ctx, "../../testdata/design"); err != nil {
			t.Fatal(err)
		}
	}
	ix, err := index.Build(ctx, &source.Repo{Path: "../../testdata/repo"}, snap)
	if err != nil {
		t.Fatal(err)
	}
	return ix
}

type toolCall func(context.Context) (*mcp.CallToolResult, any, error)

func TestGolden(t *testing.T) {
	full := NewHandlers(fixtureIndex(t, true), 12000)
	bare := NewHandlers(fixtureIndex(t, false), 12000)
	cases := []struct {
		golden string
		call   toolCall
	}{
		{"consult_tags", consult(full, ConsultDesignSystemInput{Intent: "let users select multiple tags from a long list"})},
		{"consult_delete", consult(full, ConsultDesignSystemInput{Intent: "confirm a destructive delete action", Limit: 2})},
		{"consult_toggle_alias_props", consult(full, ConsultDesignSystemInput{Intent: "toggle a setting on or off", Limit: 1})},
		{"get_component_props_toggle_alias", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentProps(ctx, nil, GetComponentPropsInput{Name: "toggle"})
		}},
		{"consult_no_match", consult(full, ConsultDesignSystemInput{Intent: "zzz qqq"})},
		{"consult_no_design", consult(bare, ConsultDesignSystemInput{Intent: "open a dialog", Limit: 1})},
		{"get_component_combobox", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponent(ctx, nil, GetComponentInput{Name: "Combobox"})
		}},
		{"get_component_no_readme", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponent(ctx, nil, GetComponentInput{Name: "@lg-chat/message"})
		}},
		{"get_component_not_found", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponent(ctx, nil, GetComponentInput{Name: "combo-box"})
		}},
		{"get_component_props_combobox", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentProps(ctx, nil, GetComponentPropsInput{Name: "combobox"})
		}},
		{"get_component_props_option", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentProps(ctx, nil, GetComponentPropsInput{Name: "combobox", Component: "ComboboxOption"})
		}},
		{"get_component_props_unknown_component", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentProps(ctx, nil, GetComponentPropsInput{Name: "button", Component: "Nope"})
		}},
		{"get_component_props_none", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentProps(ctx, nil, GetComponentPropsInput{Name: "message"})
		}},
		{"get_component_examples_button", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentExamples(ctx, nil, GetComponentExamplesInput{Name: "button", Limit: 2})
		}},
		{"get_component_examples_none", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetComponentExamples(ctx, nil, GetComponentExamplesInput{Name: "message"})
		}},
		{"get_accessibility_combobox", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetAccessibility(ctx, nil, GetAccessibilityInput{Name: "combobox"})
		}},
		{"get_accessibility_modal", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetAccessibility(ctx, nil, GetAccessibilityInput{Name: "modal"})
		}},
		{"get_accessibility_none", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetAccessibility(ctx, nil, GetAccessibilityInput{Name: "message"})
		}},
		{"get_design_guidelines_modal", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetDesignGuidelines(ctx, nil, GetDesignGuidelinesInput{Name: "modal"})
		}},
		{"get_design_guidelines_foundation", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetDesignGuidelines(ctx, nil, GetDesignGuidelinesInput{Name: "accessibility"})
		}},
		{"get_design_guidelines_missing_page", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.GetDesignGuidelines(ctx, nil, GetDesignGuidelinesInput{Name: "button"})
		}},
		{"get_design_guidelines_no_snapshot", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return bare.GetDesignGuidelines(ctx, nil, GetDesignGuidelinesInput{Name: "modal"})
		}},
		{"search_docs_keyboard", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return full.SearchDocs(ctx, nil, SearchDocsInput{Query: "keyboard", Limit: 3})
		}},
		{"search_docs_no_match", func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return bare.SearchDocs(ctx, nil, SearchDocsInput{Query: "zzzz"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.golden, func(t *testing.T) {
			res, _, err := tc.call(context.Background())
			if err != nil {
				t.Fatalf("tool error: %v", err)
			}
			if res.IsError {
				t.Fatalf("IsError result")
			}
			got := textOf(t, res)
			path := filepath.Join("testdata", tc.golden+".golden.md")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if got != string(want) {
				t.Fatalf("output differs from %s (run go test ./internal/tools/... -update and review the diff)\n--- got ---\n%s", path, got)
			}
		})
	}
}

func consult(h *Handlers, in ConsultDesignSystemInput) toolCall {
	return func(ctx context.Context) (*mcp.CallToolResult, any, error) {
		return h.ConsultDesignSystem(ctx, nil, in)
	}
}

func TestInputErrors(t *testing.T) {
	h := NewHandlers(fixtureIndex(t, false), 12000)
	ctx := context.Background()
	calls := map[string]toolCall{
		"empty intent": consult(h, ConsultDesignSystemInput{Intent: "  "}),
		"bad scope":    consult(h, ConsultDesignSystemInput{Intent: "modal", Scope: "material"}),
		"empty name": func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return h.GetComponent(ctx, nil, GetComponentInput{})
		},
		"empty query": func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return h.SearchDocs(ctx, nil, SearchDocsInput{})
		},
		"empty design name": func(ctx context.Context) (*mcp.CallToolResult, any, error) {
			return h.GetDesignGuidelines(ctx, nil, GetDesignGuidelinesInput{})
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if _, _, err := call(ctx); err == nil {
				t.Fatalf("expected an input error")
			}
		})
	}
}

func TestConsultScope(t *testing.T) {
	h := NewHandlers(fixtureIndex(t, true), 12000)
	res, _, err := h.ConsultDesignSystem(context.Background(), nil, ConsultDesignSystemInput{Intent: "message", Scope: "@lg-chat"})
	if err != nil {
		t.Fatal(err)
	}
	if got := textOf(t, res); !strings.Contains(got, "Candidates, best first: @lg-chat/message\n") {
		t.Fatalf("scope not applied:\n%s", got)
	}
}

func TestTruncation(t *testing.T) {
	h := NewHandlers(fixtureIndex(t, true), 600)
	res, _, err := h.ConsultDesignSystem(context.Background(), nil, ConsultDesignSystemInput{Intent: "select multiple tags", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "[truncated: ") || len([]rune(got)) > 700 {
		t.Fatalf("not truncated to budget (%d chars):\n%s", len([]rune(got)), got)
	}
}

func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("want 1 content item, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T", res.Content[0])
	}
	return tc.Text
}
