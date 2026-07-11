# Implementation Plan: Full ARCNET-CORE §16 Conformance Checks for `arc lint`

**Branch**: `014-lint-spec-conformance` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/014-lint-spec-conformance/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

`arc lint` today only partially covers ARCNET-CORE v0.7 §16's conformance checklist: it never compares a
node's actual predicates against what its own registered type schema (`_schema/types/<type>.md`'s
`## Requires`/`## Optional`) demands, never validates `"@id"`/`"@type"` front-matter quoting, sources its
citation-predicate vocabulary from a hardcoded Go list instead of the graph's own schema, and never
checks that a predicate's occurrence position matches its schema-declared `role`. This feature adds five
new read-only checks to the existing `internal/app/lint/service.Lint` pipeline, consuming data
`internal/app/schema/service.Resolve` (spec 011) already resolves into `core.Index` — no new port,
adapter, storage, or CLI surface is introduced; this is additive validation logic inside an existing
command. Research (research.md D2) found one piece of the user-supplied technical approach already
implemented (`checkUnrecognizedKind` already reads the Schema Index, not a legacy `MergeRuleSet`) and one
genuine implementation risk the approach flagged correctly and this plan resolves concretely
(research.md D1: `"@id"`/`"@type"` quoting is real-only via raw-text regex detection, not enforceable
post-YAML-parse) and D4/D5/D7 (precise mapping from `core.Node` fields to schema roles, with an explicit
exemption for the pre-existing inline citation-tagging convention so the new role check doesn't
false-positive against it).

## Technical Context

**Language/Version**: Go 1.26 (matches `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra` (existing `arc lint` command, unchanged), `gopkg.in/yaml.v3` (already used by `internal/core` for front-matter decode; this feature adds no new dependency — the new identity-quoting check works on raw bytes, not a second YAML pass)

**Storage**: N/A — reads an already-mounted graph via the existing `fsys.Store`/`fsys.Mounter` ports; no new storage, no writes (lint remains strictly read-only)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` — table-driven unit tests per new rule function in `internal/app/lint/service` (mirroring `rules_identity_test.go`/`rules_predicates_test.go`), plus E2E additions to `cmd/arc/lint/lint_test.go` (constitution Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; unaffected by this feature — no platform-specific code)

**Project Type**: Single Cobra CLI binary (constitution Principle III) — no new command, this feature only extends the existing `arc lint` subcommand's domain logic

**Performance Goals**: No new performance budget — the five new checks run within the existing "Checking predicates and citations" `Reporter` phase of `internal/app/lint/service.Lint`, in the same O(nodes × occurrences) walk the existing checks in that phase already perform; spec 004's existing SC-004 (~30s for several thousand nodes) is the standing budget this feature must not regress

**Constraints**: Lint remains strictly read-only (no `fsys` writes, no git mutation) per its existing, explicitly documented invariant (constitution Principle VII, spec FR-012/FR-014); the five new checks MUST NOT report a false positive against any existing fixture graph the current lint test suite already asserts "0 violations" against — research.md D6 identifies exactly which two fixture surfaces need updating and why, and Phase 3 below schedules that as an explicit precondition task before the new checks are wired into the orchestrator

**Scale/Scope**: Same graph scale as existing `arc lint` (spec 004's SC-004: several thousand nodes, single run); no change to the graph's on-disk scale or the CLI's own scope — five new node-level checks, zero new commands/flags

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Assessment |
|---|---|
| I. Architecture Documentation & ADRs | PASS — no new architectural component; existing `internal/app/lint` package boundary is reused as-is. No ADR conflict: no accepted ADR governs lint's internal rule-function shape beyond what's already established by the package's own existing pattern (one `checkXxx` function per rule, `kernel.Violation`-returning), which this feature follows. |
| II. DDD & Glossary | PASS — no new domain term is introduced; "Requires"/"Optional"/"role"/"aligned" are already-established `core.TypeDef`/`core.PredicateDef` vocabulary (spec 011), reused verbatim, not renamed or reinterpreted. |
| III. Hexagonal Architecture | PASS — all new logic lives in `internal/app/lint/service` (domain), consuming the already-injected `core.Index` parameter `cmd/arc/lint/lint.go`'s `RunE` already resolves and passes in; no new `cmd/`-level logic beyond (at most) a `Long` help-text refresh mentioning the fuller checklist. |
| IV. Functional Style | PASS — each new check is a small, single-purpose `func(node core.Node, path string, raw []byte, index core.Index) []kernel.Violation`-shaped function (or narrower, per existing convention), composed into `Lint`'s existing per-node loop; no shared mutable state. |
| V. Code Quality & Simplicity (SOLID/YAGNI) | PASS — reuses the existing one-rule-per-function file organization (`rules_type_conformance.go` alongside the four existing `rules_*.go` files); no new abstraction/interface introduced beyond what's needed (index lookups are direct map reads, matching `checkUnrecognizedKind`'s existing style). |
| VI. TDD | Applies — table-driven unit tests per new rule function, written first, following `rules_predicates_test.go`'s established pattern (`it.Then(t).Should(...)`, no other assertion library). |
| VII. External Integration & Adapters | PASS — no new external integration; `core.Index` is already resolved by the existing `internal/app/schema/service.Resolve` call in `cmd/arc/lint/lint.go` before `Lint` runs. No filesystem write path is touched. |
| VIII. E2E Acceptance Testing | Applies — one E2E test per new spec.md acceptance scenario is required; `cmd/arc/lint/lint_test.go` gains new `Test...` functions per User Story 1-5, using the existing `sut()`/`buildConformantGraph` fixtures (updated per research.md D6). |
| IX. CLI/CLIG | PASS — no new flag, no new subcommand; `NewLintCmd`'s `Long` text SHOULD be updated to mention the extended checklist coverage (cosmetic, not a breaking change). |
| X. Terminal Output | PASS — new violations render through the existing `humanLintPrinter`/`verboseLintPrinter`/`jsonPrinter` unchanged; no new renderer needed (`kernel.Violation`'s shape is unchanged, only new `Rule` string values populate its existing `rule` field). |
| XI. Configuration | N/A — no new config surface. |
| XII. Documentation & Errors | Applies lightly — no new user-facing error condition (violations are findings, not errors, per existing convention); `Long`/help text SHOULD mention the fuller §16 coverage. |
| XIII/XIV. Release/Versioning | PASS — additive, backward-compatible: existing `--json` output gains new possible `rule` string values but no schema change (not a breaking change per Principle XIV's own "breaking = command/flag/output *schema* changes" definition); no version-bump-triggering change. |

**Result**: No violations. No Complexity Tracking entries required.

## Project Structure

### Documentation (this feature)

```text
specs/014-lint-spec-conformance/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── lint-rules-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/arc/lint/
├── lint.go                            # Existing — Long help text refresh only (mentions fuller §16 coverage); no flag/RunE structural change
└── lint_test.go                       # Existing — buildConformantGraph fixtures corrected (research.md D6); new Test... functions per User Story 1-5 acceptance scenario

internal/app/lint/
├── kernel/
│   └── lint.go                        # Existing — four new Rule constants added (RuleTypeRequires, RuleTypeOptional, RuleIdentityQuoting, RulePredicateRole)
├── service/
│   ├── lint.go                        # Existing — Lint orchestrator gains four new call sites in the existing "Checking predicates and citations" phase
│   ├── rules_type_conformance.go      # NEW — checkTypeRequires, checkTypeOptional, checkPredicateRole (User Stories 1, 2, 5)
│   ├── rules_type_conformance_test.go # NEW — table-driven unit tests for the three functions above
│   ├── rules_frontmatter.go           # Existing — gains checkIdentityKeyQuoting (User Story 3)
│   ├── rules_frontmatter_test.go      # Existing — gains unit tests for checkIdentityKeyQuoting
│   ├── rules_predicates.go            # Existing — checkCitationPredicate rewritten to take a registry param (User Story 4); citoPredicates hardcoded map deleted
│   ├── rules_predicates_test.go       # Existing — checkCitationPredicate tests updated for new signature; new dynamic-registry test cases added
│   └── locate.go                      # Existing — gains locateUnquotedIdentityKey (or equivalent) raw-text helper for User Story 3
└── lint_test.go (service package)     # Existing — coreIndexFixtureLint corrected to use kernel.CoreTypeDefs/CorePredicateDefs (research.md D6)
```

**Structure Decision**: No new command or domain package. This feature extends the existing `arc lint`
command (`cmd/arc/lint`) and its existing domain package (`internal/app/lint`), adding one new rule file
(`rules_type_conformance.go`) alongside the four that already exist there, following that package's
already-established one-function-per-rule, `kernel.Violation`-returning pattern. Two existing files
(`rules_frontmatter.go`, `rules_predicates.go`) are extended/modified in place rather than duplicated,
since the new checks they gain are natural extensions of the checks already living there (front-matter
identity checks; predicate/citation checks, respectively).

## Complexity Tracking

*No entries — Constitution Check found no violations requiring justification.*

## Bugfix: BUG-001 (2026-07-09)

Adds one new touched file outside this feature's original Project Structure section:
`internal/app/schema/kernel/schema.go` (and its test, `internal/app/schema/kernel/schema_test.go`) —
owned by spec `011-machine-readable-schema`, not by this feature, but this feature's own new checks
(FR-002 specifically) are what exposed the seed-data gap (spec.md FR-014–FR-020, research.md D8–D10), so
the fix is tracked here rather than reopening 011. No change to this feature's own Constitution Check
result: the fix is additive seed-data correction (new map entries, new `Optional`-list bullets), not a new
architectural pattern, domain type, or external integration — Principles I–XIV's assessments in the
Constitution Check table above are unaffected. `schema_test.go` gains assertions on `Optional` lists
(previously only `Required` was asserted, per `TestCoreTypeDefsRequiredListsMatchCoreSection11` — the
exact test gap that let this drift go unnoticed since spec 011 shipped).

## Bugfix: BUG-002 (2026-07-10)

Adds three more touched files/packages outside this feature's original Project Structure section, all
outside `internal/app/lint` — this feature's own new checks (FR-001, again) are what exposed the gap, but
the defects live in code owned by earlier specs:
- `internal/app/schema/kernel/schema.go` (spec 011, already touched by BUG-001): register `period`,
  remove `entries`, change `cites`'s `Merge`/`Description`, update `CoreTypeDefs["timeline"].Required`.
- `internal/core/timeline.go` + `internal/core/timeline_test.go` (spec 009): `TimelineEntry` gains a
  `cites::` predicate prefix on its rendered bullet.
- `internal/core/markdown.go` (spec 010, the shared AST parser) + its test file: `listItemPattern`'s
  trailing anchor relaxed to tolerate display-only annotation after a predicate-tagged wikilink.
- `internal/app/graph/service/apply.go` (spec 003/009): `timelineEntryPattern`'s re-parse regex gains the
  same optional `cites:: ` prefix tolerance, so re-applying to an already-existing (pre-fix) period file
  doesn't lose or duplicate entries.

No change to this feature's own Constitution Check result: still no new architectural pattern, domain
type, or external integration. Principle IV (Functional Programming Style) is worth a specific note —
`TimelineEntry`'s change is a pure rendering-format change, no new side effects; `listItemPattern`'s
relaxation is a pure regex change with no new mutable state. `internal/core`'s package boundary (the
project's first core-domain package) is unaffected — no new type, no new exported function, only two
existing functions' output/matching shape corrected to match ARCNET-CORE's own documented convention
(`TimelineEntry`) or to stop silently dropping data (`listItemPattern`).
