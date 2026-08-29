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
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// contextRetrieveTool is context_retrieve's colocated mcp.Tool definition
// (research.md D2).
var contextRetrieveTool = &mcp.Tool{
	Name:        "context_retrieve",
	Description: "Assemble the full content of every node relevant to a free-text query in one call — content match, attribute match, and neighbor expansion combined, ranked, deduplicated, and truncated to limit; optionally scoped/narrowed by a filter.statements triple filter (see schema). Returns a patch-exchange document (same shape as subgraph_get) containing the ranked, deduplicated, limit-truncated set of relevant nodes.",
	InputSchema: must(contextRetrieveInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// contextRetrieveArgs is context_retrieve's input schema.
type contextRetrieveArgs struct {
	Query  string     `json:"query" jsonschema:"free text, matched literally and case-insensitively against node content and attributes"`
	Filter *mcpFilter `json:"filter,omitempty" jsonschema:"optional filter narrowing every retrieval pass (content match, attribute match, and neighbor expansion all respect it)"`
	Limit  *int       `json:"limit,omitempty" jsonschema:"maximum number of node objects to return, default 10"`
}

// contextRetrieveInputSchema derives contextRetrieveArgs's JSON Schema and
// attaches example query/limit/filter values (research.md D3/D4,
// contracts/mcp-contract.md).
func contextRetrieveInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[contextRetrieveArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "query", "post-quantum key exchange")
	withExamples(s, "limit", 10, 25)
	withExamples(s, "filter",
		map[string]any{"statements": []any{map[string]any{"predicate": "type", "target": "Source"}}},
		map[string]any{"statements": []any{map[string]any{"predicate": "tags", "target": []any{"cryptography", "protocols"}}}},
	)
	return s, nil
}

// contextRetrieveWorkflowNote documents context_retrieve's place in a
// session and when to prefer node_grep/subgraph_get instead (research.md
// D5, contracts/mcp-contract.md) — composed into the server's session-start
// Instructions by serve.go's sessionInstructions().
const contextRetrieveWorkflowNote = "Prefer context_retrieve as the default way to gather everything relevant to a topic in one call; fall back to node_grep when you specifically need line-level text matches, or to subgraph_get when you already have a seed id and want its exact neighborhood rather than a ranked, query-driven set."

// contextRetrieveHandler runs the three-pass retrieval (content match,
// attribute match, neighbor expansion) and renders the ranked, truncated
// result as one patch-exchange document, byte-shape-identical to
// subgraph_get's own reply construction (contracts/mcp-contract.md).
// contextRetrieveArgs.Limit's nil-resolution mirrors subgraphGetArgs.Depth's
// existing pattern.
func contextRetrieveHandler(dir string, cfgGrep configkernel.GrepConfig, cfgSubgraph configkernel.SubgraphConfig, index core.Index) func(context.Context, *mcp.CallToolRequest, contextRetrieveArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args contextRetrieveArgs) (*mcp.CallToolResult, any, error) {
		limit := 10
		if args.Limit != nil {
			limit = *args.Limit
		}
		logArgs := fmt.Sprintf("query=%q filter=%s limit=%d", args.Query, filterSummary(args.Filter), limit)

		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("context_retrieve", logArgs, 0, err)
			return nil, nil, err
		}

		result, err := appgraph.ContextRetrieve(ctx, fsys.Local{}, filter, args.Query, limit, cfgGrep, cfgSubgraph, dir)
		logCall("context_retrieve", logArgs, len(result.Patch.Nodes), err)
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
