// Package config resolves lg-mcp settings from flags and environment
// variables. Flags take precedence over environment variables.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Source modes.
const (
	SourceNodeModules = "node_modules"
	SourceRepo        = "repo"
)

// Config is the resolved, validated configuration.
type Config struct {
	Source      string
	Project     string
	RepoPath    string
	DesignCache string
	MaxOutput   int
	LogLevel    slog.Level
	LogFile     string
	// Design enables the design-docs snapshot in the sync subcommand.
	Design bool
}

// Load parses args (without the program name and subcommand) using getenv
// for defaults. cwd is used as the default project root.
func Load(args []string, getenv func(string) string, cwd string) (Config, error) {
	env := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}
	var (
		cfg      Config
		maxOut   string
		logLevel string
	)
	fs := flag.NewFlagSet("lg-mcp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Source, "source", env("LG_MCP_SOURCE", SourceNodeModules), "node_modules or repo")
	fs.StringVar(&cfg.Project, "project", env("LG_MCP_PROJECT", cwd), "project root containing node_modules")
	fs.StringVar(&cfg.RepoPath, "repo-path", env("LG_MCP_REPO_PATH", ""), "path to a leafygreen-ui clone")
	fs.StringVar(&cfg.DesignCache, "design-cache", env("LG_MCP_DESIGN_CACHE", defaultDesignCache(getenv)), "design docs snapshot directory")
	fs.StringVar(&maxOut, "max-output", env("LG_MCP_MAX_OUTPUT", "12000"), "max characters per tool response")
	fs.StringVar(&logLevel, "log-level", env("LG_MCP_LOG_LEVEL", "info"), "debug, info, warn or error")
	fs.StringVar(&cfg.LogFile, "log-file", env("LG_MCP_LOG_FILE", ""), "write logs to this file instead of stderr")
	fs.BoolVar(&cfg.Design, "design", false, "sync: also snapshot mongodb.design pages")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	n, err := strconv.Atoi(maxOut)
	if err != nil {
		return Config{}, fmt.Errorf("max-output %q: %w", maxOut, err)
	}
	cfg.MaxOutput = n
	if err := cfg.LogLevel.UnmarshalText([]byte(logLevel)); err != nil {
		return Config{}, fmt.Errorf("log-level %q: %w", logLevel, err)
	}
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	var errs []error
	switch c.Source {
	case SourceNodeModules:
		if c.Project == "" {
			errs = append(errs, errors.New("project is required in node_modules mode"))
		}
	case SourceRepo:
		if c.RepoPath == "" {
			errs = append(errs, errors.New("repo-path is required in repo mode (flag --repo-path or LG_MCP_REPO_PATH)"))
		}
	default:
		errs = append(errs, fmt.Errorf("source %q: want %s or %s", c.Source, SourceNodeModules, SourceRepo))
	}
	if c.MaxOutput < 500 {
		errs = append(errs, fmt.Errorf("max-output %d: must be at least 500", c.MaxOutput))
	}
	return errors.Join(errs...)
}

// Root returns the directory the configured source reads from.
func (c Config) Root() string {
	if c.Source == SourceRepo {
		return c.RepoPath
	}
	return c.Project
}

func defaultDesignCache(getenv func(string) string) string {
	if dir := getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "lg-mcp", "design")
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "lg-mcp", "design")
	}
	return ""
}
