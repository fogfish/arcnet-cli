//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel_test

import (
	"regexp"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestCoreMergeRulesFixedKinds(t *testing.T) {
	it.Then(t).Should(it.Equal(4, len(kernel.CoreMergeRules)))

	op, ok := kernel.CoreMergeRules.Lookup("source")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeNone, op))

	op, ok = kernel.CoreMergeRules.Lookup("entity")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnion, op))

	op, ok = kernel.CoreMergeRules.Lookup("resource")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnionFirstWriter, op))

	op, ok = kernel.CoreMergeRules.Lookup("timeline")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeAppend, op))
}

var camelCasePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func TestCorePredicatesThirteenDistinctCamelCaseNames(t *testing.T) {
	it.Then(t).Should(it.Equal(13, len(kernel.CorePredicates)))

	for name := range kernel.CorePredicates {
		it.Then(t).Should(it.True(camelCasePattern.MatchString(name)))
	}
}
