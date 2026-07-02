# Tasks: Apply a Document Patch to the Graph (`arc apply`)

**Input**: Design documents from `/specs/003-apply-patch/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`cli-contract.md`, `ast-contract.md`, `vcs-port-contract.md`, `config-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md); cross-referenced: `specs/002-arc-init/spec.md` (FR-017, added by this feature's planning)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3/US4, priorities P1/P2/P3/P4 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, US3, or US4 — maps to spec.md's four user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `cmd/arc/graph/` — Cobra command wiring for `arc apply` and its colocated E2E test
- `cmd/arc/ctrl/` — existing `arc init` wiring, touched for the config-seed fetch (research.md D5 revised)
- `internal/core/` — the graph AST, goldmark-backed parser/renderer, CORE §10 merge algebra, timeline derivation, kind/merge-rule vocabulary — no use-case dependency in either direction
- `internal/adapter/git/` — the git adapter, promoted from `internal/app/ctrl/adapter/git` (research.md D4)
- `internal/app/graph/{kernel,port,adapter,service}/` — the `graph` (graph I/O) domain use-case, per ADR 001's `componentX` layout
- `internal/app/config/{kernel,port,adapter,service}/` — the `config` (`.arc/config.yml`) domain use-case, same layout
- `internal/app/ctrl/` — existing `ctrl` domain, `Init`'s signature gains a `configSeed []byte` parameter

---

## Phase 0: Pre-implementation Refactoring — Git Adapter Promotion (research.md D4)

**Purpose**: Promote `internal/app/ctrl/adapter/git` to the shared `internal/adapter/git`, since both `ctrl` and the new `graph` use-case need git access (mirrors `internal/adapter/fsys`'s existing precedent). No behavior change to `arc init`. **MUST be submitted as its own PR**, before the rest of this feature's tasks, per the constitution's Task List Requirements for this optional phase.

- [ ] T001 [P] Move `internal/app/ctrl/adapter/git/git.go` to `internal/adapter/git/git.go`; update its package doc comment to describe it as the shared, cross-use-case git adapter (research.md D4), matching `internal/adapter/fsys`'s doc-comment style
- [ ] T002 Move `internal/app/ctrl/adapter/git/git_test.go` to `internal/adapter/git/git_test.go`; update package name/imports (depends on T001)
- [ ] T003 Update `cmd/arc/ctrl/init.go`'s import from `internal/app/ctrl/adapter/git` to `internal/adapter/git` (depends on T001)
- [ ] T004 Add `IsTracked(ctx context.Context, dir, path string) (bool, error)` to `internal/adapter/git.Git`, wrapping `git ls-files --error-unmatch <path>` per contracts/vcs-port-contract.md (exit 0 → `(true, nil)`; git's own "not tracked" exit status → `(false, nil)`; any other failure → `(false, err)`) (depends on T001)
- [ ] T005 [P] Add unit/integration test coverage for `IsTracked` in `internal/adapter/git/git_test.go` against a real `git` binary and `t.TempDir()`, covering tracked/untracked/error cases (depends on T004)
- [ ] T006 Run `go build ./... && go test ./...` and confirm every existing `specs/002-arc-init` E2E/unit test still passes unchanged — this promotion has zero behavior change (depends on T002, T003)

**Checkpoint**: Git adapter promoted, `arc init` behavior unchanged — ready to build the rest of this feature on top

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T007 Create the package skeleton: `internal/core/`, `internal/app/graph/{kernel,port,adapter/mock,service}/`, `internal/app/config/{kernel,port,adapter/http,adapter/mock,service}/`, `cmd/arc/graph/` directories per plan.md's Project Structure
- [ ] T008 Add `github.com/yuin/goldmark`, `github.com/yuin/goldmark-meta`, `gopkg.in/yaml.v3` to `go.mod`/`go.sum` (`go get github.com/yuin/goldmark github.com/yuin/goldmark-meta gopkg.in/yaml.v3`) per plan.md Technical Context (research.md D3, D5)
- [ ] T009 [P] Run `staticcheck ./...` and confirm it passes clean on the new (still-empty) package skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [ ] T010 Add the domain terms from spec.md Key Entities / data-model.md — Patch, Node Contribution, Source Node, Entity/Resource Node, Timeline Entry, Merge Behavior, Ingest Commit, Kind Registration — to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle I obligation, plan.md Constitution Check row I)
- [ ] T011 Verify no existing `internal/<domain>` package already defines a `Node`/`Link`/`Patch`/`MergeOp`-shaped type before introducing them in `internal/core` (none exist — this is the project's first `internal/core` package)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [ ] T012 Confirm `arc apply <patch.md>`'s bare-verb grammar (ADR 002 DS-01, research.md D9) and single positional argument against contracts/cli-contract.md
- [ ] T013 [P] Review contracts/cli-contract.md, contracts/ast-contract.md, contracts/vcs-port-contract.md, contracts/config-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [ ] T014 [P] Confirm the Phase 0-promoted `internal/adapter/git` is the sole git adapter before any new code references it, and confirm no existing adapter in the repository already covers HTTP fetch before creating `internal/app/config/adapter/http`
- [ ] T015 Define `graph.port.VCS` (`internal/app/graph/port/vcs.go`, contracts/vcs-port-contract.md) and `config.port.Fetcher` (`internal/app/config/port/fetcher.go`, contracts/config-contract.md) interface shapes as the design gate before any adapter code is written

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [ ] T016 [P] [US1] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US1's 4 acceptance scenarios (creates a new file for every patch-carried node; creates/extends the yearly and monthly timeline entries in chronological order; exactly one commit whose subject names the document and whose stats record per-kind counts; the command reports what was created) using the `sut()` helper — tests MUST compile and fail semantically (red phase)
- [ ] T017 [P] [US2] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US2's 4 acceptance scenarios (re-introducing an existing entity unions its relations with no duplicate file; a previously-empty resource field gets filled; an already-set resource field is preserved on divergence; commit stats distinguish merged from created) — red phase
- [ ] T018 [P] [US3] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US3's 3 acceptance scenarios (a registered domain kind is applied using its registered behavior; an unregistered kind is still applied — using the union default — with a warning, per FR-018; registering a kind removes the warning on the next apply) — red phase
- [ ] T019 [P] [US4] Write E2E test in `cmd/arc/graph/apply_test.go` for spec.md US4's 1 acceptance scenario (re-applying an already-tracked document makes no changes and reports clearly) — red phase
- [ ] T020 [P] Write E2E tests in `cmd/arc/graph/apply_test.go` for the Edge Cases tied to guard behavior: missing manifest field (FR-001/FR-002), malformed patch body structure (FR-002), target not an initialized graph (FR-014), and a merge conflict marker written while the commit still completes (FR-013) — per quickstart.md — red phase
- [ ] T021 [P] Write E2E tests in `cmd/arc/ctrl/init_test.go` for `specs/002-arc-init/spec.md` FR-017: the config-seed fetch succeeds (seeded `.arc/config.yml` matches the fetched content) and the fetch fails/returns malformed content (seeded file falls back to `core.CoreMergeRules`, `arc init` still succeeds with exit code 0) — via a mock `Fetcher` injected at the wiring layer — red phase

> T016–T020 all target the same new file (`cmd/arc/graph/apply_test.go`) and are therefore sequential in practice despite each being scoped to one story (mirrors `specs/002-arc-init/tasks.md`'s T010–T013 note). T021 is a separate, existing file and is genuinely parallelizable with the others.

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [ ] T022 Confirm `.arc/config.yml` is graph-root-scoped project configuration, not a secrets file, and that the config-seed fetch (research.md D5 revised) touches only a public, unauthenticated URL with no credential or secret material involved (plan.md Constitution Check row XI)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `internal/core` (AST, parser/renderer, merge algebra, timeline derivation, kind vocabulary), `internal/app/config` (load/save/resolve/default), the `internal/app/ctrl` config-seeding touch, and `internal/app/graph`'s scaffolding are all genuinely foundational — every one of US1–US4 exercises the same `service.Apply` pipeline, differing only in which guard/path fires. This phase builds that shared foundation; Phase 3+ wires story-specific behavior on top of it.

### `internal/core` — AST types and vocabulary

- [ ] T023 [P] Implement `internal/core/ast.go`: `Kind`, `MergeOp` (+ constants `MergeNone`/`MergeUnion`/`MergeUnionFirstWriter`/`MergeAppend`/`MergeValidatedOverwrite`), `Link`, `LinkBlock`, `Node`, `Patch` per data-model.md
- [ ] T024 [P] Unit tests for the plain value types' zero-value/shape behavior in `internal/core/ast_test.go` (depends on T023)
- [ ] T025 [P] Implement `internal/core/rules.go`: `CoreMergeRules`, `KnownProfileMergeRules`, `MergeRuleSet` (+ `MarshalYAML`/`UnmarshalYAML`, `Union`, `Lookup`), `ConfigPath` per data-model.md
- [ ] T026 [P] Unit tests for `MergeRuleSet.Union`/`.Lookup`/YAML round-trip in `internal/core/rules_test.go` (depends on T025)
- [ ] T027 [P] Implement `internal/core/errors.go`: `faults.Type` sentinel constants `ErrManifestInvalid`, `ErrPatchStructure` per data-model.md

### `internal/core` — Markdown parsing and rendering (goldmark, research.md D3/D3b)

- [ ] T028 Implement `internal/core/markdown.go`'s `ParsePatch(r io.Reader) (Patch, error)`: goldmark + goldmark-meta front-matter parsing, H1-kind/H2-node section walk, fenced-yaml `Attrs` block, prose/edge/link population per research.md D3 (depends on T023, T027)
- [ ] T029 Implement `internal/core/markdown.go`'s `ParseNode(r io.Reader) (Node, error)`: parses one on-disk graph node file (front-matter + body) per CORE §4/§9 (depends on T028)
- [ ] T030 Implement the bracket-strip-on-parse logic (research.md D3/D3b) shared by `ParsePatch`/`ParseNode`: recognize `[[Target]]`/`[[Target|alias]]`/`[predicate:: [[Target]]]` embedded in prose (not a standalone list-item edge), append each as a `Link` to `HRefs`, and strip the bracket markup from the string stored in `Text`/`Notes` (depends on T029)
- [ ] T031 [P] Unit tests for `ParsePatch`/`ParseNode` in `internal/core/markdown_test.go`: manifest validation (`ErrManifestInvalid`), body structure validation (`ErrPatchStructure`), fenced-yaml `Attrs` parsing, `Edges`/`Links` population, and bracket-strip-into-`HRefs` behavior (depends on T030)
- [ ] T032 Implement `internal/core/markdown.go`'s `RenderNode(n Node) ([]byte, error)`: front-matter (sorted attribute keys) + `Text` + `Edges` + `Links` (edges first, then blocks sorted by `Title`) + `Notes`, per contracts/ast-contract.md (depends on T023)
- [ ] T033 Implement `RenderNode`'s inline wikilink reconstruction (research.md D3b): walk `HRefs` in order, wrap the first eligible occurrence of each href's display substring (`Alias` if set, else `Target`) in brackets, enforcing (i) the match is not already inside brackets produced by an earlier href or pre-existing in the text, and (ii) the match starts and ends on a whitespace/text boundary, never mid-word; leave a href unlinked in prose (without dropping it from the data) when no eligible occurrence exists (depends on T032)
- [ ] T034 [P] Table-driven unit tests for `RenderNode`'s reconstruction in `internal/core/markdown_test.go`: a repeated target name (only one occurrence linked), a target substring embedded mid-word (not linked), a target immediately preceded by whitespace (linked), and a target whose display text already sits inside existing brackets (not double-wrapped) — per plan.md Testing (depends on T033)
- [ ] T035 [P] Round-trip unit tests in `internal/core/markdown_test.go`: `ParseNode` → `RenderNode` → `ParseNode` produces an equal `Node` for one representative fixture per core kind (`source`, `entity`, `resource`) (depends on T029, T033)

### `internal/core` — Merge algebra (CORE §10, research.md D6/D7)

- [ ] T036 Implement `internal/core/merge.go`'s `Merge(existing, incoming Node, op MergeOp, sourceID string) (merged Node, conflicts []string, err error)` for `MergeNone`/`MergeUnion`/`MergeUnionFirstWriter`/`MergeValidatedOverwrite` per research.md D6 (depends on T023)
- [ ] T037 Implement the conflict-marker rendering (research.md D7): embed the VISION.md-format `<<<<<<< id` / value / `=======` / value / `>>>>>>> id` string into a conflicted scalar field's value, falling back to the literal token `"existing"` when the prior writer of `existing`'s value cannot be determined (depends on T036)
- [ ] T038 [P] Table-driven unit tests for `Merge` in `internal/core/merge_test.go`: one case per `MergeOp` covering create-from-zero-`existing`, no-op (`none`), multi-valued-field union, first-writer scalar, empty-field-fill (`union-first-writer` only), and conflict-marker embedding (depends on T037)

### `internal/core` — Timeline derivation (CORE §9.4, research.md D8)

- [ ] T039 [P] Implement `internal/core/timeline.go`: `TimelinePeriods(published time.Time) (yearly, monthly string)`, `TimelineEntry(id, title string, authors []string, published time.Time) string` per research.md D8
- [ ] T040 [P] Unit tests for `TimelinePeriods`/`TimelineEntry` in `internal/core/timeline_test.go` (depends on T039)

### `internal/app/config` — `.arc/config.yml` load/save/resolve/default (research.md D5 revised)

- [ ] T041 [P] Implement `internal/app/config/kernel/config.go`: `Config{MergeRules core.MergeRuleSet}` per data-model.md
- [ ] T042 [P] Implement `internal/app/config/port/fetcher.go`: `Fetcher` interface (`Fetch(ctx, url) ([]byte, error)`) per contracts/config-contract.md
- [ ] T043 Implement `internal/app/config/adapter/http/client.go`: stdlib `net/http`-backed `Fetcher`, `http.Client{Timeout: 3 * time.Second}`, no retries, any non-2xx response treated as a failure (depends on T042)
- [ ] T044 [P] Integration test for the HTTP adapter in `internal/app/config/adapter/http/client_test.go` against `httptest.Server` (success, non-2xx status, timeout cases) (depends on T043)
- [ ] T045 [P] Implement `internal/app/config/adapter/mock/mock.go`: fake `Fetcher` with configurable return values/errors, for `Default`'s unit tests (depends on T042)
- [ ] T046 [P] Implement `internal/app/config/service/errors.go`: `faults.Safe1[string]` sentinel constants `ErrConfigMalformed`, `ErrConfigConflict` per data-model.md
- [ ] T047 Implement `internal/app/config/service/config.go`'s `Load`/`Save`/`Resolve` against `fsys.Store` per contracts/config-contract.md (`Resolve`: absent file → `core.CoreMergeRules`; malformed → `ErrConfigMalformed`; present → `core.CoreMergeRules.Union(loaded)`) (depends on T041, T046)
- [ ] T048 Implement `internal/app/config/service/config.go`'s `Default(ctx, fetcher) (kernel.Config, usedFallback bool)`: one fetch attempt, YAML-unmarshal on success, `core.CoreMergeRules` fallback on any failure (network error, non-2xx, timeout, malformed payload) — no `error` return, by construction (depends on T041, T042)
- [ ] T049 [P] Unit tests for `Load`/`Save`/`Resolve` in `internal/app/config/service/config_test.go` against a fake `fsys.Store` (depends on T047)
- [ ] T050 [P] Unit tests for `Default` in `internal/app/config/service/config_test.go` against `adapter/mock`'s `Fetcher`: fetch succeeds, fetch fails (network error), fetch returns non-2xx, fetch returns malformed YAML — each asserting `usedFallback` and the returned `Config` correctly (depends on T048, T045)
- [ ] T051 Implement `internal/app/config/component.go`: primary port `Resolve(store)`, `Save(store, cfg)`, `Default(ctx, fetcher)` as thin delegators into `service` (depends on T047, T048)
- [ ] T052 [P] Write `internal/app/config/README.md` documenting the `config` use-case per ADR 001's layout convention

### `internal/app/ctrl` — config-seed threading (research.md D5 revised)

- [ ] T053 Extend `internal/app/ctrl/component.go`'s `Init` signature to accept `configSeed []byte`, threaded through into `service.Init` (depends on T007)
- [ ] T054 Update `internal/app/ctrl/service/init.go`'s `writeLayout` call site to use a per-call copy of `kernel.DefaultLayout` with `MetaStubs[core.ConfigPath] = string(configSeed)` added (the package-level `DefaultLayout` itself stays static and config-free); extend `rollback` to also remove `core.ConfigPath` on a mid-run failure (depends on T053, T025)
- [ ] T055 [P] Update `internal/app/ctrl/service/init_test.go`'s existing unit tests for the new `Init` signature (pass a fixed `configSeed` fixture); add a case asserting `.arc/config.yml` is written with exactly the passed-in content (depends on T054)

### `internal/app/graph` — scaffolding

- [ ] T056 [P] Implement `internal/app/graph/kernel/apply.go`: `ApplyResult` per data-model.md
- [ ] T057 [P] Implement `internal/app/graph/port/vcs.go`: `VCS` interface (`IsTracked`, `StageAll`, `Commit`) per contracts/vcs-port-contract.md
- [ ] T058 [P] Implement `internal/app/graph/adapter/mock/mock.go`: in-memory fake `VCS` with configurable return values/errors and a call log, for service unit tests (depends on T057)
- [ ] T059 [P] Implement `internal/app/graph/service/errors.go`: `faults.Safe1[string]` sentinel constants `ErrNotAGraph`, `ErrPatchRead`, `ErrNodeWrite` per data-model.md
- [ ] T060 [P] Write `internal/app/graph/README.md` documenting the `graph` use-case per ADR 001's layout convention

### Wiring layer — `arc init`'s config-seed composition

- [ ] T061 Update `cmd/arc/ctrl/init.go`: construct the real `Fetcher` (`internal/app/config/adapter/http`), call `appconfig.Default(ctx, fetcher)`, marshal the resulting `kernel.Config` to YAML bytes, pass as `appctrl.Init`'s new `configSeed` argument; add the `"Fetching default configuration"` `--verbose` `Reporter` step and a `usedFallback`-conditional note, matching the existing verbose-only progress convention (depends on T043, T051, T053)

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Ingest a brand-new document into the graph (Priority: P1) 🎯 MVP

**Goal**: Applying a patch for a document not yet in the graph creates every node it carries, derives and appends timeline entries, and produces exactly one commit reporting what was created.

**Independent Test**: Apply a patch for a document that shares nothing with the current graph against a freshly `arc init`-ed graph; inspect the resulting file tree and `git log`, per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T016) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [ ] T062 [US1] Implement `internal/app/graph/service/apply.go`'s `Apply(ctx, mounter, vcs, rules, dir, patchPath)`: mount, guard `ErrNotAGraph` (`Store.Stat(".arc")`), read and parse the patch (`Store.Open` + `core.ParsePatch`, wrapping read failures as `ErrPatchRead`), and — for the all-new-content case — create every patch node via `core.Merge(zeroNode, incoming, op, patch.Document)` + `core.RenderNode` + `Store.Create` (`ErrNodeWrite`), reporting each step via `bios.Reporter` (depends on T028, T032, T036, T057, T059)
- [ ] T063 [US1] Implement timeline derivation/write-back in `internal/app/graph/service/apply.go`: `core.TimelinePeriods` + `core.TimelineEntry`, read-or-create `timeline/yearly/<YYYY>.md`/`timeline/monthly/<YYYY-MM>.md` via `core.ParseNode` + chronological insertion + `core.RenderNode`, per research.md D8 (depends on T062, T039)
- [ ] T064 [US1] Implement the commit step in `internal/app/graph/service/apply.go`: `port.VCS.StageAll` + `.Commit` with the CORE §11.3 subject/stats/`Source-Id:` trailer message; populate and return `kernel.ApplyResult{Document, Created, CommitHash, Timeline}` (depends on T063)
- [ ] T065 [US1] Implement `internal/app/graph/component.go`: primary port `Apply(ctx, mounter, vcs, rules, dir, patchPath) (kernel.ApplyResult, error)` as a thin delegator into `service.Apply` (depends on T064)
- [ ] T066 [US1] Implement `cmd/arc/graph/apply.go`: `NewApplyCmd() *cobra.Command` — calls `appconfig.Resolve(store)` then `appgraph.Apply`, wires `fsys.Local{}` and the real (promoted) git adapter, renders via `bios.Registry[kernel.ApplyResult]{Human: humanApplyPrinter{}}` per contracts/cli-contract.md (depends on T065, T051)
- [ ] T067 [US1] Populate `Short`/`Long`/`Example` help text for `arc apply` per contracts/cli-contract.md's DS-11 shape (constitution Principle XII) in `cmd/arc/graph/apply.go`
- [ ] T068 [US1] Register `graph.NewApplyCmd()` into `cmd/arc/root.go`'s command tree (depends on T066)

**Checkpoint**: At this point, User Story 1's E2E tests (T016) pass and `arc apply` is fully functional and independently testable against a graph with no overlapping content

---

## Phase 4: User Story 2 - Merge a patch's contribution into overlapping graph content (Priority: P2)

**Goal**: Applying a patch that mentions a subject already present in the graph merges into the existing node using its kind's declared merge behavior, rather than duplicating it.

**Independent Test**: Apply two patches that both reference an entity or resource of the same name; confirm only one node file exists afterward, carrying the union of both contributions, per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T017) and MUST currently be failing (red).

- [ ] T069 [US2] Extend `internal/app/graph/service/apply.go`'s per-node path (T062) to look up an existing node (`core.ParseNode` via `Store.Open`) before creating one, calling `core.Merge(existing, incoming, op, patch.Document)` instead of a bare create when the basename already exists on disk (spec FR-006) (depends on T062)
- [ ] T070 [US2] Populate `ApplyResult.Merged` (as distinct from `Created`) and `ApplyResult.Conflicts` (relative paths of node files whose merge flagged a conflict marker) in `internal/app/graph/service/apply.go` (depends on T069, T037)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Apply a patch that introduces domain-specific node kinds (Priority: P3)

**Goal**: A patch node whose kind is registered in `.arc/config.yml` is applied using its registered merge behavior; a patch node whose kind is unregistered is still applied — using the safe `union` default — with a warning, never refused.

**Independent Test**: Register a domain kind and its merge behavior for a graph, apply a patch containing a node of that kind, and confirm it uses the registered behavior; then confirm an unregistered kind still applies with a warning, per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T018) and MUST currently be failing (red).

- [ ] T071 [US3] Implement the kind-recognition step in `internal/app/graph/service/apply.go`: call `rules.Lookup(node.Kind)` before creating/merging each node; when `ok=false`, apply using `core.MergeUnion` and append a warning sentence to `ApplyResult.Warnings` naming the unrecognized kind (research.md D5-revised, spec FR-018) — the application is never refused on this basis (depends on T069, T025)
- [ ] T072 [US3] Wire `cmd/arc/graph/apply.go` to print one `bios.SCHEMA.StatusWarn`/`IconWarn`-styled stderr line per `ApplyResult.Warnings` entry (suppressed under `--quiet`/`--json`), and include `warnings` in the `--json` output shape per contracts/cli-contract.md (depends on T066, T071)
- [ ] T073 [US3] Implement the `PostRunE` conflict hint in `cmd/arc/graph/apply.go`: when `ApplyResult.Conflicts` is non-empty, print the ADR 002 DS-12 hint naming the conflicted file(s); suppressed under `--json`/`--quiet` (depends on T066, T070)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently

---

## Phase 6: User Story 4 - Re-running an already-applied patch is a safe no-op (Priority: P4)

**Goal**: Applying a patch for a document already tracked in the graph makes no changes and reports clearly, rather than duplicating a commit or node contribution.

**Independent Test**: Apply the same patch file twice in a row; confirm the second run makes no changes and produces no new commit, per quickstart.md Scenario 4.

### Implementation for User Story 4

> E2E test for this story was already written in Phase 2d (T019) and MUST currently be failing (red).

- [ ] T074 [US4] Implement the idempotency guard in `internal/app/graph/service/apply.go`: `port.VCS.IsTracked(ctx, dir, "sources/<document>.md")`, checked immediately after the `ErrNotAGraph` guard and before any write; when tracked, return `kernel.ApplyResult{Document: patch.Document, Skipped: true}` with zero filesystem/git changes (spec FR-003) (depends on T062, T057)
- [ ] T075 [US4] Implement `humanApplyPrinter`'s skip-path rendering in `cmd/arc/graph/apply.go`: `"✅ <document> is already tracked — nothing to do"` per contracts/cli-contract.md (depends on T066, T074)

**Checkpoint**: All four user stories pass their E2E tests independently — full feature complete

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [ ] T076 [P] Update `README.md`'s quick-start example to mention `arc apply` (constitution Principle XII)
- [ ] T077 [P] Manually run all 4 quickstart.md scenarios against the built binary and confirm expected output/exit codes
- [ ] T078 [P] Add table-driven unit tests in `internal/app/graph/service/apply_test.go` covering every guard/path combination (not-a-graph, already-tracked, create, merge, conflict, unregistered-kind-warning) against `adapter/mock`'s `VCS` and fakes of `fsys.Mounter`/`fsys.Store`, asserting `errors.Is(err, service.ErrXxx)` where applicable (constitution Principle VI)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [ ] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/core`, `internal/app/graph`, `internal/app/config`, the promoted `internal/adapter/git` directory-structure explanation (Principle I)
- [ ] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [ ] TN03 Command/flag surface matches the Phase 2b design exactly: `arc apply <patch.md>`, DS-03 persistent flags, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [ ] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced beyond what ADR 001/002 already cover (Principle I) — none expected for this feature; confirm during review
- [ ] TN05 Domain logic uses ports (interfaces); `cmd/arc/graph`/`cmd/arc/ctrl` wiring and `internal/adapter/{git,fsys}`/`internal/app/config/adapter/http` adapters remain separated (Principle III)
- [ ] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [ ] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [ ] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [ ] TN09 New external integrations (git subprocess, HTTP fetch, filesystem) follow the port/adapter pattern; no vendor SDK or `os.*`/`net/http` types leak through a port (Principle VII); the flagged, non-overridable HTTP fetch timeout (plan.md Complexity Tracking) is confirmed still documented, not silently dropped
- [ ] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling, including the new `SCHEMA.StatusWarn` usage (Principle X)
- [ ] TN11 Configuration precedence respected; `.arc/config.yml` is confirmed not a secrets file; the config-seed fetch touches no secret/credential material (Principle XI)
- [ ] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc apply` (Principle XII)
- [ ] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [ ] TN14 All spec.md US1–US4 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/graph/apply_test.go`; `specs/002-arc-init/spec.md` FR-017's two new scenarios pass in `cmd/arc/ctrl/init_test.go` (Principle VIII)
- [ ] TN15 Release/versioning impact assessed: `arc apply` is a new command with a new, additive `--json` `ApplyResult` schema (no prior contract to break); `arc init`'s existing `--json` `InitResult` schema is unchanged — no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Pre-implementation Refactoring (Phase 0)**: No dependencies on the rest of this feature — separate PR, run first
- **Setup (Phase 1)**: Depends on Phase 0 landing (so `internal/adapter/git` already exists for later reference) — can otherwise start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2.5 (nearly everything they need is built there); User Story 1 is the deepest since it implements `service.Apply`'s core pipeline itself — User Stories 2, 3, and 4 all extend the same file and therefore depend on Phase 3's tasks as well as Phase 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; implements the core `service.Apply` and `cmd/arc/graph/apply.go` that US2/US3/US4 extend
- **User Story 2 (P2)**: Can start after Phase 2.5, but its two tasks (T069, T070) extend the file US1 created (T062) — sequenced after US1 in practice, though its E2E test (T017) is independent and was written in Phase 2d
- **User Story 3 (P3)**: Same shape as US2 — its tasks (T071–T073) extend `service/apply.go` and `cmd/arc/graph/apply.go` from US1/US2
- **User Story 4 (P4)**: Same shape — its tasks (T074, T075) extend the same two files

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain/adapter foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Phase 0 tasks marked `[P]` can run in parallel; all Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other
- Within Phase 2.5: `internal/core`'s independent files (`ast.go`/T023, `rules.go`/T025, `errors.go`/T027, and — once T023 lands — `merge.go`/T036 and `timeline.go`/T039 in parallel with the `markdown.go` chain T028-T035), `internal/app/config`'s kernel/port (T041, T042), and `internal/app/graph`'s kernel/port/errors (T056, T057, T059) have no dependencies on each other and can run in parallel
- Once Phase 2.5 completes, User Story 1's chain (T062-T068) is mostly sequential (each depends on the previous); User Stories 2, 3, and 4 can then proceed in parallel with each other once US1 is done, since they touch different concerns within the files US1 created

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Implement internal/core/ast.go (Kind, MergeOp, Link, LinkBlock, Node, Patch)"
Task: "Implement internal/core/rules.go (CoreMergeRules, KnownProfileMergeRules, MergeRuleSet)"
Task: "Implement internal/core/errors.go sentinel constants"
Task: "Implement internal/app/config/kernel/config.go (Config)"
Task: "Implement internal/app/config/port/fetcher.go (Fetcher interface)"
Task: "Implement internal/app/graph/kernel/apply.go (ApplyResult)"
Task: "Implement internal/app/graph/port/vcs.go (VCS interface)"
Task: "Implement internal/app/graph/service/errors.go sentinel constants"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 0: Pre-implementation Refactoring (git adapter promotion, separate PR)
2. Complete Phase 1: Setup
3. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
4. Complete Phase 2.5: Foundational Infrastructure
5. Complete Phase 3: User Story 1
6. Complete Phase N: Constitution Compliance Verification
7. **STOP and VALIDATE**: Run quickstart.md Scenario 1 against the built binary
8. Deploy/demo if ready — `arc apply` against a graph with no overlapping content is already a usable MVP

### Incremental Delivery

1. Complete Phase 0 (separate PR) → Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Add User Story 4 → Verify against Phase N → Deploy/Demo
6. Each story adds value without breaking previous stories

---

## Notes

- `[P]` tasks = different files, no dependencies
- `[Story]` label maps a task to its user story for traceability
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
- Phase 0 is included (optional in the template) because this feature promotes `internal/app/ctrl/adapter/git` to a shared package — a structural change to already-shipped `002-arc-init` code, independent of this feature's new behavior
- User Stories 2, 3, and 4 are not fully file-independent from User Story 1 here (they extend `service/apply.go` and `cmd/arc/graph/apply.go` that US1 creates) — this reflects that all four stories exercise one shared `Apply` use-case, not four separate features; each remains independently *testable* via its own E2E test written in Phase 2d
- This feature also touches already-shipped `internal/app/ctrl` code outside Phase 0 (T053-T055, T061 — the config-seed signature/wiring change): unlike the git-adapter promotion, this is a genuine behavior addition (`specs/002-arc-init/spec.md` FR-017) tightly coupled to this feature's own `.arc/config.yml` design, not a pure refactor, so it is scoped to Phase 2.5 rather than Phase 0
