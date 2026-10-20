package design

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DefaultBaseURL is the design site snapshotted by Sync.
const DefaultBaseURL = "https://www.mongodb.design"

// seedFoundation is fetched first; its navigation lists every component and
// foundation page, so no hard-coded component list is needed.
const seedFoundation = "accessibility"

const maxPageBytes = 8 << 20

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Syncer snapshots design pages into Dir. It is used only by the
// `lg-mcp sync --design` subcommand.
type Syncer struct {
	Dir       string
	BaseURL   string
	Client    *http.Client
	Delay     time.Duration // pause between requests
	UserAgent string
	Log       *slog.Logger
	Now       func() time.Time
}

// Run fetches the seed foundation page, discovers all component and
// foundation pages from its navigation, fetches each one sequentially and
// writes Markdown files plus the manifest. A page that fails is logged and
// skipped; Run fails only if nothing could be fetched or written.
func (s *Syncer) Run(ctx context.Context) (Manifest, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("create design cache: %w", err)
	}
	seed, err := s.fetch(ctx, "/foundation/"+seedFoundation)
	if err != nil {
		return Manifest{}, fmt.Errorf("fetch seed page: %w", err)
	}
	m := Manifest{FetchedAt: s.now().UTC()}
	targets := s.targets(seed)
	for i, t := range targets {
		if err := ctx.Err(); err != nil {
			return Manifest{}, err
		}
		pg := seed
		if t.Kind != KindFoundation || t.Slug != seedFoundation {
			if i > 0 && s.Delay > 0 {
				time.Sleep(s.Delay)
			}
			pg, err = s.fetch(ctx, t.path)
			if err != nil {
				s.log().Warn("skip design page", "url", t.URL, "err", err)
				continue
			}
		}
		if strings.TrimSpace(pg.markdown) == "" {
			s.log().Warn("skip design page: no content extracted", "url", t.URL)
			continue
		}
		if err := s.write(t.File, withTitle(pg.markdown, t.Slug)); err != nil {
			return Manifest{}, err
		}
		m.Pages = append(m.Pages, t.Page)
	}
	if len(m.Pages) == 0 {
		return Manifest{}, fmt.Errorf("no design pages could be fetched from %s", s.BaseURL)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return Manifest{}, fmt.Errorf("encode manifest: %w", err)
	}
	return m, s.write(ManifestFile, string(b)+"\n")
}

type target struct {
	Page
	path string
}

func (s *Syncer) targets(seed page) []target {
	var out []target
	add := func(kind, slug, path, file string) {
		if !slugRe.MatchString(slug) {
			s.log().Warn("skip design page: unsafe slug", "slug", slug)
			return
		}
		out = append(out, target{Page: Page{Slug: slug, Kind: kind, URL: s.BaseURL + path, File: file}, path: path})
	}
	for _, slug := range seed.foundations {
		add(KindFoundation, slug, "/foundation/"+slug, "foundation-"+slug+".md")
	}
	for _, slug := range seed.components {
		add(KindComponent, slug, "/component/"+slug+"/design-docs", slug+".md")
	}
	return out
}

func (s *Syncer) fetch(ctx context.Context, path string) (page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+path, nil)
	if err != nil {
		return page{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Accept", "text/html")
	resp, err := s.Client.Do(req)
	if err != nil {
		return page{}, fmt.Errorf("get %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return page{}, fmt.Errorf("get %s: HTTP %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return page{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parsePage(bytes.NewReader(body))
}

// write replaces name inside Dir atomically (temp file + rename).
func (s *Syncer) write(name, content string) error {
	if !validFileName(name) {
		return fmt.Errorf("unsafe design file name %q", name)
	}
	tmp, err := os.CreateTemp(s.Dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.Dir, name)); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func (s *Syncer) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Syncer) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// withTitle prefixes md with a level-1 heading derived from slug when the
// page has none (component design-docs pages put the title outside the
// content area).
func withTitle(md, slug string) string {
	if strings.HasPrefix(md, "# ") {
		return md
	}
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return "# " + strings.Join(words, " ") + "\n\n" + md
}
