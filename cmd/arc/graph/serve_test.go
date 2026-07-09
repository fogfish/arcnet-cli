//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// connectServeSession builds the real, registered mcp.Server for dir
// (buildServer — the same construction RunE runs before selecting a
// transport) and connects a real mcp.Client to it over one half of
// mcp.NewInMemoryTransports() (research.md D7 tier 2), exercising the exact
// tool-registration/handler code path production traffic hits.
func connectServeSession(t *testing.T, ctx context.Context, dir string) *mcp.ClientSession {
	t.Helper()

	server, err := buildServer(ctx, dir)
	it.Then(t).Should(it.Nil(err))

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	go func() { server.Run(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "arc-test-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	it.Then(t).Should(it.Nil(err))
	t.Cleanup(func() { session.Close() })

	return session
}

func textOf(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	it.Then(t).Should(it.Equal(1, len(result.Content)))
	tc, ok := result.Content[0].(*mcp.TextContent)
	it.Then(t).Should(it.True(ok))
	return tc.Text
}

const serveEntityTLS = `---
"@id": Transport Layer Security
"@type": entity
category: form structure attribute process
---
# Transport Layer Security

TLS is the successor to SSL.

- [[rescorla-2026-tls13]]
`

const serveSourceTLS13 = `---
"@id": rescorla-2026-tls13
"@type": source
title: TLS 1.3
---
# rescorla-2026-tls13

TLS 1.3 is the latest version of the Transport Layer Security protocol.
`

func seedServeFixture(t *testing.T, dir string) {
	t.Helper()
	writeGrepNode(t, dir, "entities/Transport Layer Security.md", serveEntityTLS)
	writeGrepNode(t, dir, "sources/rescorla-2026-tls13.md", serveSourceTLS13)
}

// { "name": "node_get", "arguments": { "id": "Transport Layer Security" } }
// Scenario 1 from spec.md US1: node_get returns the full node object,
// matching on-disk content.
func TestServeNodeGetReturnsFullNodeContent(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_get",
		Arguments: map[string]any{"id": "Transport Layer Security"},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
	text := textOf(t, result)
	it.Then(t).
		Should(it.String(text).Contain(`"@id": Transport Layer Security`)).
		Should(it.String(text).Contain("TLS is the successor to SSL."))
}

// { "name": "node_get", "arguments": { "id": "No Such Node" } }
// Scenario 2 from spec.md US1: an unknown id returns a clear tool error, no
// node object, and the server itself keeps running.
func TestServeNodeGetUnknownIDReturnsToolError(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_get",
		Arguments: map[string]any{"id": "No Such Node"},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(result.IsError))
	it.Then(t).Should(it.String(textOf(t, result)).Contain("no node found"))

	// The server itself keeps running and answers the next call normally.
	result2, err2 := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_get",
		Arguments: map[string]any{"id": "Transport Layer Security"},
	})
	it.Then(t).Should(it.Nil(err2))
	it.Then(t).ShouldNot(it.True(result2.IsError))
}

// Scenario 3 from spec.md US1: a freshly-connected client can discover and
// invoke node_get immediately.
func TestServeFreshClientCanDiscoverAndInvokeNodeGet(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	tools, err := session.ListTools(ctx, nil)
	it.Then(t).Should(it.Nil(err))

	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	it.Then(t).Should(it.Seq(names).Contain("node_get"))

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_get",
		Arguments: map[string]any{"id": "Transport Layer Security"},
	})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
}

// { "name": "node_grep", "arguments": { "pattern": "TLS 1\\.3" } }
// Scenario 1 from spec.md US2: node_grep returns one row per matching line,
// no filter.
func TestServeNodeGrepReturnsOneRowPerMatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_grep",
		Arguments: map[string]any{"pattern": `TLS 1\.3`},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
	text := textOf(t, result)
	it.Then(t).
		Should(it.String(text).Contain("| id | kind | line | snippet |")).
		Should(it.String(text).Contain("rescorla-2026-tls13"))
}

// { "name": "node_grep", "arguments": { "pattern": "TLS", "filter": { "kind": ["source"] } } }
// Scenario 2 from spec.md US2: a filter object narrows the matched nodes.
func TestServeNodeGrepFilterNarrowsMatchedNodes(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "node_grep",
		Arguments: map[string]any{
			"pattern": "TLS",
			"filter":  map[string]any{"kind": []string{"source"}},
		},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
	text := textOf(t, result)
	it.Then(t).
		Should(it.String(text).Contain("rescorla-2026-tls13")).
		ShouldNot(it.String(text).Contain("Transport Layer Security |"))
}

// Scenario 3 from spec.md US2: a non-matching pattern returns an empty
// table, not an error.
func TestServeNodeGrepNonMatchingPatternReturnsHeaderOnly(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_grep",
		Arguments: map[string]any{"pattern": "NoSuchPatternAnywhere"},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
	text := textOf(t, result)
	it.Then(t).Should(it.String(text).Contain("| id | kind | line | snippet |"))
}

// Scenario 4 from spec.md US2: a syntactically invalid pattern returns a
// clear tool error.
func TestServeNodeGrepInvalidPatternReturnsToolError(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_grep",
		Arguments: map[string]any{"pattern": "TLS ("},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(result.IsError))
}

// { "name": "subgraph_get", "arguments": { "id": "Transport Layer Security" } }
// Scenario 1 from spec.md US3: subgraph_get with default depth returns the
// seed + direct neighbors as complete node objects.
func TestServeSubgraphGetDefaultDepthReturnsSeedAndNeighbors(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "subgraph_get",
		Arguments: map[string]any{"id": "Transport Layer Security"},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
	text := textOf(t, result)
	it.Then(t).
		Should(it.String(text).Contain("## Transport Layer Security")).
		Should(it.String(text).Contain("## rescorla-2026-tls13"))
}

const serveChainA = `---
"@id": ChainA
"@type": entity
---
# ChainA

- [[ChainB]]
`
const serveChainB = `---
"@id": ChainB
"@type": entity
---
# ChainB

- [[ChainC]]
`
const serveChainC = `---
"@id": ChainC
"@type": entity
---
# ChainC
`

// { "name": "subgraph_get", "arguments": { "id": "ChainA", "depth": 2 } }
// Scenario 2 from spec.md US3: an explicit depth widens/narrows the set.
func TestServeSubgraphGetExplicitDepthWidensSet(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/ChainA.md", serveChainA)
	writeGrepNode(t, dir, "entities/ChainB.md", serveChainB)
	writeGrepNode(t, dir, "entities/ChainC.md", serveChainC)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	depthOne := 1
	result1, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "subgraph_get",
		Arguments: map[string]any{"id": "ChainA", "depth": depthOne},
	})
	it.Then(t).Should(it.Nil(err))
	text1 := textOf(t, result1)
	it.Then(t).
		Should(it.String(text1).Contain("## ChainB")).
		ShouldNot(it.String(text1).Contain("## ChainC"))

	result2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "subgraph_get",
		Arguments: map[string]any{"id": "ChainA", "depth": 2},
	})
	it.Then(t).Should(it.Nil(err))
	text2 := textOf(t, result2)
	it.Then(t).Should(it.String(text2).Contain("## ChainC"))
}

const serveDiamondA = `---
"@id": DiamondA
"@type": entity
---
# DiamondA

- [[DiamondB]]
- [[DiamondC]]
`
const serveDiamondB = `---
"@id": DiamondB
"@type": entity
---
# DiamondB

- [[DiamondD]]
`
const serveDiamondC = `---
"@id": DiamondC
"@type": entity
---
# DiamondC

- [[DiamondD]]
`
const serveDiamondD = `---
"@id": DiamondD
"@type": entity
---
# DiamondD
`

// Scenario 3 from spec.md US3: a multi-path-reachable node appears exactly
// once.
func TestServeSubgraphGetMultiPathNodeAppearsExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/DiamondA.md", serveDiamondA)
	writeGrepNode(t, dir, "entities/DiamondB.md", serveDiamondB)
	writeGrepNode(t, dir, "entities/DiamondC.md", serveDiamondC)
	writeGrepNode(t, dir, "entities/DiamondD.md", serveDiamondD)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "subgraph_get",
		Arguments: map[string]any{"id": "DiamondA", "depth": 2},
	})

	it.Then(t).Should(it.Nil(err))
	text := textOf(t, result)
	it.Then(t).Should(it.Equal(1, strings.Count(text, "## DiamondD")))
}

// Scenario 4 from spec.md US3: an unknown seed id returns a clear tool
// error.
func TestServeSubgraphGetUnknownSeedReturnsToolError(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := connectServeSession(t, ctx, dir)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "subgraph_get",
		Arguments: map[string]any{"id": "No Such Node"},
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(result.IsError))
	it.Then(t).Should(it.String(textOf(t, result)).Contain("no node found"))
}

// Edge case (spec FR-004): the target not being an initialized graph refuses
// arc serve's RunE immediately, via ordinary sut()/RunE-direct calls.
func TestServeTargetNotAGraphRefusesImmediately(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewServeCmd(), nil)

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).Should(it.Error(out, err).Contain("initialized graph"))
}

// Edge case (spec FR-005): a syntactically invalid --http address refuses
// arc serve's RunE immediately.
func TestServeInvalidHTTPAddrRefusesImmediately(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	cmd := NewServeCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("http", "not a valid addr!!")))
	out, err := sut(cmd, nil)

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).ShouldNot(it.Nil(err))
}

// spec.md US4 acceptance scenario 1: a client connecting over SSE/Streamable
// HTTP can invoke all three tools with results identical to the in-memory
// path (research.md D7 tier 3 — httptest.NewServer wrapping the real,
// registered mcp.StreamableHTTPHandler, no real OS port chosen at random).
func TestServeHTTPTransportServesIdenticalResults(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := buildServer(ctx, dir)
	it.Then(t).Should(it.Nil(err))

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "arc-test-http-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	it.Then(t).Should(it.Nil(err))
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "node_get",
		Arguments: map[string]any{"id": "Transport Layer Security"},
	})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.True(result.IsError))
	it.Then(t).Should(it.String(textOf(t, result)).Contain("TLS is the successor to SSL."))
}

// spec.md US4 acceptance scenario 3: an invalid/in-use --http address
// refuses to start rather than silently falling back to stdio.
func TestServeHTTPAddrAlreadyInUseRefusesToStart(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	it.Then(t).Should(it.Nil(err))
	defer listener.Close()
	busyAddr := listener.Addr().String()

	cmd := NewServeCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("http", busyAddr)))
	out, err := sut(cmd, nil)

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).ShouldNot(it.Nil(err))
}

// T031: node_get's handler function, called directly (bypassing the
// transport), returns a non-nil error for an unknown id and nil content —
// mcp.AddTool's own generic wrapper is what packs that error into
// CallToolResult{IsError: true} for a connected client, already confirmed
// end-to-end by TestServeNodeGetUnknownIDReturnsToolError above.
func TestNodeGetHandlerErrorMapping(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedServeFixture(t, dir)

	handler := nodeGetHandler(dir, core.Index{})

	result, _, err := handler(context.Background(), nil, nodeGetArgs{ID: "No Such Node"})
	it.Then(t).
		Should(it.True(result == nil)).
		ShouldNot(it.Nil(err)).
		Should(it.String(err.Error()).Contain("no node found"))

	result, _, err = handler(context.Background(), nil, nodeGetArgs{ID: "Transport Layer Security"})
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(result != nil))
	it.Then(t).Should(it.String(textOf(t, result)).Contain("TLS is the successor to SSL."))
}

// T031: logCall writes exactly one line per call, ok/error(message).
func TestLogCallOutputShape(t *testing.T) {
	_, stderr, err := sutCaptureStderr(t, &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			logCall("node_get", `id="Transport Layer Security"`, nil)
			logCall("node_get", `id="No Such Node"`, errors.New("no node found with basename No Such Node"))
			return nil
		},
	}, nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.String(stderr).Contain(`serve: node_get id="Transport Layer Security" ok`)).
		Should(it.String(stderr).Contain(`serve: node_get id="No Such Node" error: no node found with basename No Such Node`))
}

// T035: renderMatchTable renders a header-only table for zero matches, and
// one row per match, in order, for multiple matches.
func TestRenderMatchTableZeroMatchesHeaderOnly(t *testing.T) {
	out := renderMatchTable(nil)

	it.Then(t).
		Should(it.String(out).Contain("| id | kind | line | snippet |")).
		ShouldNot(it.String(out).Contain("\n| "))
}

func TestRenderMatchTableMultipleMatchesOneRowEachInOrder(t *testing.T) {
	out := renderMatchTable([]kernel.Match{
		{ID: "a", Type: "source", Line: 1, Text: "first"},
		{ID: "b", Type: "entity", Line: 2, Text: "second"},
	})

	firstIdx := strings.Index(out, "| a | source | 1 | first |")
	secondIdx := strings.Index(out, "| b | entity | 2 | second |")
	it.Then(t).
		Should(it.True(firstIdx >= 0)).
		Should(it.True(secondIdx >= 0)).
		Should(it.True(firstIdx < secondIdx))
}

// T039: resolveHTTPAddr resolves a bare port/:port to loopback-only,
// preserves an explicit host unchanged, and rejects a syntactically
// invalid address.
func TestResolveHTTPAddrBarePortResolvesToLoopback(t *testing.T) {
	addr, err := resolveHTTPAddr("8080")
	it.Then(t).Should(it.Nil(err)).Should(it.Equal("127.0.0.1:8080", addr))
}

func TestResolveHTTPAddrColonPortResolvesToLoopback(t *testing.T) {
	addr, err := resolveHTTPAddr(":8080")
	it.Then(t).Should(it.Nil(err)).Should(it.Equal("127.0.0.1:8080", addr))
}

func TestResolveHTTPAddrExplicitHostPreservedUnchanged(t *testing.T) {
	addr, err := resolveHTTPAddr("0.0.0.0:8080")
	it.Then(t).Should(it.Nil(err)).Should(it.Equal("0.0.0.0:8080", addr))

	addr, err = resolveHTTPAddr("192.168.1.10:8080")
	it.Then(t).Should(it.Nil(err)).Should(it.Equal("192.168.1.10:8080", addr))
}

func TestResolveHTTPAddrInvalidAddressReturnsErrHTTPAddr(t *testing.T) {
	_, err := resolveHTTPAddr("not a valid addr!!")
	it.Then(t).Should(it.True(errors.Is(err, service.ErrHTTPAddr)))
}
