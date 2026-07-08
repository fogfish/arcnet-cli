
# Implementation Plan: Machine-Readable Predicate & Type Schema

**Branch**: `011-machine-readable-schema` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/011-machine-readable-schema/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Replace `internal/app/schema`'s existence-only schema registry (`_schema/nodes/<kind>.md` carrying only `id`/`merge`; `_schema/predicates/<name>.md` carrying only `id`) with a fully machine-readable one matching ARCNET-CORE v0.7 §9: `_schema/predicates/<name>.md` becomes a real `Property` node declaring `role`/`merge`/optional `label`/`aligned` plus a `description` body; `_schema/nodes/` is renamed to `_schema/types/` and each `<name>.md` becomes a real `Class` node declaring `required`/`optional` predicate lists plus a `description` body. A new shared `core.Index` type (`internal/core`, alongside the existing `core.MergeRuleSet`/`core.MergeOp` it retires) is built once per command invocation by `internal/app/schema/service.Resolve`, replacing the `(core.MergeRuleSet, map[string]bool)` tuple threaded through `arc apply`/`arc lint` today. `arc init`'s `Seed()` renders the complete CORE §10/§11 vocabulary (identity/content/metadata/structural/semantic/citation/type-specific/schema-own predicates; the four core types plus `Property`/`Class` themselves) instead of today's 13 predicates/4 kinds. `RegisterKind`/`RegisterPredicate` (renamed `RegisterType`/`RegisterPredicate`) auto-register a newly discovered predicate/type mid-`arc apply` with safe defaults (`role: edge`, `merge: union` for a predicate; empty `Required`/`Optional`, `merge: union` for a type) instead of a bare existence stub. `Resolve` fails fast — a missing `_schema/` folder or any malformed predicate/type document aborts the calling command before it makes any other change, reversing today's "skip a malformed document silently" tolerance.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: No new dependency. Reuses `github.com/yuin/goldmark`/`goldmark-meta` and `gopkg.in/yaml.v3` via `internal/core`'s existing, unmodified codec (`ParseNode`/`RenderNode`) — this feature parses/renders `_schema/predicates/`/`_schema/types/` documents as ordinary `core.Node` values, introducing no new parser — plus `github.com/fogfish/faults` for the new fail-fast schema errors and `github.com/fogfish/it/v2` for tests.

**Storage**: The mounted graph root, accessed exclusively through the existing, unmodified `internal/adapter/fsys` `Store`/`Mounter` — no new I/O path.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII). New table-driven unit cases in `internal/core/rules_test.go` (the new `Index`/`PredicateDef`/`TypeDef` shape) and `internal/app/schema/service/schema_test.go` (`Seed`/`Resolve`/`RegisterType`/`RegisterPredicate` against a fake `fsys.Store`, per spec 005's already-established pattern), written first per Principle VI. E2E coverage added to `cmd/arc/ctrl/init_test.go` (seeded files are spec-conformant), `cmd/arc/graph/apply_test.go` (auto-registration of a novel predicate/type writes a conformant node; a missing/malformed schema aborts the command before any write), and `cmd/arc/lint/lint_test.go` (same fail-fast case, read-only).

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`.

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — no new command, no new `internal/app/<domain>` use-case; this feature reshapes `internal/app/schema`'s existing use-case (kernel + service, no `port`/`adapter` subdirectory of its own, per its own README's existing precedent) and mechanically updates its two consumers, `internal/app/graph` and `internal/app/lint`.

**Performance Goals**: No measurable change — `Resolve` still does one directory walk per `_schema/` subfolder per command invocation, parsing a few dozen small Markdown files instead of a dozen; `arc init`'s `Seed()` renders roughly 55 documents instead of 17, still a single in-memory pass with no network I/O.

**Constraints**: No network I/O in `Seed`/`Resolve`/`RegisterType`/`RegisterPredicate` (unchanged from spec 005's D5); `RegisterType`/`RegisterPredicate` MUST NOT overwrite an existing document (unchanged FR-011 precedent); schema documents auto-registered while applying a patch MUST land in that same patch application's single commit (unchanged FR-012 precedent); `Resolve` MUST fail — not skip — on a missing `_schema/` folder or a malformed predicate/type document (spec FR-014, a deliberate reversal of spec 005's "unrecognized falls back to safe default" tolerance); `_schema/types/<name>.md` MUST additionally carry a `merge` attribute beyond CORE's own documented `Class`-node shape (spec FR-015, the interactively-resolved bridge keeping `arc apply`'s existing whole-node merge dispatch working with zero regression); rendering a type's `required`/`optional` predicates still produces one flat bulleted list, not a "## Requires"/"## Optional" headed section, because `internal/core.RenderNode`'s heading-grouped rendering is explicitly deferred to a future feature (spec 010's own Complexity Tracking; see Complexity Tracking below) — the on-disk shape is CORE-§9.1/§9.2-conformant in front matter and round-trips correctly, but not yet a byte-for-byte match of CORE §9.2's worked example.

**Scale/Scope**: Primary new/changed logic in `internal/core` (`rules.go` gains `Index`/`PredicateDef`/`TypeDef`, retiring `MergeRuleSet`) and `internal/app/schema` (`kernel/schema.go`, `service/schema.go`, `component.go`). Mechanical signature-propagation ripple through `internal/app/graph/{port/schema.go,kernel/apply.go,service/apply.go}`, `internal/app/lint/{component.go,service/lint.go,service/rules_frontmatter.go}`, `internal/app/ctrl/kernel/graph.go` (`_schema/nodes`→`_schema/types` folder rename), and their `cmd/arc/{ctrl,graph,lint}/*.go` callers, plus every corresponding `_test.go`/`testdata/` fixture in those packages. `internal/app/graph/service/apply.go`'s `coreKindFolders` (on-disk folder-name mapping) and `internal/app/lint/service/rules_predicates.go`'s `citoPredicates` (citation-predicate validation) are explicitly untouched (spec.md Assumptions) — flagged here so implementation does not conflate them with this feature's scope.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary MUST update in the same PR: "Canonical Folder" (`_schema/nodes/`→`_schema/types/`), "Node-Kind Schema Document"/"Predicate Schema Document" (rewritten as "Type Schema Node"/"Predicate Schema Node" per spec.md's Key Entities), a new "Schema Index" entry, and "Merge Behavior" (now sourced from a Type Schema Node's `merge` attribute, an arcnet-cli-specific bridge field — spec FR-015 — not from CORE's own documented `Class`-node shape). No ADR superseded — this follows ADR 001's existing use-case package layout unchanged (`internal/app/schema` keeps its kernel+service-only shape per its own README); only the domain data this use-case reads/writes gets richer. `tasks.md` MUST include this glossary-update task. |
| II — DDD & Glossary | Yes | PASS, with the same obligation — "Predicate Schema Node"/"Type Schema Node"/"Schema Index"/`PredicateDef`/`TypeDef` are the ubiquitous language a graph maintainer and every downstream spec (lint conformance, per-predicate merge, render-time grouping — see spec.md Assumptions) will use; the Glossary MUST reflect the renamed/reshaped concepts consistently with code. |
| III — Hexagonal Architecture | Yes | PASS — `internal/app/schema` keeps its existing kernel+service-only shape (no `port`/`adapter` of its own, unchanged from spec 005's precedent, since its only external dependency is the already-shared `fsys.Store`); the new `core.Index`/`PredicateDef`/`TypeDef` types live in the shared `internal/core` tier (not in `internal/app/schema/kernel`) specifically so `internal/app/graph` and `internal/app/lint` can keep consuming a schema-derived value without importing another use-case's kernel package — mirroring how `core.MergeRuleSet`/`core.MergeOp` already work today, and the exact reason those types live in `internal/core` rather than `internal/app/schema/kernel` in the first place (research.md D1). |
| IV — Functional Programming Style | Yes | PASS — new/changed functions (`core.Index` decode helpers, `Resolve`'s per-document validators, `RegisterType`/`RegisterPredicate`'s default-assignment logic) stay small and single-purpose; no inline comments beyond existing GoDoc conventions. |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — `core.Index`/`PredicateDef`/`TypeDef` is not a speculative abstraction: it directly replaces the ad hoc `(core.MergeRuleSet, map[string]bool)` tuple already threaded through `Apply`/`Lint` today with the single cohesive value spec.md's Schema Index entity requires, mirroring ARCNET-AST §8's own documented shape one-for-one — right-sized to the spec, not a speculative generalization. |
| VI — TDD | Yes | PASS — new table-driven cases in `internal/core/rules_test.go` and `internal/app/schema/service/schema_test.go`, written first per Principle VI, using `github.com/fogfish/it/v2` exclusively; each covers a CORE §10/§11 worked predicate/type plus the malformed-document and missing-`_schema/`-folder fail-fast paths. |
| VII — External Integration & Adapter Consistency | Yes | PASS — no new external integration; the only I/O this feature touches (`fsys.Store` reads/writes) is unchanged, still exclusively through `internal/adapter/fsys`. |
| VIII — E2E Acceptance Testing | Yes | PASS, with obligation — every acceptance scenario across spec.md's 3 user stories (13 scenarios total) needs a colocated E2E or unit case: US1 via `cmd/arc/ctrl/init_test.go` (seeded `_schema/predicates/`/`_schema/types/` files are spec-conformant), US2 via `cmd/arc/graph/apply_test.go` (auto-registration writes a conformant node; a missing/malformed schema aborts the command before any write — also mirrored read-only in `cmd/arc/lint/lint_test.go`), US3 (the Schema Index itself) via `internal/app/schema/service/schema_test.go` unit tests, since `Resolve` has no Cobra-level surface of its own — covered at the domain layer per Principle III's "pragmatic deviation for small tools" allowance, exactly as spec 010 already did for its own domain-layer-only scenarios — plus indirectly proven through `arc apply`/`arc lint`'s own E2E tests consuming the identical `core.Index`. |
| IX — CLIG/Cobra (ADR 002) | No | N/A — no command, flag, or help text changes; every command's CLI surface (flags, arguments, exit-code meanings) is unchanged. |
| X — Terminal Output, Color & Interactivity | No | N/A — no Reporter phase added or changed; `arc apply`'s existing auto-registration warning line changes wording only (kind→type terminology), not its rendering mechanism. |
| XI — Configuration, Environment Variables & Secrets | No | N/A — no configuration surface touched. |
| XII — Documentation & Help System | Yes | PASS — the new fail-fast schema errors (missing `_schema/` folder; a predicate/type document missing/invalid `role`/`merge`/`@type`) MUST be `faults.Type`/`faults.SafeN` constants naming the offending file and field, extending `internal/app/schema/service/errors.go`'s existing `ErrSchemaWrite` precedent with new constants (e.g. `ErrSchemaMissing`, `ErrSchemaInvalid`) rather than an ad hoc `fmt.Errorf`. |
| XIII — Distribution & Release Engineering | No | N/A — no release pipeline change. |
| XIV — Versioning, Security & Compatibility | Yes, flagged | PARTIAL, explicitly justified — this feature is a breaking change to an on-disk contract (`_schema/nodes/`→`_schema/types/` rename; `Resolve`'s skip-malformed tolerance becomes fail-fast) with no prior deprecation warning. Accepted here because the project is pre-1.0 (`0.1.x` release train per recent tags) and the user's own spec explicitly requires this — "`arc init` supports only the new schema," "the absence of a valid schema causes failing" — making this a deliberate, spec-mandated behavior change rather than an incidental one; recorded in Complexity Tracking for visibility. |

No blocking violations — the one flagged item (breaking on-disk schema contract, no prior deprecation) is a pre-1.0, spec-mandated trade-off recorded below rather than hidden.

## Project Structure

### Documentation (this feature)

```text
specs/011-machine-readable-schema/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — D1-D9 design decisions
├── data-model.md         # Phase 1 output — Index/PredicateDef/TypeDef, Predicate/Type Schema Node shapes
├── quickstart.md         # Phase 1 output — 3 runnable scenarios, one per user story
├── contracts/            # Phase 1 output
│   ├── schema-document-contract.md   # _schema/predicates/<name>.md and _schema/types/<name>.md on-disk shape
│   └── schema-index-contract.md      # core.Index/PredicateDef/TypeDef + Resolve/Seed/RegisterType/RegisterPredicate Go contract
└── tasks.md               # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/                              # shared domain tier — new schema-index types
│   ├── rules.go                        # + Index{Predicates map[string]PredicateDef, Types map[string]TypeDef}
│   │                                     #   + PredicateDef{Role, Merge, Label, Aligned}, TypeDef{Merge,
│   │                                     #   Required []string, Optional []string}; retires MergeRuleSet/
│   │                                     #   .Lookup/.Union (superseded by Index.Types[name].Merge)
│   └── rules_test.go                    # + Index/PredicateDef/TypeDef construction + lookup cases
│
└── app/
    ├── schema/
    │   ├── README.md                     # updated: _schema/types/ (not /nodes/), richer document shape
    │   ├── kernel/
    │   │   ├── schema.go                  # NodesDir->TypesDir rename; CoreMergeRules/CorePredicates/
    │   │   │                               #   coreKindDescriptions replaced by CorePredicateDefs
    │   │   │                               #   map[string]PredicateSeed and CoreTypeDefs
    │   │   │                               #   map[string]TypeSeed (role/merge/label/aligned/description
    │   │   │                               #   per CORE §10; required/optional/description/merge per
    │   │   │                               #   CORE §11 + Property/Class themselves)
    │   │   └── schema_test.go             # fixed-count assertions rewritten for full CORE vocabulary
    │   ├── service/
    │   │   ├── schema.go                  # Seed() renders real Property/Class nodes (Attrs role/merge/
    │   │   │                               #   label/aligned, Texts["description"], Class Edges
    │   │   │                               #   required/optional); Resolve(store) (core.Index, error)
    │   │   │                               #   rewritten fail-fast (missing _schema/ folder or malformed
    │   │   │                               #   document aborts, no longer skipped); RegisterKind renamed
    │   │   │                               #   RegisterType, writes a conformant Class node
    │   │   │                               #   (Merge: union default); RegisterPredicate writes a
    │   │   │                               #   conformant Property node (Role: edge, Merge: union default)
    │   │   ├── schema_test.go             # Seed/Resolve/RegisterType/RegisterPredicate against fake
    │   │   │                               #   fsys.Store; + malformed-document and missing-folder
    │   │   │                               #   fail-fast cases
    │   │   └── errors.go                   # + ErrSchemaMissing, ErrSchemaInvalid (faults.Type/SafeN)
    │   └── component.go                    # Resolve returns core.Index; RegisterKind->RegisterType
    │
    ├── graph/
    │   ├── port/schema.go                  # SchemaRegistry.RegisterKind -> RegisterType
    │   ├── kernel/apply.go                 # Warnings wording: "kind" -> "type" terminology
    │   ├── service/apply.go                # Apply(..., index core.Index, ...) replaces rules/predicates
    │   │                                     #   params; rules.Lookup(node.Type) -> index.Types[node.Type]
    │   │                                     #   lookup; predicates[name] -> index.Predicates[name];
    │   │                                     #   schema.RegisterKind -> schema.RegisterType call rename;
    │   │                                     #   coreKindFolders/nodeFolder/pluralizeKind UNCHANGED
    │   │                                     #   (out of scope, spec.md Assumptions)
    │   ├── service/apply_test.go           # fixtures updated to new Index param + richer schema docs
    │   └── component.go                     # Apply's core.MergeRuleSet param -> core.Index
    │
    ├── lint/
    │   ├── component.go                     # Lint's core.MergeRuleSet param -> core.Index
    │   ├── service/lint.go                  # Lint(..., index core.Index, dir) replaces rules/predicates
    │   │                                     #   params, passed through to the two check* calls below
    │   ├── service/rules_frontmatter.go     # checkUnrecognizedKind(node, path, index core.Index) —
    │   │                                     #   checks index.Types[node.Type] presence instead of
    │   │                                     #   core.MergeRuleSet.Lookup
    │   ├── service/rules_predicates.go      # checkPredicateRegistered(..., index.Predicates) — same
    │   │                                     #   intent, new backing type; citoPredicates UNCHANGED
    │   │                                     #   (out of scope, spec.md Assumptions)
    │   └── service/*_test.go                # fixtures updated to new Index param
    │
    └── ctrl/
        └── kernel/graph.go                  # DefaultLayout.Folders: "_schema/nodes" -> "_schema/types"

cmd/
├── arc/ctrl/init.go                      # schemaSeed() unchanged in shape (still wraps appschema.Seed())
├── arc/ctrl/init_test.go                 # + assertions that seeded files are spec-conformant
├── arc/graph/apply.go                    # appschema.Resolve(store) now returns core.Index; passed
│                                            #   through to appgraph.Apply unchanged in call shape
├── arc/graph/apply_test.go               # + auto-registration-writes-conformant-node case; +
│                                            #   missing/malformed-schema fail-fast case
├── arc/lint/lint.go                      # appschema.Resolve(store) now returns core.Index; passed
│                                            #   through to applint.Lint unchanged in call shape
└── arc/lint/lint_test.go                 # + missing/malformed-schema fail-fast case (read-only)

ARCHITECTURE.md                           # + Glossary updates: Schema Index, Predicate Schema Node,
                                            #   Type Schema Node (rewritten from Node-Kind/Predicate
                                            #   Schema Document), Canonical Folder (_schema/types/),
                                            #   Merge Behavior (now type-node `merge` bridge field)
```

**Structure Decision**: No new command, no new `internal/app/<domain>` use-case — this feature reshapes `internal/app/schema`'s existing use-case in place, introduces one new shared domain type (`core.Index`, replacing `core.MergeRuleSet`) in `internal/core`, and mechanically propagates the resulting signature change through `internal/app/graph` and `internal/app/lint` (its two existing consumers) and their `cmd/arc/{ctrl,graph,lint}` callers. `internal/adapter/fsys` is untouched (no I/O contract change), and no `cmd/` package gains new flags, commands, or business logic.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Breaking on-disk schema contract (`_schema/nodes/`→`_schema/types/` rename; `Resolve`'s skip-malformed tolerance becomes fail-fast) with no prior deprecation warning (Principle XIV) | The user's own spec explicitly mandates both changes — "`_schema/types/` (renamed from today's `_schema/nodes/`)" and "the absence of a valid schema causes failing" — as the whole point of turning an existence registry into a real, trustworthy machine-readable schema; a lenient/dual-format reader would silently mask exactly the malformed-schema condition this feature exists to surface | An additive-only migration (read both `_schema/nodes/` and `_schema/types/`, tolerate malformed documents) was rejected: spec.md FR-009/FR-016 explicitly rule it out ("`arc init` MUST NOT ... fall back to the previous existence-only schema format"; "no automatic migration path is provided"), and maintaining two parallel schema readers indefinitely contradicts Principle V (YAGNI) for a pre-1.0 tool with no production graphs predating this change (mirroring spec 005's identical, already-accepted precedent) |
| `_schema/types/<name>.md`'s `required`/`optional` predicates render as one flat bulleted list, not CORE §9.2's literal "## Requires"/"## Optional" headed sections | `internal/core.RenderNode`'s heading-grouped-by-predicate-role rendering is explicitly out of scope for this feature (a future feature owns it, per spec 010's own Complexity Tracking, which already deferred this exact capability) — implementing a narrow, type-node-only heading special case here would be a second, throwaway heuristic living alongside that future feature's eventual general solution | Waiting for the general solution was chosen over a local special case because `internal/core.ParseNode`'s existing bare-list parser already round-trips a flat `- required:: [[x]]`/`- optional:: [[y]]` bullet list correctly regardless of any heading — the data is fully correct and CORE-conformant in front matter; only the visual presentation lags, and duplicating rendering logic twice (once narrowly here, once generally later) is worse than a documented, temporary cosmetic gap |
