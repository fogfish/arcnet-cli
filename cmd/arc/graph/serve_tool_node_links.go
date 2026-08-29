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
	"github.com/fogfish/arcnet-cli/internal/core"
)

// nodeLinksTool is node_links' colocated mcp.Tool definition (contracts/
// mcp-contract.md, specs/030-mcp-node-links-backlinks).
var nodeLinksTool = &mcp.Tool{
	Name:        "node_links",
	Description: "List every outgoing relation on one node by its @id — both its structural edges and any inline prose references. Returns a markdown table, one row per relation: predicate, target.",
	InputSchema: must(nodeLinksInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// nodeLinksArgs is node_links' input schema — identical single-field shape
// to nodeGetArgs.
type nodeLinksArgs struct {
	ID string `json:"id" jsonschema:"the node's basename, e.g. the value returned as \"@id\" by node_grep/node_match"`
}

// nodeLinksInputSchema derives nodeLinksArgs's JSON Schema and attaches an
// example id value (data-model.md, contracts/mcp-contract.md).
func nodeLinksInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[nodeLinksArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "id", "tls-1-3")
	return s, nil
}

// nodeLinksWorkflowNote documents node_links' place alongside node_get/
// node_backlinks — composed into the server's session-start Instructions
// by serve.go's sessionInstructions().
const nodeLinksWorkflowNote = "Use node_links when you need only a node's own outgoing relations (predicate/target pairs) — not its full content — such as before deciding which neighbor to fetch next."

// nodeLinksHandler fetches id's outgoing relations and renders them as a
// markdown table.
func nodeLinksHandler(dir string) func(context.Context, *mcp.CallToolRequest, nodeLinksArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeLinksArgs) (*mcp.CallToolResult, any, error) {
		links, err := appgraph.NodeLinks(ctx, fsys.Local{}, dir, args.ID)
		logCall("node_links", fmt.Sprintf("id=%q", args.ID), len(links), err)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: renderLinkTable(links)}}}, nil, nil
	}
}

// renderLinkTable renders node_links' markdown reply (contracts/mcp-
// contract.md): a fixed header, one row per relation, header only when
// links is empty (spec FR-006).
func renderLinkTable(links []core.Link) string {
	var b strings.Builder
	b.WriteString("| predicate | target |\n|---|---|\n")
	for _, l := range links {
		fmt.Fprintf(&b, "| %s | %s |\n", l.Predicate, l.Target)
	}
	return b.String()
}
