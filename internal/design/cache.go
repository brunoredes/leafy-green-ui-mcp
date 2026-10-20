// Package design reads (and, from the sync subcommand only, writes) an
// offline snapshot of mongodb.design guideline pages. The server never
// fetches at runtime; it only reads the snapshot directory.
package design

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Page kinds.
const (
	KindComponent  = "component"
	KindFoundation = "foundation"
)

// ManifestFile is the snapshot index file name.
const ManifestFile = "manifest.json"

// Page describes one snapshotted page.
type Page struct {
	Slug string `json:"slug"`
	Kind string `json:"kind"`
	URL  string `json:"url"`
	File string `json:"file"`
}

// Manifest lists the pages in a snapshot.
type Manifest struct {
	FetchedAt time.Time `json:"fetched_at"`
	Pages     []Page    `json:"pages"`
}

// Doc is a page with its extracted Markdown.
type Doc struct {
	Page
	Markdown string
}

// Snapshot is a loaded design snapshot. It is read-only after Load.
type Snapshot struct {
	Dir         string
	FetchedAt   time.Time
	Components  map[string]Doc
	Foundations map[string]Doc
}

// Empty reports whether the snapshot holds no pages.
func (s *Snapshot) Empty() bool {
	return s == nil || len(s.Components)+len(s.Foundations) == 0
}

// Load reads the snapshot in dir. A missing directory or manifest is not an
// error: it yields an empty snapshot and a warning, because design docs are
// optional. Unreadable pages are skipped with a warning.
func Load(ctx context.Context, dir string) (*Snapshot, []string, error) {
	snap := &Snapshot{Dir: dir, Components: map[string]Doc{}, Foundations: map[string]Doc{}}
	if dir == "" {
		return snap, []string{"no design cache directory configured"}, nil
	}
	b, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return snap, []string{fmt.Sprintf("no design snapshot in %s; run `lg-mcp sync --design` to create one", dir)}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read design manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, nil, fmt.Errorf("parse design manifest %s: %w", dir, err)
	}
	snap.FetchedAt = m.FetchedAt
	var warnings []string
	for _, p := range m.Pages {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		doc, err := loadPage(dir, p)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		switch p.Kind {
		case KindComponent:
			snap.Components[p.Slug] = doc
		case KindFoundation:
			snap.Foundations[p.Slug] = doc
		default:
			warnings = append(warnings, fmt.Sprintf("design page %s: unknown kind %q", p.Slug, p.Kind))
		}
	}
	return snap, warnings, nil
}

func loadPage(dir string, p Page) (Doc, error) {
	if !validFileName(p.File) {
		return Doc{}, fmt.Errorf("design page %s: unsafe file name %q", p.Slug, p.File)
	}
	b, err := os.ReadFile(filepath.Join(dir, p.File))
	if err != nil {
		return Doc{}, fmt.Errorf("design page %s: %w", p.Slug, err)
	}
	return Doc{Page: p, Markdown: string(b)}, nil
}

// validFileName accepts only plain file names inside the snapshot dir.
func validFileName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`) && filepath.Base(name) == name
}
