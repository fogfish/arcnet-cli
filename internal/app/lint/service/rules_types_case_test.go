//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestCheckNodeTypeCaseValid(t *testing.T) {
	node := core.Node{Type: "Entity"}
	out := checkNodeTypeCase(node, "entities/X.md")
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckNodeTypeCaseInvalid(t *testing.T) {
	node := core.Node{Type: "entity"}
	out := checkNodeTypeCase(node, "entities/X.md")
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleTypeCase, out[0].Rule)).
		Should(it.Equal("entities/X.md", out[0].Path)).
		Should(it.String(out[0].Message).Contain("entity"))
}

func TestCheckSchemaTypeCaseAllCamelCaseNoViolation(t *testing.T) {
	index := core.Index{Types: map[string]core.TypeDef{
		"Entity": {}, "Source": {},
	}}
	out := checkSchemaTypeCase(index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckSchemaTypeCaseLowercaseKeyReportsOne(t *testing.T) {
	index := core.Index{Types: map[string]core.TypeDef{
		"Entity": {}, "widget": {},
	}}
	out := checkSchemaTypeCase(index)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleTypeCase, out[0].Rule)).
		Should(it.Equal("_schema/types/widget.md", out[0].Path)).
		Should(it.String(out[0].Message).Contain("widget"))
}
