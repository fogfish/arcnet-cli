# Implementation Plan: MCP Server (`arc serve`)

**Branch**: `008-arc-serve-mcp` | **Date**: 2026-07-05 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/008-arc-serve-mcp/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Implement `arc serve [--http <addr>]`: start a Model Context Protocol server exposing exactly three tools — `node_get(id)`, `node_grep(pattern, filter?)`, `subgraph_get(id, depth?)` — over stdio by default or SSE when `--http <addr>` is given. Per the user's explicit instruction, **the MCP server is a second primary adapter, not a fourth use-case**: it is a thin wiring layer, calling the *existing* `internal/app/graph` use-case root (`component.go`) exactly the way `cmd/arc/graph/grep.go` and `subgraph.go` already do, plus one new primary-port method (`NodeGet`, alongside the existing `Apply`/`Grep`/`Subgraph`) since no "fetch one node by id" capability exists yet. No new business logic is introduced for search or traversal — `node_grep` and `subgraph_get` delegate to the same `service.Grep`/`service.Subgraph` functions `arc grep`/`arc subgraph` already call. Per the user's second explicit instruction, every tool's reply is rendered as **markdown text**, not raw JSON: `node_get` reuses the existing `core.RenderNode` (front-matter + body, the same per-node serialization `arc apply`/`arc subgraph` already produce), `subgraph_get` reuses the existing `core.RenderPatch` (the same bytes `arc subgraph`'s own stdout already emits), and `node_grep` gets one small, new markdown-table renderer (there is no existing multi-match renderer to reuse, since `arc grep`'s own human renderer emits colorized plain-text rows, not markdown). This makes `arc serve` architecturally the pattern ADR 001 already anticipates verbatim ("a `serve` subcommand exposing the same use-cases over HTTP... itself just another primary adapter calling the same use-case root package, never a parallel implementation") — formalized here as a new ADR (003) since it is the first time a second primary-adapter *family* (MCP, alongside Cobra) actually exists in this codebase. `github.com/modelcontextprotocol/go-sdk` (the official Go MCP SDK, stdio + Streamable HTTP/SSE transports, generic `AddTool[In,Out]` registration, automatic input-schema generation and validation) is added as this feature's one new third-party dependency — the first MCP integration in this codebase, so no existing adapter can be reused (constitution Principle VII's "check whether an adapter already exists" gate).

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` (new — the official MCP Go SDK, `mcp.NewServer`/`mcp.AddTool`/`mcp.StdioTransport`/`mcp.NewStreamableHTTPHandler`; research.md D1), `github.com/spf13/cobra` (command wiring, unchanged), `github.com/fogfish/faults` (error annotation, unchanged); `github.com/charmbracelet/lipgloss` is **not** used by this command (its stdout-equivalent is MCP tool-call content, not a terminal, so there is nothing for `bios.SCHEMA` to style — same precedent `arc subgraph` already set for machine-consumable output)

**Storage**: The mounted graph root, read exclusively through the existing `internal/adapter/fsys` `Store`/`Mounter` — `arc serve`'s three tools perform **zero writes**, the fourth strictly read-only command in the codebase after `arc lint`/`arc grep`/`arc subgraph` (spec FR-015)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`. Per-tool business logic is unit-testable directly (thin wrapper functions around the existing, already-unit-tested `service.Grep`/`service.Subgraph`/new `service.NodeGet`); one true E2E test per user story exercises the *actual registered MCP tool handlers* end-to-end via `mcp.NewInMemoryTransports()` (an in-process client/server transport pair the SDK provides for exactly this purpose) plus a real `mcp.Client`, avoiding both a real subprocess and a real network port; `--http` address parsing/validation and the graph-not-initialized preflight are tested as ordinary `sut()`/`RunE`-direct E2E cases (research.md D7 — Principle VIII adaptation for a long-running command)

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — extends the existing `internal/app/graph` use-case with its fourth primary-port method (`NodeGet`, alongside `Apply`/`Grep`/`Subgraph`); adds one new Cobra command, `cmd/arc/graph/serve.go`, alongside `apply.go`/`grep.go`/`subgraph.go`

**Performance Goals**: Spec SC-003 (`node_grep` under 2s against several thousand nodes) and SC-004 (`subgraph_get` under 10s) — inherited directly from `arc grep`/`arc subgraph`'s own already-verified performance, since these tools call the identical service functions with no added overhead beyond MCP request/response marshaling

**Constraints**: Refuses to start when the target directory is not an initialized graph (spec FR-004, research.md D6 preflight); refuses to start on an invalid or already-in-use `--http` address (spec FR-005); `--http`'s bare-port/`:port` forms bind loopback-only, an explicit host binds exactly that host (spec FR-003, Clarifications); every tool call re-mounts and re-reads the graph's current on-disk state, no caching (spec FR-016 — inherited for free from `service.Grep`/`Subgraph`/new `NodeGet` each already re-enumerating per call); a per-call failure (unknown id, bad pattern, bad depth) MUST NOT terminate the server (spec FR-017 — satisfied by the SDK's own contract: a tool handler's returned `error` becomes `CallToolResult{IsError:true}`, never a protocol-level failure); one stderr line per tool call recording tool name, key arguments, and outcome (spec FR-019)

**Scale/Scope**: One new command (`arc serve`), one new primary-port method (`internal/app/graph.NodeGet`, backed by new `internal/app/graph/service/node.go` reusing the existing `enumerateNodes`/`guardIsGraph` helpers `service.Subgraph` already defines — no new traversal logic), one small new preflight delegator (`internal/app/graph.EnsureGraph`, backed by the existing private `guardIsGraph`), one new third-party dependency (`github.com/modelcontextprotocol/go-sdk`), one new markdown-table renderer for grep matches (the only genuinely new rendering code — `node_get`/`subgraph_get` reuse `core.RenderNode`/`core.RenderPatch` verbatim) — no changes to `internal/core.Node`/`RenderNode`/`RenderPatch`'s existing public contracts, no changes to `internal/adapter/fsys`, no changes to `internal/bios`, no new port anywhere (`NodeGet` needs no `port.VCS`/`port.SchemaRegistry`, exactly like `Grep`/`Subgraph`'s own precedent)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — this is the first feature to introduce a **second primary-adapter family** (MCP, alongside Cobra). ADR 001 already anticipates this exact pattern in prose ("a `serve` subcommand exposing the same use-cases over HTTP... itself just another primary adapter") but that prose has never been formalized as its own accepted decision. **`adrs/003-mcp-server-adapter.md` MUST be authored** (Status: Accepted) recording: MCP as a second primary-adapter family; the rule that an MCP tool handler MUST be a thin wrapper calling the same `internal/app/<domain>.Component` root every Cobra command calls, never a parallel reimplementation; the transport choice (`github.com/modelcontextprotocol/go-sdk`, stdio default + Streamable HTTP/SSE via `--http`); and the loopback-by-default bind-address rule (spec Clarifications). [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Directory Structure and Glossary sections MUST be updated in the same PR (new `cmd/arc/graph/serve.go` entry, `internal/app/graph.NodeGet`/`EnsureGraph`, and new glossary terms — MCP Tool, Node Object (MCP), Transport). `tasks.md` MUST schedule ADR authoring under Phase 2c (External Integration & Adapter Design) before implementation begins |
| II — DDD & Glossary | Yes | PASS — no new domain type: `node_get`/`subgraph_get` return the *existing* `core.Node`/`core.Patch` (rendered via the existing `core.RenderNode`/`RenderPatch`); the only new vocabulary is presentation-tier (an MCP `Tool`, its JSON input schema) and an incoming MCP filter object shape, which is deliberately kept a private, unexported `cmd/`-layer type (mirrors `grep.go`'s own `optsFilter`, never promoted into `internal/core`) since the domain-level `core.Filter` it converts into already exists |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/graph/serve.go` is the sole new primary adapter code: registers three MCP tools whose handlers translate MCP args → domain calls → markdown text, nothing more; `internal/app/graph/service/node.go` holds the one new domain method (`NodeGet`), reusing `enumerateNodes`/`guardIsGraph` already defined for `Subgraph` in the same package — no cobra or mcp-sdk import anywhere below `cmd/` |
| IV — Functional Programming Style | Yes | PASS — `NodeGet` is a pure index lookup after enumeration (identical shape to `Subgraph`'s own seed resolution); the MCP-filter→`core.Filter` mapping and the markdown-table renderer for grep matches are pure functions over their inputs; no inline comments |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — no fourth use-case package created; `node_grep`/`subgraph_get`'s tool handlers call the *existing* `appgraph.Grep`/`appgraph.Subgraph` verbatim (zero duplicated traversal/search logic, per the user's explicit "wiring only" instruction); the loopback-bind-address parser and the stderr call-logger are each implemented exactly once, in `serve.go`, not duplicated per tool |
| VI — TDD | Yes | PASS — `internal/app/graph/service/node_test.go` (unit tests for `NodeGet`, table-driven against a fake `fsys.Mounter`, mirroring `subgraph_test.go`'s existing shape) written first; `cmd/arc/graph/serve_test.go`'s E2E tests (research.md D7) written first and failing semantically before any handler exists |
| VII — External Integration & Adapter Consistency | Yes | PASS, with a new adapter — `github.com/modelcontextprotocol/go-sdk` is this feature's one new external integration; per Principle VII's "check whether an adapter for that capability already exists" gate, none does (first MCP integration in this codebase), so it is added directly rather than duplicating/wrapping a second client. The SDK's own `Transport` abstraction (`StdioTransport`, `NewStreamableHTTPHandler`) is used as-is, not re-wrapped in a project-private port — there is no second use-case that could need a *different* MCP transport implementation, so the ADR 001 port-isolation rule (narrowest interface the use-case actually needs) is satisfied by not introducing an unnecessary indirection at all |
| VIII — E2E Acceptance Testing | Yes, with a documented adaptation | PASS, with adaptation (research.md D7) — `arc serve`'s `RunE` runs a **long-lived** server loop, unlike every prior command's one-shot `RunE`; the constitution's `sut()`/`cmd.RunE(cmd,args)`-returns-and-captures-stdout pattern assumes `RunE` returns promptly. Adaptation: (a) `RunE` accepts `cmd.Context()` and MUST return when that context is canceled (SIGINT/SIGTERM via `signal.NotifyContext`, and directly in tests) — this is still `RunE`, still exercised directly, just no longer synchronous-return-on-success; (b) the two *refusal* scenarios (spec FR-004/FR-005 — not a graph, bad/busy `--http` address) return promptly and are tested via ordinary `sut()`; (c) the three tools' actual behavior is proven end-to-end by running the real `RunE`-registered `mcp.Server` against `mcp.NewInMemoryTransports()`'s in-process pair and a real `mcp.Client`, then canceling the context — this exercises the exact same tool-registration/handler code path production traffic hits, satisfying the principle's intent ("exercise the real production handler") without spinning a subprocess or a real network listener |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (`arc serve`, continuing `arc init`/`arc apply`/`arc lint`/`arc grep`/`arc subgraph`'s precedent); one new local flag, `--http <addr>` (string, default `""` meaning stdio), DS-02 options-struct shape; no shorthand claimed, consistent with DS-03's reserved table |
| X — Terminal Output, Color & Interactivity | Yes, narrowly | PASS, with the same documented deviation `arc subgraph` already established — `arc serve`'s "output" is MCP tool-call content (markdown text) and one stderr log line per call, neither of which is a colorized human table; `bios.SCHEMA` styling is not applicable here, and `--json`/`ResolveMode()` are not applicable either since MCP tool replies are not this command's stdout contract. Ctrl-C (SIGINT) MUST still exit promptly (research.md D7's `signal.NotifyContext`), satisfying Principle X's responsiveness rule even though the mechanism (context cancellation vs. an in-flight operation) differs from a one-shot command |
| XI — Configuration, Env & Secrets | Yes | PASS — reuses the existing `.arc/config.yml` `Config.Grep`/`Config.Subgraph` fields verbatim (loaded once at server startup, exactly like every other command loads it once per invocation — a long-running server's "one invocation" is its whole process lifetime); no new configuration file, no secrets, nothing to add to `Config` |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected failure (`--http` address invalid/in-use, target not a graph, unknown id, invalid pattern, invalid depth) declared as a `faults.Type`/`faults.SafeN` constant, extending the existing `internal/app/graph/service/errors.go` |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — no existing `--json`/`--plain` contract touched (this command has neither); adds exactly one new third-party dependency (`github.com/modelcontextprotocol/go-sdk`) for `govulncheck` to track going forward — a new, deliberate, and singular supply-chain addition, not a speculative one |

**ADR 001 port isolation rule 2** (explicit check, since a port is conspicuously *absent* here, same as `Subgraph`'s own precedent): satisfied — `internal/app/graph/service.NodeGet` needs neither `port.VCS` nor `port.SchemaRegistry`; the MCP SDK's `Transport` is consumed directly by `cmd/arc/graph/serve.go` with no project-private port wrapping it, since (per rule 2) a port narrower than "exactly what `mcp.Server.Run`/`NewStreamableHTTPHandler` already expect" would add indirection without a second implementation to justify it.

No unresolved Constitution Check conflicts, contingent on `adrs/003-mcp-server-adapter.md` being authored as a Phase 2c precondition (see Principle I row) before implementation begins. See Complexity Tracking below for the one deliberate, non-speculative trade-off this plan accepts (the E2E-testing adaptation, Principle VIII).

## Project Structure

### Documentation (this feature)

```text
specs/008-arc-serve-mcp/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md         # Phase 1 output
├── quickstart.md         # Phase 1 output
├── contracts/            # Phase 1 output
│   └── mcp-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── root.go               # + registers graph.NewServeCmd(); no new persistent flags
    └── graph/                 # existing — gains one new command file
        ├── apply.go             # unchanged
        ├── grep.go              # unchanged; its optsFilter/CLI-flag path is untouched —
        │                         #   node_grep's MCP filter object is a distinct, per-call
        │                         #   input decoded from JSON, not from CLI flags
        ├── subgraph.go          # unchanged
        ├── serve.go              # NEW — package graph: NewServeCmd() *cobra.Command; local
        │                         #   --http string flag (default ""); RunE: derives a
        │                         #   SIGINT/SIGTERM-cancelable context (research.md D7), mounts
        │                         #   and preflights via appgraph.EnsureGraph (spec FR-004),
        │                         #   loads .arc/config.yml once, parses/validates --http's
        │                         #   [host]:port (research.md D6, spec FR-003/FR-005), builds an
        │                         #   mcp.Server, registers node_get/node_grep/subgraph_get via
        │                         #   mcp.AddTool (each handler: decode args, call the matching
        │                         #   appgraph.* function, render markdown, log one stderr line
        │                         #   per FR-019), then runs StdioTransport or
        │                         #   NewStreamableHTTPHandler depending on --http
        └── serve_test.go         # NEW — E2E tests per spec.md acceptance scenario, via
                                  #   mcp.NewInMemoryTransports() + a real mcp.Client for the
                                  #   three tools' behavior, and ordinary sut()/RunE-direct
                                  #   calls for the two startup-refusal scenarios (research.md D7)

internal/
└── app/
    └── graph/                  # existing — gains NodeGet/EnsureGraph alongside Apply/Grep/Subgraph
        ├── component.go          # + NodeGet(ctx, mounter, dir, id) (core.Node, error);
        │                         #   + EnsureGraph(ctx, mounter, dir) error (preflight delegator,
        │                         #   backed by the existing private guardIsGraph)
        └── service/
            ├── node.go             # NEW — NodeGet: mount, guardIsGraph, enumerateNodes (reused
            │                        #   verbatim from subgraph.go), index lookup; EnsureGraph:
            │                        #   mount + guardIsGraph only
            ├── node_test.go         # NEW — unit tests against fake fsys.Mounter/Store
            ├── subgraph.go          # unchanged; enumerateNodes/guardIsGraph reused by node.go
            └── errors.go            # existing — + ErrHTTPAddr (invalid/in-use --http address)

ARCHITECTURE.md               # + Directory Structure/Glossary updated (Principle I obligation above)
adrs/
└── 003-mcp-server-adapter.md  # NEW — Accepted; formalizes MCP as a second primary-adapter family
                                 #   (Principle I obligation above; authored as a Phase 2c task,
                                 #   not by /speckit-plan itself)
```

**Structure Decision**: This feature extends the project's existing `internal/app/graph` use-case with a fourth primary-port method (`NodeGet`, alongside `Apply`/`Grep`/`Subgraph`) plus a small preflight delegator (`EnsureGraph`) — no new `internal/app/<domain>` package, per the user's explicit "use existing `internal/app/graph`, implement only wiring" instruction. The MCP wiring itself lives entirely in one new `cmd/arc/graph/serve.go`, following the exact same "bare-verb command in the `graph` package" placement `apply.go`/`grep.go`/`subgraph.go` already established. This is the codebase's first **second primary-adapter family** — formalized as `adrs/003-mcp-server-adapter.md` (Phase 2c task, see Constitution Check) rather than left as an unrecorded precedent. `node_get`/`subgraph_get` reuse `internal/core.RenderNode`/`RenderPatch` verbatim for their markdown replies; `node_grep` gets exactly one new, small markdown-table renderer (the only new rendering code this feature adds), living in `serve.go` itself since it is MCP-presentation-specific, not a second copy of `arc grep`'s own colorized-terminal renderer.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `RunE` no longer returns promptly on success (Principle VIII adaptation, research.md D7) | `arc serve` is inherently long-running — an MCP server that returns immediately has served nothing. This is a structural property of the feature (spec FR-002/FR-003: stdio or HTTP transport, both persistent), not a design choice this plan could avoid | Spawning a real OS subprocess per E2E test (exec the compiled binary, talk to it over a real pipe/port) was considered and rejected: it reintroduces exactly the flakiness/cost (process startup latency, port allocation races) the existing `sut()` pattern exists to avoid, and the MCP SDK's own `mcp.NewInMemoryTransports()` already provides a same-process, real-handler-dispatch alternative that keeps tests as fast and deterministic as every other command's `sut()`-based E2E suite |
