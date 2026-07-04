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

func TestMergeRuleSetLookup(t *testing.T) {
	rules := core.MergeRuleSet{"source": core.MergeNone, "entity": core.MergeUnion}

	op, ok := rules.Lookup("source")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeNone, op))

	_, ok = rules.Lookup("hypothesis")
	it.Then(t).Should(it.True(!ok))
}

func TestMergeRuleSetUnionSelfAuthoritative(t *testing.T) {
	base := core.MergeRuleSet{"source": core.MergeNone}
	other := core.MergeRuleSet{"source": core.MergeUnion, "hypothesis": core.MergeValidatedOverwrite}

	union := base.Union(other)

	sourceOp, _ := union.Lookup("source")
	hypothesisOp, ok := union.Lookup("hypothesis")

	it.Then(t).
		Should(it.Equal(core.MergeNone, sourceOp)).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeValidatedOverwrite, hypothesisOp))
}
