# Implementation Plan: ARCNET-CORE v0.11 — `Reference` Type and Type-Named Folders

**Branch**: `022-reference-type-folders` | **Date**: 2026-08-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/022-reference-type-folders/spec.md`

## Summary

Bring the tool to ARCNET-CORE v0.11 on three axes: correct the `Resource`/`Reference` role
mix-up in the seeded type vocabulary, key each type's leading body prose to the predicate that
type actually declares, and file every node in a folder named character-for-character for its
type. Accepted as a **breaking change** — no migration (spec § Clarifications).

The technical shape, after the Phase 0 verification pass, is smaller and differently placed
than the planning input assumed:

- **The folder rename is four literal edits plus one function rewrite**, not a sweep. Nothing
  in the codebase infers `@type` from a folder, traversal matches folders by exclusion rather
  than by whitelist, and edges resolve by basename. The remaining volume is test-fixture churn
  across 28 files.
- **`nodeFolder` collapses to the identity function and `coreKindFolders` is deleted.** Under
  §6 the map's every entry would be `k → k`. This dissolves the `References/`-vs-`references/`
  case hazard entirely: with no transform, there is nothing for a case-insensitive filesystem
  to hide.
- **The union-cannot-retract defect is confirmed but unreachable.** `Seed()` renders straight
  to disk without merging, so `arc init` installs exact definitions; only `arc apply schema`
  merges, and nothing routes the corrected types through it.
- **The break fails closed.** A pre-v0.11 graph lacks `_schema/Property/` and `_schema/Class/`;
  schema resolution treats a missing schema folder as a hard load failure, so every command
  refuses and names the missing folder before writing anything.

Verification also surfaced **three hazards the input did not name**: `DefaultLayout.Folders`
and `nodeFolder` are independent and can drift silently; `pluralizeKind` looks like a folder
deriver but is display-only; and `revertLeadingKey` is a hand-copied table already out of sync
with `core`'s on three domain types. All three are addressed in research.md and carried into
the test plan.

## Technical Context

**Language/Version**: Go 1.27.0 (`go.mod` declares `go 1.24`; toolchain at
`/opt/homebrew/Cellar/go/1.27.0/libexec` — `GOROOT` must be exported, the shell default is stale)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`,
`github.com/fogfish/faults`, `github.com/fogfish/it/v2`. **No new dependency.**

**Storage**: Local git working tree via `internal/adapter/fsys` (stdlib `io/fs` + the
project's writable-file extension). No new adapter.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`. Baseline is green at `f64863b`.

**Target Platform**: darwin/linux/windows, amd64+arm64 per `.goreleaser.yaml`. **Case-sensitivity
matters here**: the developer machine is APFS (case-insensitive), CI is not.

**Project Type**: Single Cobra CLI binary.

**Performance Goals**: N/A — no hot path touched.

**Constraints**:
- No new command, no new package, no new flag. The command surface is unchanged.
- Every path assertion compares the exact string handed to the store, never a filesystem stat.
- `pluralizeKind` (display) must not be conflated with `nodeFolder` (paths).

**Scale/Scope**: 8 production files changed; ~28 test files carry fixture-path churn.

## Constitution Check

*GATE: passed before Phase 0; re-checked after Phase 1 — still passing.*

| Principle | Assessment |
|---|---|
| **I — ADRs binding** | Read `adrs/001-system-architecture.md`, `002-ux-design-system.md`, `003-mcp-server-adapter.md`. No plan decision contradicts any. `ARCHITECTURE.md` **must** be updated in the same PR: the **Canonical Folder** glossary entry (line 232) enumerates the old folder set, **Predicate/Type Schema Node** (234-235) cite `_schema/predicates/`+`_schema/types/`, the package tree comment (121-122) cites both, and **Entity/Resource Node** (247) carries the retired `Resource` meaning. A `Reference` glossary entry must be added. |
| **II — DDD & glossary** | `Reference` is a new domain concept → glossary entry required (Principle I row). Ubiquitous language holds: the type name, the folder name, and the `@type` value are now literally the same string — v0.11's §6 is a glossary constraint as much as a layout one. |
| **III — Hexagonal** | No `cmd/` logic added. All changes land in `internal/core`, `internal/app/*/kernel`, `internal/app/*/service`. `cmd/` is untouched except test fixtures. |
| **IV — Functional style** | `nodeFolder` becomes a one-line pure function — a net simplification. No inline comments added; GoDoc only. Existing GoDoc that documents *why* an invariant holds (e.g. `nodeFolder` is never called with `Timeline`) is preserved, not deleted. |
| **V — YAGNI** | The declined `arc migrate` registry is the YAGNI call: a versioned migration mechanism with exactly zero migrations to run. Deleting `coreKindFolders` rather than extending it is the same principle applied to the folder map. |
| **VI — TDD** | Red first. Tests are ordered ahead of implementation in the task tranches below. `it/v2` only. |
| **VII — Adapters** | No new external integration. `fsys` unchanged. |
| **VIII — E2E traceability** | Every acceptance scenario in spec.md §US1–US3 maps to a colocated `cmd/**/..._test.go` case via `sut()`. |
| **XII — faults** | No new error type needed. `ErrSchemaMissing` already exists and already carries the failing directory. |

**No violations. Complexity Tracking table omitted — nothing to justify.**

## Project Structure

### Documentation (this feature)

```text
specs/022-reference-type-folders/
├── plan.md                                   # This file
├── spec.md
├── research.md                               # Phase 0 — folder-impact analysis + D1..D9
├── data-model.md                             # Phase 1 — changed values & functions
├── quickstart.md                             # Phase 1 — manual validation walkthrough
├── contracts/
│   ├── folder-layout-contract.md             # CORE §6 as a checkable contract
│   └── core-type-vocabulary-contract.md      # CORE §11.4/§11.6 + prose-key table
├── checklists/requirements.md
└── tasks.md                                  # Phase 2 — NOT created by /speckit-plan
```

### Source Code — production files touched

```text
internal/
├── core/
│   └── markdown.go                      # textPredicateFor: Resource→text, +Reference→relevance
└── app/
    ├── ctrl/kernel/graph.go             # DefaultLayout.Folders → the eight type-named folders
    ├── schema/kernel/schema.go          # CoreTypeDefs[Resource] redefined; [Reference] added;
    │                                    #   CoreTypeBases += Reference; TypesDir/PredicatesDir
    │                                    #   values; 5 predicate descriptions reworded
    └── graph/service/
        ├── apply.go                     # coreKindFolders DELETED; nodeFolder → identity
        └── revert.go                    # revertLeadingKey: Resource→text, +Reference

ARCHITECTURE.md                          # glossary + package tree (Principle I, mandatory)
```

**Structure Decision**: No new command and no new package. The feature is a value change in
three `kernel` packages plus a two-line behavioural change in `internal/core` and
`internal/app/graph/service`. `cmd/` gains no code — only fixture updates in its tests. This is
deliberate: `spec.md`'s no-migration ruling is what keeps the change inside `internal/`, since
the alternative (`arc migrate`) would have required a new `cmd/arc/migrate` package, a `.arc/`
version marker, and a migration registry.

## Implementation tranches

Ordered for TDD and for reviewable diffs. Each tranche is independently green.

**T1 — Guard tests for the drift hazards (red first).**
The two invariant tests from research.md §6 that do not exist today: `DefaultLayout.Folders` ↔
`nodeFolder` agreement (D3), and `core.TextPredicateFor` ↔ `revertLeadingKey` agreement for the
five core types (D6b). Written against the *target* values, so both start red.

**T2 — Type vocabulary.** `CoreTypeDefs`, `CoreTypeBases`, predicate descriptions. Seed golden
tests. Delivers spec US1.

**T3 — Prose predicate.** `textPredicateFor` + `revertLeadingKey`, the 10-case table test, and
the parse→render→parse fixtures. Turns half of T1 green. Delivers spec US2.

**T4 — Folder derivation.** Delete `coreKindFolders`, `nodeFolder` → identity, `TypesDir`/
`PredicatesDir` values, `DefaultLayout.Folders`. Turns the rest of T1 green. Delivers spec US3.
Check whether `strings` is still used in `apply.go`.

**T5 — Test-fixture migration.** The mechanical path substitution across ~28 test files. Kept
separate so the behavioural diff in T1–T4 stays reviewable.

**T6 — `ARCHITECTURE.md`.** Glossary (Canonical Folder, Predicate/Type Schema Node,
Entity/Resource Node, new Reference entry) and the package tree comment. Principle I gate.

## Risks

| Risk | Mitigation |
|---|---|
| `pluralizeKind` "fixed" to identity → reporter prints "`+3 Entity`" | research.md D4 names it explicitly; add a GoDoc line; `cmd/arc/graph/apply_test.go` already asserts the summary string |
| `DefaultLayout.Folders` / `nodeFolder` drift — silent, no runtime error | T1 invariant test (C6 in the folder contract) |
| Case error invisible on APFS | Every path assertion compares the exact store string; contract C7 makes it normative |
| A missed fixture path leaves a test green against a stale layout | T5 done as one sweep; `go test ./...` plus a repo-wide grep for the six retired literals as the completion check |
| Reviewer reads the pasted planning input and expects `references/` + `arc migrate` | research.md §0 tabulates every superseded instruction with its supersession |

## Phase 2 Design Preconditions — recorded decisions

Recorded per the constitution's Compliance Checklist (PRECONDITIONS). These four are gates
whose deliverable is a decision, not code; this section is the durable record and doubles as
the PR description's compliance block.

**T006 — No new Go type (Principles II, V).** Confirmed by inspection at `f64863b`. The whole
domain change is *values* carried by four existing declarations plus two function bodies:
`schema/kernel.CoreTypeDefs` (one entry redefined, one added), `schema/kernel.CoreTypeBases`
(one entry added), `schema/kernel.TypesDir`/`PredicatesDir` (two string literals),
`ctrl/kernel.DefaultLayout.Folders` (rewritten), `graph/service.nodeFolder` (body → identity,
`coreKindFolders` deleted), `core.textPredicateFor` + `graph/service.revertLeadingKey` (two
switch cases each). No `struct`, `interface`, named type, or exported function signature is
introduced or altered. `core.TypeDef`/`core.PredicateDef` already model everything `Reference`
needs, so the new type is a map entry, not a type.

**T007 — Command/flag surface unchanged (Principle IX).** No subcommand, no flag, no exit
code, no `--json` field, and no help-text change. `cmd/` gains no production code at all; the
only `cmd/` edits in this feature are test fixtures and one GoDoc line on `pluralizeKind`
(T039, behaviour-preserving). No `contracts/` command table is therefore required — the two
contracts this feature carries are
[contracts/folder-layout-contract.md](contracts/folder-layout-contract.md) and
[contracts/core-type-vocabulary-contract.md](contracts/core-type-vocabulary-contract.md),
both of which constrain on-disk shape rather than CLI surface. The one *observable* CLI change
is behavioural and intended: a pre-v0.11 graph now fails schema resolution with the existing
`ErrSchemaMissing`, naming `_schema/Property` (research.md §5).

**T008 — External integration N/A (Principle VII).** No new external system, no new port, no
new adapter. All I/O continues through the existing `internal/adapter/fsys` store, which is
unchanged: the feature alters *which path strings* are handed to it, never how it is reached.
No vendor SDK type is involved, and no dependency is added to `go.mod`.

**T017 — Configuration & secrets N/A (Principle XI).** No configuration value, environment
variable, XDG location, or secret is introduced, read, or read differently. The folder layout
this feature changes is a static compile-time constant (`ctrl/kernel.DefaultLayout`),
deliberately not user-configurable, and `internal/app/config` is untouched. Nothing this
feature emits is sensitive, so no logging redaction question arises.
