// Package source reads raw LeafyGreen package files from disk. Adapters only
// read files; parsing happens in internal/parse and querying in internal/index.
package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Scopes are the npm scopes indexed by lg-mcp.
var Scopes = []string{"@leafygreen-ui", "@lg-chat", "@lg-charts"}

// Per-file and per-package read limits keep a pathological package from
// blowing up memory at startup.
const (
	maxFileBytes    = 512 << 10
	maxFilesPerKind = 200
)

// Source is the seam between on-disk adapters and the index.
type Source interface {
	Name() string
	Packages(ctx context.Context) ([]RawPackage, error)
}

// RawPackage is the unparsed content of one package. File maps are keyed by
// slash-separated paths relative to Dir.
type RawPackage struct {
	Dir         string
	PackageJSON []byte
	Readme      string
	Changelog   string
	TypeFiles   map[string]string
	Stories     map[string]string
	SourceFiles map[string]string
	Warnings    []string
}

// ErrOutsideRoot is returned when a path resolves outside the source root.
var ErrOutsideRoot = errors.New("path escapes source root")

// SafeJoin joins rel onto root and guarantees the result, with symlinks
// resolved, stays inside root. It rejects absolute paths and ".." segments
// outright. The target need not exist; if it does, symlinks are resolved.
func SafeJoin(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", fmt.Errorf("%q: %w", rel, ErrOutsideRoot)
	}
	for _, seg := range strings.FieldsFunc(rel, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return "", fmt.Errorf("%q: %w", rel, ErrOutsideRoot)
		}
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %s: %w", root, err)
	}
	joined := filepath.Join(realRoot, filepath.FromSlash(rel))
	resolved, err := filepath.EvalSymlinks(joined)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return joined, nil
	case err != nil:
		return "", fmt.Errorf("resolve %s: %w", rel, err)
	}
	if !within(realRoot, resolved) {
		return "", fmt.Errorf("%q: %w", rel, ErrOutsideRoot)
	}
	return resolved, nil
}

// within reports whether path is root or below it. Both must be cleaned,
// symlink-resolved absolute paths.
func within(root, path string) bool {
	r, err := filepath.Rel(root, path)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

// readOptional reads a file if it exists. A missing file yields "" and no
// error; other failures are returned.
func readOptional(path string) (string, error) {
	b, err := readCapped(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

func readCapped(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxFileBytes {
		return nil, fmt.Errorf("%s: %d bytes exceeds %d byte limit", path, info.Size(), maxFileBytes)
	}
	return os.ReadFile(path)
}

// collect walks dir (not following directory symlinks) and reads every
// regular file accepted by match, keyed by path relative to dir. File
// symlinks are read only if they resolve inside realRoot.
func collect(ctx context.Context, realRoot, dir string, match func(rel string) bool) (map[string]string, []string, error) {
	files := map[string]string{}
	var warnings []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == dir {
				return fs.SkipAll
			}
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || (strings.HasPrefix(d.Name(), ".") && path != dir) {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !match(rel) {
			return nil
		}
		if len(files) >= maxFilesPerKind {
			warnings = append(warnings, fmt.Sprintf("skipped %s: more than %d matching files", rel, maxFilesPerKind))
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil || !within(realRoot, target) {
				warnings = append(warnings, fmt.Sprintf("skipped %s: symlink escapes source root", rel))
				return nil
			}
		}
		b, err := readCapped(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped %s: %v", rel, err))
			return nil
		}
		files[rel] = string(b)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk %s: %w", dir, err)
	}
	return files, warnings, nil
}

// readPackage reads the files common to both adapters from a resolved
// package directory.
func readPackage(dir string) (RawPackage, error) {
	pj, err := readCapped(filepath.Join(dir, "package.json"))
	if err != nil {
		return RawPackage{}, fmt.Errorf("read package.json: %w", err)
	}
	readme, err := readOptional(filepath.Join(dir, "README.md"))
	if err != nil {
		return RawPackage{}, fmt.Errorf("read README: %w", err)
	}
	changelog, err := readOptional(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		return RawPackage{}, fmt.Errorf("read CHANGELOG: %w", err)
	}
	return RawPackage{Dir: dir, PackageJSON: pj, Readme: readme, Changelog: changelog}, nil
}
