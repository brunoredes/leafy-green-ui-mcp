# Go Module Management — Reference

Scope: module layout, naming, go.mod directives, dependency workflow, private modules, and release hygiene. Targets Go 1.27; version-gated behavior is marked.

Primary sources: [Organizing a Go module](https://go.dev/doc/modules/layout), [Managing dependencies](https://go.dev/doc/modules/managing-dependencies), [go.mod reference](https://go.dev/doc/modules/gomod-ref), [Go Modules Reference](https://go.dev/ref/mod).

---

## 1. Layout

### Decision table

| Project type | Layout |
|---|---|
| Single library package | All `.go` files in the module root; package name = last element of the module path |
| Single command | `package main` files in the root; `main.go` is convention, not a requirement |
| Library or command with helpers | Helpers under `internal/` |
| Several importable packages | One directory per package, nested as needed; non-public ones under `internal/` |
| Several commands only | One directory per command (`prog1/`, `prog2/`) or under `cmd/`; shared code in `internal/` |
| Packages and commands mixed | Packages at root/subdirs, commands under `cmd/<name>/` |
| Server / service | Commands under `cmd/<name>/`, all logic under `internal/`; nothing exported |

### Rules

- **Default to `internal/`.** The go command forbids imports of `internal/` packages from outside the parent tree. That is what lets you refactor freely. Move a package out of `internal/` only when an external consumer actually needs it.
- **Use `cmd/` when a repo mixes commands and libraries**, or when the repo has many non-Go directories (deploy, web, scripts). In a commands-only repo it is optional.
- **Servers export nothing.** If a server grows something reusable, extract it into its own module. Don't expose it from the server module.
- **No `pkg/` directory.** The official guidance does not use it. It adds a path segment to every import and conveys nothing that "not in `internal/`" doesn't already convey.
- **Package names carry the meaning.** The module path is a namespace. Avoid `util`, `common`, `helpers`, and `models` as package names.
- **Tests live next to code** (`foo_test.go`). Use `package foo_test` for black-box tests of the exported API.
- **Non-Go assets** (`testdata/`, `web/`, `deploy/`) can sit anywhere. `testdata/` and directories starting with `.` or `_` are ignored by the go command. For other directories, use the `ignore` directive (§3).

### Reference trees

Library with helpers:
```
go.mod
client.go
client_test.go
internal/
  wire/
    wire.go
```

Mixed library + CLIs:
```
go.mod
client.go
retry/
  retry.go
internal/
  wire/
cmd/
  clientctl/
    main.go
```

Service:
```
go.mod
cmd/
  api/
    main.go
  worker/
    main.go
internal/
  billing/
  storage/
  httpapi/
deploy/
web/
```

Install commands with `go install <module>/cmd/<name>@latest`.

---

## 2. Naming the module

- **Use the repository location** (`github.com/org/repo`) for anything that might ever be fetched by another module. `go get` resolves code from that path.
- **Unpublished code** needs a prefix you control, such as a company domain. Avoid generic names like `app`, `utils`, or `widgets`.
- **Reserved prefixes:** `example` is for docs and tutorials, `test` is for throwaway modules created by tests.
- **Major versions ≥ 2 change the path.** The module path must end in `/vN` (`github.com/org/repo/v2`), and every import must use it. Treat a v2 as a new module for consumers. Prefer not breaking compatibility over shipping a v2.
- **Renaming a published module is a breaking change.** Pick the final path before the first tag.

---

## 3. go.mod directives

| Directive | Purpose | Notes |
|---|---|---|
| `module` | Module path | See §2 |
| `go` | Minimum language version and semantics | Gates behavior: loop vars (1.22), timers (1.23), GOMAXPROCS (1.25), traceback labels and tidy merging (1.27) |
| `toolchain` | Suggested toolchain when the local one is older | Separate from `go`; managed with `go get toolchain@…` |
| `require` | Minimum required version of a dependency | Minimal version selection picks the max of all minimums, never "latest" |
| `replace` | Redirect a module to a fork, version, or local path | **Only honored in the main module**; checksums not verified for local paths |
| `exclude` | Forbid a specific version | Main module only |
| `retract` | Mark your own published versions as bad | Add in a new release; consumers see it in `go list -m -u` and `go get` |
| `tool` | Declare a dev tool dependency (1.24+) | Replaces `tools.go` |
| `godebug` | Pin GODEBUG defaults (1.23+) | 1.27 fails the build if a removed setting is set to its old value |
| `ignore` | Directories the go command should skip (1.25+) | For large non-Go trees (e.g. `node_modules`) |

### `go` vs `toolchain`

- `go 1.N` is a promise about **semantics** and a **minimum** for consumers. Raising it forces every downstream user onto ≥ 1.N.
- **Libraries:** keep `go` as low as the code genuinely needs. **Applications/services:** raise it deliberately, one release at a time, running tests at each step.
- `go mod init` with toolchain 1.N writes `go 1.(N-1).0` (since 1.26). Raise it explicitly with `go get go@1.27` if you need newer semantics.
- `go test` runs the `stdversion` vet check by default (1.27). It catches stdlib symbols newer than your `go` directive.

### Shape of a tidy go.mod (1.27+)

Exactly one `require` block for direct dependencies and one for `// indirect`. With `go 1.27` or later, `go mod tidy` merges extra blocks automatically. On older directives you have to merge them by hand.

---

## 4. Dependency workflow

| Task | Command |
|---|---|
| Add deps for current package | `go get .` |
| Add a specific module | `go get example.com/mod` |
| Pin a version | `go get example.com/mod@v1.3.4` |
| Upgrade to latest | `go get example.com/mod@latest` |
| Patch-only upgrades for all | `go get -u=patch ./...` |
| Upgrade everything | `go get -u ./...` (review diff, run tests) |
| Get a commit / branch | `go get example.com/mod@4cf76c2` / `@branch` (records a pseudo-version) |
| Remove a dependency | `go get example.com/mod@none` (also downgrades dependents) |
| Sync go.mod with imports | `go mod tidy` (`-v` lists removals; `-diff` to check in CI) |
| List available upgrades | `go list -m -u all` |
| Why is X here? | `go mod why -m example.com/mod` |
| Full graph | `go mod graph` |
| Verify module cache | `go mod verify` |
| Vulnerabilities | `govulncheck ./...` |

### Rules

- **Commit `go.mod` and `go.sum`.** `go.sum` is how other machines verify module integrity. Never hand-edit it.
- **Run `go mod tidy` before committing**, and fail CI if it produces a diff (`go mod tidy -diff`).
- **Change requirements with go commands, not by hand.** Use `go get` and `go mod edit` so the module graph stays consistent.
- **`@latest` is not reproducible.** Use it to find a version, then commit the resulting pin.
- **Prefer tagged versions over pseudo-versions** in anything long-lived. A pseudo-version pinned to a branch commit is technical debt.
- **Upgrade in small batches.** Separate PRs for major-impact deps (gRPC, cloud SDKs, x/ modules) make regressions bisectable.
- **Fewer deps is the default.** Check the stdlib first: `ServeMux` routing, `slog`, `uuid` (1.27), `encoding/json/v2`, `crypto/*`, and `slices`/`maps` replace many common third-party modules.

---

## 5. Working on unpublished or forked code

### Local edits across modules → workspaces (preferred)

```
go work init ./app ./lib
go work use ./other
```

- `go.work` overrides resolution locally without touching any `go.mod`.
- **Generally don't commit `go.work`**, or set `GOWORK=off` in CI. Otherwise CI can pass on code that fails for anyone using only `go.mod`. The exception is a monorepo that intentionally builds as one workspace.

### `replace` (when a workspace doesn't fit)

Local path:
```
go mod edit -replace=example.com/theirmod@v0.0.0-unpublished=../theirmod
go get example.com/theirmod@v0.0.0-unpublished
```

Fork:
```
go mod edit -replace=example.com/theirmod@v1.2.3=example.com/me/theirmod@v1.2.3-fixed
```

- `replace` in a **dependency's** go.mod is ignored. Only the main module's applies. Libraries should not depend on `replace` to work.
- Local-path replacements skip checksum authentication.
- Treat every `replace` as temporary. Record why it exists and when to remove it (upstream PR link).

---

## 6. Tool dependencies (1.24+)

```
go get -tool golang.org/x/tools/cmd/stringer
go tool stringer                 # short name
go tool golang.org/x/tools/cmd/stringer   # full path if ambiguous
go tool                           # list tools
go get tool                       # upgrade all tools
go install tool                   # install all to $GOBIN
```

- Tool requirements take part in MVS like any other requirement. Thanks to module pruning, your tools' dependencies usually don't leak into your consumers' graphs.
- Delete `tools.go` blank-import files when migrating.
- Tool deps share the version graph with your code. A tool that pins an old shared dependency can hold your code back. If that becomes a real conflict, put the tool in a separate `tools/go.mod`.

---

## 7. Proxies, private modules, checksums

- Default: `GOPROXY=https://proxy.golang.org,direct`.
- `,` falls through to the next proxy only on 404/410. `|` falls through on any error.
- **Private code:** `GOPRIVATE=*.corp.example.com,github.com/org/*`. This implies `GONOPROXY` and `GONOSUMDB` for those patterns, so private paths are never sent to the public proxy or checksum DB.
- **Corporate proxy** (Athens, Artifactory, etc.): `GOPROXY=https://goproxy.corp,direct` keeps builds reproducible when upstream repos vanish.
- Set these in CI explicitly. Don't rely on developer machine `go env -w` state.
- `GOFLAGS=-mod=mod`/`-mod=readonly`: CI should use the default (`readonly`) so a build never silently edits go.mod.

### Vendoring

- `go mod vendor` copies deps into `vendor/`. With `go` ≥ 1.14 and a `vendor/` directory present, builds use it automatically.
- Use it only for a concrete reason: air-gapped builds, auditing, or a policy requirement. Otherwise a proxy gives the same reproducibility without the repo bloat.
- If you vendor, CI must check that `vendor/` matches `go.mod` (`go mod vendor` + diff).

---

## 8. Single module vs multiple modules

- **Default: one module per repository.** Multiple modules add versioning, tagging (`sub/v1.2.3` tags), and cross-module change overhead.
- **Split into a separate module only when** a part needs an independent release cadence, a different `go` minimum, or must not drag heavy dependencies onto consumers (for example, an optional integration adapter).
- Nested module tags are path-prefixed: module `github.com/org/repo/adapter` is tagged `adapter/v1.0.0`.
- Never import across nested modules through relative `replace` in released code. Use workspaces for development and real versions for releases.

---

## 9. Publishing and release hygiene

- Tag releases with semver: `v1.4.0`. `v0.x` makes no compatibility promise. `v1+` does.
- Before tagging: `go mod tidy`, `go test ./...`, `go vet ./...`, `govulncheck ./...`.
- **Mistake in a published version?** Don't delete or retag it, because proxies cache it permanently. Publish a new version with `retract v1.4.0 // reason` in go.mod.
- Keep the `go` directive in libraries conservative. Raising it is a breaking change for consumers stuck on older toolchains.
- Document public packages: `go doc pkg@version` (1.27) and pkg.go.dev read your doc comments and examples.

---

## 10. CI checklist

```
go mod tidy -diff          # go.mod/go.sum are clean
go mod verify              # cache integrity
go build ./...
go vet ./...
go test -race ./...        # also runs stdversion (1.27)
govulncheck ./...
GOWORK=off go build ./...  # if go.work exists locally
```