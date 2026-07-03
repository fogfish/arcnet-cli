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
	"gopkg.in/yaml.v3"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestCoreMergeRulesFixedKinds(t *testing.T) {
	op, ok := core.CoreMergeRules.Lookup("source")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeNone, op))

	op, ok = core.CoreMergeRules.Lookup("entity")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnion, op))

	op, ok = core.CoreMergeRules.Lookup("resource")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnionFirstWriter, op))

	op, ok = core.CoreMergeRules.Lookup("timeline")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeAppend, op))
}

func TestMergeRuleSetLookupUnregistered(t *testing.T) {
	_, ok := core.CoreMergeRules.Lookup("hypothesis")
	it.Then(t).Should(it.True(!ok))
}

func TestMergeRuleSetUnionSelfAuthoritative(t *testing.T) {
	other := core.MergeRuleSet{"source": core.MergeUnion, "hypothesis": core.MergeValidatedOverwrite}

	union := core.CoreMergeRules.Union(other)

	sourceOp, _ := union.Lookup("source")
	hypothesisOp, ok := union.Lookup("hypothesis")

	it.Then(t).
		Should(it.Equal(core.MergeNone, sourceOp)).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeValidatedOverwrite, hypothesisOp))
}

func TestMergeRuleSetYAMLRoundTrip(t *testing.T) {
	original := core.MergeRuleSet{"source": core.MergeNone, "hypothesis": core.MergeValidatedOverwrite}

	out, err := yaml.Marshal(original)
	it.Then(t).Should(it.Nil(err))

	var decoded core.MergeRuleSet
	it.Then(t).Should(it.Nil(yaml.Unmarshal(out, &decoded)))

	sourceOp, _ := decoded.Lookup("source")
	hypothesisOp, _ := decoded.Lookup("hypothesis")
	it.Then(t).
		Should(it.Equal(core.MergeNone, sourceOp)).
		Should(it.Equal(core.MergeValidatedOverwrite, hypothesisOp))
}

func TestConfigPath(t *testing.T) {
	it.Then(t).Should(it.Equal(".arc/config.yml", core.ConfigPath))
}
