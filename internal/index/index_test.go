package index

import (
	"context"
	"strings"
	"testing"

	"github.com/brunodonatelli/lg-mcp/internal/design"
	"github.com/brunodonatelli/lg-mcp/internal/source"
)

func buildFixture(t *testing.T) *Index {
	t.Helper()
	ctx := context.Background()
	snap, warns, err := design.Load(ctx, "../../testdata/design")
	if err != nil || len(warns) > 0 {
		t.Fatalf("design.Load: %v %v", err, warns)
	}
	ix, err := Build(ctx, &source.Repo{Path: "../../testdata/repo"}, snap)
	if err != nil {
		t.Fatal(err)
	}
	return ix
}

func TestBuild(t *testing.T) {
	ix := buildFixture(t)
	if len(ix.Warnings) != 0 {
		t.Fatalf("warnings: %v", ix.Warnings)
	}
	if len(ix.Packages) != 8 {
		t.Fatalf("got %d packages", len(ix.Packages))
	}
	cb, ok := ix.Lookup("combobox")
	if !ok {
		t.Fatal("combobox not found")
	}
	t.Run("metadata", func(t *testing.T) {
		if cb.Version != "12.1.0" || cb.Scope != "leafygreen-ui" || strings.Join(cb.Peers, ",") != "@leafygreen-ui/leafygreen-provider" {
			t.Fatalf("combobox = %+v", cb)
		}
		if strings.Join(cb.Imports, "|") != "import { Combobox, ComboboxOption } from '@leafygreen-ui/combobox';" {
			t.Fatalf("imports = %q", cb.Imports)
		}
	})
	t.Run("examples skip install snippets and include stories", func(t *testing.T) {
		var sources []string
		for _, e := range cb.Examples {
			sources = append(sources, e.Source)
		}
		if strings.Join(sources, "|") != "README › Example|src/Combobox.stories.tsx › Multiselect" {
			t.Fatalf("example sources = %v", sources)
		}
	})
	t.Run("a11y from every origin", func(t *testing.T) {
		kinds := map[string]int{}
		for _, n := range cb.A11y {
			kinds[n.Kind]++
		}
		if kinds[A11yDesign] != 1 || kinds[A11yReadme] != 1 || kinds[A11yProp] != 2 || kinds[A11ySource] != 1 {
			t.Fatalf("a11y kinds = %v: %+v", kinds, cb.A11y)
		}
		if cb.A11y[0].Kind != A11yDesign {
			t.Fatalf("design notes should come first: %+v", cb.A11y[0])
		}
	})
	t.Run("design attached", func(t *testing.T) {
		if cb.Design == nil || !strings.HasSuffix(cb.Design.URL, "/combobox/design-docs") {
			t.Fatalf("design = %+v", cb.Design)
		}
		btn, _ := ix.Lookup("button")
		if btn.Design != nil {
			t.Fatalf("button has no design page in the fixture snapshot")
		}
		if _, ok := ix.Foundations["accessibility"]; !ok || !ix.HasDesign() {
			t.Fatalf("foundations = %v", ix.Foundations)
		}
	})
}

func TestBuildNodeModules(t *testing.T) {
	ix, err := Build(context.Background(), &source.NodeModules{Project: "../../testdata/node_modules"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	btn, ok := ix.Lookup("Button")
	if !ok || btn.Version != "23.1.0" || ix.HasDesign() {
		t.Fatalf("button = %+v, design=%v", btn, ix.HasDesign())
	}
	if len(btn.Decls) != 2 { // type Variant + interface ButtonProps; declare const is not a type
		t.Fatalf("decls = %d", len(btn.Decls))
	}
}

func TestNormalizeName(t *testing.T) {
	tests := map[string]string{
		"button":                  "button",
		"Button":                  "button",
		" @leafygreen-ui/button ": "button",
		"leafygreen-ui/button":    "button",
		"ConfirmationModal":       "confirmation-modal",
		"confirmation modal":      "confirmation-modal",
		"text_input":              "text-input",
		"@lg-chat/message":        "message",
		"":                        "",
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			if got := NormalizeName(in); got != want {
				t.Fatalf("NormalizeName(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestLookup(t *testing.T) {
	ix := buildFixture(t)
	tests := []struct {
		in   string
		want string
	}{
		{"combobox", "@leafygreen-ui/combobox"},
		{"@leafygreen-ui/Combobox", "@leafygreen-ui/combobox"},
		{"ConfirmationModal", "@leafygreen-ui/confirmation-modal"},
		{"lg-chat/message", "@lg-chat/message"},
		{"ComboboxOption", "@leafygreen-ui/combobox"},
		{"ComboboxOptionProps", "@leafygreen-ui/combobox"},
		{"nonexistent", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			p, ok := ix.Lookup(tt.in)
			got := ""
			if ok {
				got = p.Name
			}
			if got != tt.want {
				t.Fatalf("Lookup(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSuggest(t *testing.T) {
	ix := buildFixture(t)
	got := ix.Suggest("combo-box", 3)
	if len(got) == 0 || got[0] != "@leafygreen-ui/combobox" {
		t.Fatalf("Suggest = %v", got)
	}
	if got := ix.Suggest("modl", 2); len(got) == 0 || got[0] != "@leafygreen-ui/modal" {
		t.Fatalf("Suggest(modl) = %v", got)
	}
}

func TestSearchIntent(t *testing.T) {
	ix := buildFixture(t)
	tests := []struct {
		intent string
		first  string
	}{
		{"let the user select multiple tags from a long list", "@leafygreen-ui/combobox"},
		{"pick several tags", "@leafygreen-ui/combobox"},
		{"confirm a destructive delete action", "@leafygreen-ui/confirmation-modal"},
		{"open a dialog", "@leafygreen-ui/modal"},
		{"primary button", "@leafygreen-ui/button"},
	}
	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			hits := ix.Search(tt.intent, SearchOptions{Limit: 3})
			if len(hits) == 0 || hits[0].Package.Name != tt.first {
				var names []string
				for _, h := range hits {
					names = append(names, h.Package.Name)
				}
				t.Fatalf("Search(%q) = %v, want %s first", tt.intent, names, tt.first)
			}
			if len(hits[0].Matched) == 0 {
				t.Fatalf("no match explanation")
			}
		})
	}
	if hits := ix.Search("the of and", SearchOptions{}); len(hits) != 0 {
		t.Fatalf("stopword-only query returned %d hits", len(hits))
	}
	if hits := ix.Search("message", SearchOptions{Scope: "leafygreen-ui"}); len(hits) != 0 {
		t.Fatalf("scope filter leaked: %v", hits[0].Package.Name)
	}
}

func TestSearchSections(t *testing.T) {
	ix := buildFixture(t)
	hits := ix.SearchSections("keyboard", 10, nil)
	origins := map[string]bool{}
	for _, h := range hits {
		origins[h.Ref.Origin] = true
	}
	if !origins["README"] || !origins["foundation:accessibility"] {
		t.Fatalf("origins = %v", origins)
	}
	cb, _ := ix.Lookup("combobox")
	for _, h := range ix.SearchSections("keyboard", 10, cb) {
		if h.Ref.Package != cb {
			t.Fatalf("package filter leaked %v", h.Ref.Origin)
		}
	}
}

func TestStem(t *testing.T) {
	pairs := [][2]string{
		{"tags", "tag"}, {"selected", "select"}, {"selecting", "select"},
		{"disabled", "disabl"}, {"disable", "disabl"}, {"boxes", "box"},
		{"categories", "category"}, {"class", "class"}, {"status", "status"},
		{"modal", "modal"}, {"deleted", "delet"}, {"delete", "delet"},
		{"confirmation", "confirm"}, {"confirm", "confirm"}, {"navigation", "navig"},
		{"navigate", "navig"}, {"navigates", "navig"}, {"update", "updat"}, {"updated", "updat"},
	}
	for _, p := range pairs {
		if got := stem(p[0]); got != p[1] {
			t.Errorf("stem(%q) = %q, want %q", p[0], got, p[1])
		}
	}
}

func TestExpansionsTokenize(t *testing.T) {
	// Every expansion target must survive tokenization, or it adds nothing.
	for k, vs := range expansions {
		for _, v := range vs {
			if len(tokenize(v)) == 0 {
				t.Errorf("expansion %q → %q tokenizes to nothing", k, v)
			}
		}
	}
}

func TestMainProps(t *testing.T) {
	ix := buildFixture(t)
	tests := []struct {
		pkg   string
		main  string
		parts string
	}{
		{"combobox", "ComboboxProps", ""},
		{"confirmation-modal", "ConfirmationModalProps", ""},
		{"toggle", "ToggleProps", "BaseToggleProps"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			p, _ := ix.Lookup(tt.pkg)
			main, parts, ok := p.MainProps()
			var names []string
			for _, d := range parts {
				names = append(names, d.Name)
			}
			if !ok || main.Name != tt.main || strings.Join(names, ",") != tt.parts {
				t.Fatalf("MainProps = %s %v %v", main.Name, names, ok)
			}
		})
	}
	msg, _ := ix.Lookup("message")
	if _, _, ok := msg.MainProps(); ok {
		t.Fatalf("message has no props")
	}
}

func TestEitherA11y(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"export type ToggleProps = Either<BaseToggleProps & DarkModeProps, 'aria-label' | 'aria-labelledby'>;",
			"`ToggleProps` requires one of `aria-label`, `aria-labelledby` (enforced by the type via `Either`)."},
		{"export type ComboboxProps<M extends boolean> = Either<\n  BaseComboboxProps & ComboboxMultiselectProps<M>,\n  'label' | 'aria-label'\n>;",
			"`ComboboxProps` requires one of `label`, `aria-label` (enforced by the type via `Either`)."},
		{"export type XProps = Either<A, 'href' | 'onClick'>;", ""},
		{"export type YProps = Omit<A, 'label'>;", ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			decls := buildDecls(map[string]string{"t.ts": tt.src})
			if len(decls) != 1 {
				t.Fatalf("decls = %d", len(decls))
			}
			n, ok := eitherA11y(decls[0])
			if got := n.Text; ok != (tt.want != "") || got != tt.want {
				t.Fatalf("eitherA11y = %q, %v", got, ok)
			}
		})
	}
}
