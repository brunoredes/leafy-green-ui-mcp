package config

import (
	"log/slog"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		check   func(t *testing.T, c Config)
		wantErr string
	}{
		{
			name: "defaults",
			check: func(t *testing.T, c Config) {
				if c.Source != SourceNodeModules || c.Project != "/work" || c.MaxOutput != 12000 || c.LogLevel != slog.LevelInfo {
					t.Fatalf("unexpected defaults: %+v", c)
				}
				if c.DesignCache != "/cache/lg-mcp/design" {
					t.Fatalf("design cache = %q", c.DesignCache)
				}
			},
		},
		{
			name: "env used",
			env:  map[string]string{"LG_MCP_SOURCE": "repo", "LG_MCP_REPO_PATH": "/lg", "LG_MCP_LOG_LEVEL": "debug"},
			check: func(t *testing.T, c Config) {
				if c.Source != SourceRepo || c.Root() != "/lg" || c.LogLevel != slog.LevelDebug {
					t.Fatalf("env not applied: %+v", c)
				}
			},
		},
		{
			name: "flag beats env",
			args: []string{"--max-output", "800"},
			env:  map[string]string{"LG_MCP_MAX_OUTPUT": "900"},
			check: func(t *testing.T, c Config) {
				if c.MaxOutput != 800 {
					t.Fatalf("max output = %d", c.MaxOutput)
				}
			},
		},
		{name: "repo without path", args: []string{"--source", "repo"}, wantErr: "repo-path is required"},
		{name: "bad source", args: []string{"--source", "npm"}, wantErr: `source "npm"`},
		{name: "bad max", args: []string{"--max-output", "x"}, wantErr: "max-output"},
		{name: "tiny max", args: []string{"--max-output", "10"}, wantErr: "at least 500"},
		{name: "bad level", args: []string{"--log-level", "loud"}, wantErr: "log-level"},
		{name: "stray arg", args: []string{"extra"}, wantErr: "unexpected arguments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]string{"XDG_CACHE_HOME": "/cache"}
			for k, v := range tt.env {
				env[k] = v
			}
			c, err := Load(tt.args, func(k string) string { return env[k] }, "/work")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			tt.check(t, c)
		})
	}
}
