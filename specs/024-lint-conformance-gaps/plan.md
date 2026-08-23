# Implementation Plan: Lint Conformance Gaps

**Branch**: `024-lint-conformance-gaps` | **Date**: 2026-08-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/024-lint-conformance-gaps/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

`arc lint` currently validates an `Entity`'s four-word Sowa category one word-position at a time
(144 accepted combinations) instead of against ARCNET-CORE §10.2's twelve legal rows, and has no
rule at all for §7.1's identity charset. This feature (1) replaces the four independent word-set
checks with an exact-row lookup against the twelve legal combinations, suggesting the closest
legal row on rejection; (2) adds a new `arc lint` rule flagging any node identity — content or
schema — containing a forbidden filesystem character, naming each character and its position;
and (3) enforces that same charset rule inside `arc apply`, rejecting the whole operation before
any file is written, so an unsafe identity can never enter the graph in the first place. All three
checks reuse existing plumbing already established by adjacent rules (`ValidSowaCategory`,
`checkSchemaTypeCase`'s graph-spanning pattern, `guardNoOldFormatNodes`'s pre-write-scan pattern,
`isCamelCase`/`ErrTypeCasing`'s shared-primitive-in-`internal/core` pattern) rather than
introducing new architecture.

## Technical Context

**Language/Version**: Go 1.26.5 (`go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`,
`github.com/fogfish/faults` (error annotation, Principle XII) — no new dependency added by this
feature.

**Storage**: Local git-versioned graph directory (existing `internal/adapter/fsys`/`internal/adapter/git`
adapters, unchanged — this feature adds no new I/O surface).

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII).

**Target Platform**: linux/darwin/windows, amd64+arm64 (`.goreleaser.yaml`) — unchanged.

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature extends two
existing commands (`arc lint`, `arc apply`); it adds no new command.

**Performance Goals**: N/A — both checks are O(1) table lookups / O(n) rune scans over short
strings, negligible next to existing graph-wide file I/O.

**Constraints**: Detection-only for both `arc lint` checks (spec.md FR-006) — no file rewritten,
no `--fix` behavior. `arc apply` enforcement (spec.md FR-001–FR-003) must be atomic: either the
whole patch applies or nothing is written, using the existing `rollback(store, createdPaths)`
mechanism (`internal/app/graph/service/apply.go`) already exercised by every other per-node
failure in `Apply` — no new rollback mechanism.

**Scale/Scope**: Two existing packages touched (`internal/app/lint/*`, `internal/app/graph/service`),
one new shared primitive in `internal/core`. No new package, no new command.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (ADRs binding)**: No architectural decision here contradicts [ADR
  001](../../adrs/001-system-architecture.md) (hexagonal layering) or [ADR
  002](../../adrs/002-ux-design-system.md) (CLI/UX) — this feature adds domain logic inside
  existing `kernel`/`service` layers of two existing domains, no new adapter, no new port. PASS.
- **Principle III (hexagonal boundaries)**: All new logic lives in `internal/app/lint/kernel`,
  `internal/app/lint/service`, `internal/app/graph/service`, and `internal/core` — no `cobra`
  import, no `cmd/`-package type reaches any of them. `cmd/arc/lint` and `cmd/arc/graph` gain no
  new logic beyond existing E2E test fixtures. PASS.
- **Principle IV/V (functional style, SOLID/YAGNI)**: The Sowa-table replacement and the
  charset-scan primitive are both pure functions over pure data — no mutation, no new interface,
  no premature abstraction (research.md D6 explicitly rejects a shared package between
  `internal/app/lint` and `internal/app/graph/service`, keeping `internal/core` as the one
  pre-existing shared dependency both already have). PASS.
- **Principle VI (TDD)**: Test strategy (below) writes the 144-combination table test, the
  10-forbidden-character unit test, and new E2E fixtures before implementation, per the feature's
  own plan input. PASS, enforced during Phase 2/implementation, not by this planning phase itself.
- **Principle VII (adapters)**: No new external integration; no new adapter. PASS trivially — N/A.
- **Principle VIII (E2E traceability)**: Every spec.md acceptance scenario maps to an E2E test
  extending the existing `cmd/arc/lint/lint_test.go` and `cmd/arc/graph/apply_test.go` (both
  already exist and already use the `sut()` pattern) — no new test tree. PASS.
- **Principle XII (error messages)**: New `arc apply` rejection uses `faults.SafeN`
  (`ErrIdentityCharset`, data-model.md §5), not ad hoc `fmt.Errorf`, matching every other error in
  `internal/app/graph/service/errors.go`. PASS.

No violations requiring justification — Complexity Tracking table is empty (omitted).

## Project Structure

### Documentation (this feature)

```text
specs/024-lint-conformance-gaps/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── sowa-category-contract.md
│   └── identity-charset-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/
│   ├── identity.go          # NEW — forbiddenIdentityChars + scan primitive (data-model.md §3, research.md D6)
│   ├── identity_test.go     # NEW — unit: 10 forbidden chars + 1 legal control (C2.1)
│   └── errors.go            # + ErrIdentityCharset (data-model.md §5)
│
├── app/lint/
│   ├── kernel/
│   │   ├── lint.go          # sowaPosition1..3/sowaLeaf → sowaCategories table; + RuleIdentityCharset
│   │   └── lint_test.go     # unit: exhaustive 144-combination table (C1.1)
│   └── service/
│       ├── rules_identity.go        # + checkIdentityCharset (content nodes)
│       ├── rules_types_case.go      # + checkSchemaIdentityCharset (schema nodes, mirrors checkSchemaTypeCase)
│       └── lint.go                  # wire both new checks into Lint's existing loops
│
└── app/graph/service/
    ├── apply.go              # + pre-scan guard (patch.Nodes identities) + in-loop checks (node.Type, predicate names)
    └── errors.go             # (see internal/core/errors.go above; apply.go references it)

cmd/arc/
├── lint/lint_test.go         # + E2E fixtures: one identityCharset (content), one identityCharset (schema), one entityCategory violation
└── graph/apply_test.go       # + E2E fixture: patch with unsafe identity → rejected, graph unchanged

testdata/                     # fixtures colocated with the E2E tests above (Principle VIII)
```

**Structure Decision**: This feature adds no new command and no new domain package. It extends
`arc lint` (`internal/app/lint/kernel`, `internal/app/lint/service`) and `arc apply`
(`internal/app/graph/service`), and introduces one new small file, `internal/core/identity.go`,
holding the one shared forbidden-character primitive both commands consume (research.md D6) — the
same package that already hosts the analogous `isCamelCase`/`ErrTypeCasing` precedent for a
different node-shape invariant (spec 019).

## Complexity Tracking

*No entries — Constitution Check recorded no violations.*
