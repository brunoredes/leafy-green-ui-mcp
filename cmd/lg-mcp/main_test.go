package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "lg-mcp.log")
	tests := []struct {
		name       string
		args       []string
		wantErr    string
		wantStdout []string
	}{
		{name: "version", args: []string{"version"}, wantStdout: []string{"lg-mcp dev"}},
		{name: "unknown command", args: []string{"bogus"}, wantErr: `unknown command "bogus"`},
		{name: "bad flag", args: []string{"index", "--source", "npm"}, wantErr: `source "npm"`},
		{
			name: "index summary",
			args: []string{"index", "--source", "repo", "--repo-path", "../../testdata/repo",
				"--design-cache", "../../testdata/design", "--log-file", logFile},
			wantStdout: []string{"packages:      8 (leafygreen-ui 7, lg-chat 1)", "design:        2 packages with guidelines", "warnings:      0"},
		},
		{
			name:    "sync with nothing to do",
			args:    []string{"sync", "--project", "../../testdata/node_modules", "--log-file", logFile},
			wantErr: "nothing to sync",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(context.Background(), tt.args, &stdout, &stderr)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("stdout missing %q:\n%s", want, stdout.String())
				}
			}
		})
	}
}
