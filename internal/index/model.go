// Package index builds the read-only, in-memory model of LeafyGreen packages
// that tools query. It knows nothing about MCP.
package index

import (
	"time"

	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
	"github.com/brunodonatelli/lg-mcp/internal/parse/tstypes"
)

// Package is everything known about one npm package.
type Package struct {
	Name        string // npm name, e.g. "@leafygreen-ui/combobox"
	Scope       string // scope without "@", e.g. "leafygreen-ui"
	Short       string // name without scope, e.g. "combobox"
	Version     string
	Description string
	Deprecated  string
	Peers       []string
	Imports     []string // import statements found in README examples
	HasReadme   bool
	Readme      []markdown.Section
	Decls       []Decl
	Examples    []Example
	A11y        []A11yNote
	Design      *DesignDoc
}

// Decl is an exported type declaration with the file it came from.
type Decl struct {
	tstypes.Decl
	File string
}

// Example is one code example and where it came from.
type Example struct {
	Source string // e.g. "README › Example" or "src/Button.stories.tsx › Primary"
	Lang   string
	Code   string
}

// A11yNote kinds, in the order tools present them.
const (
	A11yDesign = "design"
	A11yReadme = "readme"
	A11yProp   = "prop"
	A11ySource = "source"
)

// A11yNote is one accessibility fact with provenance.
type A11yNote struct {
	Kind   string
	Source string
	Text   string
}

// DesignDoc is a component's design guideline page from the snapshot.
type DesignDoc struct {
	URL      string
	Sections []markdown.Section
}

// Foundation is a cross-cutting design page such as Accessibility.
type Foundation struct {
	Slug     string
	URL      string
	Sections []markdown.Section
}

// Index is the read-only package catalog. It is safe for concurrent use
// once Build returns.
type Index struct {
	SourceName      string
	Packages        []*Package // sorted by Name
	Foundations     map[string]Foundation
	DesignFetchedAt time.Time
	DesignDir       string
	Warnings        []string

	byName   map[string]*Package
	byDecl   map[string]*Package
	pkgDocs  *corpus
	sections *corpus
	secRefs  []SectionRef
}

// HasDesign reports whether a design snapshot was loaded.
func (ix *Index) HasDesign() bool { return !ix.DesignFetchedAt.IsZero() }

// SectionRef identifies a searchable documentation section.
type SectionRef struct {
	Package *Package // nil for foundation pages
	Origin  string   // "README", "design" or "foundation:<slug>"
	Section markdown.Section
}
