package tools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/output"
	"github.com/brunodonatelli/lg-mcp/internal/parse/tstypes"
)

// GetComponentPropsInput is the input of get_component_props.
type GetComponentPropsInput struct {
	Name      string `json:"name" jsonschema:"component or package name, e.g. combobox or @leafygreen-ui/combobox"`
	Component string `json:"component,omitempty" jsonschema:"optional declaration to show when a package exports several, e.g. ComboboxOption or ComboboxOptionProps"`
}

// GetComponentProps returns exported prop declarations with JSDoc.
func (h *Handlers) GetComponentProps(_ context.Context, _ *mcp.CallToolRequest, in GetComponentPropsInput) (*mcp.CallToolResult, any, error) {
	p, nf, err := h.lookup(in.Name)
	if p == nil {
		return nf, nil, err
	}
	var b strings.Builder
	b.WriteString(header("Props", p))
	if len(p.Decls) == 0 {
		fmt.Fprintf(&b, "No props declaration found for %s (no exported interface or type in its %s).\n", p.Name, typeFilesLabel(h.idx.SourceName))
		return h.result(b.String()), nil, nil
	}
	decls := orderedDecls(p)
	if in.Component != "" {
		d, ok := findDecl(p.Decls, in.Component)
		if !ok {
			fmt.Fprintf(&b, "No declaration named %q in %s.\n\nAvailable: %s\n", in.Component, p.Name, declNames(decls))
			return h.result(b.String()), nil, nil
		}
		decls = []index.Decl{d}
	}
	for _, d := range decls {
		writeDecl(&b, d)
	}
	b.WriteString("Note: members inherited through `extends` are not expanded; look up those types separately if needed.\n")
	return h.result(b.String()), nil, nil
}

func typeFilesLabel(source string) string {
	if source == "repo" {
		return "src/**/*.types.ts and types.ts files"
	}
	return ".d.ts files"
}

// orderedDecls lists the main props declaration first, then the interfaces
// it is composed of, then other *Props interfaces, other interfaces and
// type aliases, keeping source order within each group.
func orderedDecls(p *index.Package) []index.Decl {
	main, parts, _ := p.MainProps()
	first := map[string]int{main.Name: -2}
	for _, d := range parts {
		first[d.Name] = -1
	}
	out := slices.Clone(p.Decls)
	slices.SortStableFunc(out, func(a, b index.Decl) int {
		return propsRank(a, first) - propsRank(b, first)
	})
	return out
}

func propsRank(d index.Decl, first map[string]int) int {
	if r, ok := first[d.Name]; ok {
		return r
	}
	switch {
	case d.Kind == tstypes.Interface && strings.HasSuffix(d.Name, "Props"):
		return 0
	case d.Kind == tstypes.Interface:
		return 1
	}
	return 2
}

func findDecl(decls []index.Decl, name string) (index.Decl, bool) {
	for _, d := range decls {
		if strings.EqualFold(d.Name, name) || strings.EqualFold(d.Name, name+"Props") {
			return d, true
		}
	}
	return index.Decl{}, false
}

func declNames(decls []index.Decl) string {
	names := make([]string, len(decls))
	for i, d := range decls {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

func writeDecl(b *strings.Builder, d index.Decl) {
	fmt.Fprintf(b, "## %s%s\n\n", d.Name, d.TypeParams)
	fmt.Fprintf(b, "%s in `%s`", d.Kind, d.File)
	if d.Extends != "" {
		fmt.Fprintf(b, " · extends `%s`", d.Extends)
	}
	b.WriteString("\n\n")
	if d.JSDoc != "" {
		b.WriteString(d.JSDoc)
		b.WriteString("\n\n")
	}
	if d.Kind == tstypes.Interface && len(d.Members) > 0 {
		writeMemberTable(b, d.Members)
		return
	}
	b.WriteString(output.CodeBlock("ts", output.Clip(d.Source, 1500)))
	b.WriteString("\n")
}

func writeMemberTable(b *strings.Builder, members []tstypes.Member) {
	b.WriteString("| Prop | Type | Required | Description |\n| --- | --- | --- | --- |\n")
	for _, m := range members {
		req := "yes"
		if m.Optional {
			req = "no"
		}
		fmt.Fprintf(b, "| `%s` | `%s` | %s | %s |\n", m.Name, cell(m.Type), req, cell(m.JSDoc))
	}
	b.WriteString("\n")
}
