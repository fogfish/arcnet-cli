# Phase 1 Data Model: `arc lint`

Value types are immutable (constitution Principle IV) and carry no Cobra, `os/exec`, raw `os.*` filesystem, or goldmark AST types. `internal/core.Node`/`ParseNode` are reused unchanged (research.md D3) — no new type is added to `internal/core` by this feature.

## Application values (`internal/app/lint/kernel`)

### Rule

`Rule` is a `string` constant identifying exactly one CORE §14 checklist item, so every `Violation` names precisely which rule fired without a second lookup table:

| Constant | CORE reference | Checked by (research.md) |
|---|---|---|
| `RuleFrontMatter` | §14 "valid YAML front-matter and `kind`" | D2 |
| `RuleUniqueBasename` | §3.2 | D4 |
| `RuleLinkResolves` | §3.2 | D5 |
| `RuleSourceCitekey` | §6.2 | D6 |
| `RuleEntityCategory` | §9.2.1 | D7 |
| `RuleDerivedProvenance` | §3.4 | D8 |
| `RulePredicateCase` | §7.3 (camelCase) | D9 |
| `RulePredicateRegistered` | §7.3 (registration) | D9 |
| `RuleCitationPredicate` | §8 | D10 |
| `RuleUnrecognizedKind` | §10/§14 (extension conformance) | D11 |
| `RuleIngestCommit` | §11.1 | D12 |
| `RuleMergeConflict` | (§14 "no active merge conflicts") | D13 |

### Violation

The domain value one failed check produces.

| Field | Type | Notes |
|---|---|---|
| `Rule` | `Rule` | Which checklist item failed |
| `Path` | `string` | Node file path, relative to the graph root |
| `Line` | `int` | 1-based line number within `Path`; `0` means "not applicable" (spec FR-015 — e.g. a basename collision spanning two files, or a missing-provenance-link absence) |
| `Message` | `string` | Human-readable detail (e.g. the unresolved target name, the colliding basenames, the commit count found) |
| `RelatedPaths` | `[]string` | Populated only for violations spanning more than one file (`RuleUniqueBasename`'s colliding paths); empty otherwise |

### NodeStatus

One enumerated node's overall outcome, the unit `--verbose` output lists one of per node (research.md D14).

| Field | Type | Notes |
|---|---|---|
| `Path` | `string` | Relative to graph root |
| `ID` | `string` | Parsed node identity (empty when `RuleFrontMatter` itself failed and `core.ParseNode` never ran) |
| `Kind` | `core.Kind` | Empty when unparseable |
| `Violations` | `[]Violation` | Empty means this node passed every applicable check |

### LintResult

The domain value `component.go`'s `Lint` returns to `cmd/arc/lint`, rendered by `bios.Registry[LintResult]`.

| Field | Type | Notes |
|---|---|---|
| `Root` | `string` | The graph root that was linted |
| `Nodes` | `[]NodeStatus` | Every enumerated node, in walk order — the `Verbose` renderer's source; the `Human` renderer filters this to entries with non-empty `Violations` |
| `Violations` | `[]Violation` | Flattened view of every `NodeStatus.Violations` plus file-spanning violations with no single owning node (`RuleUniqueBasename`, `RuleIngestCommit`) — the `--json` schema's primary array |
| `Passing` | `int` | Count of nodes with zero violations |
| `Failing` | `int` | Count of nodes with at least one violation |

`Passing + Failing == len(Nodes)`. `LintResult` carries no field for "did the run succeed" beyond `len(Violations) == 0` — `cmd/arc/lint` derives the exit code (research.md D14) from that count directly, rather than duplicating it as a redundant boolean.

## Ports

### `internal/app/lint/port.VCS`

The narrowest of the three `port.VCS` interfaces in this codebase (`ctrl`, `graph`, now `lint`) — lint never initializes, stages, or commits.

```go
type VCS interface {
    CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)
}
```

Satisfied structurally by the same shared `internal/adapter/git.Git` concrete type `ctrl.port.VCS`/`graph.port.VCS` already use (research.md D12, ADR 001 port isolation rule 1) — `git.Git` gains one new method, no new adapter package.

## Filesystem I/O

All reads go through `fsys.Store` (`internal/adapter/fsys`, already shared — no changes to that package). `arc lint` mounts the graph root the same way `arc apply` does and calls `fsys.Store.Stat(".arc")` for the same "is this an initialized graph" guard `graph.service.guardIsGraph` already implements (research.md/plan.md Constraints) — **lint never calls `Store.Create`, `Store.Remove`, or any `File` write method anywhere in its execution path** (spec FR-014); this is verified by SC-006 (byte-for-byte identical graph state before/after any run).

## Reporter events (`internal/bios.Reporter`, ADR 002 DS-06)

| Label | Emitted around |
|---|---|
| `"Reading graph"` | Enumerating every node file (research.md D2) and parsing each via `core.ParseNode` |
| `"Checking basenames and links"` | D4/D5 — uniqueness and link-resolution passes |
| `"Checking predicates and citations"` | D9/D10 |
| `"Checking commit history"` | D12 — one `port.VCS.CommitsMatching` call per distinct `source` node |

Each label gets one `Start`/`Done` pair (flat `Reporter`, matching `arc init`/`arc apply`'s precedent — lint has no multi-phase task tree deep enough to warrant DS-08's richer pattern). No `Reporter.Step` per-node calls are added (unlike `arc apply`'s FR-021) — `--verbose`'s per-node detail is the *primary result* (`LintResult.Nodes`, rendered by the `Verbose` printer), not incidental progress narration, so it belongs in the renderer, not the `Reporter`.

## Error sentinels (`github.com/fogfish/faults`)

### `internal/app/lint/service` (`errors.go`)

| Constant | Kind | Message |
|---|---|---|
| `ErrNotAGraph` | `faults.Safe1[string]` | `"%s is not an initialized graph"` |
| `ErrPredicatesUnreadable` | `faults.Safe1[string]` | `"failed to read %s"` (a real I/O failure reading `_meta/predicates.md` — distinct from the file simply not existing, which is not an error per research.md D9) |

A per-node parse failure (`RuleFrontMatter`) is **not** an error sentinel — it is a `Violation`, since spec FR-013 requires the run to continue past it, not abort (unlike `arc apply`'s all-or-nothing manifest validation, lint's entire purpose is to surface exactly this kind of per-file defect without stopping).

## Validation rules (from spec Functional Requirements)

| Rule | Source | Enforced in |
|---|---|---|
| Every node has valid front-matter + `kind` | FR-001 | `service.checkFrontMatter`, first check per file (after D13's conflict-marker pre-pass) |
| Basenames unique across the graph | FR-002 | `service.checkUniqueBasenames`, over the full D2 enumeration |
| Every `[[link]]` resolves | FR-003 | `service.checkLinksResolve`, second pass once the basename index exists |
| `source.id == basename` | FR-004 | `service.checkSourceCitekey` |
| `entity.category` is a valid four-word Sowa bag (positional array, this codebase's established on-disk form) | FR-005 | `service.checkEntityCategory` against research.md D7's fixed positional word-sets |
| Derived node links to at least one `source` | FR-006 | `service.checkDerivedProvenance` |
| Predicate is camelCase | FR-007 | `service.checkPredicateCase` |
| Predicate is registered in `_meta/predicates.md` | FR-008 | `service.checkPredicateRegistered` against research.md D9's parsed registry |
| Citation predicate is `cito:`-aligned | FR-009 | `service.checkCitationPredicate` against research.md D10's fixed set, over `HRefs` entries with a non-empty `Predicate` |
| One `graph(ingest):` commit per document | FR-010 | `service.checkIngestCommit`, via `port.VCS.CommitsMatching` |
| Extension-kind profile conformance (scoped, research.md D11) | FR-011 | `service.checkUnrecognizedKind` against the resolved `core.MergeRuleSet` |
| No active merge-conflict markers | FR-012 | `service.checkConflictMarkers`, pre-pass before any other check for that file (research.md D13) |
| Never stop at the first violation | FR-013 | `service.Lint`'s top-level loop collects every check's result before returning, for every enumerated node |
| Never write to the graph | FR-014 | Structural — `service.Lint` never receives anything but read-only `fsys.Store` methods and `port.VCS.CommitsMatching` in its call graph |
| Every violation names rule + file + line (or "not applicable") | FR-015 | `kernel.Violation`'s shape itself; `Line == 0` is the "not applicable" case |
| Exit status distinguishes pass/fail | FR-016 | `cmd/arc/lint`, derived from `len(LintResult.Violations)` |
| Refuse when target is not an initialized graph | FR-017 | `service.Lint` guard, before any enumeration begins |
| Unrecognized kind is itself a violation, not silently passed | FR-018 | `service.checkUnrecognizedKind` |
