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
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// subgraphGetTool is subgraph_get's colocated mcp.Tool definition
// (research.md D2).
var subgraphGetTool = &mcp.Tool{
	Name:        "subgraph_get",
	Description: "Return the fully-resolved subgraph rooted at a node, to a given hop depth, optionally scoped/narrowed by a filter.statements triple filter (see schema) — a predicate-only statement restricts which relations traversal follows. Returns a patch-exchange document: the seed node plus every node reached within depth hops, each rendered with its own attributes and relations, in the same format arc subgraph produces on the command line.",
	InputSchema: must(subgraphGetInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// subgraphGetArgs is subgraph_get's input schema. Filter is new
// (research.md D8) — subgraph_get had no filter argument before this
// feature, so both flat-inclusion narrowing and predicate-scoped traversal
// were unreachable from it.
type subgraphGetArgs struct {
	ID     string     `json:"id" jsonschema:"seed node basename to expand outward from"`
	Depth  *int       `json:"depth,omitempty" jsonschema:"number of hops to traverse from the seed, default 1"`
	Filter *mcpFilter `json:"filter,omitempty" jsonschema:"optional filter; a predicate-only statement scopes which relations traversal follows, a statement also naming source/target narrows which reached nodes are kept in the result"`
}

// subgraphGetInputSchema derives subgraphGetArgs's JSON Schema and attaches
// example id/depth/filter values (research.md D3/D4, contracts/mcp-contract.md).
func subgraphGetInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[subgraphGetArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "id", "tls-1-3")
	withExamples(s, "depth", 1, 2)
	withExamples(s, "filter",
		map[string]any{"statements": []any{map[string]any{"predicate": "cites"}}},
		map[string]any{"statements": []any{map[string]any{"predicate": "type", "target": "Source"}}},
	)
	return s, nil
}

// subgraphGetWorkflowNote documents subgraph_get's place in a session and
// when to prefer it over context_retrieve (research.md D5,
// contracts/mcp-contract.md) — composed into the server's session-start
// Instructions by serve.go's sessionInstructions().
const subgraphGetWorkflowNote = "Prefer subgraph_get when you already know the seed node and want its resolved neighborhood as one connected document; prefer context_retrieve when you don't yet know a good seed and want to start from a free-text query instead."

// subgraphGetHandler extracts the seed plus every node reachable within
// depth hops and renders the result as one patch-exchange document, byte-
// identical to arc subgraph's own stdout for the same seed/depth.
func subgraphGetHandler(dir string, cfg configkernel.SubgraphConfig, index core.Index) func(context.Context, *mcp.CallToolRequest, subgraphGetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args subgraphGetArgs) (*mcp.CallToolResult, any, error) {
		depth := 1
		if args.Depth != nil {
			depth = *args.Depth
		}

		logArgs := fmt.Sprintf("id=%q depth=%d", args.ID, depth)
		if depth < 0 {
			err := service.ErrInvalidDepth.With(errNoCause, strconv.Itoa(depth))
			logCall("subgraph_get", logArgs, err)
			return nil, nil, err
		}

		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("subgraph_get", logArgs, err)
			return nil, nil, err
		}

		result, err := appgraph.Subgraph(ctx, fsys.Local{}, filter, args.ID, depth, cfg, dir, false)
		logCall("subgraph_get", logArgs, err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderPatch(result.Patch, index)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	}
}
