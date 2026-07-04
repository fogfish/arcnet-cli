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

var coreMergeRulesFixture = core.MergeRuleSet{
	"source":   core.MergeNone,
	"entity":   core.MergeUnion,
	"resource": core.MergeUnionFirstWriter,
	"timeline": core.MergeAppend,
}

func TestCheckUniqueBasenamesNoCollision(t *testing.T) {
	index := map[string][]string{
		"foo": {"sources/foo.md"},
		"bar": {"entities/bar.md"},
	}
	out := checkUniqueBasenames(index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckUniqueBasenamesTwoWayCollision(t *testing.T) {
	index := map[string][]string{
		"rfc8446": {"resources/rfc8446.md", "entities/rfc8446.md"},
	}
	out := checkUniqueBasenames(index)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleUniqueBasename, out[0].Rule)).
		Should(it.Equal("", out[0].Path)).
		Should(it.Equal(2, len(out[0].RelatedPaths))).
		Should(it.String(out[0].Message).Contain("rfc8446"))
}

func TestCheckUniqueBasenamesThreeWayCollisionNamesEveryFile(t *testing.T) {
	index := map[string][]string{
		"widget": {"a/widget.md", "b/widget.md", "c/widget.md"},
	}
	out := checkUniqueBasenames(index)
	it.Then(t).
		Should(it.Equal(1, len(out))).
		Should(it.Equal(3, len(out[0].RelatedPaths)))
}

func TestCheckUnrecognizedKindRecognized(t *testing.T) {
	node := core.Node{Kind: "source"}
	out := checkUnrecognizedKind(node, "sources/foo.md", coreMergeRulesFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckUnrecognizedKindUnrecognized(t *testing.T) {
	node := core.Node{Kind: "hypothesis"}
	out := checkUnrecognizedKind(node, "hypothesis/foo.md", coreMergeRulesFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleUnrecognizedKind, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("hypothesis"))
}

func TestCheckUnrecognizedKindConfigRegistered(t *testing.T) {
	rules := coreMergeRulesFixture.Union(core.MergeRuleSet{"hypothesis": core.MergeValidatedOverwrite})
	node := core.Node{Kind: "hypothesis"}
	out := checkUnrecognizedKind(node, "hypothesis/foo.md", rules)
	it.Then(t).Should(it.Equal(0, len(out)))
}
