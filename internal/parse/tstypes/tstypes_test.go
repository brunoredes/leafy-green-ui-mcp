package tstypes

import (
	"strings"
	"testing"
)

const src = `import { X } from 'y';

export const Variant = { Primary: 'primary' } as const;

export type Variant = (typeof Variant)[keyof typeof Variant];

/**
 * Props for Combobox.
 * Generic over multiselect.
 */
export interface ComboboxProps<M extends boolean, T = Array<string>> extends DarkModeProps,
  Omit<HTMLElementProps<'input'>, 'onChange'> {
  /** Allows multiple. */
  multiselect?: M;

  // plain comment is ignored
  'aria-label'?: string;

  /** Called on change. */
  onChange?: (value: M extends true ? Array<string> : string | null) => void;

  overflow?: { kind: 'expand-y' | 'scroll-x'; max: number };

  size:
    | 'small'
    | 'default';

  readonly label: string
  render(item: T): ReactNode;
  [key: string]: unknown;
  /** Brace in string: "}" should not close. */
  sep?: '}';
}

interface NotExported { a: string }

export declare type Callback<T> = (v: T) => void;

export type Point = {
  /** X coord. */
  x: number;
  y?: number;
};
`

func TestExtract(t *testing.T) {
	decls := Extract(src)
	var names []string
	for _, d := range decls {
		names = append(names, string(d.Kind)+":"+d.Name)
	}
	want := "type:Variant|interface:ComboboxProps|type:Callback|type:Point"
	if got := strings.Join(names, "|"); got != want {
		t.Fatalf("decls = %s, want %s", got, want)
	}

	cb := decls[1]
	t.Run("interface header", func(t *testing.T) {
		if cb.TypeParams != "<M extends boolean, T = Array<string>>" {
			t.Errorf("TypeParams = %q", cb.TypeParams)
		}
		if cb.Extends != "DarkModeProps, Omit<HTMLElementProps<'input'>, 'onChange'>" {
			t.Errorf("Extends = %q", cb.Extends)
		}
		if cb.JSDoc != "Props for Combobox.\nGeneric over multiselect." {
			t.Errorf("JSDoc = %q", cb.JSDoc)
		}
		if !strings.HasSuffix(cb.Source, "sep?: '}';\n}") {
			t.Errorf("Source does not end at the closing brace: %q", cb.Source[len(cb.Source)-30:])
		}
	})

	t.Run("members", func(t *testing.T) {
		want := []struct {
			name, typ, doc string
			optional       bool
		}{
			{"multiselect", "M", "Allows multiple.", true},
			{"aria-label", "string", "", true},
			{"onChange", "(value: M extends true ? Array<string> : string | null) => void", "Called on change.", true},
			{"overflow", "{ kind: 'expand-y' | 'scroll-x'; max: number }", "", true},
			{"size", "| 'small' | 'default'", "", false},
			{"label", "string", "", false},
			{"render", "(item: T): ReactNode", "", false},
			{"sep", "'}'", `Brace in string: "}" should not close.`, true},
		}
		if len(cb.Members) != len(want) {
			t.Fatalf("got %d members: %+v", len(cb.Members), cb.Members)
		}
		for i, w := range want {
			m := cb.Members[i]
			if m.Name != w.name || m.Type != w.typ || m.JSDoc != w.doc || m.Optional != w.optional {
				t.Errorf("member %d = %+v, want %+v", i, m, w)
			}
		}
	})

	t.Run("type alias", func(t *testing.T) {
		if decls[0].Source != "export type Variant = (typeof Variant)[keyof typeof Variant];" {
			t.Errorf("Variant source = %q", decls[0].Source)
		}
		if decls[2].TypeParams != "<T>" || !strings.HasSuffix(decls[2].Source, "=> void;") {
			t.Errorf("Callback = %+v", decls[2])
		}
		pt := decls[3]
		if len(pt.Members) != 2 || pt.Members[0].JSDoc != "X coord." || !pt.Members[1].Optional {
			t.Errorf("Point members = %+v", pt.Members)
		}
	})
}

func TestExtractUnterminated(t *testing.T) {
	if got := Extract("export interface Broken {\n a: string;\n"); len(got) != 0 {
		t.Fatalf("expected no decls from unterminated interface, got %+v", got)
	}
}

func TestAriaProps(t *testing.T) {
	d := Decl{Members: []Member{
		{Name: "aria-label"},
		{Name: "role"},
		{Name: "label", JSDoc: "Visible label. Required unless aria-label is provided."},
		{Name: "initialFocus", JSDoc: "Element that receives focus when opened."},
		{Name: "size", JSDoc: "Size of the input."},
	}}
	var got []string
	for _, m := range AriaProps(d) {
		got = append(got, m.Name)
	}
	if strings.Join(got, ",") != "aria-label,role,label,initialFocus" {
		t.Fatalf("AriaProps = %v", got)
	}
}
