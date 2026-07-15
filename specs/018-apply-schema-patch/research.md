# Research: Import Schema Definitions via `arc apply schema`

## D1: Source-kind detection (local file vs. URL vs. `arcnet:` shorthand)

**Decision**: Classify the single positional argument in this order:
1. Literal prefix match on `"arcnet:"` (via `strings.CutPrefix`) — if present, the remainder is the catalog suffix (D1a below).
2. Otherwise, parse with stdlib `net/url.Parse`; if the result has `Scheme` equal to `"http"` or `"https"`, treat it as a URL and fetch it through the new `Fetcher` port.
3. Otherwise, treat it as a local filesystem path, mirroring `internal/app/graph/service/apply.go`'s existing `readPatch` (mount the parent directory via `fsys.Mounter`, `Open` the base name).

**Rationale**: No new flag is needed to disambiguate (Principle IX: a single positional "subject" argument, no redundant flag); `arcnet:` is checked first since it is not a valid URL scheme `url.Parse` would otherwise recognize on its own, and a bare local path never legitimately starts with `arcnet:` (colon-containing relative paths are not a realistic concern on any of the project's supported platforms — Windows drive letters are a single letter, not `arcnet`).

**Alternatives considered**: A `--url`/`--file`/`--catalog` flag triad — rejected, adds ceremony for a distinction the input's own shape already communicates unambiguously. Sniffing via `os.Stat` first (file-exists check) — rejected, a relative path colliding with a URL/`arcnet:`-shaped string is not a realistic concern and prefix/`Scheme` detection is simpler and used identically elsewhere in the Go ecosystem (`go get`, `git clone`).

## D1a: `arcnet:` prefix resolution

**Decision**: A literal `arcnet:<suffix>` input resolves to
`https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/<suffix>`
— a fixed, hardcoded base URL joined with `<suffix>` as-is (no additional
escaping beyond what `net/url` already guarantees for a valid path segment)
— then fetched exactly like any other URL input via the same `Fetcher` port.
An empty `<suffix>` (bare `"arcnet:"`) is rejected before any fetch attempt,
with the same "disallowed/invalid input" error shape as a malformed
local-path argument.

**Rationale**: Spec FR-002a's whole purpose is letting a maintainer name an
official arcnet extension by its short catalog name instead of memorizing
or copy-pasting the full `raw.githubusercontent.com` URL every time. The
base is a compile-time constant (not read from config/environment/flag) per
the spec's own Assumptions — this feature does not need the base to be
overridable, since it names one specific, official catalog.

**Alternatives considered**: Making the base URL configurable via a flag or
`.arc/config.yml` entry — rejected as scope creep; the spec explicitly
assumes a fixed base, and no user story asks for a second catalog.

**E2E test seam**: Principle VIII forbids live network access in the E2E
suite, yet this is the codebase's first genuinely network-calling feature
(fsys and git, the only external systems every prior command's E2E tests
exercise, are both local/hermetic). Two techniques cover it without ever
contacting a real host: a directly supplied `http(s)://` URL input is
pointed at a stdlib `httptest.Server` in the test itself (the input string
*is* the seam, no indirection needed); the `arcnet:` prefix's fixed base
(`kernel.ArcnetCatalogBaseURL`, D1a) is declared as a package-level `var`
rather than a `const` specifically so a test can temporarily repoint it at
an `httptest.Server` for the one assertion that needs to observe the
resolved URL, then restore it — the same package-var-indirection technique
`internal/app/ctrl/service.go`'s `resolveLocalRoot`/`removeLocalRoot` already
uses for the same reason (test seam without a constructor parameter).

## D2: New `internal/adapter/http` package

**Decision**: Add a new, shared adapter package `internal/adapter/http`, alongside the existing `internal/adapter/fsys`/`internal/adapter/git`, exposing a small `Client` type with one capability: `Fetch(ctx context.Context, url string) (io.ReadCloser, error)`, backed by `net/http.Client` with a default timeout (30s) applied via `http.NewRequestWithContext` plus `context.WithTimeout`, overridable by a `--timeout` flag on `arc apply schema`.

**Rationale**: A repo-wide search confirms no existing HTTP client/adapter exists anywhere in the codebase — this is a genuinely new external integration, and Principle VII requires it live behind a port/adapter pair, not be called inline from `cmd/` or `service/`. Placed at `internal/adapter/` (not nested under `internal/app/schema/adapter/`) since URL fetching is a capability other future commands could reuse (mirrors `fsys`/`git`'s own shared placement), even though today only schema's own port consumes it.

**Alternatives considered**: A generic `curl`/`wget` subprocess shell-out (mirroring how `git` is invoked) — rejected, stdlib `net/http` is sufficient, avoids an external binary dependency, and keeps error handling in Go rather than parsing subprocess output.

## D3: `internal/app/schema/port.VCS` — narrow, schema-scoped

**Decision**: Add `internal/app/schema/port.VCS` with exactly two methods: `StageAll(ctx, dir) error` and `Commit(ctx, dir, message) (hash string, err error)`. No `IsAvailable`/`Init` (unlike `ctrl.port.VCS`) and no `IsTracked` (unlike `graph.port.VCS`).

**Rationale**: Follows the existing per-use-case narrow-port precedent exactly (`internal/app/ctrl/port.VCS` needs `IsAvailable`/`Init` because it bootstraps a repo from nothing; `internal/app/graph/port.VCS` needs `IsTracked` for source-document idempotency and the `arc revert` surface). `arc apply schema` always runs against an already-`arc init`-ed graph (git already initialized) and has no source-document idempotency concept (schema patches carry no `source` node) — so it needs neither. The existing shared `internal/adapter/git.VCS` concrete type already implements `StageAll`/`Commit` and therefore satisfies this new interface structurally with zero new adapter code (ADR 001 port isolation rule 1 — the same technique `internal/app/schema/component.go`'s own doc comment already cites for `internal/adapter/git.Git`).

**Alternatives considered**: Reusing `graph.port.VCS` directly — rejected, it would pull in five methods (`IsTracked`, `CommitsMatching`, `ChangedPaths`, `CommitsTouching`, `RevertCommit`, `Blame`, `ShowFile`) this feature never calls, violating Interface Segregation (Principle V).

## D4: All-or-nothing validation strategy

**Decision**: Before writing anything, iterate over the parsed patch's `Nodes` once and classify every node's `Type`. If any node's type is not exactly `"Property"` or `"Class"`, return `ErrDisallowedNodeType` immediately — no store writes have occurred yet, so no rollback bookkeeping is needed (unlike `graph.Apply`'s `rollback(store, createdPaths)`, which exists because that command interleaves validation and writes per-node).

**Rationale**: Spec FR-005 requires the entire operation to fail with zero partial writes, including for otherwise-valid `Property`/`Class` sections in the same document. Validating in a dedicated pass before the create/merge loop is strictly simpler than write-then-rollback, and correctness does not depend on write ordering.

**Alternatives considered**: Validate-and-rollback (mirroring `graph.Apply`'s pattern) — rejected as unnecessary complexity here, since the validation pass is pure (no I/O) and can run to completion before any write begins, unlike `graph.Apply` where an unrecognized *type* is a warning-and-auto-register case rather than a hard failure.

## D5: Reuse of existing Property/Class decode/validate logic

**Decision**: The create/merge loop calls the same `decodePredicateDef`/`decodeTypeDef` validators `service/schema.go`'s `Resolve` already uses when loading `_schema/` from disk, applied to each patch-carried `Property`/`Class` node before it is written. A node that fails this validation (bad `role`/`merge`, missing `description`) is rejected with the same `ErrSchemaInvalid`-shaped error `Resolve` already produces for a malformed on-disk document.

**Rationale**: Guarantees a schema patch can never introduce a document `Resolve` would later refuse to load — the write-time and read-time validation are the exact same function, so there is no drift between "what this command accepts" and "what the schema loader accepts."

## D6: Create/merge mechanics

**Decision**: For each valid node, read the existing on-disk document (if any) the same way `graph/service/apply.go`'s `readExistingNode` does, then either write a new file (`store.Create`) or run `core.Merge(existing, incoming, index, patch.Document)` and write the merged result — the same two-branch shape `graph.Apply`'s per-node loop already uses, scoped to `_schema/predicates/<name>.md` / `_schema/types/<name>.md` instead of content folders.

**Rationale**: Reuses proven, already-tested merge semantics (each predicate's own declared `merge` op) rather than inventing a second merge code path for schema documents.

## D7: Idempotency (no source-tracking check)

**Decision**: No `IsTracked`-style check gates re-application. Re-applying an unchanged patch runs the full create/merge loop, finds every node's merged rendering byte-identical to the existing document (mirroring `graph/service/apply.go`'s `nodeContentChanged`), and reports zero created/merged definitions with no commit produced (nothing to stage).

**Rationale**: Schema patches carry no `source` node and no natural "document identity" to track for idempotency the way `graph.Apply` tracks `sources/<document>.md`; content-level no-op detection is both sufficient (spec FR-011) and consistent with how `graph.Apply` already reports "no real change" at the node level.

## D8: CLI attachment point

**Decision**: `cmd/arc/ctrl/apply_schema.go` exports `NewApplySchemaCmd() *cobra.Command` (`Use: "schema <patch.md>|<url>"`). `cmd/arc/root.go` attaches it as a child of the existing `graph.NewApplyCmd()` return value: `applyCmd := graph.NewApplyCmd(); applyCmd.AddCommand(ctrl.NewApplySchemaCmd()); cmd.AddCommand(applyCmd)`.

**Rationale**: Matches the user's explicit direction: the command's business logic and conceptual home is schema/config management (grouped with `arc init` in `ctrl`), but it is surfaced under the pre-existing `apply` verb for naming consistency with `arc apply <patch.md>` rather than inventing a second top-level verb. Cobra resolves `arc apply schema ...` to the child command before consulting the parent's own `RunE`/`Args`, so the parent's existing single-file behavior (`arc apply <patch.md>`) is unaffected.

**Alternatives considered**: A new top-level `arc schema apply <patch.md>` command — rejected per the user's explicit instruction to reuse the `apply` verb.
