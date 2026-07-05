# Phase 0 Research: MCP Server (`arc serve`)

## D1: MCP server library

**Decision**: `github.com/modelcontextprotocol/go-sdk` (the official Go SDK for the Model Context Protocol), pinned at `v1.6.1` at plan time.

**Rationale**: It is the canonically maintained implementation (published under the `modelcontextprotocol` GitHub org, in lockstep with the protocol spec itself), it ships both transports this feature needs out of the box (`mcp.StdioTransport` and `mcp.NewStreamableHTTPHandler`/`mcp.NewSSEHandler` for `--http`), and its generic `mcp.AddTool[In, Out]` registration automatically derives a JSON input schema from a plain Go struct's `json`/`jsonschema` tags and validates incoming arguments before the handler runs — removing an entire category of hand-written argument-validation code this feature would otherwise need. Confirmed importable (`go list -m -versions` against the real module proxy resolved 20+ published versions through `v1.6.1`).

**Alternatives considered**:
- `github.com/mark3labs/mcp-go` — a popular, earlier third-party server library. Rejected: it predates the official SDK, is not maintained by the protocol's own org, and this codebase has no existing dependency on it to justify continuity; the official SDK is the safer long-term bet for spec conformance as MCP itself evolves.
- Hand-rolled JSON-RPC 2.0 over stdio/HTTP — rejected outright: it would mean re-implementing MCP's initialize/list-tools/call-tool handshake and schema-validation logic from scratch, exactly the "second, divergent client for the same external system" constitution Principle VII forbids, for a protocol that already has an official, well-tested SDK.

## D2: Tool reply format — markdown, reusing existing renderers

**Decision**: `node_get`'s and `subgraph_get`'s MCP tool replies are literally `string(core.RenderNode(node))` and `string(core.RenderPatch(result.Patch))` respectively — the exact same bytes `arc apply`'s node writer and `arc subgraph`'s human-mode stdout already produce. `node_grep`'s reply is one new, small markdown table (`| id | kind | line | snippet |`), since no existing renderer produces a multi-match markdown table (`arc grep`'s own human renderer emits colorized plain-text rows for a terminal, not markdown).

**Rationale**: Per the user's explicit instruction ("Reply data in markdown format for MCP client"), and per the user's "implement only wiring" instruction — reusing `RenderNode`/`RenderPatch` verbatim means zero new core-domain rendering code for two of the three tools. Markdown is also what an LLM client consuming these tools will parse/display most naturally, and it is the same format the rest of this codebase already treats as its canonical human/LLM-facing document shape (patch-exchange format, CORE §12.2).

**Alternatives considered**: Returning `mcp.CallToolResult.StructuredContent` (a typed JSON value, auto-populated by `AddTool` when the handler's `Out` type is a concrete struct) was considered and rejected for this increment — the user's instruction is explicit about markdown text, and a structured-JSON contract would be a second, parallel reply shape to keep in sync with the markdown one for no immediate consumer benefit.

## D3: `node_get`/`EnsureGraph` placement and implementation

**Decision**: Add `NodeGet` and `EnsureGraph` to `internal/app/graph/service` (new `node.go`), delegated through `component.go` exactly like `Apply`/`Grep`/`Subgraph` already are. `NodeGet` reuses `subgraph.go`'s existing, unexported `enumerateNodes`/`guardIsGraph` helpers verbatim: mount, guard, enumerate every node into the `nodeIndex`, look up `id`, return `ErrSeedNotFound.With(errNoCause, id)` on a miss (the same error `Subgraph`'s own seed-resolution already returns for the identical failure mode). `EnsureGraph` is a two-line function: mount, then `guardIsGraph` — nothing else.

**Rationale**: `enumerateNodes` already returns a fully-parsed `map[string]core.Node` keyed by id; `node_get`'s entire job is that one map's lookup. Reusing it rather than writing a bespoke single-file reader keeps `node_get`'s error behavior (unknown id, not-a-graph) automatically consistent with `subgraph_get`'s already-shipped seed-resolution behavior, and requires zero new traversal or parsing logic. `EnsureGraph` exists only because `arc serve` needs to refuse to *start* (spec FR-004) before entering its serve loop, distinct from every prior command's per-invocation guard-then-proceed shape — reusing `guardIsGraph` (already private to `service`) avoids a second, divergent "is this a graph" check.

**Alternatives considered**: A dedicated single-file lookup (open exactly the one file `id` should map to, skip the full-graph enumeration) was considered for `NodeGet`'s performance, but rejected for this increment: it would require a basename→path resolution rule independent of `enumerateNodes`'s own (which already walks and parses every node), risking the two falling out of sync (e.g. a node the enumeration excludes for a parse error would still "exist" to a single-file reader). `arc grep`/`arc subgraph` already pay the full-enumeration cost per invocation with no index (Phase 4 of VISION.md is unbuilt); `node_get` paying the same, already-accepted cost keeps behavior uniform rather than introducing a second, faster-but-inconsistent path.

## D4: MCP filter object → `core.Filter` mapping

**Decision**: A small, unexported `cmd/arc/graph/serve.go`-local struct (`mcpFilter{Kind []string; Tags []string; Attrs map[string]string; AttrPatterns map[string]string}`) decoded directly from `node_grep`'s JSON arguments via `mcp.AddTool`'s automatic input-schema binding, converted to `core.Filter` by one small pure function (`Kind` → `Filter.Kinds`, `Tags` → `Filter.Tags`, `Attrs` → `Filter.Attrs`, `AttrPatterns` compiled via `regexp.Compile` into `Filter.AttrPatterns`, returning a clear error on an invalid pattern).

**Rationale**: This mirrors `grep.go`'s own `optsFilter.build()` exactly — a `cmd/`-layer (primary-adapter) translation from one driver's native input shape (CLI flags there, MCP JSON here) into the same shared domain type, `core.Filter`. Keeping it private to `serve.go` (not promoted to `internal/core` or shared with `optsFilter`) matches VISION.md's own MCP filter object schema, which is intentionally not identical in shape to the CLI's repeatable `--attr name=value`/`name~=pattern` flag syntax (JSON gives `attrs`/`attrPatterns` as two separate maps instead of one repeated flag needing runtime `~=` splitting).

**Alternatives considered**: Sharing `optsFilter` itself between the CLI and MCP paths was considered and rejected — `optsFilter` is a Cobra-flag options struct (DS-02); its `apply(cmd)` method has no meaning for an MCP tool's JSON schema, and forcing the JSON filter through it would mean synthesizing fake `--attr` strings just to reuse `build()`, more indirection than the two small, independent mapping functions this decision uses instead.

## D5: `--http` address parsing and default bind

**Decision**: One small pure function, `resolveHTTPAddr(addr string) (string, error)`, using `net.SplitHostPort`: if `addr` has no host component (`"8080"`, `":8080"`), the resolved address is `"127.0.0.1:<port>"`; if `addr` already has an explicit host (`"0.0.0.0:8080"`, `"192.168.1.10:8080"`), it is used exactly as given; a syntactically invalid address is a clear startup error (spec FR-005).

**Rationale**: Implements the spec's Clarifications session decision directly (loopback-only unless a host is explicitly given) using only the stdlib `net` package — no new dependency, no new port, since address parsing is a pure string transform, not an I/O operation.

**Alternatives considered**: A separate `--bind`/`--host` flag alongside `--port` was considered (this was Option C in the clarification question) and rejected in favor of the single `[host]:port` address spec the user actually chose, since it is one flag instead of two and matches a convention already familiar from `net/http`'s own `ListenAndServe(addr string, ...)` signature.

## D6: Startup preflight (spec FR-004)

**Decision**: `serve.go`'s `RunE` calls `appgraph.EnsureGraph(ctx, mounter, dir)` once, before registering any MCP tool or starting either transport; a non-nil error is returned from `RunE` immediately (Cobra's normal `SilenceUsage`/`SilenceErrors` error path, DS-07), and no server is ever started.

**Rationale**: Every existing command already guards "is this a graph" per invocation (`guardIsGraph`, called inside `Apply`/`Grep`/`Subgraph`); `arc serve` is the first command where "per invocation" and "per tool call" are different moments — a long-running server that started successfully but then fails every subsequent tool call with "not a graph" would be a confusing, silent-until-first-use failure mode. One explicit preflight call keeps the existing per-call guard inside each tool's own service function unchanged (still needed, since the graph's `.arc/` directory could theoretically disappear mid-session) while giving an operator immediate, correct feedback at the moment they run the command.

## D7: E2E testing strategy for a long-running command (Principle VIII adaptation)

**Decision**: Three testing tiers, all still exercising the real, production `RunE`/handler code:
1. **Refusal scenarios** (not a graph; bad/in-use `--http` address) — ordinary `sut()`/`cmd.RunE(cmd, args)` calls; `RunE` returns an error promptly, exactly like every other command's error path.
2. **Tool-behavior scenarios** (the bulk of spec.md's acceptance scenarios) — `RunE` is invoked with a `context.Context` derived from `context.WithCancel`, in a goroutine; the test builds an `mcp.Client`, connects it to the server via one half of `mcp.NewInMemoryTransports()` (the SDK's own in-process transport pair, intended for exactly this kind of test), calls `session.CallTool(ctx, ...)` for each of the three tools, asserts on the returned `CallToolResult`'s content and `IsError` flag via `github.com/fogfish/it/v2`, then cancels the context and confirms `RunE`'s goroutine returns (server shutdown).
3. **Startup wiring smoke test** — one test starts the real `--http` path against `httptest.NewServer` wrapping the registered `mcp.StreamableHTTPHandler`, confirming the HTTP transport is reachable end-to-end (not just the in-memory transport), satisfying spec User Story 4's acceptance scenarios without binding a real OS port chosen at random.

**Rationale**: Tier 2 is what makes this adaptation still compliant with Principle VIII's actual intent — "exercising the same function Cobra dispatches to in normal operation" — even though the mechanism (context cancellation instead of return-on-completion) differs from every prior command. `mcp.NewInMemoryTransports()` existing in the SDK specifically for this purpose (confirmed via `go doc`) means no bespoke test harness has to be invented.

**Alternatives considered**: Exec-ing the compiled `arc` binary as a real subprocess and talking to it over real stdio/a real TCP port (closer to a literal end-user invocation) was considered and rejected as the *primary* test mechanism — it reintroduces process-startup latency and port-allocation flakiness into `go test ./...`, exactly what `sut()` was introduced to avoid for every other command; it remains available as an optional, non-gating smoke script per constitution Principle VI's "Bash/shell scripts... reserved for optional smoke-test scripts" if a project maintainer wants one later.

## D8: Concurrency (spec FR-018)

**Decision**: No new synchronization code. Every tool handler independently mounts (`fsys.Local{}.Mount(dir)`) and calls a read-only service function (`Grep`/`Subgraph`/`NodeGet`) that itself opens and parses files fresh, with no shared mutable state across calls; the MCP SDK dispatches concurrent tool calls (over `--http`, potentially many simultaneous sessions) as ordinary concurrent Go function invocations.

**Rationale**: The three tools are already, individually, side-effect-free pure reads of the filesystem — the same property that already makes `arc grep`/`arc subgraph` safe to imagine running concurrently with themselves. There is no in-process cache or mutable index (D3/VISION.md Phase 4 is unbuilt) for concurrent calls to race over.

## D9: Operational logging (spec FR-019)

**Decision**: One small unexported helper in `serve.go`, called by every tool handler after it returns: `logCall(tool string, args string, err error)`, writing one line to `os.Stderr` (`fmt.Fprintf`) — tool name, a short rendering of the key argument(s) (id/pattern/depth), and `ok`/`error: <message>`.

**Rationale**: Directly implements spec FR-019/SC-008 with the minimum necessary mechanism — no new port, no structured logging library, consistent with this codebase's existing stderr-for-diagnostics convention (constitution Principle X) and with the spec's own Clarifications decision (one line per call, no metrics/tracing layer in this increment).

**Alternatives considered**: Threading ADR 002 DS-06's `Reporter` port (`Start`/`Step`/`Done`/`Error`) through the MCP handlers was considered and rejected — `Reporter` models a single command's own multi-phase progress narrative (`Start` → `Step`s → `Done`), not a long-running server's independent, discrete per-request access log; forcing the two into one abstraction would blur a real semantic difference for no reuse benefit (nothing else in this feature has a `Reporter`-shaped progress narrative).
