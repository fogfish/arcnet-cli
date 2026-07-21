# Tasks: Predicate-First Graph Node Model

**Input**: Design documents from `/specs/010-predicate-node-model/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`ast-contract.md`, `subgraph-json-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story. Because this feature reshapes the one domain type every existing use-case imports, Go's compiler enforces a single, whole-module Foundational phase before any story-specific behavior can even build — see Phase 2.5.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `internal/core/{ast.go,rules.go,markdown.go,merge.go,filter.go,errors.go}` + `_test.go` siblings — the primary domain reshape
- `internal/app/{schema,graph,lint}/**` + their `_test.go` siblings — mechanical compile-fix ripple (no new business logic, plan.md Summary)
- `cmd/arc/{graph,lint}/**` — E2E tests only; no `cmd/` production-code/flag changes
- No new package, no new command, no new port/adapter (plan.md Structure Decision)

**Bugfix**: 2026-07-07 — BUG-001 Updated from bugfix patch: reopened T021/T025/T029 (⚠️ Reopened markers below) and added T079-T082 to fix `parsePatchBody`'s over-broad requirement that every patch node's own yaml fence redundantly duplicate `"@id"`/`"@type"`, which rejected well-formed, non-legacy patches (e.g. `fogfish/bots`' output) that derive identity/type from their `"## <ID>"`/`"# <Type>"` section headings alone, per CORE §12.2's pre-existing convention.

**Bugfix**: 2026-07-20 — BUG-002 Updated from bugfix patch: added Phase 6 (T083-T091) to thread `core.Index` into patch/node parsing and resolve a `**Label**`-headed body block's predicate identity and role against the schema, instead of silently dropping any block that isn't wikilink-shaped. No task reopened as a false completion — T023/T024 correctly implemented what they were scoped to, with the resulting limitation explicitly flagged and deferred rather than hidden; this phase is the deferred follow-up, not a correction.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the ground this feature builds on, before touching any file

- [X] T001 Confirm no new package/directory is needed — this feature reshapes the existing `internal/core.Node` and mechanically updates its existing callers (plan.md Project Structure); no scaffolding step required
- [X] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged (plan.md Technical Context: reuses existing `goldmark`/`goldmark-meta`/`yaml.v3`/`fogfish/faults`/`fogfish/it/v2`)
- [X] T003 [P] Run `staticcheck ./...` and `go build ./...` and confirm both pass clean before any change, establishing a baseline to diff against

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Update [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary: reshape the **Node** entry (`@id`/`@type`, `Attrs`/`Predicate`, `Texts`, unified `Edges`), add a **Predicate** entry, add a **Text Predicate / Prose Field** entry, and remove/retire the **Link Block** entry it supersedes (plan.md Constitution Check rows I/II obligation, data-model.md)
- [X] T005 Verify no existing `internal/core` type already models "one value or one reference, exactly one of two" before introducing `Predicate` — confirmed none exists (research.md D2)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm this feature introduces **zero** new/changed Cobra commands, flags, or help text — the only `--json` schema change is the already-documented, intentional break in contracts/subgraph-json-contract.md (gate check, no `cmd/` flag/command changes)
- [X] T007 [P] Review contracts/ast-contract.md and contracts/subgraph-json-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's 17 functional requirements — gate check, no changes expected

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T008 Confirm this feature introduces no new external integration or adapter — the only I/O touched (`fsys.Store` via the existing, unmodified `Store`/`Mounter`) is unchanged; no new port

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [ ] T009 [P] [US1] Write/rewrite E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US1's 5 acceptance scenarios (a node's `"@id"`/`"@type"` establish identity/type with no fallback; several named `Texts` fields survive a round-trip individually; single- and multi-valued `Attrs` both parse as lists; flat-bulleted and heading-grouped links both land in one `Edges` list; a round-tripped node is byte-stable on a second round-trip), using `sut()` against predicate-first fixtures — tests MUST compile and fail semantically (red phase)
- [ ] T010 [P] [US2] Write/rewrite E2E tests in `cmd/arc/graph/apply_test.go`, `cmd/arc/graph/grep_opts_test.go`, `cmd/arc/graph/subgraph_test.go`, `cmd/arc/graph/serve_test.go`, and `cmd/arc/lint/lint_test.go` for spec.md US2's 5 acceptance scenarios (patch merges into the node identified by `"@id"`; lint evaluates real attrs/links; grep matches content in a non-default named `Texts` predicate; subgraph Markdown and `--json` agree on identity/attrs-as-lists/texts/unified-edges; serve displays identity/prose/attrs/links correctly) against predicate-first fixtures — red phase
- [ ] T011 [P] [US3] Write/rewrite E2E tests in `cmd/arc/graph/apply_test.go` and `cmd/arc/lint/lint_test.go` for spec.md US3's 4 acceptance scenarios (legacy `kind` field with no `"@id"`/`"@type"` rejected; `"@type"`-only node rejected, no title/period fallback; `"@id"` ≠ basename rejected; a failed read makes zero writes) against old-format fixtures, asserting non-zero exit and an unchanged working tree — red phase

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T012 Confirm this feature introduces no new configuration surface and no secret/credential material anywhere in `internal/core` or the packages it ripples into

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `internal/core.Node`'s reshape (identity, `Attrs`, `Texts`, unified `Edges`) is the one change every user story and every downstream package depends on — nothing in the module compiles until it and its mechanical ripple are done (plan.md Summary: "Go won't compile otherwise"). This phase builds that shared foundation once; Phase 3+ then only needs to turn each story's own E2E tests from Phase 2d green.

### `internal/core` — type reshape (data-model.md, research.md D1/D2/D5)

- [X] T013 [P] Add `Predicate struct { Value any; Target string; Alias string }` to `internal/core/ast.go` (research.md D2)
- [X] T014 Replace `Node.Kind Kind` with `Node.Type string` in `internal/core/ast.go`; delete the `Kind` named type entirely (research.md D1)
- [X] T015 Replace `Node.Attrs map[string]any` with `Node.Attrs map[string][]Predicate` in `internal/core/ast.go` (depends on T013)
- [X] T016 Replace `Node.Text string` + `Node.Notes string` with `Node.Texts map[string]string` in `internal/core/ast.go` (data-model.md)
- [X] T017 Replace `Node.Edges []Link` + `Node.Links map[string]LinkBlock` with a single `Node.Edges []Link`; delete the `LinkBlock` type entirely in `internal/core/ast.go` (research.md D5)
- [X] T018 [P] Update `MergeRuleSet` from `map[Kind]MergeOp` to `map[string]MergeOp`, and `Lookup(kind Kind)` to `Lookup(kind string)`, in `internal/core/rules.go` (depends on T014)
- [X] T019 [P] Unit tests in `internal/core/ast_test.go`: a `Predicate` with only `Value` set vs. only `Target` set; an `Attrs` entry with one vs. several `Predicate`s; a `Texts` map with multiple distinct keys; a single `Edges` slice mixing what were previously bare and grouped links — written first, MUST fail to compile/pass until T013-T017 land (red phase)

**Checkpoint**: `internal/core.Node`'s new shape exists; nothing outside `internal/core` compiles yet — expected, next steps fix that

### `internal/core` — parser (`markdown.go`) (research.md D3/D4/D5/D7)

- [X] T020 Delete `deriveNodeID`'s `id`/`title`/`period` fallback chain in `internal/core/markdown.go`; read `"@id"` only, front matter absent/empty `"@id"` is an error (depends on T014)
- [X] T021 ⚠️ Reopened (BUG-001) Read `"@type"` instead of `"kind"` for `Node.Type` in `ParseNode`/`parsePatchBody`, `internal/core/markdown.go` (depends on T014) — reopened because the mechanical "instead of kind" reading was implemented as "stop deriving from the `"# <Type>"` heading in `parsePatchBody` entirely, require an explicit `"@type"` fence key," a bigger behavioral change than this task described; see T079-T080
- [X] T022 Wrap every remaining front-matter key's value into `[]Predicate` (scalar → one-element list, YAML sequence → one element per item) when building `Attrs`, in `internal/core/markdown.go` (depends on T015)
- [X] T023 Implement `textPredicateFor(nodeType string, leading bool) string`, the hardcoded `@type`→text-predicate lookup table (`source`→`abstract`/`notes`, `entity`→`definition`/`notes`, `resource`→`relevance`/`notes`, `hypothesis`→`claim`/`notes`, `aporia`→`tension`/`notes`, `thought`→`claim`/`notes`, fallback `text`/`notes`), in `internal/core/markdown.go`, with a GoDoc comment flagging it as a temporary stopgap superseded by spec 011's Schema Index (research.md D4)
- [X] T024 Update `walkNodeBody` in `internal/core/markdown.go`: keep its existing leading-prose/optional-bare-list/heading-or-bold-label-blocks/trailing-prose structural recognition, but (a) route leading/trailing prose into `Texts[textPredicateFor(...)]` instead of `Text`/`Notes`, and (b) flatten the bare list's items and every heading/bold-label block's items into one `Edges` slice in document order, with no per-block grouping key retained (depends on T016, T017, T023)
- [X] T025 ⚠️ Reopened (BUG-001) Implement old-format detection in `internal/core/markdown.go`/`internal/core/errors.go`: a front-matter `kind` key present at all, or `"@id"` absent/empty, or `"@type"` absent/empty, or (for `ParseNode`) `"@id"` ≠ the file's basename, each returns `ErrManifestInvalid` with a message naming the specific file and the specific problem — no partial `Node` is ever constructed (depends on T020, T021; research.md D7) — reopened because "`"@id"`/`"@type"` absent/empty" was applied to `parsePatchBody` as "absent from the yaml fence," rejecting well-formed, non-legacy patches whose identity/type come from their `"## <ID>"`/`"# <Type>"` section headings instead (CORE §12.2's own convention); see T079-T080
- [X] T026 Update `renderAttrYAML`/`renderFrontMatter` in `internal/core/markdown.go`: render `"@id"` and `"@type"` first, both quoted; render every other `Attrs` key sorted alphabetically, a single-element `[]Predicate` as a bare scalar and a multi-element list as a YAML sequence (depends on T015, T022; research.md D3)
- [X] T027 Update `renderNodeBody` in `internal/core/markdown.go`: render `Texts` (leading-slot key, then any other keys sorted alphabetically, then the trailing-slot key), then `Edges` as one flat bulleted list only — delete grouped-heading rendering entirely (depends on T024; research.md D6)
- [X] T028 Update `RenderPatch`'s per-node fence and `"# <Type>"` heading grouping to use `Node.Type` instead of `Node.Kind`, in `internal/core/markdown.go` (depends on T014, T026)
- [X] T029 [P] ⚠️ Reopened (BUG-001) Unit tests in `internal/core/markdown_test.go`: `ParseNode`/`RenderNode`/`RenderPatch` round-trip for every ARCNET-CORE §11 worked example (`source`, `entity`, `resource`, `timeline`) plus one DOMAIN-ARTICLE `hypothesis` example (`derivedFrom`/`assumes`/`addresses`) — written first, red phase (depends on T020-T028 existing as compile targets) — reopened because these round-trip cases only exercised `parsePatchBody`'s own chosen fence-duplication convention, never a CORE §12.2-canonical, heading-only patch shape (no yaml-fence `"@id"`/`"@type"` keys), so they could not have caught BUG-001; see T081
- [X] T030 [P] Unit tests in `internal/core/markdown_test.go`: old-format rejection — legacy `kind` field, missing `"@id"`, missing `"@type"`, `"@id"` ≠ basename — each producing `ErrManifestInvalid` with the specific offending field named, zero partial `Node` returned (depends on T025)
- [X] T031 [P] Unit tests in `internal/core/markdown_test.go`: idempotent round-trip (`RenderNode(ParseNode(RenderNode(n)))` byte-equal to `RenderNode(n)`) and the explicitly permitted cosmetic exception (a node originally written with a `"## <Label>"` grouped link block renders flat on re-render, with content/connectivity unchanged) (depends on T027)

### BUG-001 fix: `parsePatchBody` must accept CORE §12.2-canonical, heading-only patch sections

- [X] T079 [BUG-001] [P] Confirm data-model.md's `Node.Type`/`Patch.Nodes` rows, contracts/ast-contract.md's Parsing section, and spec.md's FR-011/FR-018/Edge Cases entry now consistently state that a patch-document node contribution's `"@id"`/`"@type"` are satisfied by its own `"## <ID>"`/`"# <Type>"` section headings, with an explicit yaml-fence key optional and (if present) cross-checked for agreement — design-artifact gate check, no code change (already patched by this bugfix pass)
- [X] T080 [BUG-001] Fix `parsePatchBody` in `internal/core/markdown.go`: derive `Node.Type` from the enclosing `"# <Type>"` section heading and `Node.ID` from the `"## <ID>"` heading when no explicit `"@id"`/`"@type"` key is present in the node's own yaml fence; when an explicit key IS present, cross-check it against the corresponding heading and reject the contribution as inconsistent on disagreement, rather than requiring the key unconditionally — reject only on a legacy `kind` field, or when neither a heading nor an explicit key establishes identity/type at all (depends on T079; supersedes T021/T025's over-broad patch-body rejection)
- [X] T081 [BUG-001] [P] Add/rewrite unit tests in `internal/core/markdown_test.go` for `parsePatchBody`: a CORE §12.2-canonical, heading-only patch shape (no yaml-fence `"@id"`/`"@type"` keys, mirroring the pre-spec-010 fixture shape and real external patch producers) succeeds; an explicit fence key agreeing with its heading succeeds; an explicit fence key disagreeing with its heading is rejected; confirm no regression to T030's legacy-`kind`-field/missing-identity rejection cases (depends on T080; supersedes T029's patch-body coverage gap)
- [X] T082 [BUG-001] [P] Add an E2E fixture/test in `cmd/arc/graph/apply_test.go` using a heading-only patch shape (no yaml-fence `"@id"`/`"@type"` keys) to confirm `arc apply` accepts it end-to-end, matching the shape real external patch-generating tools (e.g. `fogfish/bots`) already produce (depends on T080)

**Checkpoint**: `internal/core`'s parser/renderer round-trips the new shape; `internal/core/markdown_test.go` passes; BUG-001's heading-only patch shape is accepted by both `internal/core.ParsePatch` and `arc apply` end-to-end

### `internal/core` — merge (`merge.go`) (data-model.md "Relationships / Lifecycle")

- [X] T032 Rewrite `mergeCore`'s prose handling in `internal/core/merge.go`: merge `Texts` key-by-key over the union of both nodes' keys, applying the existing scalar-merge policy per key (was: two fixed `Text`/`Notes` calls) (depends on T016)
- [X] T033 Rewrite `mergeCore`'s `Attrs` handling in `internal/core/merge.go`: merge each key's `[]Predicate` list per the node kind's existing `MergeOp` policy (list-level, not element-level) — no change to which `MergeOp` applies to which kind (depends on T015)
- [X] T034 Rewrite `mergeCore`'s edge handling in `internal/core/merge.go`: union `Edges` as one list; delete `mergeLinkBlocks` and any `Links`-specific merge code (depends on T017)
- [X] T035 [P] Unit tests in `internal/core/merge_test.go`: `Texts` per-key merge across multiple keys, `Attrs`-list merge (single- and multi-valued), unified-`Edges` union — one case per existing `MergeOp` (`none`/`union`/`union-first-writer`/`append`/`validated-overwrite`) — written first, red phase (depends on T032-T034 existing as compile targets)

**Checkpoint**: `internal/core`'s merge algebra operates correctly over the new shape; `internal/core/merge_test.go` passes

### `internal/core` — filter (`filter.go`)

- [X] T036 [P] Update `internal/core/filter.go`: `node.Kind` → `node.Type`; `node.Attrs[name]` accessed list-aware (iterate `[]Predicate`, read `.Value`) for tag/attribute matching (depends on T014, T015)
- [X] T037 [P] Unit tests in `internal/core/filter_test.go`: list-valued `Attrs` match cases (single value, multiple values, no match) — written first, red phase (depends on T036 existing as compile target)

**Checkpoint**: `internal/core` fully reshaped and internally self-consistent — `go test ./internal/core/...` passes

### Mechanical downstream compile-fix ripple (no new business logic, plan.md Summary)

- [X] T038 [P] Update `internal/app/schema/component.go`: `core.Kind` → `string`
- [X] T039 [P] Update `internal/app/schema/kernel/schema.go`: `core.Kind` → `string` (`SchemaKind`, `coreKindDescriptions`, `KindDescription`)
- [X] T040 Update `internal/app/schema/service/schema.go`: `core.Kind` → `string`; `node.Attrs["merge"]` read list-aware (first `Predicate`'s `.Value.(string)`) (depends on T015, T018)
- [X] T041 [P] Update `internal/app/schema/service/schema_test.go` fixtures to the predicate-first shape (depends on T040)
- [X] T042 [P] Update `internal/app/graph/port/schema.go`: `core.Kind` → `string`
- [X] T043 [P] Update `internal/app/graph/kernel/apply.go`: `map[core.Kind]int` → `map[string]int` (`Created`, `Merged`)
- [X] T044 [P] Update `internal/app/graph/kernel/grep.go`: `Kind core.Kind` JSON field → `Type string`
- [X] T045 Update `internal/app/graph/service/apply.go`: `node.Kind` → `node.Type`, `core.Kind` → `string` throughout (`coreKindFolders`, `nodeFolder`), `isStub` checks `Attrs`/`Texts`/`HRefs`/`Edges` (no more `Text`/`Notes`/`Links`) (depends on T014-T017)
- [X] T046 [P] Update `internal/app/graph/service/apply_test.go` fixtures to the predicate-first shape; add cases asserting zero writes on an old-format input (depends on T045)
- [X] T047 Update `internal/app/graph/service/grep.go`: `node.Kind` → `node.Type`; match against `Texts` values instead of `Text` (depends on T016)
- [X] T048 [P] Update `internal/app/graph/service/grep_test.go` fixtures, including a case matching inside a non-default named `Texts` predicate (depends on T047)
- [X] T049 Update `internal/app/graph/service/subgraph.go`: `n.Edges`/`n.Links` collapse to `n.Edges` only; `target.Kind` → `target.Type` (depends on T017)
- [X] T050 [P] Update `internal/app/graph/service/subgraph_test.go` fixtures and `--json` golden output to match contracts/subgraph-json-contract.md's "After" shape exactly (depends on T049)
- [X] T051 [P] Update `internal/app/lint/service/rules_frontmatter.go`: `node.Kind` → `node.Type`
- [X] T052 [P] Update `internal/app/lint/service/rules_history.go`: `node.Kind` → `node.Type`
- [X] T053 Update `internal/app/lint/service/rules_identity.go`: `node.Kind` → `node.Type`; `node.Attrs["category"]` read list-aware (depends on T015)
- [X] T054 Update `internal/app/lint/service/rules_links.go`: delete `sortedLinkKeys`/`LinkBlock` iteration; iterate the single `node.Edges` (plus `node.HRefs`); `node.Kind` → `node.Type` (depends on T017)
- [X] T055 Update `internal/app/lint/service/rules_predicates.go`: delete `node.Links` iteration; use `node.Edges` only; `node.Kind` → `node.Type` (depends on T017)
- [X] T056 [P] Update `internal/app/lint/kernel/lint.go`: `Kind core.Kind` JSON field → `Type string`
- [X] T057 Update `internal/app/lint/service/lint.go`: `kindIndex map[string]core.Kind` → `map[string]string`; `p.Node.Kind` → `p.Node.Type`; `status.Kind` → `status.Type` (depends on T014, T056)
- [X] T058 [P] Update `internal/app/lint/service/rules_links_test.go` fixtures to the predicate-first shape (depends on T054)
- [X] T059 [P] Update `internal/app/lint/service/rules_predicates_test.go` fixtures to the predicate-first shape (depends on T055)
- [X] T060 [P] Update `cmd/arc/graph/apply.go` internal `Node`/`Kind` references (no CLI-visible change) (depends on T045)
- [X] T061 [P] Update `cmd/arc/graph/grep.go` internal `Node`/`Kind` references (no CLI-visible change) (depends on T047)
- [X] T062 [P] Update `cmd/arc/graph/serve.go` internal `Node`/`Kind` references (no CLI-visible change) (depends on T049)
- [X] T063 [P] Update `cmd/arc/graph/grep_opts_test.go` fixtures to the predicate-first shape
- [X] T064 [P] Update `cmd/arc/graph/serve_test.go` fixtures to the predicate-first shape
- [X] T065 Run `go build ./...` and `staticcheck ./...` and confirm the whole module compiles and lints clean (depends on T038-T064)

**Checkpoint**: Foundation ready — `go build ./...` succeeds, every existing pre-feature test that doesn't itself assert old-format-specific behavior compiles; Phase 2d's E2E tests (T009-T011) can now actually run (red or green) instead of failing to compile

---

## Phase 3: User Story 1 - Author a node with open, predicate-keyed identity and content (Priority: P1) 🎯 MVP

**Goal**: A graph maintainer's node file, written with `"@id"`/`"@type"`, several named prose fields, list-shaped attributes, and links in either original layout, survives `arc apply` and a subsequent round-trip byte-for-byte (except permitted cosmetic edges-grouping normalization).

**Independent Test**: Apply a patch whose node sections use the new shape into a fresh graph, inspect every created node file, and re-apply an unchanged follow-up to confirm stability — per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red) until Phase 2.5 completes. Phase 2.5 already contains all the production code this story needs — this phase is where its own E2E tests are confirmed green with no further src changes.

- [X] T066 [US1] Confirm E2E tests T009 in `cmd/arc/graph/apply_test.go` pass against the Phase 2.5 foundation with no further production-code changes (depends on T065, T009)
- [X] T067 [P] [US1] Add/finalize `testdata/` fixtures for `cmd/arc/graph/apply_test.go` covering every ARCNET-CORE §11 worked example referenced by T009, if not already present from T029's `internal/core` fixtures (depends on T029, T009)
- [X] T068 [US1] Verify, via `cmd/arc/graph/apply_test.go`, that re-applying an already-applied patch a second time produces a byte-identical node file (idempotent round-trip, spec FR-015) (depends on T066)

**Checkpoint**: At this point, User Story 1's E2E tests (T009) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Operate the full toolchain against a predicate-first graph (Priority: P2)

**Goal**: `arc apply`, `arc lint`, `arc grep`, `arc subgraph` (Markdown and `--json`), and `arc serve` all read/write/display the predicate-first shape correctly and consistently with each other.

**Independent Test**: Exercise each command against a small predicate-first fixture graph and confirm correct output — per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T010) and MUST currently be failing (red) until Phase 2.5 completes. Phase 2.5 already contains all the production code this story needs — this phase confirms each command's E2E coverage is both green and complete.

- [X] T069 [US2] Confirm E2E tests T010 across `cmd/arc/graph/apply_test.go`, `cmd/arc/graph/grep_opts_test.go`, `cmd/arc/graph/subgraph_test.go`, `cmd/arc/graph/serve_test.go`, and `cmd/arc/lint/lint_test.go` pass against the Phase 2.5 foundation (depends on T065, T010)
- [X] T070 [P] [US2] Verify `arc subgraph --json` output matches contracts/subgraph-json-contract.md's "After" shape exactly (field names, `attrs`-as-arrays, `texts` map, single `edges` array) via `subgraph_test.go` golden-output assertions (depends on T050, T069)
- [X] T071 [P] [US2] Verify `arc grep` matches content inside a non-default named `Texts` predicate (e.g. `definition`, not just the fallback `text`) via `grep_opts_test.go` (depends on T048, T069)
- [X] T072 [P] [US2] Verify `arc lint`'s identity/frontmatter/links/predicates rules evaluate the real `Attrs`/`Edges` shapes (not a stale two-slot assumption) via `cmd/arc/lint/lint_test.go` (depends on T058, T059, T069)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Fail safely on a pre-0.5 graph instead of misreading it (Priority: P3)

**Goal**: Every arc command detects an old-format graph file (legacy `kind` field, missing `"@id"`/`"@type"`, or `"@id"` ≠ basename) and exits with a clear error and zero writes, never a silent misread.

**Independent Test**: Point arc at a fixture graph still written in the pre-0.5 shape and confirm every command exits with an error rather than partial/corrupted output — per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T011) and MUST currently be failing (red) until Phase 2.5 completes. Phase 2.5's T025 already implements the rejection logic — this phase confirms every command surfaces it correctly and makes no write.

- [X] T073 [US3] Confirm E2E tests T011 in `cmd/arc/graph/apply_test.go` and `cmd/arc/lint/lint_test.go` pass against T025's old-format detection (depends on T065, T025, T011)
- [X] T074 [P] [US3] Verify, via `cmd/arc/graph/apply_test.go`, that an old-format graph file causes `arc apply` to make zero file writes and zero commits (git state unchanged) (depends on T073)
- [X] T075 [P] [US3] Add `testdata/` fixtures for all four old-format rejection cases (legacy `kind` field, missing `"@id"`, missing `"@type"`, `"@id"` ≠ basename) under `cmd/arc/graph/testdata/` and `cmd/arc/lint/testdata/`, if not already present from T011 (depends on T011)

**Checkpoint**: User Story 3's E2E tests (T011) pass; all three user stories' E2E tests are green together

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T076 [P] Manually run all three quickstart.md scenarios end-to-end against a built `arc` binary, confirming every documented output matches actual behavior
- [X] T077 [P] Remove now-dead helpers left over from the old shape (`sortedLinkKeys`, `linkBlockKey`, `camelizeTitle` if no longer referenced, `mergeLinkBlocks`) across every touched file
- [X] T078 Search `README.md` and any generated command reference for node-file examples still shown in the old `kind`/`Text`/`Notes`/`Links` shape; update to the predicate-first shape

---

## Phase 6: Bugfix BUG-002 — Schema-Blind Parsing Silently Drops Non-Wikilink Body Content

**Purpose**: Addresses [bugs/BUG-002.md](bugs/BUG-002.md): `arc apply` silently drops a `**Label**`-headed body block's content whenever it doesn't structurally look like a wikilink list, violating FR-005/FR-006/FR-009's already-live verbatim-preservation guarantees. Root cause: `walkNodeBody`/`blockTitle`/`collectListLinks` (`internal/core/markdown.go`) take no `core.Index`, so a block's predicate identity and role are never consulted — this feature's own plan.md Complexity Tracking table named and deferred exactly this gap ("exactly the Schema Index question spec 011 owns"), and neither spec 011 nor spec 013 (which explicitly scoped its own schema-role dispatch to rendering only) ever closed it.

- [X] T083 [P] Thread `core.Index` into `ParsePatch`/`ParseNode`/`parsePatchBody`/`walkNodeBody` in `internal/core/markdown.go`, mirroring `RenderNode(n Node, index Index)`/`RenderPatch(p Patch, index Index)`'s existing signature shape; update every call site (`internal/app/graph/service.Apply`'s `readPatch`, and any other `ParsePatch`/`ParseNode` caller) to pass the `core.Index` already in scope there (FR-019)
- [X] T084 [P] Implement inverse label resolution: build a `label → predicate id` map from `index.Predicates` by computing each predicate's resolved display label (mirroring `labelFor`'s existing predicate→label logic) and inverting it; used by `walkNodeBody`/`blockTitle` to resolve a `**Label**`-prefixed block's predicate identity (FR-019, depends on T083)
- [X] T085 (reopened — BUG-003, reclosed by T092) Dispatch a resolved `**Label**` block's content by its predicate's declared `Role` in `walkNodeBody`: `edge`/`link` role → parse bullets as edges (existing wikilink-target extraction unchanged, role no longer inferred from shape); `text` role → aggregate the block's content (paragraph or bulleted lines) into `Texts[predicateID]` (depends on T084; FR-019). Correctly implemented for its original scope (content is captured, not dropped); reopened because that scope did not extend to preserving the captured content's own list-item markup — closed by T092 (FR-020).
- [X] T086 (reopened — BUG-004, reclosed by T101/T102) [P] A bare list with no preceding label keeps today's per-line classification (explicit `predicate::` tag or wikilink shape); a non-matching line now falls through to text instead of being dropped, in `collectListLinks`/`walkNodeBody` (`internal/core/markdown.go`) (FR-019, Edge Cases). Correctly implemented per-line classification for a single bare list; reopened because (a) it never considered a *second* untitled plain list appearing later in the body, which permanently derails the headed-blocks loop's title+list pairing for everything after it, and (b) a non-matching line falling through to text lost its bulleted shape (joined into blank-line-separated paragraphs) instead of being reconstructed as a list — closed by T101/T102 (FR-023).
- [X] T087 (reopened — BUG-003, reclosed by T094-T096) Extend `RegisterPredicate` in `internal/app/schema/service` to default a newly auto-discovered predicate to `role: text, merge: append` when the observed content isn't wikilink-shaped, instead of always `role: edge, merge: union` — closes spec 011 research.md's own flagged "future feature" auto-discovery gap (depends on T085). Correctly implemented the `text`-vs-`edge` role default; reopened because it did not populate a `label` attribute, and had no signal to distinguish "edge content under its own `**Label**` block" (should be `role: link`) from "edge content in the bare list" (`role: edge`) — closed by T094/T095/T096 (FR-021/FR-022).
- [X] T088 [P] Delete `textPredicateFor`'s dead `case "hypothesis":`/`case "aporia":`/`case "thought":` branches in `internal/core/markdown.go` (unreachable since spec 019 made every real `@type` CamelCase, already silently falling through to `default: "text"`); do NOT touch `Source`/`Entity`/`Resource`'s own cases (`abstract`/`definition`/`relevance`) — `CoreTypeDefs.Required` and `arc lint`'s `checkTypeRequires` depend on those exact predicate names for the three built-in content types
- [X] T089 [P] Add regression tests: a labeled `text`-role block (plain prose or non-wikilink list) round-trips through `arc apply` intact and merges per its predicate's declared behavior on re-apply; a labeled `edge`/`link`-role block is unaffected; an unregistered label with non-wikilink content auto-registers as `role: text, merge: append` and its content survives; a mixed list (some wikilink lines, some not) preserves both the extracted edges and the remaining text (`internal/core/markdown_test.go`, `internal/app/graph/service/apply_test.go`, `cmd/arc/graph/apply_test.go`)
- [X] T090 Run `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; confirm all green
- [X] T091 Manually re-verify against the exact patch that triggered this report (`fogfish/bots/dmitry-2026-article.md` applied to a real graph) — confirm the `Hypothesis` nodes' `**Assumptions**`/`**References**` content now survives

**Bugfix**: 2026-07-21 — BUG-003 Updated from bugfix patch: reopened T085/T087 with correction notes (not false completions — each correctly implemented its original, narrower scope; see notes above) and added Phase 7 (T092-T100) to close the formatting/round-trip gap BUG-002 left open: a `**Label**`-resolved block's content survived (FR-019) but its list-item markup, recoverable label, and per-block grouping did not (FR-020/FR-021/FR-022).

---

**Checkpoint**: BUG-002 fixed — a `**Label**`-headed body block's predicate identity and role are resolved against the schema at parse time, closing the loop spec 010's own plan.md named and deferred, and spec 011/013 each subsequently deferred again without ever picking up.

---

## Phase 7: Bugfix BUG-003 — Content Survives But Formatting Is Lost (Wikilinks, List Shape, Labels, Grouping)

**Purpose**: Addresses [bugs/BUG-003.md](bugs/BUG-003.md): BUG-002's fix stopped `arc apply` from dropping a `**Label**`-headed block's content, but the surviving content comes back reformatted — a `text`-role list's wikilink brackets and list markers are stripped, the block's own `**Label**` never reappears on write, and distinct labeled edge blocks (e.g. `**Assumes**`, `**Derived From**`, `**Related Aporias**`) collapse into one undifferentiated flat list. Root causes: (1) text-role list-item content is routed through `extractInlineLinks`/`reconstructHRefs`, a heuristic built for free-flowing paragraph prose, not discrete list items; (2) `RegisterPredicate` never populates a `label` attribute for an auto-discovered predicate; (3) `renderNodeBody`'s "other Texts keys" loop never emits a heading for any `role: text` predicate; (4) auto-discovery's observed-role signal only distinguishes `text` vs `edge`, with no way to know an edge occurred under its own `**Label**` block (which should register `role: link` to preserve grouping).

- [X] T092 [P] Preserve list-item source formatting for `text`-role content verbatim (literal wikilink brackets, inline `predicate::` tags) instead of routing list-item lines through `extractInlineLinks`/`reconstructHRefs`, in `listItemLines`/`walkNodeBody` (`internal/core/markdown.go`) (FR-020)
- [X] T093 [P] Fix, or confirm no longer reachable after T092, `reconstructHRefs`'s `followedByBoundary` gap for the `[[Target]]<suffix>` case (a wikilink immediately followed by a non-whitespace, non-punctuation character) for any content still going through that heuristic (`internal/core/markdown.go`) (FR-020, depends on T092)
- [X] T094 Introduce a parse-time carrier for an auto-discovered predicate's original literal `**Label**` text — populated by `walkNodeBody` exactly when a block's label does not resolve against the schema, for both the text-role and edge-role fallback paths — threaded through to `Apply`'s auto-registration call site (`internal/core/markdown.go`, `internal/core/ast.go`, `internal/app/graph/service/apply.go`) (depends on T083-T087; prerequisite for T095/T096)
- [X] T095 Populate the `label` attribute on an auto-registered `role: text` predicate from T094's carried literal label text, in `RegisterPredicate` (`internal/app/schema/service/schema.go`, `internal/app/graph/port/schema.go`, `internal/app/schema/component.go`) (FR-021, depends on T094)
- [X] T096 Give auto-discovery a third observed-role signal: edge content carried with a T094 label (occurred under its own `**Label**` block) → register `role: link` with that `label`; edge content with no carried label (bare/ungrouped list) → register `role: edge` exactly as today, in `distinctPredicates`/`RegisterPredicate` (`internal/app/graph/service/apply.go`, `internal/app/schema/service/schema.go`) (FR-022, depends on T094)
- [X] T097 Render a heading/bold-label for a `role: text` predicate's `Texts` entry on write, mirroring `renderEdges`'s existing `role: link` heading logic (`## Label` for `RenderNode`, `**Label**` for `RenderPatch`) in `renderNodeBody`'s "other Texts keys" loop (`internal/core/markdown.go`) (FR-021, depends on T095)
- [X] T098 [P] Add regression tests asserting *shape* equivalence, not just content presence: a text-role list item's wikilink brackets/list markers survive a round trip (including the `[[Target]]<suffix>` case); an auto-registered text-role predicate's heading reappears on write; an auto-registered edge-role predicate discovered under its own label renders as its own distinct grouped block, not merged with another label's occurrences (`internal/core/markdown_test.go`, `internal/app/graph/service/apply_test.go`, `cmd/arc/graph/apply_test.go`) (depends on T092-T097)
- [X] T099 Run `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; confirm all green
- [X] T100 Manually re-verify against the exact BUG-003 reproduction (a Hypothesis node with `**Assumptions**`, `**Related Aporias**`, `**References**`, `**Assumes**`, `**Derived From**` blocks) applied to a real graph — confirm wikilink markup, list shape, block labels, and per-block edge grouping all survive a round trip, not just the words

**Checkpoint**: BUG-003 fixed — a `**Label**`-resolved block's content now survives a write with its shape intact: literal wikilink brackets and list markers preserved verbatim for text-role content, the block's own label recoverable (via a new transient `Node.Labels` parse-time carrier and the auto-registered `label` attribute it feeds), and a labeled edge block registers `role: link` instead of the flat `role: edge` default so distinct blocks (`**Assumes**`/`**Derived From**`/`**Related Aporias**`) stay separate groups rather than collapsing together.

**Bugfix**: 2026-07-21 — BUG-004 Updated from bugfix patch: reopened T086 with a correction note (correctly implemented its original, narrower scope — a single bare list — see note above) and added Phase 8 (T101-T105) to fix the underlying single-forward-pass constraint in `walkNodeBody`'s title+list pairing: a body containing more than one untitled plain list permanently stops pairing before it reaches later, well-formed `**Label**` blocks, splitting their titles from their own list content (FR-023).

---

## Phase 8: Bugfix BUG-004 — Untitled-List Pairing Loop Isn't Resumable, Stranding Later `**Label**` Blocks

**Purpose**: Addresses [bugs/BUG-004.md](bugs/BUG-004.md): `walkNodeBody`'s title+list pairing loop (BUG-002/FR-019) is a single, non-resumable forward pass — the moment it meets a list not immediately preceded by a title, it `break`s for good, so a body with more than one untitled, non-wikilink-shaped plain list strands every later `**Label**` block's title from its own list content (the label reappears as orphaned trailing prose while its list is separately captured as edges, producing duplicated/split output). Independently, a plain list's items lose their bulleted shape whenever demoted to prose in either the leading bare-list slot or the trailing fallback loop, neither of which reuses `renderTextListLines`'s existing bulleted reconstruction. Root cause and fix approach: replace the single-forward-pass structure with a two-pass parse — pass 1 classifies every top-level body child (prose / list / title) and decides title+list pairing for the *entire* body, not just up to the first unpaired list; pass 2 converts the classified sequence into `Texts`/`Edges`/`labels`, reusing the existing role/label resolution logic unchanged.

- [X] T101 [P] Restructure `walkNodeBody`'s block recognition into two passes: pass 1 (new `classifyNodeBody`) walks all of `children` once, classifying each top-level node as prose, a list, or a title+list pair (`blockTitle`), deciding pairing for *every* list in the body — a title immediately followed by a list pairs with it regardless of how many untitled lists appeared earlier; pass 2 (`walkNodeBody` itself) converts the classified sequence into `Texts`/`Edges`/`labels`, reusing `resolveLabelPredicate`/`classifyListItems`/`collectListLinks`/`listItemLines` unchanged for the per-block decisions those already make correctly. The leading/trailing boundary itself is kept exactly as before (contiguous leading prose + at most one immediately-following list forms the leading slot; everything else that isn't a titled pair is trailing) — generalizing only pairing resumability, not the two-slot physical layout `renderNodeBody`/`revert_internal_test.go` already depend on (`internal/core/markdown.go`) (FR-023)
- [X] T102 [P] Preserve a plain (non-wikilink) list's bulleted shape wherever its items land in `Texts` — reused via a new `renderBareTextListLines` (not `renderTextListLines` itself) for the leading bare-list slot's text lines and any other untitled list's text lines. Uses `"*"` rather than `"-"`: an untitled list is never preceded by a heading, so a `"-"`-marked reconstruction could silently re-merge with an adjacent bare/headed Edges list (also `"-"`) into one loose list on a later parse — discovered via T103's idempotency test — losing non-wikilink items that a resolved edge/link-role block's `collectListLinks` silently drops. `renderTextListLines` itself (the named text-role-block path, T092) is untouched — always preceded by its own heading, so no merge risk — and keeps `"-"`, matching all its existing test fixtures (`internal/core/markdown.go`) (FR-023, depends on T101)
- [X] T103 [P] Add regression tests: (a) a body with two untitled plain lists (short + long) before two titled blocks — both lists' bulleted shape survives; (b) a `**Label**` block occurring after both untitled lists still resolves against the schema and stays paired with its own list, exactly as if it were the first block in the body; (c) round-trip and idempotent round-trip stability (FR-014/FR-015) for the same shape, with predicates pre-registered `role: link` (mirroring the schema state after `apply.go` auto-registration in the real pipeline) (`internal/core/markdown_test.go`: `TestParsePatchMultipleUntitledListsPreserveBulletShapeAndTitlePairing`, `TestParsePatchLabeledBlockAfterMultipleUntitledListsStillResolvesAgainstSchema`, `TestRoundTripMultipleUntitledListsIdempotent`). `apply_test.go`/`cmd/arc/graph/apply_test.go` coverage folded into T105's manual E2E verification instead of duplicated, since the parse/render shape is already exhaustively covered at the `internal/core` level (depends on T101, T102)
- [X] T104 Ran `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; all green (depends on T101-T103)
- [X] T105 Manually re-verified against the exact BUG-004 reproduction (the `kolesnikov-2026-graph-metadata` Source node, with its two untitled plain lists followed by `**Mentions**`/`**Cites**`/`**Thoughts**` blocks) applied to a real graph via the built `arc` binary — both plain lists keep their bulleted (`*`) shape, each `**Label**` block renders exactly once (no orphaned duplicate labels), and a re-apply of the same patch is a byte-identical no-op (`git commit`: "nothing to commit, working tree clean") (depends on T104)

**Checkpoint**: BUG-004 fixed — `walkNodeBody`'s title+list pairing is no longer a single non-resumable forward pass; every list in a body gets a chance to pair with an immediately preceding title regardless of how many untitled lists came before it, and a plain list's bulleted shape survives wherever it lands in `Texts`.

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — Glossary-only change (T004), no Directory Structure change
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — Node (reshaped), Predicate, Text Predicate / Prose Field (T004)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: zero changes (Principle IX) — confirmed by T006

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — N/A, no new pattern; this feature reshapes the existing `internal/core.Node` in place
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — unchanged; no `cmd/` business logic introduced
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — T019/T029-T031/T035/T037 before T013-T018/T020-T028/T032-T034/T036
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration (T008)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A, no output-formatting change beyond error message text
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A (T012)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — N/A, no command changed; the old-format rejection error (T025) uses `faults.Type`/`faults.SafeN`-style human-readable guidance, not a raw parse error
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII) — T009-T011 turned green by T066-T075; changes to existing E2E tests are fixture-shape rewrites, not new test logic, per plan.md's Testing note
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — all 14 acceptance scenarios across US1-US3
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — **Yes, flagged**: `arc subgraph --json`'s `Node` schema breaks non-additively (plan.md Complexity Tracking, contracts/subgraph-json-contract.md, research.md D8); accepted pre-1.0 (`0.0.x` release train), not hidden

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion. Internally sequential in three layers: (1) `internal/core` type reshape (T013-T019) → (2) `internal/core` parser/merge/filter (T020-T037, each sub-area depends only on the type reshape, not on each other, so parser/merge/filter can proceed in parallel) → (3) downstream mechanical ripple (T038-T065, depends on the relevant `internal/core` pieces per task, otherwise parallel across packages)
- **User Stories (Phase 3-5)**: All depend on Phase 2.5's final checkpoint (T065, whole-module build green) — none can even compile before that
  - US1/US2/US3 are independently testable once Phase 2.5 completes; none depends on another story's own tasks, only on the shared foundation
- **Additional Polish**: Depends on all three user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases
- **BUG-001 fix (T079-T082)**: Sequential — T079 (design-artifact gate check) → T080 (`parsePatchBody` fix) → T081/T082 (parallel, both depend only on T080). Independent of Phase N and Additional Polish; can proceed as soon as Phase 2.5's `internal/core` checkpoint (T065-adjacent, `markdown_test.go` passing) is reached, since it only touches `internal/core/markdown.go`/`markdown_test.go` and one `cmd/arc/graph/apply_test.go` fixture.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 (T065) — no dependency on US2/US3
- **User Story 2 (P2)**: Can start after Phase 2.5 (T065) — independently testable via T010's E2E tests regardless of US1/US3's own confirmation tasks
- **User Story 3 (P3)**: Can start after Phase 2.5 (T065) — independently testable via T011's E2E tests regardless of US1/US2

### Within Each User Story

- E2E tests (Phase 2d) already written and failing-to-compile-or-red before Phase 2.5 completes
- Phase 2.5 (`internal/core` + downstream ripple) before any Phase 3+ confirmation task
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Phase 2a-2e subsections marked [P] can run in parallel with each other
- T009/T010/T011 (E2E test authoring for US1/US2/US3) can run in parallel, coordinating on shared test files (`apply_test.go` is touched by all three)
- T013 and T014 (independent `ast.go` edits) can run in parallel; T018 depends on T014
- Within Phase 2.5's parser/merge/filter layer: `markdown.go` tasks (T020-T031), `merge.go` tasks (T032-T035), and `filter.go` tasks (T036-T037) touch different files and can proceed in parallel once T013-T019 land
- Within the downstream ripple: every `[P]`-marked task touches a distinct file and can run in parallel; non-`[P]` tasks in the same package (e.g. T045 before T046) have an intra-file or fixture dependency
- Once Phase 2.5 (T065) completes, US1/US2/US3 confirmation work can proceed in parallel (if staffed)

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Independent internal/core type reshape:
Task: "Add Predicate struct to internal/core/ast.go"
Task: "Replace Node.Kind Kind with Node.Type string in internal/core/ast.go"

# Once the type reshape lands, independent internal/core subsystems:
Task: "Rewrite deriveNodeID/parse manifest logic in internal/core/markdown.go"
Task: "Rewrite mergeCore's Texts/Attrs/Edges handling in internal/core/merge.go"
Task: "Update node.Kind/node.Attrs access in internal/core/filter.go"

# Independent downstream compile-fix files, once internal/core is done:
Task: "Update internal/app/schema/component.go: core.Kind -> string"
Task: "Update internal/app/graph/port/schema.go: core.Kind -> string"
Task: "Update internal/app/lint/service/rules_frontmatter.go: node.Kind -> node.Type"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure (CRITICAL — the whole module must compile)
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenario 1)
7. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready (whole module builds, US1/US2/US3 E2E tests can all run)
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions together
2. Foundational Infrastructure is inherently a shared, largely sequential effort (the whole module must compile as one unit) — best staffed as: Developer A owns `internal/core`'s type reshape + parser (T013-T031), Developer B owns `internal/core`'s merge + filter (T032-T037) once the type reshape lands, Developer C starts the downstream ripple's independent `[P]` files (T038-T044, T051-T052, T056) as soon as each relevant `internal/core` piece is done
3. Once Phase 2.5 (T065) completes:
   - Developer A: User Story 1 confirmation
   - Developer B: User Story 2 confirmation
   - Developer C: User Story 3 confirmation
4. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable once Phase 2.5 completes
- E2E tests (Phase 2d) MUST already be written (compiling-but-red, or failing-to-compile until Phase 2.5 lands) before their story's confirmation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections MUST be retained verbatim across features (constitution Governance > Task List Requirements) — only task descriptions are adapted
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- One scope note carried from plan.md applies throughout Phase 2.5/3: the `arc subgraph --json` break is intentional (TN15). ~~`Texts`' open-key representation is not yet matched by open-key *parsing* (a node still gets at most two `Texts` keys from this feature, research.md D4's known limitation) — do not expand `walkNodeBody` to recognize arbitrary named prose headings while implementing T024; that is explicitly deferred, not an oversight to fix inline~~ Closed by Phase 6 (Bugfix BUG-002, 2026-07-20): `walkNodeBody` now resolves a `**Label**`-headed block's predicate identity against the schema index (T083-T086)
