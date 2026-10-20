package index

import (
	"regexp"
	"strings"

	"github.com/brunodonatelli/lg-mcp/internal/parse/tstypes"
)

var propsIdentRe = regexp.MustCompile(`\b[A-Z][A-Za-z0-9_]*Props\b`)

// MainProps returns the package's primary props declaration:
// <PascalName>Props, else Base<PascalName>Props, else the first exported
// *Props interface. When the main declaration is a type alias (LeafyGreen
// often composes props as `Either<BaseXProps & YProps, ...>`), parts holds
// the interfaces declared in this package that the alias references, in
// order of appearance. Types from other packages are not resolved.
func (p *Package) MainProps() (main Decl, parts []Decl, ok bool) {
	pascal := pascalCase(p.Short)
	main, ok = p.decl(pascal + "Props")
	if !ok {
		main, ok = p.decl("Base" + pascal + "Props")
	}
	if !ok {
		for _, d := range p.Decls {
			if d.Kind == tstypes.Interface && strings.HasSuffix(d.Name, "Props") {
				main, ok = d, true
				break
			}
		}
	}
	if !ok || main.Kind != tstypes.TypeAlias {
		return main, nil, ok
	}
	seen := map[string]bool{main.Name: true}
	for _, name := range propsIdentRe.FindAllString(main.Source, -1) {
		if seen[name] {
			continue
		}
		seen[name] = true
		if d, found := p.decl(name); found && d.Kind == tstypes.Interface {
			parts = append(parts, d)
		}
	}
	return main, parts, true
}

func (p *Package) decl(name string) (Decl, bool) {
	for _, d := range p.Decls {
		if d.Name == name {
			return d, true
		}
	}
	return Decl{}, false
}

// IsDeprecated reports whether a member's JSDoc carries @deprecated.
func IsDeprecated(m tstypes.Member) bool {
	return strings.Contains(m.JSDoc, "@deprecated")
}

// pascalCase turns a kebab-case package name into PascalCase:
// "confirmation-modal" → "ConfirmationModal".
func pascalCase(kebab string) string {
	var b strings.Builder
	for _, part := range strings.Split(kebab, "-") {
		if part != "" {
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}
