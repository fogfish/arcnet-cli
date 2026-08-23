# Implementation Plan: Patch Manifest Identity (`@type: patch`)

**Branch**: `021-patch-type-manifest` | **Date**: 2026-08-23 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/021-patch-type-manifest/spec.md`

## Summary

The patch exchange format still declares itself with the pre-0.5 `kind: patch` key while the
node parser has long since migrated to `@id`/`@type`. Two functions in `internal/core` carry
the whole defect: `decodePatchManifest` gates on `kind`, and `renderPatchManifest` emits it.

The fix replaces that single gate with a five-case total decision over the pair
(`@type`, `kind`) — accept, conflict, legacy, not-a-patch, absent — expressed as four
`faults` constants in `internal/core/errors.go`. `LooksLikePatch` widens to recognize *both*
keys, but strictly for **error routing**, never acceptance: it is what keeps a legacy patch in
a batch directory from being silently counted as ordinary Markdown (FR-008). `RenderPatch`
switches to the `appendQuotedKeyYAMLPair` helper that already renders node `@id`/`@type`.

No new field is added to `core.Patch`, so the `--json` contract is untouched by construction
(FR-011). The bulk of the change is the fixture migration: 33 inline constants in
`cmd/arc/graph/apply_test.go` alone, 10 `testdata/**` files, and 9 runnable `quickstart.md`
guides.

## Technical Context

**Language/Version**: Go 1.26.5 (per `go.mod`)

**Primary Dependencies**: `github.com/yuin/goldmark` + `goldmark-meta` (front-matter parsing),
`gopkg.in/yaml.v3` (manifest rendering), `github.com/fogfish/faults` (error constants),
`github.com/spf13/cobra`. No new dependency.

**Storage**: Local git-versioned Markdown graph, reached through `internal/adapter/fsys`.
Unchanged by this feature.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E via the `sut()` helper
colocated in `cmd/arc/graph` and `cmd/arc/ctrl`.

**Target Platform**: linux/darwin/windows amd64+arm64 per `.goreleaser.yaml`.

**Project Type**: Single Cobra CLI binary.

**Performance Goals**: N/A — the change is one map lookup per patch parse.

**Constraints**:

- `internal/core` MUST NOT import `internal/bios` (Principle III). Every new error is a
  `faults` constant returned as a value; the file path is attached by the caller that knows
  it.
- `core.Patch`'s json tags are the public `--json` contract (spec 007). **No field is added**
  — see D2, which supersedes the `Deprecations []string` design in the pasted technical
  context.
- Manifest keys arrive through `normalizeYAMLMap` as `map[string]any`; the `@` prefix
  survives that path unchanged (verified empirically, D1).
- Current released version is `0.1.12` (pre-1.0). See the Principle XIV entry in
  Complexity Tracking.

**Scale/Scope**: ~120 lines of production change across 4 files; ~50 files touched overall,
dominated by fixture migration.

### Correction to the pasted technical context

The `/speckit-plan` input reproduces `CORE-FIX.md` §3.7 verbatim, which was written against
that document's §3.3 FR-003 — a one-release transitional acceptance of `kind: patch` with a
deprecation notice. **`spec.md` supersedes that**: the user's `/speckit-specify` input states
"compatibility with a legacy `kind: patch` manifest IS NOT REQUIRED", and the spec records it
as an Assumption. There is therefore no deprecation notice to route, and consequently:

- **No `Deprecations []string` field** on `core.Patch`, and no `json:"-"` question to answer.
- **No `bios.Reporter` plumbing** in `internal/app/graph/service.Apply` for this feature.

The hexagonal constraint the pasted context was protecting still binds and is still honored —
`internal/core` gains no `bios` import — it is simply satisfied by returning a typed error
rather than by carrying notice data on the value. See [research.md](./research.md) D2.

## Constitution Check

*GATE: evaluated before Phase 0, re-evaluated after Phase 1 design.*

| Principle | Gate | Status |
|---|---|---|
| **I** — Architecture docs & ADRs | ARCHITECTURE.md updated in the same PR; no accepted ADR contradicted | **PASS** — glossary rows **Patch** (`ARCHITECTURE.md:244`) and **Batch Plan** (`:273`) both name `kind: patch` and are updated by T-DOC1. ADRs 001/002/003 reviewed: none constrains the exchange-format identity key. No new ADR needed — this implements an external spec (ARCNET-CORE §14.2.1), it does not decide architecture. |
| **II** — DDD & glossary | Domain terms in the glossary | **PASS** — no new domain concept; the **Patch** row's field list is corrected. |
| **III** — Hexagonal boundaries | `internal/core` free of `cobra`/`bios`/`cmd` imports | **PASS** — see D2. New errors are `faults` values; path attachment happens in `internal/app/*/service`. |
| **IV** — Functional style | Small pure functions, no in-body comments | **PASS** — `decodePatchManifest` stays under 25 lines by extracting `patchManifestType` (D4). Doc comments above declarations follow the established `markdown.go` pattern. |
| **V** — SOLID / YAGNI | Build only what FRs require | **PASS** — the `Deprecations` field and the shared lint/core locator are both rejected as unneeded (D2, D3). |
| **VI** — TDD | Unit tests written first, compile, fail semantically; `it/v2` only; table-driven | **PASS** — the five-case table in `internal/core/markdown_test.go` is T001, before any production edit. |
| **VII** — Adapters | No new external integration | **N/A** |
| **VIII** — E2E & spec traceability | Every acceptance scenario has a colocated E2E test via `sut()` | **PASS** — 12 scenarios map 1:1 to tests in `cmd/arc/graph/{apply,batch,subgraph}_test.go` and `cmd/arc/ctrl/apply_schema_test.go`. See [contracts/cli-contract.md](./contracts/cli-contract.md). |
| **IX** — Command & flag design | No command/flag surface change | **PASS** — no new command, flag, or alias. |
| **X** — Terminal output | stdout/stderr split, exit codes | **PASS** — errors to stderr, exit 1, unchanged mechanism. |
| **XII** — Docs & errors | `faults.Type`/`SafeN` constants, human-readable guidance | **PASS** — four new constants in `internal/core/errors.go`; each message is itself the guidance (SC-007). |
| **XIII** — Release engineering | GoReleaser unchanged | **N/A** |
| **XIV** — Versioning & compatibility | Breaking format change; deprecation warning in a prior minor release | **DEVIATION — justified below** |

**Post-Phase-1 re-evaluation**: unchanged. The Phase 1 design added no new package, no new
dependency, and no new external integration; D3 and D6 were both resolved toward the option
that keeps the blast radius inside `internal/core` plus two one-line service call sites.

## Project Structure

### Documentation (this feature)

```text
specs/021-patch-type-manifest/
├── plan.md              # This file
├── research.md          # Phase 0 — D1..D8
├── data-model.md        # Phase 1 — the manifest decision table
├── quickstart.md        # Phase 1 — runnable validation guide
├── contracts/
│   ├── patch-manifest.md   # Normative manifest grammar + error contract
│   └── cli-contract.md     # Per-command behaviour, scenario→test map
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
cmd/arc/
├── graph/
│   ├── apply_test.go          # E2E: US1 S1-S3, US3 S1-S3, S5   (33 inline constants migrate)
│   ├── batch_test.go          # E2E: US3 S4                      (6 constants migrate)
│   ├── revert_test.go         # E2E: regression only             (5 constants migrate)
│   ├── subgraph_test.go       # E2E: US2 S1-S3
│   └── testdata/
│       ├── batch/**           # 7 fixtures migrate; +1 new legacy-kind fixture (D8)
│       └── batch-failfast/**  # 3 fixtures migrate
└── ctrl/
    └── apply_schema_test.go   # E2E: US1 S4                      (9 constants migrate)

internal/
├── core/
│   ├── errors.go              # +ErrManifestTypeConflict, +ErrManifestLegacyKind,
│   │                          #  +ErrManifestNotAPatch, +ErrIdentityQuoting
│   ├── markdown.go            # decodePatchManifest, patchManifestType (new),
│   │                          #  LooksLikePatch, renderPatchManifest, ParsePatch
│   └── markdown_test.go       # unit: 5-case table + round-trip   (27 constants migrate)
└── app/
    ├── graph/service/
    │   ├── apply.go           # readPatch: wrap parse errors with the path (D5)
    │   ├── revert.go          # isPatchDocument: adopt LooksLikePatch (D6)
    │   └── batch.go           # doc comment only — classifyPatchFiles is already correct
    └── schema/service/
        └── apply.go           # already wraps with ErrPatchRead — no change

ARCHITECTURE.md                # glossary rows Patch (:244), Batch Plan (:273)
specs/VISION.md                # `arc apply` roadmap line
```

**Structure Decision**: The production change is confined to `internal/core` (the AST/
serialization domain, which owns the exchange-format grammar) plus two call sites in
`internal/app/graph/service` that attach the file path to a parse error. No `cmd/` production
file changes — the command surface is untouched (FR: no flag or command change), so `cmd/`
appears above only as the E2E test and fixture surface.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| **Principle XIV**: a breaking change to a script-consumed format ships without the "SHOULD be preceded by a deprecation warning on `stderr` in at least one prior minor release" grace period | The user's `/speckit-specify` input states the requirement directly: "The compatibility with a legacy `kind: patch` manifest IS NOT REQUIRED." Recorded as the first Assumption in `spec.md`. The grace period is also *load-bearing only if someone can be reading legacy patches successfully today* — and no one can: `arc apply` already rejects every spec-conformant patch, so the format is broken in both directions and has no working ecosystem to deprecate out of. The project is at `0.1.12`, pre-1.0, where SemVer does not promise compatibility. | The one-release transitional acceptance (`CORE-FIX.md` §3.3 FR-003, and the `Deprecations []string` design in §3.7) was rejected by the user. Keeping it would also require importing notice-carrying state into `core.Patch` and threading a `bios.Reporter` for a warning nobody can currently trigger. |
| **Principle XIV**: no major-version bump accompanies the format break | Pre-1.0 (`0.x`) versioning; the `--json` and `--plain` contracts — the two the constitution names as stable and scriptable — are provably unaffected (FR-011, verified by T014). | Bumping to `1.0.0` would assert a stability guarantee across the *rest* of the surface that the three in-flight conformance features (`CORE-FIX.md` §2: 022, 023, 024) are about to break again. |

**Mitigation for both**: FR-003 requires the rejection message to name the retired key and its
replacement, and FR-008 requires batch runs to fail loudly rather than skip. Together these
turn the absent deprecation window into a single, self-explaining error at the exact moment a
user hits it — which is what the deprecation warning would have bought.
