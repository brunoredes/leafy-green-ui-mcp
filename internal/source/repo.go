package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// repoDirs are the top-level directories of a leafygreen-ui clone that hold
// published packages.
var repoDirs = []string{"packages", "chat", "charts"}

// Repo reads packages from a local clone of mongodb/leafygreen-ui.
type Repo struct {
	Path string
	Log  *slog.Logger
}

// Name implements Source.
func (r *Repo) Name() string { return "repo" }

// Packages implements Source.
func (r *Repo) Packages(ctx context.Context) ([]RawPackage, error) {
	realRoot, err := filepath.EvalSymlinks(r.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve repo %s: %w", r.Path, err)
	}
	var pkgs []RawPackage
	for _, top := range repoDirs {
		entries, err := os.ReadDir(filepath.Join(realRoot, top))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", top, err)
		}
		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !e.IsDir() {
				continue
			}
			p, ok := r.readOne(ctx, realRoot, top+"/"+e.Name())
			if ok {
				pkgs = append(pkgs, p)
			}
		}
	}
	return pkgs, nil
}

func (r *Repo) readOne(ctx context.Context, realRoot, rel string) (RawPackage, bool) {
	dir, err := SafeJoin(realRoot, rel)
	if err != nil {
		r.log().Warn("skip package", "path", rel, "err", err)
		return RawPackage{}, false
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		return RawPackage{}, false
	}
	p, err := readPackage(dir)
	if err != nil {
		r.log().Warn("skip package", "path", rel, "err", err)
		return RawPackage{}, false
	}
	src := filepath.Join(dir, "src")
	kinds := []struct {
		dst   *map[string]string
		match func(string) bool
	}{
		{&p.TypeFiles, isTypeFile},
		{&p.Stories, func(f string) bool { return strings.HasSuffix(f, ".stories.tsx") }},
		{&p.SourceFiles, isComponentSource},
	}
	for _, k := range kinds {
		files, warns, err := collect(ctx, realRoot, src, k.match)
		if err != nil {
			r.log().Warn("read sources", "path", rel, "err", err)
			continue
		}
		*k.dst = prefixKeys(files, "src/")
		p.Warnings = append(p.Warnings, warns...)
	}
	return p, true
}

func isComponentSource(rel string) bool {
	if !strings.HasSuffix(rel, ".tsx") {
		return false
	}
	for _, skip := range []string{".stories.", ".spec.", ".test.", ".testutils."} {
		if strings.Contains(rel, skip) {
			return false
		}
	}
	return true
}

func prefixKeys(m map[string]string, prefix string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[prefix+k] = v
	}
	return out
}

func (r *Repo) log() *slog.Logger {
	if r.Log != nil {
		return r.Log
	}
	return slog.Default()
}

// isTypeFile accepts "*.types.ts" and plain "types.ts" files, the two
// conventions LeafyGreen packages use for prop declarations.
func isTypeFile(rel string) bool {
	return strings.HasSuffix(rel, ".types.ts") || rel == "types.ts" || strings.HasSuffix(rel, "/types.ts")
}
