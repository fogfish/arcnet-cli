//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestNodeZeroValue(t *testing.T) {
	var n core.Node

	it.Then(t).
		Should(it.Equal("", n.ID)).
		Should(it.Equal(core.Kind(""), n.Kind)).
		Should(it.Equal(0, len(n.Attrs))).
		Should(it.Equal(0, len(n.HRefs))).
		Should(it.Equal(0, len(n.Edges))).
		Should(it.Equal(0, len(n.Links)))
}

func TestPatchZeroValue(t *testing.T) {
	var p core.Patch

	it.Then(t).
		Should(it.Equal("", p.Document)).
		Should(it.True(p.Published.IsZero())).
		Should(it.Equal(0, len(p.Nodes)))
}

func TestMergeOpConstants(t *testing.T) {
	it.Then(t).
		Should(it.Equal(core.MergeOp("none"), core.MergeNone)).
		Should(it.Equal(core.MergeOp("union"), core.MergeUnion)).
		Should(it.Equal(core.MergeOp("union-first-writer"), core.MergeUnionFirstWriter)).
		Should(it.Equal(core.MergeOp("append"), core.MergeAppend)).
		Should(it.Equal(core.MergeOp("validated-overwrite"), core.MergeValidatedOverwrite))
}

func TestLinkBlockShape(t *testing.T) {
	lb := core.LinkBlock{
		Title: "Mentions",
		Seq:   []core.Link{{Predicate: "mentions", Target: "Transport Layer Security"}},
	}

	it.Then(t).
		Should(it.Equal("Mentions", lb.Title)).
		Should(it.Equal(1, len(lb.Seq)))
}
