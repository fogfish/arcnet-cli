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
	"maps"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fogfish/arcnet-cli/internal/core"
)

// schemaTool is schema's colocated mcp.Tool definition (research.md D2).
var schemaTool = &mcp.Tool{
	Name:        "schema",
	Description: "Return the knowledge graph's semantic layer and vocabulary. It returns defined classes (rdfs:Class) and properties (rdfs:Property) with descriptions. Use this to discover what vocabulary is available before reading or writing the graph. It is recommended as the first tool call of a session.",
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}

// schemaArgs is schema's input schema — empty, since schema takes no
// arguments (spec FR-006).
type schemaArgs struct{}

// schemaWorkflowNote documents schema's place in a session (research.md D5,
// contracts/mcp-contract.md) — composed into the server's session-start
// Instructions by serve.go's sessionInstructions().
const schemaWorkflowNote = "Always call schema first in a new session. Usage of other tools, arguments and filters depends on the understanding of the graph's semantic layer and vocabulary."

// renderSchema renders index as schema's markdown reply (research.md D2,
// contracts/mcp-contract.md, data-model.md): a sorted-by-name Predicates
// bullet list (name + description only), followed by a sorted-by-name
// Classes section, one subsection per class with its description and its
// required/optional predicate names ("(none)" for an empty list). Pure
// function, no I/O — mirrors renderMatchTable's existing shape.
func renderSchema(index core.Index) string {
	var b strings.Builder

	// TODO: Explain the filters
	b.WriteString("# Graph Schema\n\n")
	b.WriteString(`A knowledge graph here is a set of linked Markdown files: each file is a node, 
and [[wiki-links]] between files are edges. A bare [[Link]] embedded inside
the text is an untyped association; and a property:: [[Target]] bullet is
an explicit, typed edge. Both links file's @id. Underneath that surface,
every node is really a bag of RDF triples about one subject — (@id, predicate, value).
The service offers a tools to query the graph and discover either individual
statements, nodes or interconnected subgraphs. Use the schema below to discover
the graph's vocabulary and the predicates available for each class.
	`)

	b.WriteString("\n\n")
	b.WriteString("## Predicates\n\n")
	for _, name := range slices.Sorted(maps.Keys(index.Predicates)) {
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", name, index.Predicates[name].Description)
	}

	b.WriteString("\n## Classes\n\n")
	for _, name := range slices.Sorted(maps.Keys(index.Types)) {
		def := index.Types[name]
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", name, def.Description)
		fmt.Fprintf(&b, "Required: %s\n", joinOrNone(def.Required))
		fmt.Fprintf(&b, "Optional: %s\n\n", joinOrNone(def.Optional))
	}

	return b.String()
}

// joinOrNone joins names with ", ", or renders "(none)" for an empty list
// (research.md D2).
func joinOrNone(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

// schemaHandler renders the already-resolved index as schema's reply — the
// operation cannot fail once index is already in hand, so the returned
// error is always nil (kept only because mcp.AddTool's handler signature
// requires an error return). dir mirrors nodeGetHandler/subgraphGetHandler's
// existing factory shape (data-model.md) though schema makes no domain call
// of its own and never reads it.
func schemaHandler(dir string, index core.Index) func(context.Context, *mcp.CallToolRequest, schemaArgs) (*mcp.CallToolResult, any, error) {
	return func(context.Context, *mcp.CallToolRequest, schemaArgs) (*mcp.CallToolResult, any, error) {
		text := renderSchema(index)
		logCall("schema", "", len(index.Predicates)+len(index.Types), nil)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
	}
}
