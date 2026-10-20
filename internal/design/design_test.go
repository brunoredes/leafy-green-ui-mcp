package design

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	snap, warns, err := Load(context.Background(), "../../testdata/design")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if len(snap.Components) != 2 || len(snap.Foundations) != 1 {
		t.Fatalf("components %d, foundations %d", len(snap.Components), len(snap.Foundations))
	}
	if !strings.Contains(snap.Components["combobox"].Markdown, "## How to use this component") {
		t.Fatalf("combobox markdown not loaded")
	}
	if snap.FetchedAt.IsZero() {
		t.Fatalf("FetchedAt not set")
	}
}

func TestLoadMissingIsWarning(t *testing.T) {
	snap, warns, err := Load(context.Background(), filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Empty() || len(warns) != 1 || !strings.Contains(warns[0], "lg-mcp sync --design") {
		t.Fatalf("snap empty=%v warns=%v", snap.Empty(), warns)
	}
}

func TestLoadRejectsUnsafeFile(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"pages":[{"slug":"x","kind":"component","file":"../secret.md"},{"slug":"y","kind":"component","file":"y.md"}]}`
	writeFile(t, filepath.Join(dir, ManifestFile), manifest)
	writeFile(t, filepath.Join(dir, "y.md"), "# Y\n")
	snap, warns, err := Load(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snap.Components["x"]; ok || len(snap.Components) != 1 {
		t.Fatalf("components = %v", snap.Components)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "unsafe file name") {
		t.Fatalf("warns = %v", warns)
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		file        string
		want        []string
		notWant     []string
		comps       string
		foundations string
	}{
		{
			file: "foundation-accessibility.html",
			want: []string{
				"# Accessibility\n\nAccessible design helps everyone.",
				"## Keyboard navigation\n\nAll components should be reachable with the keyboard. Use `tabIndex` sparingly.",
				"## Contrast\n\n- Text: 4.5:1\n- Components: 3:1",
			},
			notWant:     []string{"Anatomy", "Component list", "not content", "Copy link"},
			comps:       "combobox,modal",
			foundations: "accessibility,grid",
		},
		{
			file: "component-combobox.html",
			want: []string{
				"## Overview\n\nComboboxes let users",
				"| Alternative | When |\n| --- | --- |\n| Select | Short lists \\| under ten |",
				"## How to use this component\n\nUse multiselect for tags.",
			},
			notWant: []string{"Live Example", "Design Documentation"},
			comps:   "combobox",
		},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			f, err := os.Open(filepath.Join("../../testdata/design-html", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			p, err := parsePage(f)
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(p.markdown, w) {
					t.Errorf("markdown missing %q:\n%s", w, p.markdown)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(p.markdown, nw) {
					t.Errorf("markdown contains %q:\n%s", nw, p.markdown)
				}
			}
			if got := strings.Join(p.components, ","); got != tt.comps {
				t.Errorf("components = %s, want %s", got, tt.comps)
			}
			if got := strings.Join(p.foundations, ","); got != tt.foundations {
				t.Errorf("foundations = %s, want %s", got, tt.foundations)
			}
		})
	}
}

func TestSyncerRun(t *testing.T) {
	pages := map[string]string{
		"/foundation/accessibility":       "foundation-accessibility.html",
		"/component/combobox/design-docs": "component-combobox.html",
	}
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		if r.Header.Get("User-Agent") != "lg-mcp-test" {
			t.Errorf("missing user agent on %s", r.URL.Path)
		}
		file, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join("../../testdata/design-html", file))
	}))
	defer srv.Close()

	dir := filepath.Join(t.TempDir(), "design")
	s := &Syncer{
		Dir: dir, BaseURL: srv.URL, Client: srv.Client(), UserAgent: "lg-mcp-test",
		Now: func() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) },
	}
	m, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// grid and modal 404 and are skipped; the seed is not fetched twice.
	var slugs []string
	for _, p := range m.Pages {
		slugs = append(slugs, p.Kind+":"+p.Slug)
	}
	if got := strings.Join(slugs, ","); got != "foundation:accessibility,component:combobox" {
		t.Fatalf("pages = %s", got)
	}
	if n := strings.Count(strings.Join(requests, ","), "/foundation/accessibility"); n != 1 {
		t.Fatalf("seed fetched %d times", n)
	}

	snap, warns, err := Load(context.Background(), dir)
	if err != nil || len(warns) != 0 {
		t.Fatalf("Load after sync: %v %v", err, warns)
	}
	cb := snap.Components["combobox"].Markdown
	if !strings.HasPrefix(cb, "# Combobox\n\n## Overview") {
		t.Fatalf("component page missing title:\n%s", cb)
	}
	if !snap.FetchedAt.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("FetchedAt = %v", snap.FetchedAt)
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".tmp-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp files left behind: %v", leftovers)
	}
}

func TestSyncerRunSeedFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	s := &Syncer{Dir: t.TempDir(), BaseURL: srv.URL, Client: srv.Client()}
	if _, err := s.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "seed") {
		t.Fatalf("err = %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
