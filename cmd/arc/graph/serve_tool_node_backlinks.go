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

// nodeBacklinksTool is node_backlinks' colocated mcp.Tool definition
// (contracts/mcp-contract.md, specs/030-mcp-node-links-backlinks).
var nodeBacklinksTool = &mcp.Tool{
	Name:        "node_backlinks",
	Description: "List every relation elsewhere in the graph — structural edge or inline prose reference — that targets one node by its @id. Returns a markdown table, one row per incoming relation: source, predicate.",
	InputSchema: must(nodeBacklinksInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// nodeBacklinksArgs is node_backlinks' input schema — identical single-field
// shape to nodeLinksArgs/nodeGetArgs.
type nodeBacklinksArgs struct {
	ID string `json:"id" jsonschema:"the node's basename, e.g. the value returned as \"@id\" by node_grep/node_match"`
}

// nodeBacklinksInputSchema derives nodeBacklinksArgs's JSON Schema and
// attaches an example id value (data-model.md, contracts/mcp-contract.md).
func nodeBacklinksInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[nodeBacklinksArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "id", "rescorla-tls13")
	return s, nil
}

// nodeBacklinksWorkflowNote documents node_backlinks' place alongside
// node_links — composed into the server's session-start Instructions by
// serve.go's sessionInstructions().
const nodeBacklinksWorkflowNote = "Use node_backlinks to discover what elsewhere in the graph references a node — every incoming relation, structural or inline — without scanning the graph by hand."

// nodeBacklinksHandler fetches every relation targeting id and renders them
// as a markdown table.
func nodeBacklinksHandler(dir string) func(context.Context, *mcp.CallToolRequest, nodeBacklinksArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeBacklinksArgs) (*mcp.CallToolResult, any, error) {
		backlinks, err := appgraph.NodeBacklinks(ctx, fsys.Local{}, dir, args.ID)
		logCall("node_backlinks", fmt.Sprintf("id=%q", args.ID), len(backlinks), err)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: renderBacklinkTable(backlinks)}}}, nil, nil
	}
}

// renderBacklinkTable renders node_backlinks' markdown reply (contracts/
// mcp-contract.md): a fixed header, one row per incoming relation, header
// only when backlinks is empty (spec FR-006).
func renderBacklinkTable(backlinks []kernel.BacklinkEntry) string {
	var b strings.Builder
	b.WriteString("| source | predicate |\n|---|---|\n")
	for _, e := range backlinks {
		fmt.Fprintf(&b, "| %s | %s |\n", e.Source, e.Predicate)
	}
	return b.String()
}
