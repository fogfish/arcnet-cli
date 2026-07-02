# Phase 1 Data Model: `arc apply`

Value types are immutable (constitution Principle IV) and carry no Cobra, `os/exec`, or raw `os.*` filesystem types; goldmark's own AST types never appear outside `internal/core`'s parsing functions (research.md D2, D3).

## Core AST (`internal/core`)

### Kind / MergeOp

| Type | Values |
|---|---|
| `Kind` (`string`) | Open vocabulary (AST §4 invariant 5); this feature recognizes `source`, `entity`, `resource`, `timeline`, plus whatever a graph's `.arc/config.yml` additionally registers |
| `MergeOp` (`string`) | `none`, `union`, `union-first-writer`, `append`, `validated-overwrite` (CORE §10's fixed menu) |

### MergeRuleSet

`map[Kind]MergeOp`, with `MarshalYAML`/`UnmarshalYAML` (`gopkg.in/yaml.v3`) so the same type is both `internal/app/ctrl`'s seed content for `.arc/config.yml` and `internal/app/config`'s load/save shape (research.md D5) — one type, zero duplication.

| Value | Contents | Owner |
|---|---|---|
| `CoreMergeRules` | `{source: none, entity: union, resource: union-first-writer, timeline: append}` | `internal/core` — the format's own fixed kinds; always recognized, never requires registration (spec FR-018) |
| `KnownProfileMergeRules` | `{hypothesis: validated-overwrite, aporia: validated-overwrite, thought: union}` | `internal/core` — the two example domain profiles documented in `github.com/fogfish/arcnet-spec` (`ARCNET-DOMAIN-ARTICLE.md`, `ARCNET-DOMAIN-CORE-THOUGHT.md`); ready-made values a user copies into `.arc/config.yml` to opt in, never auto-registered |

`ConfigPath` — the constant `".arc/config.yml"`, shared by `internal/app/ctrl` (seeding) and `internal/app/config` (load/save), avoiding two independently-maintained path literals.

`Lookup(kind Kind) (op MergeOp, ok bool)` — the one query method a caller needs (research.md D5-revised); `ok=false` when `kind` is absent from the set, the exact condition `internal/app/graph/service.Apply` uses to decide "apply with the safe `union` default and warn" instead of failing.

### Link / LinkBlock

| Field | Type | Notes |
|---|---|---|
| `Predicate` | `string` | camelCase per CORE §7.3; empty for an untyped `[[Target]]` mention |
| `Target` | `string` | basename reference (CORE §7.1) |
| `Alias` | `string` | optional display text, `[[Target|text]]` |

`LinkBlock{ Title string; Seq []Link }` — one predicate-grouped body block (AST §6.5); `Title` is the display heading (e.g. `"Mentions"`), derived and never independently re-derived by a consumer once parsed.

### Node

| Field | Type | Notes |
|---|---|---|
| `ID` | `string` | Basename, equal to filename without `.md` (CORE §6, AST §4) |
| `Kind` | `Kind` | Mandatory |
| `Attrs` | `map[string]any` | Front-matter scalars, excluding `kind` (AST §4); unrecognized keys preserved verbatim (AST invariant 5, spec FR-017) |
| `Text` | `string` | Leading prose block (`abstract`/`definition`/etc. per kind); empty when the kind has none. **Bracket-stripped**: any `[[Target]]`/`[[Target\|alias]]` markup originally embedded in the prose is removed and recorded in `HRefs` instead (research.md D3/D3b) — `Text` itself carries only the plain display substring |
| `Notes` | `string` | Trailing prose block, rendered after `Edges`/`Links`; bracket-stripped exactly like `Text` (D3b) |
| `HRefs` | `[]Link` | Inline links originally embedded in `Text`/`Notes`, extracted at parse time; never a source of navigable edges (AST invariant 3). `RenderNode` reconstructs the bracket markup back into the serialized `Text`/`Notes` from this list (research.md D3b) — `HRefs` is the sole record of "where a link goes" once parsed |
| `Edges` | `[]Link` | Ungrouped structural edges, order-preserving |
| `Links` | `map[string]LinkBlock` | Predicate-grouped structural edges |

### Patch

| Field | Type | Notes |
|---|---|---|
| `Document` | `string` | The source citekey this patch contributes (CORE §12.2 manifest, mandatory) |
| `Published` | `time.Time` | Manifest `published` (mandatory); drives timeline derivation (D8) |
| `Title` | `string` | Manifest `title` (recommended) |
| `Stats` | `map[string]any` | Manifest `stats` (recommended); carried through, not independently validated against actual counts |
| `Nodes` | `[]Node` | Every H1/H2 node section, in document order |

## Application values (`internal/app/graph/kernel`)

### ApplyResult

The domain value `component.go`'s `Apply` returns to `cmd/arc/graph`, rendered by `bios.Registry[ApplyResult]`.

| Field | Type | Notes |
|---|---|---|
| `Document` | `string` | The applied patch's source id |
| `Skipped` | `bool` | `true` when the idempotency check (FR-003) found the document already tracked; every other field is zero-valued in that case |
| `Created` | `map[core.Kind]int` | Node counts by kind, newly created |
| `Merged` | `map[core.Kind]int` | Node counts by kind, merged into existing nodes |
| `Conflicts` | `[]string` | Relative paths of node files that received a conflict marker (FR-013), for the `PostRunE` hint (research.md D9) |
| `Warnings` | `[]string` | One human-readable sentence per node whose kind was not found in the resolved `MergeRuleSet` and was therefore applied using the default `union` behavior (spec FR-018, research.md D5-revised); empty when every kind was recognized |
| `CommitHash` | `string` | Short hash of the single resulting commit; empty when `Skipped` |
| `Timeline` | `[]string` | Period codes touched (`"2026"`, `"2026-04"`), for human/JSON output |

## Ports

### `internal/app/graph/port.VCS`

Narrower than `internal/app/ctrl/port.VCS` — `apply` never initializes a repository or re-checks git availability (a graph it operates on is already `arc init`-ed).

```go
type VCS interface {
    IsTracked(ctx context.Context, dir, path string) (bool, error)
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

Both this and `ctrl.port.VCS` are satisfied structurally by the one promoted `internal/adapter/git.Git` type (research.md D4, ADR 001 port isolation rule 1).

### `internal/app/config`

Depends on `fsys.Store` directly for load/save (no port needed, research.md D5 — same exception `ctrl` established), plus one new use-case-private port for the seed fetch (research.md D5-revised):

```go
// port/fetcher.go
type Fetcher interface {
    Fetch(ctx context.Context, url string) ([]byte, error)
}

// component.go
func Resolve(store fsys.Store) (core.MergeRuleSet, error)
func Save(store fsys.Store, cfg kernel.Config) error
func Default(ctx context.Context, fetcher port.Fetcher) (cfg kernel.Config, usedFallback bool)
```

`Default` has no `error` return, by construction — a fetch/parse failure of any kind is not an error condition, it is the `usedFallback = true` path returning `core.CoreMergeRules` (research.md D5-revised, satisfying `specs/002-arc-init/spec.md` FR-017's "initialization MUST NOT fail... on this basis alone"). `DefaultSourceURL = "https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/config.yml"` is a package constant. The real `Fetcher` (`internal/app/config/adapter/http`) wraps stdlib `net/http` with a fixed 3-second timeout, no retries; a mock `Fetcher` (`internal/app/config/adapter/mock`) backs `Default`'s unit tests (constitution Principle VI — no real network access in `go test`).

## Filesystem I/O

All reads/writes go through `fsys.Store`/`fsys.Mounter` (`internal/adapter/fsys`, already shared — no changes to that package). `arc apply` mounts the same way `arc init` does; unlike `init`, it does **not** call `fsys.ResolveLocalRoot` (the target must already be a resolved, initialized graph — spec FR-014) and never calls `fsys.RemoveLocalRoot` (rollback on mid-run failure is per-path `Store.Remove`/`Store.Create`-then-`Discard`, matching `arc init`'s research.md D4 pattern, scoped to exactly the files this run itself wrote).

## Reporter events (`internal/bios.Reporter`, ADR 002 DS-06)

| Label | Emitted around |
|---|---|
| `"Reading patch file"` | `fsys.Store.Open` + `core.ParsePatch` |
| `"Checking idempotency"` | `port.VCS.IsTracked` |
| `"Applying node contributions"` | Per-node create (`Store.Create`) or merge (`core.Merge` + `Store.Create`/rewrite) |
| `"Updating timeline"` | `core.TimelinePeriods` + yearly/monthly file read-insert-write |
| `"Committing"` | `port.VCS.StageAll` + `.Commit` |

Each label gets one `Start`/`Done` (or `Error`) pair (flat `Reporter`, ADR 002 DS-08 — sufficient at this scale, matching `arc init`'s precedent).

## Error sentinels (`github.com/fogfish/faults`, research.md D10)

### `internal/core` (`errors.go`)

| Constant | Kind | Message | Produced by |
|---|---|---|---|
| `ErrManifestInvalid` | `faults.Type` | `"patch manifest is missing a mandatory field (kind: patch, document, published)"` | `ParsePatch` (spec FR-001, FR-002) |
| `ErrPatchStructure` | `faults.Type` | `"patch body does not follow the H1-kind/H2-node section structure"` | `ParsePatch` (spec FR-002) |

An unrecognized node kind is **not** an error sentinel (research.md D5-revised, D10) — it produces a `kernel.ApplyResult.Warnings` entry, not a returned `error`.

### `internal/app/config` (`service/errors.go`)

| Constant | Kind | Message |
|---|---|---|
| `ErrConfigMalformed` | `faults.Safe1[string]` | `"%s is not valid YAML"` |
| `ErrConfigConflict` | `faults.Safe1[string]` | `"%s already registers a different merge rule for this kind"` (spec FR-019 duplicate-registration guard) |

### `internal/app/graph/service` (`errors.go`)

| Constant | Kind | Message |
|---|---|---|
| `ErrNotAGraph` | `faults.Safe1[string]` | `"%s is not an initialized graph"` |
| `ErrPatchRead` | `faults.Safe1[string]` | `"failed to read patch file %s"` |
| `ErrNodeWrite` | `faults.Safe1[string]` | `"failed to write %s"` |

## Validation rules (from spec Functional Requirements)

| Rule | Source | Enforced in |
|---|---|---|
| Manifest missing mandatory field → refuse, no writes | FR-001, FR-002 | `core.ParsePatch`, before `service.Apply` does anything else |
| Document already tracked → skip, no writes, no commit | FR-003 | `service.Apply` guard (`port.VCS.IsTracked`), before any write |
| Target not an initialized graph → refuse, no writes | FR-014 | `service.Apply` guard (`Store.Stat(".arc")`), before any write |
| Node kind not in resolved `MergeRuleSet` → apply with default `union` behavior, warn, do not refuse | FR-018 | `service.Apply`, via `MergeRuleSet.Lookup` for every node; appends to `ApplyResult.Warnings` (research.md D5-revised) |
| `arc init`'s config-seed fetch failing for any reason → fall back to core-only defaults, never fail initialization | `specs/002-arc-init/spec.md` FR-017 | `config.Default` (research.md D5-revised) — no `error` return by construction |
| Mid-run failure → no partial state | FR-015 | `service.Apply`, per-path `Store.Remove`/`File.Discard` against exactly the paths this run attempted (no `fsys.RemoveLocalRoot` — the graph root predates this run) |
| Scalar merge conflict → first-writer wins, flagged, commit still proceeds | FR-013 | `core.Merge` (research.md D6, D7) |
