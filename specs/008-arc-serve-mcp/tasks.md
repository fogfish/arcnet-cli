# Tasks: MCP Server (`arc serve`)

**Input**: Design documents from `/specs/008-arc-serve-mcp/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`mcp-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor). Per plan.md's Principle VIII adaptation (research.md D7), this feature's E2E tests exercise the real, registered MCP tool handlers via `mcp.NewInMemoryTransports()` + a real `mcp.Client`, not a synchronous `sut()` return, since `arc serve`'s `RunE` is long-running.

**Organization**: Tasks are grouped by user story (US1-US4, priorities P1-P4 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, US3, or US4 — maps to spec.md's four user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `cmd/arc/graph/serve.go` — new Cobra wiring file for `arc serve`, alongside existing `apply.go`/`grep.go`/`subgraph.go`, plus its colocated `serve_test.go`
- `internal/app/graph/service/node.go` — new file; `NodeGet`/`EnsureGraph`, reusing `subgraph.go`'s existing `enumerateNodes`/`guardIsGraph`
- `internal/app/graph/component.go` — existing `graph` use-case primary port; gains `NodeGet`/`EnsureGraph` alongside `Apply`/`Grep`/`Subgraph`
- `adrs/003-mcp-server-adapter.md` — new ADR, Status: Accepted (plan.md Constitution Check, Principle I obligation)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Confirm no new package tier is required beyond `internal/app/graph/service/node.go` and `cmd/arc/graph/serve.go` — `internal/core`, `internal/app/graph/{kernel,service,component.go}`, `cmd/arc/graph`, `internal/app/config` all already exist (plan.md Project Structure)
- [X] T002 Add `github.com/modelcontextprotocol/go-sdk` (`v1.6.1`) to `go.mod`/`go.sum` via `go get github.com/modelcontextprotocol/go-sdk@v1.6.1` (research.md D1) — this feature's one new third-party dependency
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the current tree before starting (baseline)

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add the new glossary terms from data-model.md/contracts/mcp-contract.md — **MCP Tool**, **Transport** (stdio default / HTTP-SSE via `--http`), **Bind Address** (loopback-by-default rule, spec Clarifications) — to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle I obligation, plan.md Constitution Check row I)
- [X] T005 Verify `core.Node`, `core.Patch`, `core.Filter`, `kernel.GrepResult`/`Match`, `kernel.SubgraphResult` are reused as-is (data-model.md "Reused, unchanged domain types") — confirm no new domain type is introduced beyond the small, private `mcpFilter`/`nodeGetArgs`/`nodeGrepArgs`/`subgraphGetArgs` presentation shapes confined to `cmd/arc/graph/serve.go`

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc serve`'s bare-verb grammar and single local `--http <addr>` string flag (default `""`) against contracts/mcp-contract.md
- [X] T007 [P] Review contracts/mcp-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's 19 functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 [P] Confirm no existing adapter in this codebase covers MCP server functionality (Principle VII "check whether an adapter already exists" gate) — this is the first MCP integration
- [X] T009 **Author `adrs/003-mcp-server-adapter.md` (Status: Accepted)** per plan.md Constitution Check row I: records MCP as a second primary-adapter family, the rule that an MCP tool handler is a thin wrapper calling the same `internal/app/<domain>.Component` root every Cobra command calls, the transport choice (`github.com/modelcontextprotocol/go-sdk`, stdio default + Streamable HTTP/SSE via `--http`), and the loopback-by-default bind-address rule — **blocks all user story implementation below**
- [X] T010 Confirm `mcp.Server`'s `Transport` (`StdioTransport`, `NewStreamableHTTPHandler`) is consumed directly in `cmd/arc/graph/serve.go` with no project-private port wrapping it (ADR 001 port isolation rule 2, research.md D1/D7 — ADR 003 from T009 formalizes this)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T011 [P] [US1] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md US1's 3 acceptance scenarios (`node_get` returns the full node object matching on-disk content; an unknown id returns a clear tool error with no node object; a freshly-connected client can discover and invoke `node_get` immediately) using `mcp.NewInMemoryTransports()` + a real `mcp.Client` calling `session.CallTool` against the server built by a `RunE` invoked in a goroutine — tests MUST compile and fail semantically (red phase)
- [X] T012 [P] [US2] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md US2's 4 acceptance scenarios (`node_grep` returns one row per matching line with no filter; a `filter` object narrows the matched nodes; a non-matching pattern returns an empty table, not an error; a syntactically invalid pattern returns a clear tool error) — red phase
- [X] T013 [P] [US3] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md US3's 4 acceptance scenarios (`subgraph_get` with default depth returns the seed + direct neighbors as complete node objects; an explicit `depth` widens/narrows the set; a multi-path-reachable node appears exactly once; an unknown seed id returns a clear tool error) — red phase
- [X] T014 [P] [US4] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md US4's 3 acceptance scenarios, using `httptest.NewServer` wrapping the real, `RunE`-registered `mcp.StreamableHTTPHandler` (a client connecting over SSE can invoke all three tools with results identical to the in-memory/stdio path; omitting `--http` opens no network port; an invalid/in-use `--http` address refuses to start) — red phase
- [X] T015 [P] Write E2E tests in `cmd/arc/graph/serve_test.go` for the Edge Cases tied to guard/startup behavior via ordinary `sut()`/`cmd.RunE(cmd, args)` calls: the target not being an initialized graph refuses immediately (FR-004), and a syntactically invalid `--http` address refuses immediately (FR-005) — red phase

> T011-T015 all target the same new file (`cmd/arc/graph/serve_test.go`) and are therefore sequential in practice despite each being scoped to one story (mirrors `specs/007-arc-subgraph/tasks.md`'s T009-T012 note).

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T016 Confirm `--http`'s address is a CLI flag only (no new `.arc/config.yml` field), `.arc/config.yml`'s existing `Grep`/`Subgraph` sections are reused verbatim and loaded once at server startup, and no secret or credential material is introduced anywhere in this feature (plan.md Constitution Check row XI)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `NodeGet`/`EnsureGraph`, the MCP server bootstrap (address parsing, filter mapping, call logging, command scaffold), and tool registration are genuinely foundational — every one of US1-US4 registers into and runs through the same `mcp.Server`/`RunE`. This phase builds that shared foundation; Phase 3+ adds each story's specific tool/behavior on top of it.

### `internal/app/graph/service` — `NodeGet`/`EnsureGraph` (research.md D3, D6)

- [X] T017 [P] Implement `internal/app/graph/service/node.go`'s `NodeGet(ctx context.Context, mounter fsys.Mounter, dir, id string) (core.Node, error)`: mount, `guardIsGraph`, `enumerateNodes` (reused verbatim from `subgraph.go`), index lookup, `ErrSeedNotFound.With(errNoCause, id)` on a miss
- [X] T018 [P] Implement `internal/app/graph/service/node.go`'s `EnsureGraph(ctx context.Context, mounter fsys.Mounter, dir string) error`: mount + `guardIsGraph` only — the preflight `arc serve`'s `RunE` calls before starting any transport (spec FR-004)
- [X] T019 [P] Add `ErrHTTPAddr` and `ErrInvalidFilterPattern` `faults.Safe1[string]` sentinel constants to `internal/app/graph/service/errors.go` (extending the existing file)
- [X] T020 [P] Unit tests in `internal/app/graph/service/node_test.go` against a fake `fsys.Mounter`/`fsys.Store` (mirrors `subgraph_test.go`'s existing shape): `NodeGet` returns the matching node's full content, an unknown id returns `ErrSeedNotFound`, a not-yet-initialized graph returns `ErrNotAGraph` before any lookup; `EnsureGraph` returns `nil` for a valid graph and `ErrNotAGraph` otherwise (depends on T017, T018)
- [X] T021 [P] Implement `internal/app/graph/component.go`'s `NodeGet(ctx, mounter, dir, id) (core.Node, error)` and `EnsureGraph(ctx, mounter, dir) error` delegators, alongside the existing `Apply`/`Grep`/`Subgraph` (depends on T017, T018)

### `cmd/arc/graph/serve.go` — command scaffold and shared wiring helpers

- [X] T022 Scaffold `cmd/arc/graph/serve.go`: `NewServeCmd() *cobra.Command` with a local `--http` string flag (default `""`), and `RunE` returning a "not implemented" placeholder error (empty-but-compiling scaffold)
- [X] T023 [P] Implement `resolveHTTPAddr(addr string) (string, error)` in `serve.go` using `net.SplitHostPort`: a bare port or `:port` (no host) resolves to `"127.0.0.1:<port>"`; an explicit host is used as-is; a syntactically invalid address returns `ErrHTTPAddr` (research.md D5, spec FR-003)
- [X] T024 [P] Implement `mcpFilter` struct (`Kind []string; Tags []string; Attrs map[string]string; AttrPatterns map[string]string`) and its `toCoreFilter() (core.Filter, error)` conversion in `serve.go` — `AttrPatterns` values compiled via `regexp.Compile`, returning `ErrInvalidFilterPattern` on failure (research.md D4, data-model.md)
- [X] T025 [P] Implement `logCall(tool, args string, err error)` in `serve.go`: one `fmt.Fprintf(os.Stderr, ...)` line per call — tool name, key arguments, `ok` or `error: <message>` (research.md D9, spec FR-019)
- [X] T026 Register `graph.NewServeCmd()` into `cmd/arc/root.go`'s command tree (depends on T022)

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Fetch a node's full context by id (Priority: P1) 🎯 MVP

**Goal**: `node_get(id)` returns a node's complete content — attrs, text, edges, links — as markdown, matching the on-disk file exactly.

**Independent Test**: Connect an MCP client over the default stdio transport, call `node_get` with a known id, and confirm the returned markdown matches that node's content, per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T011, T015) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T027 [US1] Implement `nodeGetArgs{ID string}` and the `node_get` tool handler function in `serve.go`: call `appgraph.NodeGet(ctx, fsys.Local{}, dir, args.ID)`, on success return one `mcp.TextContent{Text: string(core.RenderNode(node))}`, on error return the error directly (the SDK auto-populates `IsError`/content text per `AddTool`'s documented contract) — `logCall("node_get", ...)` on both paths (depends on T021, T025)
- [X] T028 [US1] Register `node_get` via `mcp.AddTool` in `RunE`'s server-build step, with `Tool.Annotations = &mcp.ToolAnnotations{ReadOnlyHint: true}` and `Description` per contracts/mcp-contract.md (depends on T027)
- [X] T029 [US1] Implement `RunE`'s real startup sequence in `serve.go`: derive a `signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)`-canceled context, call `appgraph.EnsureGraph` as the preflight (spec FR-004), mount `fsys.Local{}`, load `.arc/config.yml` once via `appconfig.Load`, build the `mcp.Server`, register the tools built so far, then run `server.Run(ctx, &mcp.StdioTransport{})` when `--http` is unset (depends on T018, T022, T023, T026, T028)
- [X] T030 [US1] Populate `Short`/`Long`/`Example` help text for `arc serve` in `serve.go` per contracts/mcp-contract.md's DS-11 shape (constitution Principle XII)
- [X] T031 [P] [US1] Add a direct unit test in `cmd/arc/graph/serve_test.go` for the `node_get` handler function's error-to-`IsError`-result mapping (bypassing the transport), and for `logCall`'s output line shape (depends on T027, T025)

**Checkpoint**: At this point, User Story 1's E2E tests (T011) pass and `arc serve` is fully functional and independently testable for `node_get` over stdio

---

## Phase 4: User Story 2 - Search node content to find relevant nodes (Priority: P2)

**Goal**: `node_grep(pattern, filter?)` returns one markdown table row per matching content line, optionally narrowed by an MCP filter object.

**Independent Test**: Call `node_grep` with a pattern matching a known subset of nodes and confirm the returned table has exactly one row per matching line, per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E test for this story was already written in Phase 2d (T012) and MUST currently be failing (red) until this phase lands.

- [X] T032 [US2] Implement the markdown-table renderer for grep matches in `serve.go`: `renderMatchTable(matches []kernel.Match) string` — `| id | kind | line | snippet |` header, one row per match, header-only when `matches` is empty (research.md D2, contracts/mcp-contract.md)
- [X] T033 [US2] Implement `nodeGrepArgs{Pattern string; Filter *mcpFilter}` and the `node_grep` tool handler function in `serve.go`: convert `Filter` via T024's `toCoreFilter()` (nil/empty → `core.Filter{}`), call `appgraph.Grep(ctx, fsys.Local{}, filter, args.Pattern, cfg.Grep, dir)`, render via T032, `logCall("node_grep", ...)` (depends on T024, T032)
- [X] T034 [US2] Register `node_grep` via `mcp.AddTool` in `RunE`, with `ReadOnlyHint: true` and `Description` per contracts/mcp-contract.md (depends on T029, T033)
- [X] T035 [P] [US2] Unit tests in `cmd/arc/graph/serve_test.go` for `renderMatchTable`: zero matches produces header-only output; multiple matches produce one row each in order (depends on T032)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Expand a node into its neighborhood in one call (Priority: P3)

**Goal**: `subgraph_get(id, depth?)` returns the seed plus every node reachable within `depth` hops (default `1`) as one markdown patch-exchange document.

**Independent Test**: Call `subgraph_get` with a seed id and confirm the returned markdown matches `arc subgraph`'s own stdout for the same seed/depth, per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T013) and MUST currently be failing (red) until this phase lands.

- [X] T036 [US3] Implement `subgraphGetArgs{ID string; Depth *int}` and the `subgraph_get` tool handler function in `serve.go`: resolve `Depth` (nil → `1`, negative → `ErrInvalidDepth`), call `appgraph.Subgraph(ctx, fsys.Local{}, core.Filter{}, args.ID, depth, cfg.Subgraph, dir, false)` (no filter, `stubs=false` — neither is exposed by this tool per spec.md), render `string(core.RenderPatch(result.Patch))`, `logCall("subgraph_get", ...)` (depends on T029)
- [X] T037 [US3] Register `subgraph_get` via `mcp.AddTool` in `RunE`, with `ReadOnlyHint: true` and `Description` per contracts/mcp-contract.md (depends on T036)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently — all three tools functional over stdio

---

## Phase 6: User Story 4 - Reach the server over a network connection (Priority: P4)

**Goal**: `--http <addr>` serves the identical three tools over Streamable HTTP/SSE; a bare port or `:port` binds loopback-only, an explicit host binds exactly that host; an invalid/in-use address refuses to start.

**Independent Test**: Start with `--http :8080`, connect an SSE-capable client, and confirm all three tools return results identical to the stdio path, per quickstart.md Scenario 4.

### Implementation for User Story 4

> E2E test for this story was already written in Phase 2d (T014) and MUST currently be failing (red) until this phase lands.

- [X] T038 [US4] Implement the `--http` branch of `RunE` in `serve.go`: when `--http` is set, resolve/validate the address via T023's `resolveHTTPAddr`, `net.Listen("tcp", addr)` (a bind failure — invalid address or port in use — returns `ErrHTTPAddr` immediately, spec FR-005), wrap the already-built `mcp.Server` in `mcp.NewStreamableHTTPHandler`, and serve via `http.Serve` on an `http.Server` that shuts down when `ctx` is canceled (depends on T029, T023, T037)
- [X] T039 [P] [US4] Unit tests in `cmd/arc/graph/serve_test.go` for `resolveHTTPAddr`: a bare port and a `:port` form both resolve to `127.0.0.1:<port>`; an explicit host (`0.0.0.0:8080`, `192.168.1.10:8080`) is preserved unchanged; a syntactically invalid address returns `ErrHTTPAddr` (depends on T023)

**Checkpoint**: All four user stories pass their E2E tests independently — feature complete

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T040 [P] Update `README.md`'s command reference to mention `arc serve` (constitution Principle XII)
- [X] T041 [P] Manually run all quickstart.md scenarios (Scenarios 1-4, read-only verification, logging verification) against the built binary and confirm expected markdown output, error behavior, and stderr log lines
- [X] T042 [P] Run `govulncheck ./...` and confirm `github.com/modelcontextprotocol/go-sdk` introduces no known-critical vulnerability (constitution Principle XIV, Mandatory Libraries & Tooling)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/app/graph`'s new `NodeGet`/`EnsureGraph` members and the new `cmd/arc/graph/serve.go` entry (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary — MCP Tool, Transport, Bind Address (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: `arc serve --http <addr>`, error/exit behavior (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 `adrs/003-mcp-server-adapter.md` authored (Status: Accepted) with correct numbering, recording MCP as a second primary-adapter family (Principle I) — from T009
- [X] TN05 Domain logic uses ports (interfaces) where needed; `internal/app/graph/service.NodeGet`/`EnsureGraph` need no `port.VCS`/`port.SchemaRegistry` (mirrors `Subgraph`'s own precedent); `cmd/arc/graph/serve.go` wiring and `internal/app/graph/service` remain separated (Principle III)
- [X] TN06 Unit tests (`node_test.go`, `renderMatchTable`/`resolveHTTPAddr` tests) were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 `github.com/modelcontextprotocol/go-sdk` is consumed directly with no vendor SDK type leaking through `internal/app/graph`'s primary port (`NodeGet`/`EnsureGraph` return only `core.Node`/`error`); no new `os.*` filesystem calls outside `internal/adapter/fsys` (Principle VII)
- [X] TN10 `arc serve` makes no use of `bios.SCHEMA`'s styling (its output is MCP tool content and stderr log lines, not a colorized terminal table — same documented deviation `arc subgraph` already established); Ctrl-C (SIGINT)/SIGTERM exit promptly via `signal.NotifyContext` (Principle X, research.md D7)
- [X] TN11 No new `.arc/config.yml` field introduced; existing `Grep`/`Subgraph` config sections loaded once at startup; no secrets logged or involved (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc serve` (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md US1-US4 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/graph/serve_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `arc serve` is an entirely new command with no existing `--json`/`--plain` contract touched; `github.com/modelcontextprotocol/go-sdk` is one new, deliberate third-party dependency for `govulncheck` to track — no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other, **except T009's ADR authoring, which blocks Phase 2.5 and everything after it** (Constitution Principle I)
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion (specifically T009's ADR)
- **User Stories (Phase 3+)**: All depend on Phase 2.5; User Story 1 is the deepest since it implements the server's full startup sequence — User Stories 2, 3, and 4 extend the same `RunE`/`serve.go` and therefore depend on Phase 3's tasks as well as Phase 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; implements `RunE`'s full startup sequence (preflight, mount, config load, stdio transport) that US2-US4 extend
- **User Story 2 (P2)**: Can start after Phase 2.5, but its tool-registration task (T034) attaches to the `RunE`/server-build sequence US1 creates (T029) — sequenced after US1 in practice, though its E2E test (T012) is independent and was written in Phase 2d
- **User Story 3 (P3)**: Can start after Phase 2.5, same attachment pattern as US2 (sequenced after US1's T029, independent E2E test T013)
- **User Story 4 (P4)**: Can start after Phase 2.5, but its `--http` branch (T038) wraps the same `mcp.Server` US1-US3 build — sequenced after US1/US2/US3's tool registrations land, independent E2E test T014

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- `NodeGet`/`EnsureGraph`/scaffold foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other (except T009, which gates Phase 2.5)
- Within Phase 2.5: `internal/app/graph/service` (T017-T021) and `cmd/arc/graph/serve.go`'s pure helpers (T023-T025) have no cross-dependencies and can proceed in parallel
- Once Phase 3 lands, User Stories 2, 3, and 4's tool-specific logic (renderer/handler/registration for each) can be developed in parallel, though all three attach to the same `RunE` sequence

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Implement internal/app/graph/service/node.go (NodeGet, EnsureGraph)"
Task: "Implement resolveHTTPAddr in cmd/arc/graph/serve.go"
Task: "Implement mcpFilter.toCoreFilter() in cmd/arc/graph/serve.go"
Task: "Implement logCall in cmd/arc/graph/serve.go"
```

## Parallel Example: Phase 4 User Story 2

```bash
# Once T029 (RunE startup sequence) exists, launch together:
Task: "Implement renderMatchTable in cmd/arc/graph/serve.go"
Task: "Unit tests for renderMatchTable in cmd/arc/graph/serve_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories; includes authoring ADR 003)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Run quickstart.md Scenario 1 against the built binary
7. Deploy/demo if ready — `arc serve` already answers `node_get` over stdio at this point, missing `node_grep` (US2), `subgraph_get` (US3), and `--http` (US4)

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
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
- No Phase 0 (Pre-implementation Refactoring) is included — unlike `specs/007-arc-subgraph`, this feature renames nothing existing; `NodeGet` is additive and reuses `enumerateNodes`/`guardIsGraph` without modifying them
- User Stories 2, 3, and 4 are not fully file-independent from User Story 1 here (they extend `serve.go`'s `RunE`/tool registration US1 creates) — this reflects that all four stories share one MCP server process and one startup sequence, not four separate features; each remains independently *testable* via its own E2E test written in Phase 2d
- T009 (authoring ADR 003) is the one task in this feature's Phase 2 with a hard blocking dependency on everything after it, per constitution Principle I's "deviation from an accepted ADR is NOT permitted" and "before implementing, read every ADR referenced in the plan's Constitution Check section" rules — this feature's Constitution Check names an ADR that does not exist yet, so it must be created before Phase 2.5 starts
