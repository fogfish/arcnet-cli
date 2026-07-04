//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel_test

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
)

func TestNewLintResultDerivesPassingFailing(t *testing.T) {
	nodes := []kernel.NodeStatus{
		{Path: "sources/a.md", ID: "a", Kind: "source"},
		{Path: "entities/b.md", ID: "b", Kind: "entity", Violations: []kernel.Violation{
			{Rule: kernel.RuleLinkResolves, Path: "entities/b.md", Line: 3, Message: "boom"},
		}},
		{Path: "entities/c.md", ID: "c", Kind: "entity"},
	}

	result := kernel.NewLintResult("/graph", nodes)

	it.Then(t).
		Should(it.Equal(2, result.Passing)).
		Should(it.Equal(1, result.Failing)).
		Should(it.Equal(1, len(result.Violations)))
}

func TestNewLintResultIncludesGraphSpanningFirst(t *testing.T) {
	nodes := []kernel.NodeStatus{
		{Path: "a.md", ID: "a", Violations: []kernel.Violation{
			{Rule: kernel.RuleLinkResolves, Path: "a.md", Line: 1, Message: "boom"},
		}},
	}
	spanning := kernel.Violation{Rule: kernel.RuleUniqueBasename, Message: "collision", RelatedPaths: []string{"a.md", "b.md"}}

	result := kernel.NewLintResult("/graph", nodes, spanning)

	it.Then(t).
		Should(it.Equal(2, len(result.Violations))).
		Should(it.Equal(kernel.RuleUniqueBasename, result.Violations[0].Rule))
}

func TestRuleConstantsAreDistinct(t *testing.T) {
	rules := []kernel.Rule{
		kernel.RuleFrontMatter, kernel.RuleUniqueBasename, kernel.RuleLinkResolves,
		kernel.RuleSourceCitekey, kernel.RuleEntityCategory, kernel.RuleDerivedProvenance,
		kernel.RulePredicateCase, kernel.RulePredicateRegistered, kernel.RuleCitationPredicate,
		kernel.RuleUnrecognizedKind, kernel.RuleIngestCommit, kernel.RuleMergeConflict,
	}

	seen := map[kernel.Rule]bool{}
	for _, r := range rules {
		it.Then(t).Should(it.True(!seen[r]))
		seen[r] = true
	}
	it.Then(t).Should(it.Equal(12, len(seen)))
}

func TestValidSowaCategory(t *testing.T) {
	ok, reason := kernel.ValidSowaCategory([]string{"independent", "abstract", "occurrent", "script"})
	it.Then(t).Should(it.True(ok)).Should(it.Equal("", reason))
}

func TestValidSowaCategoryWrongLength(t *testing.T) {
	ok, reason := kernel.ValidSowaCategory([]string{"independent", "abstract", "occurrent"})
	it.Then(t).
		Should(it.True(!ok)).
		Should(it.String(reason).Contain("found 3"))
}

func TestValidSowaCategoryBadWord(t *testing.T) {
	ok, _ := kernel.ValidSowaCategory([]string{"bogus", "abstract", "occurrent", "script"})
	it.Then(t).Should(it.True(!ok))
}
