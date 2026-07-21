# Implementation Plan: CamelCase Node Class Names

**Branch**: `019-camelcase-node-types` | **Date**: 2026-07-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/019-camelcase-node-types/spec.md`

## Summary

Make CamelCase (first letter uppercase) the enforced convention for node
class/type names everywhere they are produced or checked: the built-in
schema's four content types (`source`/`entity`/`resource`/`timeline` are
renamed to `Source`/`Entity`/`Resource`/`Timeline` in
`internal/app/schema/kernel/schema.go`, which `arc init`'s existing
`Seed()` automatically propagates into seeded filenames/`@type` values);
`internal/core.patchNodeIdentity` stops lowercasing a patch's H1 heading
and instead preserves its casing verbatim, rejecting (via a new
`ErrTypeCasing`) any H1 heading or explicit `@type` that does not start
with an uppercase letter; and `arc lint` gains a new `RuleTypeCase`
violation (mirroring the existing `RulePredicateCase` pattern, inverted)
that flags any schema type definition or node's `@type` reference that
isn't CamelCase. Every remaining literal `"source"`/`"entity"`/
`"resource"`/`"timeline"` string comparison across
`internal/app/graph/service`, `internal/app/lint/service`, and
`cmd/arc/graph/apply.go` is mechanically renamed to match, while the
graph's physical directory layout (`sources/`, `entities/`, `resources/`,
`timeline/`) deliberately stays lowercase-plural and unchanged
(research.md D5) — this feature changes the `@type` value's casing and the
`_schema/types/*.md` filenames, not the graph's on-disk folder structure.

## Technical Context

**Language/Version**: Go 1.26.5 (go.mod)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults` (new `internal/core.ErrTypeCasing` constant), `github.com/fogfish/it/v2`; no new external dependency

**Storage**: Local filesystem via `internal/adapter/fsys` (`_schema/types/*.md` filenames/content change casing; graph content folders unchanged — research.md D5)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated with `cmd/arc/graph/apply_test.go`, `cmd/arc/ctrl/init_test.go`, `cmd/arc/lint/lint_test.go` via the `sut()` helper (Principle VIII), one per spec.md acceptance scenario; unit tests in `internal/core/markdown_test.go` and a new `internal/app/lint/service/rules_types_case_test.go`

**Target Platform**: linux/darwin/windows amd64+arm64 (`.goreleaser.yaml`, unchanged by this feature)

**Project Type**: Single Cobra CLI binary (Principle III)

**Performance Goals**: N/A — parsing/validation cost is unchanged order-of-magnitude (one rune check per heading/type, one regex match per node/schema-type in lint); no throughput target

**Constraints**: No graph mutation on rejection (FR-005) — `patchNodeIdentity` fails before any node is constructed, so `internal/app/graph/service.Apply`'s existing all-or-nothing patch semantics require no new rollback logic

**Scale/Scope**: One domain-level validation rule (`internal/core`), one kernel data rename (`internal/app/schema/kernel`), one new lint rule (`internal/app/lint`), ~7 mechanical literal renames across `internal/app/graph/service`/`internal/app/lint/service`/`cmd/arc/graph`, plus an estimated ~28 existing test files whose fixtures/assertions use the old lowercase literals (research.md D8) — no new command, port, or adapter

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (ADRs are binding)**: Checked ADR 001 (system architecture) and ADR 002 (UX design system) for any existing casing convention for node/type names — none found; ADR 002's only casing rules concern the CLI *binary name* (DS-01), unrelated to graph content types. No conflict, no new ADR needed. PASS.
- **Principle III (Hexagonal Architecture)**: The CamelCase validation gate lives in `internal/core` (patch-parsing domain logic, already Cobra-free); the new lint rule lives in `internal/app/lint/service` (existing domain package); `cmd/arc/graph/apply.go`/`cmd/arc/lint/lint.go` are unchanged except for the mechanical `pluralizeKind` literal rename — no business logic added to `cmd/`. PASS.
- **Principle IV/V (Functional style, SOLID)**: New code (`isCamelCase` helper, `checkNodeTypeCase`/`checkSchemaTypeCase`) mirrors the existing `camelCasePattern`/`checkPredicateCase` shape exactly (research.md D6) — no new abstraction invented, same small-function/single-responsibility shape as its precedent. PASS.
- **Principle VI/VIII (TDD & E2E Testing)**: Every spec.md acceptance scenario gets a colocated E2E test (`cmd/arc/graph/apply_test.go` for US1, `cmd/arc/ctrl/init_test.go` for US2, `cmd/arc/lint/lint_test.go` for US3), written first per red-green-refactor; unit tests for `patchNodeIdentity`'s new casing gate in `internal/core/markdown_test.go`, and for `checkNodeTypeCase`/`checkSchemaTypeCase` in a new `internal/app/lint/service/rules_types_case_test.go`, mirroring `rules_predicates_test.go`'s existing structure. PASS.
- **Principle VII (External Integration)**: No new external system, port, or adapter. N/A.
- **Principle IX (CLIG Compliance)**: No command, flag, or positional-argument change; the new `ErrTypeCasing` message is itself the human-readable guidance Principle XII requires (`faults.Safe1`, matching every other error in the codebase). PASS.
- **Principle XIV (Versioning)**: This is a scriptable-behavior change without a flag/field rename — contracts/cli-contract.md documents it explicitly as a breaking behavior change requiring a major-version release note per Principle XIV, even though no `--json` field is renamed. Flagged for the release process, not a plan blocker.

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/019-camelcase-node-types/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── cli-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/core/
├── errors.go               # new: ErrTypeCasing = faults.Safe1[string](...)
├── markdown.go              # patchNodeIdentity: drop strings.ToLower(typeHeading);
│                            #   new isCamelCase helper gates heading + explicit @type (FR-004/005/008);
│                            #   textPredicateFor switch keys renamed (research.md D7)
└── markdown_test.go         # new/updated cases: accepted CamelCase heading, rejected
                             #   lowercase heading, rejected lowercase explicit @type

internal/app/schema/kernel/
└── schema.go                # CoreTypeDefs/CoreTypeBases: source/entity/resource/timeline
                              #   keys renamed to Source/Entity/Resource/Timeline (research.md D4)

internal/app/graph/service/
├── apply.go                  # coreKindFolders keys renamed; node.Type=="timeline"/"source"
│                              #   literal comparisons renamed (research.md D5/D7)
└── apply_test.go             # existing Node{Type: "..."} literals updated

internal/app/lint/kernel/
├── lint.go                   # new: RuleTypeCase Rule = "typeCase"
└── lint_test.go               # TestRuleConstantsAreDistinct gains RuleTypeCase

internal/app/lint/service/
├── rules_types_case.go        # new: checkNodeTypeCase, checkSchemaTypeCase (research.md D6)
├── rules_types_case_test.go   # new: unit tests mirroring rules_predicates_test.go
├── rules_identity.go           # node.Type != "source"/"entity" literals renamed
├── rules_links.go              # node.Type == "source"/"timeline" literals renamed
├── rules_history.go            # node.Type != "source" literal renamed
└── lint.go                     # wires checkNodeTypeCase into per-node loop;
                                 #   checkSchemaTypeCase called once against index

cmd/arc/graph/
├── apply.go                    # pluralizeKind's kind == "entity" literal renamed
└── apply_test.go               # new E2E: reject-lowercase-H1, accept-CamelCase-H1

cmd/arc/ctrl/
└── init_test.go                 # new/updated E2E: assert seeded schema filenames/@type CamelCase

cmd/arc/lint/
└── lint_test.go                 # new E2E: typeCase violation reported for a hand-authored
                                  #   lowercase _schema/types/ document
```

**Structure Decision**: No new package, command, port, or adapter. This
feature is a validation-rule addition (`internal/core`, `internal/app/lint`)
plus a data rename (`internal/app/schema/kernel`) plus mechanical literal
updates in the two existing consumers of the old lowercase type strings
(`internal/app/graph/service`, `cmd/arc/graph`). All new code follows the
directly-precedented shape already established by `RulePredicateCase`/
`camelCasePattern` (research.md D6) and by the existing `faults.Safe1`
error constants (research.md D3) — no new architectural pattern is
introduced.

## Complexity Tracking

*No violations — table omitted.*
