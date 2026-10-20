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

// NodeModules reads installed packages from <project>/node_modules.
type NodeModules struct {
	Project string
	Log     *slog.Logger
}

// Name implements Source.
func (n *NodeModules) Name() string { return "node_modules" }

// Packages implements Source. Package directories may be symlinks (pnpm);
// they are followed only when they resolve inside the project root.
func (n *NodeModules) Packages(ctx context.Context) ([]RawPackage, error) {
	realRoot, err := filepath.EvalSymlinks(n.Project)
	if err != nil {
		return nil, fmt.Errorf("resolve project %s: %w", n.Project, err)
	}
	var pkgs []RawPackage
	for _, scope := range Scopes {
		scopeDir, err := SafeJoin(realRoot, "node_modules/"+scope)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(scopeDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", scopeDir, err)
		}
		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			p, ok := n.readOne(ctx, realRoot, scope, e.Name())
			if ok {
				pkgs = append(pkgs, p)
			}
		}
	}
	return pkgs, nil
}

func (n *NodeModules) readOne(ctx context.Context, realRoot, scope, name string) (RawPackage, bool) {
	rel := "node_modules/" + scope + "/" + name
	dir, err := SafeJoin(realRoot, rel)
	if err != nil {
		n.log().Warn("skip package", "path", rel, "err", err)
		return RawPackage{}, false
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return RawPackage{}, false
	}
	p, err := readPackage(dir)
	if err != nil {
		n.log().Warn("skip package", "path", rel, "err", err)
		return RawPackage{}, false
	}
	types, warns, err := collect(ctx, realRoot, dir, func(r string) bool { return strings.HasSuffix(r, ".d.ts") })
	if err != nil {
		n.log().Warn("read types", "path", rel, "err", err)
	}
	p.TypeFiles = types
	p.Warnings = append(p.Warnings, warns...)
	return p, true
}

func (n *NodeModules) log() *slog.Logger {
	if n.Log != nil {
		return n.Log
	}
	return slog.Default()
}
