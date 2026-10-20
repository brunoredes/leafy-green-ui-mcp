package source

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mustMkdir(t, filepath.Join(root, "pkg"))
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "pkg"), filepath.Join(root, "inside")); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		rel     string
		wantErr bool
	}{
		{name: "plain", rel: "pkg"},
		{name: "missing is allowed", rel: "pkg/nope.md"},
		{name: "symlink inside root", rel: "inside"},
		{name: "dotdot", rel: "pkg/../../etc", wantErr: true},
		{name: "leading dotdot", rel: "../x", wantErr: true},
		{name: "absolute", rel: "/etc/passwd", wantErr: true},
		{name: "empty", rel: "", wantErr: true},
		{name: "symlink escape", rel: "escape", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeJoin(root, tt.rel)
			if tt.wantErr {
				if !errors.Is(err, ErrOutsideRoot) {
					t.Fatalf("SafeJoin(%q) = %q, %v; want ErrOutsideRoot", tt.rel, got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SafeJoin(%q): %v", tt.rel, err)
			}
		})
	}
}

func TestRepoPackages(t *testing.T) {
	pkgs, err := (&Repo{Path: "../../testdata/repo"}).Packages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]RawPackage{}
	for _, p := range pkgs {
		byName[filepath.Base(p.Dir)] = p
	}
	for _, want := range []string{"button", "combobox", "modal", "confirmation-modal", "a11y", "leafygreen-provider", "message", "toggle"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("missing package %s", want)
		}
	}
	cb := byName["combobox"]
	if !strings.Contains(cb.Readme, "# Combobox") {
		t.Errorf("combobox README not read")
	}
	if _, ok := cb.TypeFiles["src/Combobox/Combobox.types.ts"]; !ok {
		t.Errorf("combobox types not read: %v", keys(cb.TypeFiles))
	}
	if _, ok := cb.Stories["src/Combobox.stories.tsx"]; !ok {
		t.Errorf("combobox stories not read: %v", keys(cb.Stories))
	}
	if _, ok := cb.SourceFiles["src/Combobox/Combobox.tsx"]; !ok || len(cb.SourceFiles) != 1 {
		t.Errorf("combobox sources = %v", keys(cb.SourceFiles))
	}
	if byName["message"].Readme != "" {
		t.Errorf("message should have no README")
	}
}

func TestNodeModulesPackages(t *testing.T) {
	pkgs, err := (&NodeModules{Project: "../../testdata/node_modules"}).Packages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	for _, p := range pkgs {
		if len(p.TypeFiles) != 1 {
			t.Errorf("%s: type files = %v", p.Dir, keys(p.TypeFiles))
		}
	}
}

func TestNodeModulesSkipsEscapingPackage(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "package.json"), `{"name":"@leafygreen-ui/evil"}`)
	scope := filepath.Join(project, "node_modules", "@leafygreen-ui")
	mustMkdir(t, scope)
	if err := os.Symlink(outside, filepath.Join(scope, "evil")); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(scope, "ok", "package.json"), `{"name":"@leafygreen-ui/ok"}`)
	mustWrite(t, filepath.Join(scope, "ok", "dist", "x.d.ts"), `export type X = 1;`)
	if err := os.Symlink(filepath.Join(outside, "package.json"), filepath.Join(scope, "ok", "dist", "leak.d.ts")); err != nil {
		t.Fatal(err)
	}
	pkgs, err := (&NodeModules{Project: project}).Packages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || filepath.Base(pkgs[0].Dir) != "ok" {
		t.Fatalf("packages = %+v", pkgs)
	}
	if _, ok := pkgs[0].TypeFiles["dist/leak.d.ts"]; ok {
		t.Fatalf("read a file through an escaping symlink")
	}
	if len(pkgs[0].Warnings) == 0 {
		t.Fatalf("expected a warning for the escaping symlink")
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsTypeFile(t *testing.T) {
	tests := map[string]bool{
		"types.ts": true, "Button/Button.types.ts": true, "Button/types.ts": true,
		"mytypes.ts": false, "types.tsx": false, "Button.tsx": false,
	}
	for in, want := range tests {
		if got := isTypeFile(in); got != want {
			t.Errorf("isTypeFile(%q) = %v, want %v", in, got, want)
		}
	}
}
