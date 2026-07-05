# ADR 003 - MCP Server as a Second Primary-Adapter Family

**Status**: Accepted
**Date**: 2026-07-05

## Context

Every command shipped so far (`arc init`, `arc apply`, `arc grep`, `arc subgraph`, `arc lint`) is reached through exactly one primary (driving) adapter family: the Cobra command tree under `/cmd`. ADR 001 already anticipates a second family in prose — "a `serve` subcommand exposing the same use-cases over HTTP... itself just another primary adapter calling the same use-case root package, never a parallel implementation" — but that sentence has never been formalized as its own accepted decision, and no second family has existed until now.

`specs/008-arc-serve-mcp` introduces `arc serve`: a Model Context Protocol (MCP) server exposing `node_get`, `node_grep`, and `subgraph_get` as MCP Tools, so an LLM client can read the graph directly. This is the first time a second primary-adapter family actually exists in this codebase, and the first third-party dependency this project has ever added purely to speak a driving protocol (as opposed to a secondary/driven integration like `git`).

## Decision

1. **MCP is a second primary-adapter family, alongside Cobra.** An MCP tool handler MUST be a thin wrapper — decode the tool's JSON arguments, call the identical `internal/app/<domain>.Component` primary-port function every Cobra command already calls, render the result, nothing more. It is never a parallel reimplementation of business logic already expressed for the Cobra path. `arc serve`'s three tools delegate verbatim into `internal/app/graph.NodeGet`/`Grep`/`Subgraph` — the same functions `arc grep`/`arc subgraph` call.

2. **Transport**: `github.com/modelcontextprotocol/go-sdk`, the official Go SDK for MCP, is this feature's one new third-party dependency (research.md D1 in `specs/008-arc-serve-mcp`). `mcp.StdioTransport` is the default transport; `mcp.NewStreamableHTTPHandler` (Streamable HTTP/SSE) is used when `--http <addr>` is given. The SDK's `Transport` abstraction is consumed directly by `cmd/arc/graph/serve.go` — it is not re-wrapped in a project-private port, since (per ADR 001 port isolation rule 2) a port narrower than what the SDK already expects would add indirection with no second implementation to justify it.

3. **Loopback-by-default bind address**: `--http <addr>`'s value is a `[host]:port` address spec. A bare port or `:port` (no explicit host) binds `127.0.0.1` only; an explicit host binds exactly that host. An operator must opt in to a non-loopback bind by naming a host explicitly — the default is never reachable from another machine.

## Neglected

- **A hand-rolled JSON-RPC 2.0 transport** was rejected outright: it would mean re-implementing MCP's initialize/list-tools/call-tool handshake and schema-validation logic from scratch, the exact "second, divergent client for the same external system" constitution Principle VII forbids, for a protocol that already has an official, well-tested SDK.
- **`github.com/mark3labs/mcp-go`**, a popular third-party MCP server library, was considered and rejected: it predates the official SDK, is not maintained by the protocol's own org, and this codebase has no existing dependency on it to justify continuity.
- **A project-private port wrapping `mcp.Transport`** was considered and rejected: there is no second use-case that could need a *different* MCP transport implementation, so the indirection would exist for its own sake.

## To Achieve

A second primary-adapter family that reaches the same use-case roots every Cobra command already reaches, so an LLM client and a human operator are guaranteed to see identical graph-read behavior — never two diverging code paths for "how do we fetch a node" or "how do we search content."

## Accepting

`arc serve` produces no `stdout` document of its own and uses none of `bios.SCHEMA`'s terminal styling — its "output" is MCP tool-call content (markdown text) and one stderr log line per call. This is the same documented deviation `arc subgraph` already established for machine/LLM-consumable output, extended here to a second adapter family entirely. We accept a long-running `RunE` (blocks until its context is canceled) as a structural property of a server command — not a deviation this ADR could design away — with its E2E-testing adaptation recorded in `specs/008-arc-serve-mcp/research.md` D7.

## Implementation notes

- `cmd/arc/graph/serve.go` is the sole new primary-adapter code for this feature: it registers the three MCP tools, translating MCP args → domain calls → markdown text.
- `internal/app/graph/service/node.go` holds the one new domain method, `NodeGet`, reusing `enumerateNodes`/`guardIsGraph` already defined for `Subgraph` in the same package — no new traversal or parsing logic.
- No `cobra` or `mcp-sdk` import appears anywhere below `cmd/`, preserving ADR 001's hexagonal boundary for this second adapter family exactly as it already holds for Cobra.
