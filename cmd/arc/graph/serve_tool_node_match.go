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
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
)

// nodeMatchTool is node_match's colocated mcp.Tool definition (research.md
// D2).
var nodeMatchTool = &mcp.Tool{
	Name:        "node_match",
	Description: "List every distinct fact (node_id, property, value) from the graph that fully satisfy a required filter argument. It returns a markdown table, one row per distinct fact: id, property, value. Use filters to express the intent behind the search criteria and scope facts.",
	InputSchema: must(nodeMatchInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// nodeMatchArgs is node_match's input schema. Unlike node_grep/
// subgraph_get/context_retrieve, Filter is REQUIRED (non-pointer, no
// "omitempty") — spec FR-001/FR-005.
type nodeMatchArgs struct {
	Filter mcpFilter `json:"filter" jsonschema:"required filter; at least one statement — an empty filter is rejected since node_match's output is evidence of why nodes matched, and every node vacuously matches an empty filter"`
}

// nodeMatchInputSchema derives nodeMatchArgs's JSON Schema and attaches
// example filter values (research.md D3/D4, contracts/mcp-contract.md) —
// both non-trivial, since an empty filter is rejected (service.ErrEmptyFilter).
func nodeMatchInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[nodeMatchArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "filter",
		map[string]any{"statements": []any{map[string]any{"predicate": "type", "target": "Source"}}},
		map[string]any{"statements": []any{map[string]any{"predicate": "title", "targetPattern": "^TLS"}}},
	)
	return s, nil
}

// nodeMatchWorkflowNote documents node_match's place alongside node_grep/
// context_retrieve (research.md D5, contracts/mcp-contract.md) — composed
// into the server's session-start Instructions by serve.go's
// sessionInstructions().
const nodeMatchWorkflowNote = "Use node_match instead of node_grep/context_retrieve when you need to know which facts justified a match (for citation or explanation), not the node's content itself."

// nodeMatchHandler evaluates filter against every node's own facts and
// renders one markdown table row per distinct fact that satisfied at
// least one statement (contracts/mcp-contract.md). args.Filter is a
// concrete, always-non-nil value here, unlike the other tools' *mcpFilter
// — toCoreFilter is called on &args.Filter; the empty-statements check
// itself happens in service.Match, not here, so the validation lives in
// one place regardless of transport.
func nodeMatchHandler(dir string) func(context.Context, *mcp.CallToolRequest, nodeMatchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeMatchArgs) (*mcp.CallToolResult, any, error) {
		logArgs := fmt.Sprintf("filter=%s", filterSummary(&args.Filter))

		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("node_match", logArgs, 0, err)
			return nil, nil, err
		}

		result, err := appgraph.Match(ctx, fsys.Local{}, filter, dir)
		logCall("node_match", logArgs, len(result.Matches), err)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: renderFactTable(result.Matches)}}}, nil, nil
	}
}

// renderFactTable renders node_match's markdown reply (specs/028-node-
// match-filter contracts/mcp-contract.md): a fixed header, one row per
// matching fact, header only when matches is empty (spec FR-007).
func renderFactTable(matches []kernel.MatchEntry) string {
	var b strings.Builder
	b.WriteString("| id | property | value |\n|---|---|---|\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", m.ID, m.Property, m.Value)
	}
	return b.String()
}
