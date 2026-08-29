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

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// nodeGetTool is node_get's colocated mcp.Tool definition (research.md D2).
var nodeGetTool = &mcp.Tool{
	Name:        "node_get",
	Description: "Fetch one node's full stored content by its id. Returns the node exactly as stored: its front-matter attributes and body, including its outgoing relations, rendered as markdown.",
	InputSchema: must(nodeGetInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// nodeGetArgs is node_get's input schema.
type nodeGetArgs struct {
	ID string `json:"id" jsonschema:"the node's basename (filename without extension), e.g. the value returned as \"id\" by node_grep/node_match"`
}

// nodeGetInputSchema derives nodeGetArgs's JSON Schema and attaches an
// example id value (research.md D3, contracts/mcp-contract.md).
func nodeGetInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[nodeGetArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "id", "tls-1-3")
	return s, nil
}

// nodeGetWorkflowNote documents node_get's place in a session (research.md
// D5, contracts/mcp-contract.md) — composed into the server's session-start
// Instructions by serve.go's sessionInstructions().
const nodeGetWorkflowNote = "Use node_get once you already have a specific node id — from node_grep, node_match, subgraph_get, or a prior context_retrieve result — and need that one node's complete content."

// nodeGetHandler fetches one node by id and renders it exactly as
// core.RenderNode already serializes it on disk (research.md D2).
func nodeGetHandler(dir string, index core.Index) func(context.Context, *mcp.CallToolRequest, nodeGetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeGetArgs) (*mcp.CallToolResult, any, error) {
		node, err := appgraph.NodeGet(ctx, fsys.Local{}, dir, args.ID)
		logCall("node_get", fmt.Sprintf("id=%q", args.ID), err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderNode(node, index)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	}
}
