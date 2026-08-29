//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// allServeTools lists every registered mcp.Tool var (data-model.md
// Validation Rules) — used by every metadata-completeness test below so a
// tool added to buildServer but forgotten here fails loudly.
var allServeTools = []*mcp.Tool{
	nodeGetTool, nodeGrepTool, subgraphGetTool, contextRetrieveTool, schemaTool, nodeMatchTool,
}

// TestServeToolVarsRegistered is a regression guard over Phase 0's file
// split (spec.md US1, data-model.md Validation Rules): every tool var must
// be non-nil and carry the wire name buildServer registers it under.
func TestServeToolVarsRegistered(t *testing.T) {
	names := map[*mcp.Tool]string{
		nodeGetTool:         "node_get",
		nodeGrepTool:        "node_grep",
		subgraphGetTool:     "subgraph_get",
		contextRetrieveTool: "context_retrieve",
		schemaTool:          "schema",
		nodeMatchTool:       "node_match",
	}
	for tool, name := range names {
		t.Run(name, func(t *testing.T) {
			it.Then(t).
				ShouldNot(it.Nil(tool)).
				Should(it.Equal(tool.Name, name))
		})
	}
}

// TestServeToolParameterMetadata (spec.md US2, spec SC-002): every parameter
// on every tool's input schema carries a non-empty description and at least
// one example value. schemaTool takes no arguments and is skipped.
func TestServeToolParameterMetadata(t *testing.T) {
	for _, tool := range allServeTools {
		schema, ok := tool.InputSchema.(*jsonschema.Schema)
		if !ok || schema == nil {
			continue
		}
		for name, prop := range schema.Properties {
			t.Run(tool.Name+"/"+name, func(t *testing.T) {
				it.Then(t).
					Should(it.True(prop.Description != "")).
					Should(it.GreaterOrEqual(len(prop.Examples), 1))
			})
		}
	}
}

// TestServeFilterArgumentExamples (spec.md US3, spec SC-003): the shared
// filter argument on every filter-accepting tool carries at least two
// example values, each demonstrating a distinct usage pattern.
func TestServeFilterArgumentExamples(t *testing.T) {
	for _, tool := range []*mcp.Tool{nodeGrepTool, subgraphGetTool, contextRetrieveTool, nodeMatchTool} {
		t.Run(tool.Name, func(t *testing.T) {
			schema, ok := tool.InputSchema.(*jsonschema.Schema)
			it.Then(t).Should(it.True(ok))

			filter, ok := schema.Properties["filter"]
			it.Then(t).Should(it.True(ok))
			it.Then(t).Should(it.GreaterOrEqual(len(filter.Examples), 2))
		})
	}
}

// TestServeSessionInstructions (spec.md US4, spec SC-004/SC-005): the
// server's session-start Instructions carry every tool's workflow/
// preference guidance, composed from data-model.md's Server Workflow
// Guidance.
func TestServeSessionInstructions(t *testing.T) {
	instructions := sessionInstructions()

	notes := []string{
		nodeGetWorkflowNote,
		nodeGrepWorkflowNote,
		subgraphGetWorkflowNote,
		contextRetrieveWorkflowNote,
		schemaWorkflowNote,
		nodeMatchWorkflowNote,
	}
	for _, note := range notes {
		it.Then(t).Should(it.String(instructions).Contain(note))
	}
}
