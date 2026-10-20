# AGENTS.md

Instructions for coding agents (Claude Code, Codex, Cursor, etc.) working in this repository. Read this before making changes. `README.md` describes what the project does; this file describes how to work on it.

## Project in one paragraph

`lg-mcp` is a local MCP server in Go that exposes MongoDB's LeafyGreen UI design system (components, props, examples, tokens, changelogs) to MCP clients over stdio. It indexes either a project's `node_modules/@leafygreen-ui|@lg-chat|@lg-charts/*` or a local clone of `mongodb/leafygreen-ui` into memory at startup, then serves read-only tools. There is no database and no network access at runtime.

## Commands

```bash
make build      # build ./bin/lg-mcp
make test       # go test ./... -race
make lint       # golangci-lint run
make fmt        # gofmt + goimports
make index      # run the indexer against testdata and print a summary
make inspect    # launch the MCP Inspector against the binary
```

Run `make fmt lint test` before declaring any task complete. All three must pass.

## Stack

- Go, version pinned in `go.mod`. Do not lower it.
- MCP SDK: `github.com/modelcontextprotocol/go-sdk` (package `mcp`). Do not introduce `mark3labs/mcp-go` or a second SDK.
- Logging: `log/slog` from the standard library.
- Prefer the standard library. Any new third-party dependency needs a one-line justification in the PR/commit message. Do not add dependencies for things the stdlib covers (HTTP, JSON, flags, filepath walking, testing).

## Architecture

```
cmd/lg-mcp            wiring only: config → source → index → server
internal/config       flag/env resolution and validation
internal/source       Source interface; nodemodules.go and repo.go adapters
internal/design       design-guideline snapshot: cache.go (read, runtime) and sync.go (write, sync only)
internal/index        Package model, in-memory index, lexical search
internal/parse/...    markdown, TypeScript-declaration and a11y-source extraction (pure functions)
internal/tools        one file per tool + register.go
internal/resources    MCP resource handlers
internal/output       truncation and Markdown formatting
testdata/             fixtures; the only filesystem tests may touch
```

Dependency direction is one-way: `cmd → tools/resources → index → source/design/parse`. `parse` and `output` import nothing internal. Do not create import cycles or have `index` know about MCP types. Only `cmd` may use `design.Syncer` (the one HTTP client in the codebase).

The `Source` interface is the seam between adapters and the index:

```go
type Source interface {
    Name() string
    Packages(ctx context.Context) ([]RawPackage, error)
}
```

Adapters only read files and return raw contents. All parsing happens in `parse`, all querying in `index`.

## Hard rules

1. **Never write to stdout** outside the MCP transport. No `fmt.Print*`, no `log.Print*` with default output, no `os.Stdout` writes. stdout carries JSON-RPC; any stray byte breaks the client connection. Use `slog` configured to stderr or the log file.
2. **No network access at runtime.** Tools must not fetch from GitHub, npm, or mongodb.design. The only network operations live in the explicit `lg-mcp sync` subcommand: it shells out to `git` (repo mode) and, with `--design`, makes HTTP GET requests to mongodb.design to refresh the local design snapshot. Nothing else may open a network connection.
3. **All tools are read-only.** No tool writes to the user's filesystem or modifies `node_modules` or the repo clone.
4. **Every tool response goes through `output.Truncate`.** No tool returns unbounded text.
5. **Path safety.** Any file path derived from tool input (e.g. a package name) must be resolved and checked to stay inside the configured source root. Reject `..`, absolute paths and symlinks escaping the root.
6. **Do not invent LeafyGreen data.** If a README, type or token is missing, the tool says so explicitly ("No props declaration found for X"). Never fill gaps with guessed props or examples.
7. **Tests never depend on the real LeafyGreen packages or the network.** Use `testdata/`. If a test needs a new case, add a minimal fixture.

## Conventions

### Go

- Errors: wrap with context, `fmt.Errorf("index %s: %w", name, err)`. No `panic` outside `main` setup.
- Pass `context.Context` as the first argument to anything that does I/O.
- No package-level mutable state. The index is built once and passed by pointer; it is read-only after construction, so tools can run concurrently without locks.
- Exported identifiers get doc comments. Unexported ones only when non-obvious.
- Table-driven tests with `t.Run`. Use `t.TempDir()` for anything that writes.
- Keep functions under ~60 lines; split parsing logic into small pure functions that are easy to test.

### Tools

Each tool lives in `internal/tools/<snake_name>.go` and follows this shape:

```go
type GetComponentInput struct {
    Name string `json:"name" jsonschema:"component or package name, e.g. button or @leafygreen-ui/button"`
}

func (h *Handlers) GetComponent(ctx context.Context, req *mcp.CallToolRequest, in GetComponentInput) (*mcp.CallToolResult, any, error) {
    // 1. normalize input  2. query index  3. format  4. truncate
}
```

- Tool names: `snake_case`, verb first (`get_`, `list_`, `search_`).
- Descriptions are written for the model, not for humans: say when to use the tool and what it returns, in one or two sentences. Tool descriptions are part of the product; review them as carefully as code.
- Name inputs leniently: accept `button`, `Button`, `@leafygreen-ui/button`. Normalize in one shared helper (`index.NormalizeName`).
- On "not found", return a normal result (not a protocol error) listing the closest matches, so the model can retry.
- Output format is Markdown with a stable structure. Changing a tool's output format requires updating its golden file.

### Parsing

- Markdown: split by headings, extract fenced code blocks with their language tag. Do not render Markdown.
- TypeScript: extract exported `interface` and `type` declarations with their preceding JSDoc. This is text-level, not a full parser. Handle nested braces and generics; record `extends` clauses verbatim without resolving them. If extraction becomes unreliable, stop and raise it rather than adding increasingly fragile regexes.

## Testing

- Unit tests sit next to the code (`foo_test.go`).
- Tool tests use golden files under `internal/tools/testdata/*.golden.md`. Regenerate with `go test ./internal/tools/... -update`, then read the diff before committing.
- `internal/server_test.go` (or equivalent) runs an in-memory MCP client against the server and calls every registered tool at least once.
- New behavior without a test is not done.

## Definition of done

- [ ] `make fmt lint test` passes
- [ ] New or changed tool has tests and an updated golden file
- [ ] `README.md` Tools/Configuration tables updated if the public surface changed
- [ ] No new stdout writes, network calls, or unbounded outputs
- [ ] `lg-mcp index` against `testdata/` shows no new warnings

## Things to ask the human about instead of deciding

- Adding a third-party dependency
- Adding a write-capable tool or any network access
- Changing a tool's name or input schema (breaks existing client usage)
- Introducing a Node/TypeScript sidecar for type resolution
- Changing the default source mode

## Out of scope

- HTTP/SSE transport, authentication, multi-user deployment
- Rendering or screenshotting components
- Generating application code; the server supplies facts, the client writes code