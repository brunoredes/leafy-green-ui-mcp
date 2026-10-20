# lg-mcp

A local [Model Context Protocol](https://modelcontextprotocol.io) server, written in Go, that gives Claude Code (or any MCP client) structured, version-accurate access to MongoDB's [LeafyGreen UI](https://github.com/mongodb/leafygreen-ui) design system.

Describe what the UI should do ("let users pick several tags from a long list") and `consult_design_system` crosses four sources for the best-matching components and returns one compact answer:

1. **Design documentation**: usage guidance from mongodb.design, read from a local snapshot.
2. **Code documentation**: package, version, import, and prop interfaces with JSDoc.
3. **Code examples**: from READMEs and Storybook stories.
4. **Accessibility**: focus, keyboard and ARIA notes, required accessible names, and roles and attributes set in source.

> **Status:** personal project, pre-1.0. Not affiliated with or endorsed by MongoDB.

---

## Table of contents

- [Why this exists](#why-this-exists)
- [How it works](#how-it-works)
- [Requirements](#requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Using it with Claude Code](#using-it-with-claude-code)
- [Tools](#tools)
- [Resources](#resources)
- [Project layout](#project-layout)
- [Development](#development)
- [Testing](#testing)
- [Debugging](#debugging)
- [Design decisions](#design-decisions)
- [Limitations](#limitations)
- [Roadmap](#roadmap)
- [License](#license)

---

## Why this exists

When Claude Code writes LeafyGreen code without help, it tends to:

- import components from the wrong package (`@leafygreen-ui/button` default vs named export, `@lg-chat/*`, `@lg-charts/*`);
- invent props or use props from an older major version;
- hard-code colors and spacing instead of using `@leafygreen-ui/tokens` / `@leafygreen-ui/palette`;
- miss required providers (`LeafyGreenProvider`, `darkMode` handling).

This server answers those questions from the actual source instead of model memory, and keeps answers small enough not to flood the context window.

### Before building: check what already exists

The LeafyGreen repository ships agent skills (`setup-leafygreen`, `leafygreen-authoring`, installable with `npx skills add https://github.com/mongodb/leafygreen-ui --skill <name>`) and has an `mcp/` directory at its root. That directory holds MCP-UI pieces (`embeddable-uis`, `hooks`, `ui-resource-renderer`), not a documentation server. This project covers what those don't: version-pinned answers from your project's own `node_modules`, prop and type lookup, and design guidance and accessibility facts cross-referenced for a stated intent.

---

## How it works

```
┌──────────────┐  stdio (JSON-RPC)  ┌───────────────────────────┐
│ Claude Code  │ ◄────────────────► │ lg-mcp                    │
└──────────────┘                    │                           │
                                    │  tools ──► index (memory) │
                                    │              ▲            │
                                    │              │ built at   │
                                    │              │ startup    │
                                    │   source adapter          │
                                    │   ├─ node_modules (default)
                                    │   └─ repo (local clone)   │
                                    └───────────────────────────┘
```

1. On startup, a **source adapter** walks either your project's `node_modules/@leafygreen-ui|@lg-chat|@lg-charts/*` or a local clone of `mongodb/leafygreen-ui`.
2. It reads each package's `package.json`, `README.md`, `CHANGELOG.md` and type declarations (`*.d.ts`, or `src/**/*.types.ts` and `src/**/types.ts`). In repo mode it also reads `*.stories.tsx` and component source, which is scanned for accessibility behaviour.
3. If a **design snapshot** exists (created once with `lg-mcp sync --design`), mongodb.design guideline pages are attached to the matching packages, and foundation pages such as Accessibility are loaded.
4. Everything is held in an **in-memory index**. No database, and no network access at runtime. Indexing the full repository plus the snapshot takes about 0.5 s.
5. **Tools** query the index and return compact, truncated Markdown.

### Source modes

| Mode | Reads from | Pros | Cons |
|---|---|---|---|
| `node_modules` (default) | `<project>/node_modules/@leafygreen-ui/*` etc. | Answers match the versions you actually installed | Only packages you installed; no stories; types come from compiled `.d.ts` |
| `repo` | Local clone of `mongodb/leafygreen-ui` | Full catalog, stories, source types | Reflects `main`, which may be ahead of your installed versions |

`node_modules` is the default because a wrong-version answer is worse than a missing one.

---

## Requirements

- Go (the version pinned in `go.mod`; the MCP Go SDK only supports currently maintained Go releases)
- `git` (only for `repo` mode)
- Claude Code, or another MCP client that supports stdio servers

Dependencies:

- [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) — the official Go SDK, maintained with Google
- [`golang.org/x/net/html`](https://pkg.go.dev/golang.org/x/net/html) — HTML parsing for `lg-mcp sync --design` only (the standard library has no HTML parser)
- Standard library for everything else

---

## Installation

### From source

```bash
git clone <this-repo-url> lg-mcp
cd lg-mcp
make build            # produces ./bin/lg-mcp
```

Or install into `$GOBIN`:

```bash
go install ./cmd/lg-mcp
```

### Repo mode only: clone LeafyGreen

```bash
git clone --depth 1 https://github.com/mongodb/leafygreen-ui ~/src/leafygreen-ui
```

Refresh it later with:

```bash
lg-mcp sync --repo-path ~/src/leafygreen-ui
```

### Design guidelines snapshot (recommended)

The design guidance that `consult_design_system` and `get_design_guidelines` return lives only on mongodb.design, not in the repository. Download it once:

```bash
lg-mcp sync --design          # writes to $XDG_CACHE_HOME/lg-mcp/design
```

This is the only command that contacts mongodb.design. It fetches about 70 pages one at a time, extracts their text and writes Markdown files plus a `manifest.json`. The server only ever reads that directory. Re-run it to refresh, then restart the server. Without a snapshot, every tool still works and lists the missing design guidance as a gap.

---

## Configuration

Flags take precedence over environment variables.

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--source` | `LG_MCP_SOURCE` | `node_modules` | `node_modules` or `repo` |
| `--project` | `LG_MCP_PROJECT` | current working directory | Project root containing `node_modules` (node_modules mode) |
| `--repo-path` | `LG_MCP_REPO_PATH` | — | Path to the LeafyGreen clone (repo mode, required) |
| `--design-cache` | `LG_MCP_DESIGN_CACHE` | `$XDG_CACHE_HOME/lg-mcp/design` (OS user cache dir) | Directory holding the mongodb.design snapshot |
| `--max-output` | `LG_MCP_MAX_OUTPUT` | `12000` | Max characters per tool response before truncation (minimum 500) |
| `--log-level` | `LG_MCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `--log-file` | `LG_MCP_LOG_FILE` | — | Write logs to a file (logs always go to stderr otherwise, never stdout) |
| `--design` | — | off | `sync` only: also snapshot mongodb.design guidelines |

Subcommands:

| Command | Purpose |
|---|---|
| `lg-mcp` / `lg-mcp serve` | Run the MCP server over stdio |
| `lg-mcp index` | Build the index and print a summary (packages, declarations, examples, a11y notes, design coverage, warnings). Useful to validate a source without an MCP client |
| `lg-mcp sync` | `git pull --ff-only` on the clone given by `--repo-path`; with `--design`, refresh the design snapshot |
| `lg-mcp version` | Print build version and commit |

---

## Using it with Claude Code

### Per project (recommended for node_modules mode)

Commit a `.mcp.json` at the root of the frontend project:

```json
{
  "mcpServers": {
    "leafygreen": {
      "command": "lg-mcp",
      "args": ["--source", "node_modules"]
    }
  }
}
```

Claude Code starts the server from the project directory, so `--project` defaults to the right place.

### User-wide (repo mode)

```bash
claude mcp add leafygreen --scope user \
  -e LG_MCP_SOURCE=repo \
  -e LG_MCP_REPO_PATH=$HOME/src/leafygreen-ui \
  -- lg-mcp
```

### Verify

Inside Claude Code, run `/mcp` and confirm `leafygreen` is connected and lists its 7 tools and 1 prompt. Then try:

> I need users to pick several tags from a long list. Which LeafyGreen component should I use, and how do I make it accessible?

You can also start from the prompt: `/mcp__leafygreen__build_with_leafygreen` with your intent.

### Steering Claude to use it

MCP tools are only used when the model decides to. Add a line to the consuming project's `CLAUDE.md`:

```markdown
For any LeafyGreen UI work, call the `leafygreen` MCP tool `consult_design_system` with what the UI should do, then check props and accessibility with the `get_*` tools before writing code. Do not rely on memory for LeafyGreen APIs.
```

---

## Tools

All tools are read-only (annotated `readOnlyHint`) and return Markdown. Responses longer than `--max-output` are truncated with an explicit `[truncated: N more characters — narrow the query]` marker. A name that matches nothing returns a normal result listing the closest package names. Data the sources lack is reported as missing, never filled in.

| Tool | Input | Returns |
|---|---|---|
| `consult_design_system` | `intent` (what the UI should do), `limit?` (default 3, max 5), `scope?` (`leafygreen-ui` \| `lg-chat` \| `lg-charts` \| `all`) | Ranked candidate components. Each one includes why it matched, install/import and peers, design-guidance excerpts, key props (deprecated last), the most relevant code example and accessibility notes. The response ends with cross-cutting guidance (accessibility foundation, `LeafyGreenProvider`), an explicit **Gaps** list and next steps |
| `get_component` | `name` (e.g. `button`, `Combobox`, `@leafygreen-ui/combobox`) | Package, version, install command, import statement(s), peer dependencies, deprecation notice, README overview and the available detail |
| `get_component_props` | `name`, `component?` (e.g. `ComboboxOption`) | Exported interfaces and types as declared in source, with JSDoc. The main `<Name>Props` comes first, followed by the local interfaces it is composed of. `extends` clauses are listed but not expanded |
| `get_component_examples` | `name`, `limit?` (default 3, max 10) | Code examples from the README and, in repo mode, Storybook stories |
| `get_accessibility` | `name` | Design-guideline a11y sections, README keyboard/focus/ARIA sections, accessibility props (including `Either<…, 'label' \| 'aria-label'>` requirements), behaviour seen in source (validators, focus traps, roles and `aria-*` attributes per file) and pointers to the a11y foundation and `@leafygreen-ui/a11y` |
| `get_design_guidelines` | `name` (component, or foundation slug such as `accessibility`) | The full mongodb.design page from the snapshot, with its URL and fetch date |
| `search_docs` | `query`, `limit?` (default 5, max 20) | Ranked sections across READMEs, design pages and foundation pages |

Prompt:

| Prompt | Arguments | Purpose |
|---|---|---|
| `build_with_leafygreen` | `intent` | Tells the client to consult, check props and accessibility, report gaps instead of guessing, then write code |

Names are lenient: `button`, `Button`, `@leafygreen-ui/button`, `ConfirmationModal`, `confirmation modal` and exported component names such as `ComboboxOption` all resolve.

### Example

Request:

```json
{ "name": "consult_design_system", "arguments": { "intent": "confirm a destructive delete action", "limit": 1 } }
```

Response (abridged, real repository and snapshot):

```markdown
# LeafyGreen consult: "confirm a destructive delete action"

Candidates, best first: @leafygreen-ui/confirmation-modal

## 1. @leafygreen-ui/confirmation-modal 11.0.2

Why it matched: `confirm` (name, description, design guidance, README, design docs, props); `destructive` (design guidance); …
Import: `import ConfirmationModal from '@leafygreen-ui/confirmation-modal';`

### Design guidance

**How to use this component > Delete Confirmation Modal:** Use the delete confirmation modal only when the action is destructive. …

### Key props (ConfirmationModalProps, extends `Omit<ModalProps, 'size'>`)

- `variant?: Variant` — Variant of the modal that represents the type of action handled by the modal.
- `confirmButtonProps?: CustomConfirmButtonProps` — An object that accepts all Button props except for the `variant` prop. …

### Example (README › Example)
…

### Accessibility
…

## Gaps (do not fill these from memory)
…
```

### Planned

`list_components`, `get_tokens` and `get_changelog` (see [Roadmap](#roadmap)).

---

## Resources

Not implemented yet. Planned for clients that support MCP resources (attach with `@` in Claude Code):

| URI | Content |
|---|---|
| `lg://components` | Full component catalog |
| `lg://components/{name}/readme` | Raw README for one package |
| `lg://tokens/{category}` | Token table for one category |

---

## Project layout

```
.
├── cmd/
│   └── lg-mcp/            # main package: flag parsing, subcommands, wiring
├── internal/
│   ├── config/            # flags + env resolution, validation
│   ├── source/            # Source interface, node_modules and repo adapters, SafeJoin
│   ├── design/            # mongodb.design snapshot: cache reader (runtime) + syncer (sync only)
│   ├── index/             # package model, name lookup, BM25 search, main-props detection
│   ├── parse/
│   │   ├── markdown/      # README/design section + code block extraction
│   │   ├── tstypes/       # exported interfaces/types (+ members, JSDoc) from .ts/.d.ts
│   │   └── a11ysrc/       # accessibility behaviour found in component source
│   ├── tools/             # one file per MCP tool, the prompt, registration
│   └── output/            # truncation and Markdown formatting helpers
├── testdata/
│   ├── node_modules/      # minimal fake project with @leafygreen-ui packages
│   ├── repo/              # minimal fake leafygreen-ui checkout
│   ├── design/            # minimal fake design snapshot
│   └── design-html/       # HTML fixtures for the snapshot extractor
├── AGENTS.md
├── Makefile
├── go.mod
└── README.md
```

---

## Development

```bash
make build      # go build -o bin/lg-mcp ./cmd/lg-mcp
make test       # go test ./... -race (the race detector needs cgo and a C compiler)
make lint       # golangci-lint run (v2 config in .golangci.yml)
make fmt        # gofmt + goimports
make inspect    # run under the MCP Inspector against testdata
make index      # bin/lg-mcp index against testdata, prints summary
```

Without `golangci-lint` installed, `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run ./...` gives the same result.

### Adding a tool

1. Create `internal/tools/<tool_name>.go` with an input struct (JSON tags + `jsonschema` descriptions) and a handler.
2. Register it in `internal/tools/register.go`.
3. Add golden cases in `internal/tools/tools_test.go` and a call in `server_test.go`, using the `testdata/` fixtures.
4. Document it in the [Tools](#tools) table above.

See `AGENTS.md` for the conventions that apply.

---

## Testing

- **Unit tests** run against the fixtures in `testdata/`, never against a real `node_modules` or the network. The design syncer is tested against an `httptest` server.
- **Golden files** (`internal/tools/testdata/*.golden.md`) capture tool output. Update with `go test ./internal/tools/... -update` and review the diff.
- **Protocol test** (`internal/tools/server_test.go`): an in-process client/server pair on the SDK's in-memory transport lists the tools and prompt and calls each one end to end.
- **Manual**: `make inspect` opens the [MCP Inspector](https://github.com/modelcontextprotocol/inspector) against the built binary.

---

## Debugging

- **Nothing on stdout but protocol messages.** Any stray `fmt.Println` corrupts the JSON-RPC stream and Claude Code will drop the connection. Logs go to stderr (or `--log-file`). The linter's `forbidigo` rule rejects `fmt.Print*`, `log.Print*` and `os.Stdout` outside `cmd/`.
- Claude Code shows server stderr in its MCP logs; `claude --debug` gives more detail.
- `lg-mcp index` prints coverage (packages without a README or a11y notes, design pages attached) and every warning, such as files skipped for size or escaping symlinks.

---

## Design decisions

| Decision | Choice | Reason | Revisit if |
|---|---|---|---|
| Language | Go | Single static binary, fast startup, no Node runtime in the server | Type extraction proves too fragile without the TypeScript compiler |
| SDK | Official `modelcontextprotocol/go-sdk` | Tracks the spec, maintained with Google; `mark3labs/mcp-go` is the main alternative | — |
| Transport | stdio only | Single user, local client; no auth or ports to manage | You want to share it across machines |
| Storage | In-memory, rebuilt at startup | The full repository plus design snapshot (100 packages) indexes in about 0.5 s; no cache invalidation | Startup exceeds ~1s |
| Default source | `node_modules` | Version accuracy over completeness | — |
| Type extraction | Text-level extraction of exported `interface`/`type` blocks | No TS toolchain dependency | Answers miss inherited/generic props too often (see Limitations) |
| Design guidance | Offline snapshot written by `lg-mcp sync --design` | mongodb.design is the only source of usage guidelines; a snapshot keeps the server offline and deterministic | mongodb.design publishes a machine-readable feed |
| HTML extraction | `golang.org/x/net/html`, used only by the syncer | The standard library has no HTML parser; regex over HTML is fragile | — |
| Search | BM25 over weighted fields, a name-match bonus, and a small tested query-expansion table | Explainable ranking ("why it matched"), no model or embeddings | Intents routinely miss (see Limitations) |
| Output | Markdown, hard character cap, per-dimension budgets in `consult_design_system` | Readable by the model, protects the context window, and truncation never drops a whole dimension | — |

---

## Limitations

- **Props are not fully resolved.** LeafyGreen props often extend `HTMLElementProps<'button'>`, `DarkModeProps` or polymorphic helpers. Text-level extraction returns the declared interface, names what it extends, and lists the members of local interfaces that a `<Name>Props` alias is composed of. It does not expand types from other packages. A faithful resolver would need the TypeScript compiler (e.g. a small Node sidecar using `ts-morph`), which this project deliberately avoids for now.
- **No visual information.** The server cannot show what a component looks like; use Storybook or mongodb.design for that. Snapshot pages drop images, so sections that are only images (e.g. "Anatomy") are omitted.
- **The design snapshot can go stale.** It is a copy of mongodb.design from the date shown in every response. Re-run `lg-mcp sync --design` to refresh it. The content remains MongoDB's and is stored only on your machine.
- **Repo mode reflects `main`.** It may document props that are not yet in your installed version.
- **Search is lexical.** There are no embeddings, so an intent has to share vocabulary with the docs. The expansion table bridges common gaps ("dialog" → modal, "temporary" → toast), but ambiguous phrasing can still miss. For example, "collapsible sections of content" does not surface Expandable Card, whose docs say "show or hide". Naming the kind of control helps.
- **Accessibility facts are only what the sources state.** Notes come from design pages, READMEs, prop types and a text scan of component source (roles, `aria-*` attributes, known a11y utilities). They are not an audit.

---

## Roadmap

- [x] `node_modules` and `repo` source adapters
- [x] Core tools: `get_component`, `search_docs`
- [x] `get_component_props` with JSDoc
- [x] Intent tool `consult_design_system` crossing design guidance, props, examples and accessibility
- [x] Design guidelines snapshot (`lg-mcp sync --design`, `get_design_guidelines`)
- [x] Accessibility notes per component (`get_accessibility`)
- [ ] `list_components`
- [ ] `get_tokens`, `get_changelog`
- [ ] Resources
- [ ] Optional: `check_usage` tool that flags hard-coded colors/spacing in a snippet and suggests tokens

---

## License

MIT for this project's code. LeafyGreen UI content read at runtime, and the mongodb.design snapshot stored locally by `lg-mcp sync --design`, remain under their owners' terms (the leafygreen-ui repository is Apache-2.0).
