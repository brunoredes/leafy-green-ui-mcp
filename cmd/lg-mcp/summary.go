package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/brunodonatelli/lg-mcp/internal/config"
	"github.com/brunodonatelli/lg-mcp/internal/index"
)

// summary renders the `lg-mcp index` report.
func summary(cfg config.Config, ix *index.Index) string {
	var w strings.Builder
	scopes := map[string]int{}
	var decls, examples, a11y, designPages int
	var noReadme, noA11y []string
	for _, p := range ix.Packages {
		scopes[p.Scope]++
		decls += len(p.Decls)
		examples += len(p.Examples)
		a11y += len(p.A11y)
		if p.Design != nil {
			designPages++
		}
		if !p.HasReadme {
			noReadme = append(noReadme, p.Name)
		}
		if len(p.A11y) == 0 {
			noA11y = append(noA11y, p.Name)
		}
	}
	var perScope []string
	for _, s := range slices.Sorted(maps.Keys(scopes)) {
		perScope = append(perScope, fmt.Sprintf("%s %d", s, scopes[s]))
	}
	fmt.Fprintf(&w, "source:        %s (%s)\n", ix.SourceName, cfg.Root())
	fmt.Fprintf(&w, "packages:      %d (%s)\n", len(ix.Packages), strings.Join(perScope, ", "))
	fmt.Fprintf(&w, "declarations:  %d\nexamples:      %d\na11y notes:    %d\n", decls, examples, a11y)
	if ix.HasDesign() {
		fmt.Fprintf(&w, "design:        %d packages with guidelines, %d foundation pages (snapshot %s, %s)\n",
			designPages, len(ix.Foundations), ix.DesignFetchedAt.Format("2006-01-02"), ix.DesignDir)
	} else {
		fmt.Fprintf(&w, "design:        no snapshot loaded from %s\n", cfg.DesignCache)
	}
	if len(noReadme) > 0 {
		fmt.Fprintf(&w, "no README:     %s\n", strings.Join(noReadme, ", "))
	}
	if len(noA11y) > 0 {
		fmt.Fprintf(&w, "no a11y notes: %s\n", strings.Join(noA11y, ", "))
	}
	fmt.Fprintf(&w, "warnings:      %d\n", len(ix.Warnings))
	for _, warn := range ix.Warnings {
		fmt.Fprintf(&w, "  - %s\n", warn)
	}
	return w.String()
}
