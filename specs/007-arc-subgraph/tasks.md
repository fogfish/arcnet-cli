# Tasks: Extract a Self-Contained Subgraph (`arc subgraph`)

**Input**: Design documents from `/specs/007-arc-subgraph/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`cli-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story.

**Bugfix**: 2026-07-05 — [BUG-001](bugs/BUG-001.md) Reopened T017/T025/T027/T029/T044 and added Phase 6 (T048-T054) for the opt-in `--stubs` flag (spec FR-017): an extraction-boundary structural link with no corresponding node section produced a dangling reference once applied into a graph lacking the excluded targets.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `cmd/arc/graph/` — existing Cobra wiring package for the `graph` domain; gains `subgraph.go` and its colocated E2E test
- `internal/core/markdown.go` — existing shared core domain codec; gains `RenderPatch`
- `internal/app/config/` — existing use-case; `kernel/config.go` gains its second real field (`Subgraph`)
- `internal/app/graph/{kernel,service,component.go}` — existing `graph` use-case; gains `Subgraph` alongside `Apply`/`Grep`

---

## Phase 0: Pre-implementation Refactoring (OPTIONAL)

**Include only when the feature requires significant changes to existing code.** MUST be submitted as a separate PR from feature work. All existing tests MUST pass after refactoring.

- [X] T000 Rename `walkGrepNodeFiles` to a direction-neutral `walkNodeFiles` in `internal/app/graph/service/grep.go` (research.md D7); update its one call site inside `service.Grep`; confirm `internal/app/graph/service/grep_test.go` and `cmd/arc/graph/grep_test.go` still pass unmodified — this feature's `Subgraph` will be the function's second caller

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Confirm no new package tier is required — `internal/core`, `internal/app/graph/{kernel,service}`, `cmd/arc/graph`, `internal/app/config/kernel` all already exist (plan.md Project Structure); this feature only adds new files/fields within them
- [X] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged per plan.md Technical Context
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the current tree before starting (baseline)

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add the domain terms from spec.md Key Entities / data-model.md — Seed Node, Reachable Node, Subgraph, Traversal Cap — to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle I obligation, plan.md Constitution Check row I)
- [X] T005 Verify `core.Patch`/`core.Node` (already defined for `ParsePatch`) are reused as-is for `RenderPatch`'s input — confirm no new node/patch type is introduced (research.md D2, Principle V)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc subgraph`'s bare-verb grammar, single positional `<basename>` argument, the new local `--depth` int flag, and the reused `optsFilter` flags against contracts/cli-contract.md (research.md D6)
- [X] T007 [P] Review contracts/cli-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T008 Confirm `arc subgraph` needs no new port/adapter — `internal/app/graph/service.Subgraph` depends only on `fsys.Mounter`, mirroring `arc grep`'s own precedent (research.md D8, ADR 001 port isolation rule 2)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T009 [P] [US1] Write E2E tests in `cmd/arc/graph/subgraph_test.go` for spec.md US1's 4 acceptance scenarios (seed + directly-connected nodes of several kinds are all included, grouped by kind, front-matter/body verbatim; a seed with no connections yields the seed alone; the extracted output re-ingests via `arc apply` without a structural error; an unknown basename refuses with a clear error and no output) using the `sut()` helper — tests MUST compile and fail semantically (red phase)
- [X] T010 [P] [US2] Write E2E tests in `cmd/arc/graph/subgraph_test.go` for spec.md US2's 4 acceptance scenarios (`--depth 2` includes exactly what's within 2 hops and excludes anything farther; `--depth 0` yields the seed alone; omitting `--depth` behaves as `--depth 1`; a node reachable by more than one path of different lengths appears exactly once) — red phase
- [X] T011 [P] [US3] Write E2E tests in `cmd/arc/graph/subgraph_test.go` for spec.md US3's 4 acceptance scenarios (the seed is included even when its own kind doesn't match `--kind`; `--kind` restricts which reachable nodes are added; a filter matching zero reachable nodes still outputs the seed alone with no error; combined `--kind`+`--tag`+`--attr` narrows further per the shared Filtering rules) — red phase
- [X] T012 [P] Write E2E tests in `cmd/arc/graph/subgraph_test.go` for the Edge Cases tied to guard/UX behavior: an unknown `<basename>` refuses (FR-011), a negative or non-integer `--depth` refuses (FR-012/FR-013), the target not being an initialized graph refuses (FR-010), a dangling link target is silently excluded rather than failing the run (FR-006), and a cycle in the graph does not loop forever or duplicate a node (FR-004) — red phase

> T009–T012 all target the same new file (`cmd/arc/graph/subgraph_test.go`) and are therefore sequential in practice despite each being scoped to one story (mirrors `specs/006-arc-grep-content-search/tasks.md`'s T010–T013 note).

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T013 Confirm the new `.arc/config.yml` fields (`subgraph.directCap`, `subgraph.backlinkCap`) follow the flag → env → project config → user config → system config precedence (project-config-only, no flag/env override introduced) and that no secret or credential material is involved (plan.md Constitution Check row XI, research.md D6)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `internal/core.RenderPatch`, the `kernel`/`config` value-type extensions, and `service.Subgraph`'s enumeration+index+traversal+capping+orchestration are genuinely foundational — every one of US1–US3 runs the same extraction and reports through the same `kernel.SubgraphResult` shape, differing only in which flags their own E2E scenarios exercise. This phase builds that shared foundation; Phase 3+ adds each story's specific behavior on top of it.

### `internal/core` — `RenderPatch` (research.md D2, D9)

- [X] T014 [P] Extract the shared node-body rendering (Text/Edges/Links/Notes + HRef reconstruction) out of `RenderNode`'s tail into an unexported helper in `internal/core/markdown.go`, so both `RenderNode` and the new `RenderPatch` share one body-rendering implementation (research.md D2) — refactor only; `RenderNode`'s existing output and tests are unchanged
- [X] T015 Implement `internal/core/markdown.go`'s `RenderPatch(Patch) ([]byte, error)`: manifest front-matter (`kind: patch`, `document`, `published`, `title`, `stats`) as a `---`-delimited YAML block, then nodes grouped by `Kind` (alphabetical) and by `ID` (alphabetical) within each kind, under `# <Kind>`/`## <ID>` headings, a fenced `yaml` front-matter block excluding `kind`, and the body via T014's shared helper (research.md D2, D9) (depends on T014)
- [X] T016 [P] Unit tests in `internal/core/markdown_test.go`: the round-trip property `ParsePatch(RenderPatch(p))` reproduces `p`'s node set, for a table of `Patch` fixtures (single node; multiple kinds; nodes carrying Edges/Links/Notes/HRefs); deterministic kind/ID ordering regardless of the input slice's original order (depends on T015)

### `internal/app/graph/kernel` — value types

- [X] T017 [P] Implement `internal/app/graph/kernel/subgraph.go`: `SubgraphResult{Root, Seed, Depth, Patch, DirectReachable, DirectIncluded, DirectTruncated, BacklinkReachable, BacklinkIncluded, BacklinkTruncated}` per data-model.md

### `internal/app/config` — second real `Config` field (research.md D6)

- [X] T018 [P] Extend `internal/app/config/kernel/config.go`: add `Config.Subgraph SubgraphConfig{DirectCap, BacklinkCap}` per data-model.md
- [X] T019 [P] Unit tests in `internal/app/config/service/config_test.go`: `Subgraph.DirectCap`/`BacklinkCap` round-trip through `Load`/`Save`, and an absent/zero value loads as the zero `SubgraphConfig{}` (defaulting happens at the `cmd/` wiring layer, not here) (depends on T018)

### `internal/app/graph/service` — errors, index, traversal, orchestration

- [X] T020 [P] Add `ErrSeedNotFound`, `ErrInvalidDepth` `faults.Safe1[string]` sentinel constants to `internal/app/graph/service/errors.go` (extending the existing file)
- [X] T021 Implement the node-enumeration+index pass in `internal/app/graph/service/subgraph.go`: reuse T000's `walkNodeFiles`, parse each remaining `*.md` via `core.ParseNode` into an `id → core.Node` index; an unreadable or unparseable file is excluded (mirrors `service.Grep`'s own enumeration, research.md D7) (depends on T000)
- [X] T022 Implement the reverse-edge index and degree function in `internal/app/graph/service/subgraph.go`: build `map[string][]string` from every node's `Edges`/`Links` targets; `degree(id)` = out-edge count (`len(Edges)` + Σ `len(block.Seq)`) + in-edge count (`len(reverse[id])`) (research.md D4) (depends on T021)
- [X] T023 Implement the two independent BFS passes in `internal/app/graph/service/subgraph.go`: "direct" (outgoing `Edges`/`Links` only) and "backlink" (reverse index only), each bounded by `depth` hops over the full index, each producing its own reachable-node-ID set with hop distances; a dangling target (absent from the index) is silently skipped; both passes run to full completion before any capping (research.md D3, D5) (depends on T022)
- [X] T024 Implement post-traversal degree-ranked capping in `internal/app/graph/service/subgraph.go`: for each pool (direct/backlink) whose discovered size exceeds its configured cap, sort candidates by `degree` descending (ties broken by `ID` ascending) and retain only the top `cap` entries, recording each pool's pre-/post-cap counts (research.md D4, D5) (depends on T023)
- [X] T025 Implement `internal/app/graph/service.Subgraph(ctx, mounter, filter core.Filter, basename string, depth int, cfg kernel.SubgraphConfig, dir string) (kernel.SubgraphResult, error)`: guard `ErrNotAGraph`, run T021's enumeration, resolve the seed by `basename` (`ErrSeedNotFound` if absent), run T023/T024's traversal+capping, apply `filter` to non-seed candidates only (the seed is always included), synthesize the `Patch` manifest (`Document` derived from the seed's basename + extraction timestamp, `Published` = now, `Stats` from T024's counts), assemble `core.Patch{Nodes: seed + surviving candidates}` (depends on T015, T020, T024)
- [X] T026 [P] Unit tests in `internal/app/graph/service/subgraph_test.go` against a fake `fsys.Mounter`/`fsys.Store`: not-a-graph guard refuses before any traversal, an unknown seed returns `ErrSeedNotFound`, `depth=0` yields the seed alone, a cycle does not loop and does not duplicate a node, a dangling link target is excluded, the filter excludes non-seed candidates only and never the seed, and a pool exceeding its configured cap retains exactly the highest-degree candidates (SC-007) (depends on T025)

### Wiring skeleton

- [X] T027 [P] Implement `internal/app/graph/component.go`'s `Subgraph(ctx, mounter, filter, basename, depth, cfg, dir) (kernel.SubgraphResult, error)` delegator, alongside the existing `Apply`/`Grep` (depends on T025)
- [X] T028 [P] Scaffold `cmd/arc/graph/subgraph.go`: `NewSubgraphCmd() *cobra.Command` with `Args: cobra.ExactArgs(1)`, a local `--depth` int flag (default `1`), reusing `grep.go`'s existing `optsFilter{kind, tag, attr}` (no redeclaration — research.md D6), and `RunE` returning a "not implemented" placeholder error (empty-but-compiling scaffold)

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Pull a node and its immediate context into a portable document (Priority: P1) 🎯 MVP

**Goal**: With default settings (depth 1, no filter), the seed plus every directly connected node is extracted and serialized as a valid, kind-grouped patch-exchange document.

**Independent Test**: Run `arc subgraph <basename>` against a graph with a known, fixed set of direct connections and confirm the output contains exactly the seed plus those connections, correctly grouped and formatted, per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T009, T012) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T029 [US1] Implement `cmd/arc/graph/subgraph.go`'s real `RunE`: `filepath.Abs(".")`, `fsys.Local{}.Mount`, `internal/app/config.Load`, resolve `SubgraphConfig` defaults (`DirectCap <= 0` → `4096`, `BacklinkCap <= 0` → `1024`), depth defaulting to `1` when the flag is unset, an empty `core.Filter{}` when no filter flags are given, call `appgraph.Subgraph` (depends on T027, T028)
- [X] T030 [US1] Implement the `Human` renderer in `cmd/arc/graph/subgraph.go`: write `core.RenderPatch(result.Patch)`'s bytes verbatim to stdout — no `bios.SCHEMA` reference, no styling of any kind (research.md D10) (depends on T015, T029)
- [X] T031 [US1] Construct `bios.Registry[kernel.SubgraphResult]{Human: ...}` (no distinct `Verbose` renderer — it falls back to `Human`, research.md D10) and resolve/print via `bios.ResolveMode()` in `RunE`, giving `--json` for free via the generic `jsonPrinter` (depends on T030)
- [X] T032 [US1] Implement the DS-07 exit-code contract in `cmd/arc/graph/subgraph.go`: `RunE` returns `nil` whenever extraction completes — no `bios.ErrSilent` path (research.md D11); a genuine refusal (seed not found, invalid depth, not a graph) returns a real error before anything is printed
- [X] T033 [US1] Populate `Short`/`Long`/`Example` help text for `arc subgraph` per contracts/cli-contract.md's DS-11 shape (constitution Principle XII) in `cmd/arc/graph/subgraph.go`
- [X] T034 [US1] Register `graph.NewSubgraphCmd()` into `cmd/arc/root.go`'s command tree (depends on T029)
- [X] T035 [P] [US1] Add unit tests in `internal/app/graph/service/subgraph_test.go` covering User Story 1's acceptance scenarios specifically: default depth-1, no-filter extraction includes the seed plus every directly connected node grouped by kind, and a seed with no connections yields a one-node `Patch` (depends on T025)

**Checkpoint**: At this point, User Story 1's E2E tests (T009, T012) pass and `arc subgraph` is fully functional and independently testable for the default one-hop case

---

## Phase 4: User Story 2 - Widen or narrow the reach of the extraction (Priority: P2)

**Goal**: `--depth` precisely controls how many hops are traversed; `0` yields the seed alone, omission behaves as `1`, and a multi-path node is never duplicated.

**Independent Test**: Run `arc subgraph <basename> --depth <n>` at several depths against a graph with a known hop-chain and confirm exact inclusion/exclusion at each boundary, per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E test for this story was already written in Phase 2d (T010) and MUST currently be failing (red) until this phase's flag wiring lands.

- [X] T036 [US2] Implement `--depth` flag registration in `cmd/arc/graph/subgraph.go`: an `IntVar` local flag, default `1` (DS-02) (depends on T028)
- [X] T037 [US2] Implement `--depth` validation in `cmd/arc/graph/subgraph.go`: reject a negative value with `ErrInvalidDepth` before calling `appgraph.Subgraph`; a non-integer value is already rejected by Cobra's own `IntVar` parsing (depends on T020, T036)
- [X] T038 [US2] Wire the validated depth into `RunE`, replacing T029's hardcoded default-`1` call with the resolved `--depth` value (depends on T029, T037)
- [X] T039 [P] [US2] Add unit tests in `internal/app/graph/service/subgraph_test.go` covering depth `0`, `2`, and `3` against a known hop-chain fixture, confirming exact inclusion/exclusion at the boundary and that a node reachable by more than one path of different lengths appears exactly once (depends on T025)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Keep the extraction focused with a filter (Priority: P3)

**Goal**: `--kind`/`--tag`/`--attr` restrict which reachable (non-seed) nodes are included; the seed is never excluded by the filter regardless of its own kind/tags/attributes.

**Independent Test**: Run `arc subgraph <basename> --kind <kind>` against a graph where the seed's own kind differs from the filter and confirm the seed still appears alongside only the matching reachable nodes, per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T011) and MUST currently be failing (red) until this phase's flag wiring lands.

- [X] T040 [US3] Wire `grep.go`'s existing `optsFilter.apply(cmd)`/`.build()` into `cmd/arc/graph/subgraph.go`'s command construction and `RunE`, replacing T029's hardcoded empty `core.Filter{}` with the parsed filter (no redeclaration of `optsFilter` — research.md D6) (depends on T029)
- [X] T041 [US3] Verify (and adjust if needed) `internal/app/graph/service.Subgraph`'s filter application (T025) never excludes the seed even when `optsFilter.build()` returns a filter the seed itself does not match — a hardening pass against T011's E2E scenario (depends on T025, T040)
- [X] T042 [P] [US3] Add unit tests in `internal/app/graph/service/subgraph_test.go` covering a filter matching zero reachable nodes (seed-only output, no error) and a combined `--kind`+`--tag`+`--attr` filter narrowing to the exact expected subset (depends on T025)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently — feature complete

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T043 [P] Implement the stderr truncation notice in `cmd/arc/graph/subgraph.go`: one plain, unstyled diagnostic line when `SubgraphResult.DirectTruncated` or `BacklinkTruncated` is `true` (research.md D10)
- [X] T044 [P] Carry `DirectReachable`/`DirectIncluded`/`DirectTruncated` and the `Backlink*` counts into the synthesized `Patch.Stats` map in `internal/app/graph/service.Subgraph`, per contracts/cli-contract.md's example stats block (depends on T025)
- [X] T045 [P] Add table-driven unit tests proving SC-007: on a fixture graph whose backlink pool exceeds a small configured `BacklinkCap`, the retained nodes are always exactly the highest-degree candidates, deterministic across repeated runs (depends on T024)
- [X] T046 [P] Update `README.md`'s quick-start example to mention `arc subgraph` (constitution Principle XII)
- [X] T047 [P] Manually run all quickstart.md scenarios (Scenarios 1-3, cap configuration, read-only verification) against the built binary and confirm expected output, exit codes, and stats/truncation behavior

---

## Phase 6: Bugfix BUG-001 — Stub emission for extraction-boundary referential integrity

**Purpose**: Fixes [BUG-001](bugs/BUG-001.md) (spec FR-017/SC-008, added 2026-07-05): an included node's structural links to a target excluded from the extraction boundary (`--depth`, a truncation cap, or the filter) were rendered verbatim with no corresponding node section, so applying extracted output into a graph that does not already contain the excluded targets (e.g. an empty graph) produced a dangling reference. Adds an opt-in `--stubs` flag that emits a minimal (kind + id only, empty body, never itself expanded) node section for every such boundary target.

**Reopened tasks** (implementations that must be extended, not tasks that were done wrong relative to their original scope):

- [X] T017 [P] ⚠️ Reopened (BUG-001) Implement `internal/app/graph/kernel/subgraph.go`: `SubgraphResult{Root, Seed, Depth, Patch, DirectReachable, DirectIncluded, DirectTruncated, BacklinkReachable, BacklinkIncluded, BacklinkTruncated}` per data-model.md — reopened to add a trailing `Stubs int` field (count of stub node sections emitted, mirroring the existing Direct/BacklinkIncluded counts) (reopened — BUG-001)
- [X] T025 ⚠️ Reopened (BUG-001) Implement `internal/app/graph/service.Subgraph(ctx, mounter, filter core.Filter, basename string, depth int, cfg kernel.SubgraphConfig, dir string) (kernel.SubgraphResult, error)` — reopened: signature gains a trailing `stubs bool` parameter; when `true`, after assembling the seed + surviving candidates, compute the boundary set (T049) and append one minimal stub `core.Node` per boundary target (T050), recording the stub count for T044 (depends on T015, T020, T024, T049, T050) (reopened — BUG-001)
- [X] T027 [P] ⚠️ Reopened (BUG-001) Implement `internal/app/graph/component.go`'s `Subgraph(ctx, mounter, filter, basename, depth, cfg, dir) (kernel.SubgraphResult, error)` delegator — reopened: signature gains a trailing `stubs bool` parameter, threaded straight through to `service.Subgraph` (depends on T025) (reopened — BUG-001)
- [X] T029 [US1] ⚠️ Reopened (BUG-001) Implement `cmd/arc/graph/subgraph.go`'s real `RunE` — reopened: also read the new `--stubs` flag (T048) and pass it through to `appgraph.Subgraph` (depends on T027, T028, T048) (reopened — BUG-001)
- [X] T044 [P] ⚠️ Reopened (BUG-001) Carry `DirectReachable`/`DirectIncluded`/`DirectTruncated` and the `Backlink*` counts into the synthesized `Patch.Stats` map — reopened: also carry the stub count (`stats["stubs"]`) when `--stubs` is enabled (depends on T025) (reopened — BUG-001)

**New tasks**:

- [X] T048 [P] [BUG-001] Add a local `--stubs` bool flag (default `false`) to `cmd/arc/graph/subgraph.go`'s command construction, alongside the existing `--depth`/`optsFilter` flags
- [X] T049 [BUG-001] Implement boundary-set computation in `internal/app/graph/service/subgraph.go`: for every node in the final included set (seed + surviving direct/backlink candidates), collect every `Edges`/`Links` target present in `nodeIndex` but not itself in the included set, deduped by ID (depends on T025's existing candidate assembly)
- [X] T050 [BUG-001] Implement stub-node emission in `internal/app/graph/service/subgraph.go`: when `stubs` is `true`, append one `core.Node{ID: id, Kind: nodeIndex[id].Kind}` per boundary-set entry (T049) to `Patch.Nodes` — no `Attrs`, `Text`, `Notes`, `Edges`, or `Links` — and never traverse a stub's own connections (no recursive expansion, or the `--depth` bound becomes meaningless) (depends on T049)
- [X] T051 [P] [BUG-001] Unit tests in `internal/app/graph/service/subgraph_test.go`: a boundary target gets a stub node only when `stubs=true` (absent when `stubs=false`, unchanged from pre-bugfix behavior); a stub node carries no attributes beyond kind/id and an empty body; a stub is never itself expanded even when another already-included node would otherwise put it within `--depth` (depends on T050)
- [X] T052 [P] [BUG-001] E2E regression test in `cmd/arc/graph/subgraph_test.go`: extract with `--stubs` from a graph where an included node references an excluded boundary target, apply the result into a freshly initialized, otherwise empty graph, then run `arc lint` and confirm no resolvable-links violation is reported (spec SC-008) — this is the specific apply-into-a-different-graph flow that surfaced BUG-001
- [X] T053 [P] [BUG-001] Populate the `--stubs` flag's own description text (Cobra `Flags().BoolVar` help string) per contracts/cli-contract.md's DS-11 shape (constitution Principle XII)
- [X] T054 [BUG-001] Manually verify the BUG-001 scenario against the built binary: `arc subgraph <basename> --stubs` on a graph with a boundary-excluded reference, apply the output into a fresh empty graph, run `arc lint`, confirm zero violations

**Checkpoint**: BUG-001 fixed — `--stubs` extraction is referentially self-contained even when applied into a graph that does not already contain the excluded targets; default (no `--stubs`) behavior is unchanged

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/core.RenderPatch` and `internal/app/graph`'s new `Subgraph` member (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: `arc subgraph <basename> --depth`, `--kind`/`--tag`/`--attr`, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced beyond what ADR 001/002 already cover (Principle I) — none expected; `RenderPatch`'s phase-1 placement is already covered by ADR 001's own domain-evolution model, confirm during review
- [X] TN05 Domain logic uses ports (interfaces) where needed; `cmd/arc/graph` wiring, `internal/core`, and `internal/app/graph/service` remain separated; no port was declared where none is needed (research.md D8) (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 `internal/core.RenderPatch` introduces no new third-party dependency; `internal/app/graph/service.Subgraph` makes zero `os.*` filesystem calls, using only `fsys.Store` (Principle VII)
- [X] TN10 Terminal output: `arc subgraph` deliberately makes no use of `bios.SCHEMA`'s styling per the user's explicit instruction, confirmed by inspection that `cmd/arc/graph/subgraph.go` never references `bios.SCHEMA`'s styled fields; `--quiet`/`--verbose`/`--json` still function correctly (Principle X, research.md D10)
- [X] TN11 Configuration precedence respected for the new `subgraph.directCap`/`subgraph.backlinkCap` fields; no secrets logged or involved (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc subgraph` (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md US1–US3 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/graph/subgraph_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `arc subgraph` is a new command with a new, additive `--json` `SubgraphResult` schema; `kernel.GrepResult`/`ApplyResult`'s existing `--json` contracts are untouched; no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Pre-implementation Refactoring (Phase 0)**: No dependencies — separate PR, run first
- **Setup (Phase 1)**: Depends on Phase 0 landing — can start immediately after
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2.5; User Story 1 is the deepest since it implements the full command surface and rendering — User Stories 2 and 3 extend the same files and therefore depend on Phase 3's tasks as well as Phase 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Bugfix BUG-001 (Phase 6)**: Depends on Additional Polish (extends T017/T025/T027/T029/T044, all already implemented) — must land before the final compliance gate re-runs
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases, including Phase 6

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; implements `cmd/arc/graph/subgraph.go`'s full render/exit-code surface that US2/US3 extend
- **User Story 2 (P2)**: Can start after Phase 2.5, but its flag-wiring tasks (T036-T038) attach to the `RunE` US1 builds (T029) — sequenced after US1 in practice, though its E2E test (T010) is independent and was written in Phase 2d
- **User Story 3 (P3)**: Can start after Phase 2.5, but its flag-wiring tasks (T040-T041) attach to the same `RunE` — sequenced after US1, though its E2E test (T011) is independent

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain/index/traversal foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other
- Within Phase 2.5: `internal/core` (T014-T016), `internal/app/graph/kernel` (T017), and `internal/app/config` (T018-T019) have no cross-dependencies and can proceed in parallel; `internal/app/graph/service` (T020-T026) depends on several of these landing first
- Once Phase 3 lands, User Stories 2 and 3 can proceed in parallel with each other

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Implement internal/core/markdown.go's RenderPatch and its shared body-rendering helper"
Task: "Implement internal/app/graph/kernel/subgraph.go (SubgraphResult)"
Task: "Extend internal/app/config/kernel/config.go (Config.Subgraph)"
```

## Parallel Example: Phase 3 User Story 1

```bash
# Once T027/T028 (component delegator + cmd scaffold) exist, launch together:
Task: "Implement RunE mount/config/Subgraph-call wiring in cmd/arc/graph/subgraph.go"
Task: "Implement the Human renderer (core.RenderPatch, no color) in cmd/arc/graph/subgraph.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 0: Pre-implementation Refactoring
2. Complete Phase 1: Setup
3. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
4. Complete Phase 2.5: Foundational Infrastructure
5. Complete Phase 3: User Story 1
6. Complete Phase N: Constitution Compliance Verification
7. **STOP and VALIDATE**: Run quickstart.md Scenario 1 against the built binary
8. Deploy/demo if ready — `arc subgraph` already extracts and serializes a one-hop, unfiltered subgraph at this point, missing only `--depth` control (US2) and `--kind`/`--tag`/`--attr` narrowing (US3)

### Incremental Delivery

1. Complete Phase 0 + Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Each story adds value without breaking previous stories

---

## Notes

- `[P]` tasks = different files, no dependencies
- `[Story]` label maps a task to its user story for traceability
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
- Phase 0 is included (unlike 006) because this feature's first task renames an existing, shared helper (`walkGrepNodeFiles` → `walkNodeFiles`) that `arc grep`'s own tests already depend on — a small, real rename of existing code, submitted separately from the new `Subgraph` behavior it enables
- User Stories 2 and 3 are not fully file-independent from User Story 1 here (they extend `cmd/arc/graph/subgraph.go` US1 creates) — this reflects that all three stories exercise one shared `Subgraph` use-case and one shared renderer, not three separate features; each remains independently *testable* via its own E2E test written in Phase 2d
- The two-independent-BFS-passes tradeoff (research.md D3, plan.md Complexity Tracking) is accepted as-is — no task above attempts to collapse it into one combined pass, since doing so would reintroduce the traversal-order-dependent capping research.md D3/D5 deliberately reject
- Phase 6 (BUG-001) reopens T017/T025/T027/T029/T044 rather than replacing them: each reopened task's original scope was correctly implemented against the spec as it stood at the time; the bug is a spec gap (no requirement previously governed referential integrity across the extraction boundary), not an implementation error, so reopening extends rather than corrects those tasks
