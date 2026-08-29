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
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
)

// nodeGrepTool is node_grep's colocated mcp.Tool definition (research.md D2).
var nodeGrepTool = &mcp.Tool{
	Name:        "node_grep",
	Description: "Search every node's content for lines matching a regexp pattern, optionally narrowed to a subset of nodes by filter.statements. Returns a markdown table, one row per matching line: id, type, line (1-based line number), snippet (the matching line's text).",
	InputSchema: must(nodeGrepInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// nodeGrepArgs is node_grep's input schema.
type nodeGrepArgs struct {
	Pattern string     `json:"pattern" jsonschema:"regexp pattern to search node content for"`
	Filter  *mcpFilter `json:"filter,omitempty" jsonschema:"optional filter narrowing which nodes are scanned; omit to scan every node"`
}

// nodeGrepInputSchema derives nodeGrepArgs's JSON Schema and attaches
// example pattern/filter values (research.md D3/D4, contracts/mcp-contract.md).
func nodeGrepInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[nodeGrepArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "pattern", "TODO", `func\s+\w+\(`)
	withExamples(s, "filter",
		map[string]any{"statements": []any{map[string]any{"predicate": "type", "target": "Source"}}},
		map[string]any{"statements": []any{map[string]any{"predicate": "title", "targetPattern": "^TLS"}}},
	)
	return s, nil
}

// nodeGrepWorkflowNote documents node_grep's place in a session and when to
// prefer it over context_retrieve/subgraph_get (research.md D5,
// contracts/mcp-contract.md) — composed into the server's session-start
// Instructions by serve.go's sessionInstructions().
const nodeGrepWorkflowNote = "Prefer node_grep when you need the exact matching line(s) of a node's raw content (e.g. to quote a passage); prefer context_retrieve when you want whole, ranked node objects instead of individual line matches."

// nodeGrepHandler searches node content for pattern, narrowed by an optional
// filter, and renders one markdown table row per matching line.
func nodeGrepHandler(dir string, cfg configkernel.GrepConfig) func(context.Context, *mcp.CallToolRequest, nodeGrepArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeGrepArgs) (*mcp.CallToolResult, any, error) {
		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("node_grep", fmt.Sprintf("pattern=%q", args.Pattern), err)
			return nil, nil, err
		}

		result, err := appgraph.Grep(ctx, fsys.Local{}, filter, args.Pattern, cfg, dir)
		logCall("node_grep", fmt.Sprintf("pattern=%q", args.Pattern), err)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: renderMatchTable(result.Matches)}}}, nil, nil
	}
}

// renderMatchTable renders matches as node_grep's markdown reply (research.md
// D2, contracts/mcp-contract.md): a fixed header, one row per match, header
// only when matches is empty (spec FR-009).
func renderMatchTable(matches []kernel.Match) string {
	var b strings.Builder
	b.WriteString("| id | type | line | snippet |\n|---|---|---|---|\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", m.ID, m.Type, m.Line, m.Text)
	}
	return b.String()
}
