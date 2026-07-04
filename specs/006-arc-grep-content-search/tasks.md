# Tasks: Search Graph Content by Pattern (`arc grep`)

**Input**: Design documents from `/specs/006-arc-grep-content-search/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`cli-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `cmd/arc/graph/` — existing Cobra wiring package for the `graph` domain; gains `grep.go` and its colocated E2E test
- `internal/pkg/grep/` — NEW, first occupant of the `internal/pkg` tier: reusable, dependency-free, `fs.FS`-based content-search library
- `internal/core/` — existing shared core domain; gains `filter.go`
- `internal/bios/` — existing shared kernel; `theme.go` gains one field
- `internal/app/config/` — existing use-case; `kernel/config.go` gains its first real field
- `internal/app/graph/{kernel,service,component.go}` — existing `graph` use-case; gains `Grep` alongside `Apply`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create the package skeleton: `internal/pkg/grep/` directory, per plan.md's Project Structure (all other touched packages already exist)
- [X] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged per plan.md Technical Context (`internal/pkg/grep` is stdlib-only, research.md D3)
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the new (still-empty) `internal/pkg/grep` skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add the domain terms from spec.md Key Entities / data-model.md — Filter, Match, Grep Run — to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle I obligation, plan.md Constitution Check row I)
- [X] T005 Verify no existing `internal/<domain>` package already defines a `Filter`/node-selection type before introducing `internal/core.Filter` (research.md D8 — none exists; `arc grep` is the first Filtering-section command to ship)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc grep`'s bare-verb grammar, single positional `<pattern>` argument, and the three new local `--kind`/`--tag`/`--attr` flags against contracts/cli-contract.md (research.md D14 — no new persistent/global flag is introduced)
- [X] T007 [P] Review contracts/cli-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T008 [P] Confirm no existing package already provides an `fs.FS`-based, parallel content-search capability before creating `internal/pkg/grep` (research.md D2 — none exists; this is the first `internal/pkg/<lib>` occupant in the codebase)
- [X] T009 Define `internal/pkg/grep`'s public contract (`Match`, `Options`, `Result`, `Search`) per data-model.md as the design gate before any implementation code is written — confirm the signature depends only on stdlib `io/fs` (`fs.FS`/`fs.ReadDirFS`), never `os.*` (constitution Principle VII, research.md D2)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T010 [P] [US1] Write E2E tests in `cmd/arc/graph/grep_test.go` for spec.md US1's 3 acceptance scenarios (every occurrence of a pattern is reported with `kind`/`id`/line across the whole graph with no filter; a pattern matching nothing produces no output and a non-zero exit; a node matching on more than one line reports each line separately) using the `sut()` helper — tests MUST compile and fail semantically (red phase)
- [X] T011 [P] [US2] Write E2E tests in `cmd/arc/graph/grep_test.go` for spec.md US2's 4 acceptance scenarios (`--kind` restricts to that kind; `--tag` restricts to that tag; combined `--kind`+`--attr` narrows further; a filter matching zero nodes produces no output and a non-zero exit) — red phase
- [X] T012 [P] [US3] Write E2E tests in `cmd/arc/graph/grep_test.go` for spec.md US3's 3 acceptance scenarios (output piped through a line-counting tool yields the exact match count with no extra lines; output piped through a field-extraction tool splits `kind`/`id`/`line`/text cleanly; a single matched line never spans more than one output line) — red phase
- [X] T013 [P] Write E2E tests in `cmd/arc/graph/grep_test.go` for the Edge Cases tied to guard/UX behavior: an invalid `<pattern>` regexp refuses with a clear error and no scan (FR-008), target not an initialized graph (FR-011), an unreadable/unparseable node file is excluded and does not abort the run (FR-012), and `--verbose` shows the full untruncated line while default mode may truncate a long match per quickstart.md Scenario 4 — red phase

> T010–T013 all target the same new file (`cmd/arc/graph/grep_test.go`) and are therefore sequential in practice despite each being scoped to one story (mirrors `specs/004-arc-lint/tasks.md`'s T010–T013 note).

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T014 Confirm the new `.arc/config.yml` fields (`grep.workers`, `grep.maxLineWidth`) follow the flag → env → project config → user config → system config precedence (they are project-config-only, no flag/env override introduced) and that no secret or credential material is involved (plan.md Constitution Check row XI, research.md D10)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `internal/core.Filter`, `internal/pkg/grep.Search`, the `kernel`/`config`/`bios` value-type extensions, and `service.Grep`'s enumeration+orchestration are genuinely foundational — every one of US1–US3 runs the same scan and reports through the same `kernel.GrepResult` shape, differing only in which flags/output behavior their own E2E scenarios exercise. This phase builds that shared foundation; Phase 3+ adds each story's specific behavior and hardening on top of it.

### `internal/core` — shared `Filter` type (research.md D8)

- [X] T015 [P] Implement `internal/core/filter.go`: `Filter{Kinds, Tags, Attrs, AttrPatterns}` and `Filter.Match(Node) bool` per data-model.md
- [X] T016 [P] Unit tests in `internal/core/filter_test.go`: `Kinds` OR semantics, `Tags` AND semantics (against `node.Attrs["tags"]`), `Attrs` exact-match AND (case-insensitive scalar, array membership), `AttrPatterns` regexp-match AND (scalar and array), zero-value `Filter{}` matches every node (depends on T015)

### `internal/pkg/grep` — reusable content-search library (research.md D2-D7)

- [X] T017 [P] Implement `internal/pkg/grep/grep.go`'s public shape: `Match`, `Options{Extension, Workers, Include}`, `Result{Matches, Unreadable}`, and the literal-vs-regex pattern classification (`regexp.QuoteMeta(pattern) == pattern`) per research.md D4
- [X] T018 Implement the bounded-pool parallel walker+scanner in `internal/pkg/grep/grep.go`: a channel-based semaphore sized `Options.Workers` (default 8) shared by directory-listing goroutines (one per subdirectory, via `fs.ReadDir`) and file-scanning goroutines (one per matched, `Include`-passing file), coordinated by one `sync.WaitGroup` (research.md D3, D7) (depends on T017)
- [X] T019 Implement buffered/pooled line scanning in `internal/pkg/grep/grep.go`: a `sync.Pool` of `*bufio.Reader`, `.Reset(f)` per file, `ReadBytes('\n')` line loop with 1-based line numbering, `.Reset(nil)` before returning to the pool, deferred `f.Close()` per file (research.md D5) (depends on T018)
- [X] T020 Implement `Search`'s result assembly in `internal/pkg/grep/grep.go`: per-file open/read failures collected into `Result.Unreadable` (scan continues), a hard `error` return only for an invalid `pattern` or a root-listing failure, and a final `sort.Slice` of `Matches` by `(Path, Line)` (research.md D6) (depends on T018, T019)
- [X] T021 [P] Unit tests in `internal/pkg/grep/grep_test.go` against `fstest.MapFS` (no real filesystem): literal vs. regex dispatch produce identical matches for a metacharacter-free pattern, a line matching more than once collapses to one `Match`, an unreadable file is recorded in `Unreadable` and the scan continues, an invalid pattern returns a hard error before any file is opened, `Matches` ordering is deterministic across repeated runs, and `-race` is clean with `Workers` > 1 (depends on T017, T018, T019, T020)

### `internal/app/graph/kernel` — value types

- [X] T022 [P] Implement `internal/app/graph/kernel/grep.go`: `Match{Kind, ID, Path, Line, Text, Start, End}` and `GrepResult{Root, Pattern, Matches, Unreadable}` per data-model.md

### `internal/app/config` — first real `Config` field (research.md D10)

- [X] T023 [P] Extend `internal/app/config/kernel/config.go`: add `Config.Grep GrepConfig{Workers, MaxLineWidth}` per data-model.md
- [X] T024 [P] Unit tests in `internal/app/config/service/config_test.go`: `Grep.Workers`/`Grep.MaxLineWidth` round-trip through `Load`/`Save`, and an absent/zero value loads as the zero `GrepConfig{}` (defaulting happens at the `cmd/` wiring layer, not here) (depends on T023)

### `internal/bios` — `Schema.Match` (research.md D11)

- [X] T025 [P] Extend `internal/bios/theme.go`: add `Schema.Match lipgloss.Style`, a no-op in `SCHEMA_PLAIN`, a distinct bold/colored style in `SCHEMA_COLOR`

### `internal/app/graph/service` — errors and enumeration

- [X] T026 [P] Add `ErrInvalidPattern`, `ErrInvalidAttrFlag` `faults.Safe1[string]` sentinel constants to `internal/app/graph/service/errors.go` (extending the existing file)
- [X] T027 Implement the node-enumeration pass in `internal/app/graph/service/grep.go`: recursive `fsys.Store.ReadDir` excluding `.arc/` and `_schema/` (mirrors `internal/app/lint/service.walkNodeFiles`), parsing each remaining `*.md` via `core.ParseNode`; builds a `path → core.Node` index for `kind`/`id` labeling and a `Filter`-membership set simultaneously; an unreadable or unparseable file is recorded and excluded from the scan (research.md D9) (depends on T022)
- [X] T028 Implement `internal/app/graph/service.Grep(ctx, mounter, filter core.Filter, pattern string, cfg kernel.GrepConfig, dir string) (kernel.GrepResult, error)`: guard `ErrNotAGraph` (`Store.Stat(".arc")`), run T027's enumeration, build `grep.Options{Extension: ".md", Workers: cfg.Workers-or-default, Include: filter-membership lookup}`, call `internal/pkg/grep.Search`, map each `grep.Match` into `kernel.Match` via T027's index, merge `Unreadable` lists (depends on T015, T017–T020, T023, T026, T027)
- [X] T029 [P] Unit tests in `internal/app/graph/service/grep_test.go` against a fake `fsys.Mounter`/`fsys.Store`: not-a-graph guard refuses before scanning, an empty `Filter{}` scans every node, a populated `Filter` excludes non-matching nodes from the scan entirely, an unreadable/unparseable node is excluded and reported in `Unreadable`, an invalid pattern returns `ErrInvalidPattern` (depends on T028)

### Wiring skeleton

- [X] T030 [P] Implement `internal/app/graph/component.go`'s `Grep(ctx, mounter, filter, pattern, cfg, dir) (kernel.GrepResult, error)` delegator, alongside the existing `Apply` (depends on T028)
- [X] T031 [P] Scaffold `cmd/arc/graph/grep.go`: `NewGrepCmd() *cobra.Command` with `Args: cobra.ExactArgs(1)`, a local `optsFilter{kind, tag, attr []string}` options struct (DS-02, research.md D14, empty `apply`/`build` stubs), and `RunE` returning a "not implemented" placeholder error (empty-but-compiling scaffold)

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Find every occurrence of a term across the whole graph (Priority: P1) 🎯 MVP

**Goal**: With no filter, every node file in the graph is scanned; every matching line is reported exactly once, labeled with its owning node's `kind`/`id` and its line number; a pattern matching nothing produces no output and a distinguishable non-zero exit.

**Independent Test**: Run `arc grep <pattern>` against a graph with a known term appearing on known lines in known nodes and confirm every occurrence is reported correctly, per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T010, T013) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T032 [US1] Implement `cmd/arc/graph/grep.go`'s real `RunE`: `filepath.Abs(".")`, `fsys.Local{}.Mount`, `internal/app/config.Load`, resolve `GrepConfig` defaults (`Workers <= 0` → `8`, `MaxLineWidth <= 0` → `80`), build `core.Filter{}` (empty when no filter flags given), call `appgraph.Grep` (depends on T030, T031)
- [X] T033 [US1] Implement the highlight/line-fitting transform in `cmd/arc/graph/grep.go`: wrap `Text[Start:End]` in `SCHEMA.Match.Render(...)` only when `bios.SCHEMA == bios.SCHEMA_COLOR`; when the line exceeds the resolved `MaxLineWidth`, ellipsis-fit a window centered on `[Start:End)` (research.md D11) (depends on T025)
- [X] T034 [US1] Implement `humanGrepPrinter` (the `bios.Registry`'s `Human` renderer) in `cmd/arc/graph/grep.go`: one row per match, `<kind>  <id>  <line>  <text>`, applying T033's transform, no header/footer/summary line, per contracts/cli-contract.md
- [X] T035 [US1] Implement `verboseGrepPrinter` (the `Verbose` renderer) in `cmd/arc/graph/grep.go`: identical row format, T033's highlight applied but truncation disabled — the full line is always shown (research.md D11)
- [X] T036 [US1] Construct `bios.Registry[kernel.GrepResult]{Human: humanGrepPrinter{...}, Verbose: verboseGrepPrinter{...}}` per-invocation (it needs the resolved `MaxLineWidth`) and resolve/print via `bios.ResolveMode()` in `cmd/arc/graph/grep.go`'s `RunE` (depends on T032, T034, T035)
- [X] T037 [US1] Implement the DS-07 exit-code contract in `cmd/arc/graph/grep.go`: after the result is printed, return `bios.ErrSilent` when `GrepResult.Matches` is empty (research.md D12); a genuine refusal (invalid pattern, not a graph) returns a real error before anything is printed
- [X] T038 [US1] Populate `Short`/`Long`/`Example` help text for `arc grep` per contracts/cli-contract.md's DS-11 shape (constitution Principle XII) in `cmd/arc/graph/grep.go`
- [X] T039 [US1] Register `graph.NewGrepCmd()` into `cmd/arc/root.go`'s command tree (depends on T032)
- [X] T040 [P] [US1] Add unit tests in `internal/app/graph/service/grep_test.go` covering spec User Story 1's acceptance scenarios specifically: no filter scans every node, every match is labeled with the correct `kind`/`id`, and a node matching on multiple lines produces one `kernel.Match` per line, in line order (constitution Principle VI) (depends on T028)

**Checkpoint**: At this point, User Story 1's E2E tests (T010, T013) pass and `arc grep` is fully functional and independently testable for the unfiltered case

---

## Phase 4: User Story 2 - Narrow the search to a subset of nodes (Priority: P2)

**Goal**: `--kind`/`--tag`/`--attr` restrict the scan to exactly the nodes satisfying every given condition, using VISION.md's Filtering semantics (OR within `--kind`, AND across `--tag`/`--attr`/groups).

**Independent Test**: Run `arc grep --kind <kind> <pattern>` against a graph with matches both inside and outside that kind and confirm only the in-kind matches are reported, per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E test for this story was already written in Phase 2d (T011) and MUST currently be failing (red) until this phase's flag wiring lands.

- [X] T041 [US2] Implement `optsFilter.apply(cmd *cobra.Command)` in `cmd/arc/graph/grep.go`: register `--kind`, `--tag`, `--attr` as repeatable `StringArrayVar` local flags (DS-02) (depends on T031)
- [X] T042 [US2] Implement `optsFilter.build() (core.Filter, error)` in `cmd/arc/graph/grep.go`: parse each `--attr` value as `name=value` (→ `Filter.Attrs`) or `name~=pattern` (→ `Filter.AttrPatterns`), returning `ErrInvalidAttrFlag` for a value matching neither shape; assemble `core.Filter{Kinds, Tags, Attrs, AttrPatterns}` (depends on T026, T041)
- [X] T043 [US2] Wire `optsFilter.build()`'s result into `RunE`, replacing T032's empty `core.Filter{}` with the parsed filter before calling `appgraph.Grep` (depends on T032, T042)
- [X] T044 [P] [US2] Add unit tests in `internal/core/filter_test.go` covering combined `Kinds`+`Tags`+`Attrs`+`AttrPatterns` AND-across-groups composition and a filter matching zero nodes (constitution Principle VI) (depends on T015)
- [X] T045 [P] [US2] Add unit tests in `cmd/arc/graph/grep_test.go` (or a colocated `_test.go` for `optsFilter`) covering `--attr` parsing: a valid `name=value`, a valid `name~=pattern`, and a malformed value returning `ErrInvalidAttrFlag` (depends on T042)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Pipe search results into other command-line tools (Priority: P3)

**Goal**: Output is strictly one match per output line, whitespace-delimited, with no header/footer/summary mixed in, in a stable order — safe to pipe into `wc`, `cut`/`awk`, `sort`/`uniq`, or `xargs` without a parsing workaround.

**Independent Test**: Pipe `arc grep <pattern>`'s output through a line-counting tool and a field-extraction tool against a graph with a known number of matches, per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T012) and MUST currently be failing (red) until this phase verifies/hardens the exact output shape Phase 3 already produces.

- [X] T046 [US3] Verify `humanGrepPrinter`/`verboseGrepPrinter` (T034/T035) emit zero non-match lines to `stdout` — no summary/count line (unlike `arc lint`'s summary line; spec FR-007 forbids one here) — adjust if T012's E2E test reveals extra output (depends on T034, T035)
- [X] T047 [US3] Verify `grep.Search`'s `Result.Matches` ordering (T020) is stable and identical across repeated runs against the same graph, including when `Options.Workers` > 1 (depends on T020)
- [X] T048 [P] [US3] Add unit tests confirming `SCHEMA_PLAIN` output (simulating piped/non-TTY) is the full, untruncated, unstyled line in every case — i.e. T033's truncation/highlight transform never fires when `bios.SCHEMA == bios.SCHEMA_PLAIN` (research.md D11, spec FR-006/FR-007) (depends on T033)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently — feature complete

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T049 [P] Update `README.md`'s quick-start example to mention `arc grep` (constitution Principle XII)
- [X] T050 [P] Manually run all quickstart.md scenarios (1-4, plus the config and read-only verification sections) against the built binary and confirm expected output, highlighting, truncation, and exit codes
- [X] T051 [P] Add table-driven unit tests in `internal/pkg/grep/grep_test.go` covering `Options.Workers` configurability (1, 8, 32) and `Options.Extension` configurability (`.md` default vs. a custom extension) end-to-end against `fstest.MapFS`

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/pkg/grep` (first occupant of that tier), `internal/core.Filter`, and `internal/app/graph`'s new `Grep` member (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: `arc grep <pattern>`, `--kind`/`--tag`/`--attr`, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced beyond what ADR 001/002 already cover (Principle I) — none expected; `internal/pkg` is already documented in ADR 001's own domain-evolution model, confirm during review
- [X] TN05 Domain logic uses ports (interfaces) where needed; `cmd/arc/graph` wiring, `internal/pkg/grep`, and `internal/app/graph/service` remain separated; no port was declared where none is needed (research.md D13) (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 `internal/pkg/grep` contains zero `os.*` filesystem calls — verified by inspection/grep of the package, confirming it depends only on stdlib `io/fs` (Principle VII, research.md D2)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for the new `Schema.Match` style (Principle X)
- [X] TN11 Configuration precedence respected for the new `grep.workers`/`grep.maxLineWidth` fields; no secrets logged or involved (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc grep` (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md US1–US3 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/graph/grep_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `arc grep` is a new command with a new, additive `--json` `GrepResult` schema; `kernel.ApplyResult`'s existing `--json` contract is untouched; no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2.5; User Story 1 is the deepest since it implements the full command surface and rendering — User Stories 2 and 3 extend the same files and therefore depend on Phase 3's tasks as well as Phase 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; implements `cmd/arc/graph/grep.go`'s full render/exit-code surface that US2/US3 extend
- **User Story 2 (P2)**: Can start after Phase 2.5, but its flag-wiring tasks (T041-T043) attach to the `RunE` US1 builds (T032) — sequenced after US1 in practice, though its E2E test (T011) is independent and was written in Phase 2d
- **User Story 3 (P3)**: Its tasks (T046-T048) verify/harden output shape US1 already produces — sequenced after US1, though its E2E test (T012) is independent

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain/library foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other
- Within Phase 2.5: `internal/core/filter.go` (T015-T016), `internal/pkg/grep` (T017-T021), `internal/app/graph/kernel` (T022), `internal/app/config` (T023-T024), and `internal/bios` (T025) have no cross-dependencies and can proceed in parallel; `internal/app/graph/service/grep.go` (T027-T029) depends on several of these landing first
- Once Phase 3 lands, User Stories 2 and 3 can proceed in parallel with each other

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Implement internal/core/filter.go (Filter, Filter.Match)"
Task: "Implement internal/pkg/grep/grep.go's public shape and classification"
Task: "Implement internal/app/graph/kernel/grep.go (Match, GrepResult)"
Task: "Extend internal/app/config/kernel/config.go (Config.Grep)"
Task: "Extend internal/bios/theme.go (Schema.Match)"
```

## Parallel Example: Phase 3 User Story 1

```bash
# Once T030/T031 (component delegator + cmd scaffold) exist, launch together:
Task: "Implement RunE mount/config/Grep-call wiring in cmd/arc/graph/grep.go"
Task: "Implement the highlight/line-fitting transform in cmd/arc/graph/grep.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Run quickstart.md Scenario 1 against the built binary
7. Deploy/demo if ready — `arc grep` already scans and reports every occurrence with color/truncation at this point, missing only `--kind`/`--tag`/`--attr` narrowing (US2) and US3's piping-specific hardening

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
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
- No Phase 0 (Pre-implementation Refactoring) is included — this feature only adds new files/fields to existing packages; nothing existing is renamed, restructured, or split
- User Stories 2 and 3 are not fully file-independent from User Story 1 here (they extend `cmd/arc/graph/grep.go` US1 creates) — this reflects that all three stories exercise one shared `Grep` use-case and one shared renderer, not three separate features; each remains independently *testable* via its own E2E test written in Phase 2d
- The double-read tradeoff (research.md D9, plan.md Complexity Tracking) is accepted as-is — no task above attempts to eliminate it, since doing so would require teaching `internal/pkg/grep` about `core.Node`/YAML, which research.md D2/D7 deliberately reject
