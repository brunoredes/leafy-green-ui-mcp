// Command lg-mcp is a local MCP server exposing the LeafyGreen UI design
// system (components, props, examples, design guidelines, accessibility) to
// MCP clients over stdio.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/brunodonatelli/lg-mcp/internal/config"
	"github.com/brunodonatelli/lg-mcp/internal/design"
	"github.com/brunodonatelli/lg-mcp/internal/index"
	"github.com/brunodonatelli/lg-mcp/internal/source"
	"github.com/brunodonatelli/lg-mcp/internal/tools"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage: lg-mcp [command] [flags]

Commands:
  serve     run the MCP server over stdio (default)
  index     build the index and print a summary
  sync      update sources: git pull the repo clone (repo mode) and,
            with --design, snapshot mongodb.design guidelines
  version   print the version

Flags (env var in brackets):
  --source        node_modules | repo        [LG_MCP_SOURCE]
  --project       project root               [LG_MCP_PROJECT]
  --repo-path     leafygreen-ui clone        [LG_MCP_REPO_PATH]
  --design-cache  design snapshot directory  [LG_MCP_DESIGN_CACHE]
  --max-output    max characters per reply   [LG_MCP_MAX_OUTPUT]
  --log-level     debug|info|warn|error      [LG_MCP_LOG_LEVEL]
  --log-file      log file (default stderr)  [LG_MCP_LOG_FILE]
  --design        sync only: snapshot design guidelines
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "lg-mcp:", err)
		os.Exit(1)
	}
}

// run dispatches subcommands. stdout is used only by non-serve commands;
// while serving, stdout belongs to the MCP transport.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cmd := "serve"
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "help", "-h", "--help":
		_, err := fmt.Fprint(stderr, usage)
		return err
	case "version":
		_, err := fmt.Fprintf(stdout, "lg-mcp %s%s\n", version, commit())
		return err
	case "serve", "index", "sync":
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	cfg, err := config.Load(args, os.Getenv, cwd)
	if err != nil {
		return err
	}
	log, closeLog, err := newLogger(cfg, stderr)
	if err != nil {
		return err
	}
	defer closeLog()
	switch cmd {
	case "index":
		return runIndex(ctx, cfg, log, stdout)
	case "sync":
		return runSync(ctx, cfg, log, stdout)
	}
	return runServe(ctx, cfg, log)
}

func newLogger(cfg config.Config, stderr io.Writer) (*slog.Logger, func(), error) {
	w, closeFn := stderr, func() {}
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		w, closeFn = f, func() { _ = f.Close() }
	}
	log := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)
	return log, closeFn, nil
}

func newSource(cfg config.Config, log *slog.Logger) source.Source {
	if cfg.Source == config.SourceRepo {
		return &source.Repo{Path: cfg.RepoPath, Log: log}
	}
	return &source.NodeModules{Project: cfg.Project, Log: log}
}

func buildIndex(ctx context.Context, cfg config.Config, log *slog.Logger) (*index.Index, error) {
	start := time.Now()
	snap, warns, err := design.Load(ctx, cfg.DesignCache)
	if err != nil {
		return nil, err
	}
	for _, w := range warns {
		log.Warn("design snapshot", "detail", w)
	}
	ix, err := index.Build(ctx, newSource(cfg, log), snap)
	if err != nil {
		return nil, err
	}
	for _, w := range ix.Warnings {
		log.Warn("index", "detail", w)
	}
	log.Info("index built", "source", ix.SourceName, "root", cfg.Root(), "packages", len(ix.Packages),
		"design", ix.HasDesign(), "took", time.Since(start).Round(time.Millisecond))
	return ix, nil
}

func runServe(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	ix, err := buildIndex(ctx, cfg, log)
	if err != nil {
		return err
	}
	if len(ix.Packages) == 0 {
		log.Warn("no LeafyGreen packages found; tools will report not found", "source", ix.SourceName, "root", cfg.Root())
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "lg-mcp", Version: version}, &mcp.ServerOptions{
		Instructions: "LeafyGreen UI design system facts from " + ix.SourceName + ". Call consult_design_system " +
			"first for a UI intent; use the get_* tools to drill down. Never invent props or examples the tools do not return.",
		Logger: log,
	})
	tools.Register(srv, tools.NewHandlers(ix, cfg.MaxOutput))
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func runIndex(ctx context.Context, cfg config.Config, log *slog.Logger, stdout io.Writer) error {
	ix, err := buildIndex(ctx, cfg, log)
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, summary(cfg, ix))
	return err
}

func runSync(ctx context.Context, cfg config.Config, log *slog.Logger, stdout io.Writer) error {
	did := false
	if cfg.RepoPath != "" {
		cmd := exec.CommandContext(ctx, "git", "-C", cfg.RepoPath, "pull", "--ff-only")
		cmd.Stdout, cmd.Stderr = stdout, stdout
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git pull in %s: %w", cfg.RepoPath, err)
		}
		did = true
	}
	if cfg.Design {
		s := &design.Syncer{
			Dir:       cfg.DesignCache,
			BaseURL:   design.DefaultBaseURL,
			Client:    &http.Client{Timeout: 30 * time.Second},
			Delay:     250 * time.Millisecond,
			UserAgent: "lg-mcp/" + version + " (design snapshot)",
			Log:       log,
		}
		m, err := s.Run(ctx)
		if err != nil {
			return fmt.Errorf("design sync: %w", err)
		}
		if _, err := fmt.Fprintf(stdout, "design snapshot: %d pages written to %s\n", len(m.Pages), cfg.DesignCache); err != nil {
			return err
		}
		did = true
	}
	if !did {
		return errors.New("nothing to sync: pass --repo-path to pull the leafygreen-ui clone, and/or --design to snapshot design guidelines")
	}
	return nil
}

func commit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return " (" + s.Value[:7] + ")"
		}
	}
	return ""
}
