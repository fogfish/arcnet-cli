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
		{Path: "Source/a.md", ID: "a", Type: "Source"},
		{Path: "Entity/b.md", ID: "b", Type: "Entity", Violations: []kernel.Violation{
			{Rule: kernel.RuleLinkResolves, Path: "Entity/b.md", Line: 3, Message: "boom"},
		}},
		{Path: "Entity/c.md", ID: "c", Type: "Entity"},
	}

	result := kernel.NewLintResult("/graph", nodes)

	it.Then(t).
		Should(it.Equal(2, result.Passing)).
		Should(it.Equal(1, result.Failing)).
		Should(it.Equal(1, len(result.Violations)))
}

// TestRuleDefinitionsCoverEveryRule confirms kernel.RuleDefinitions'
// Invariant A (data-model.md): its length matches the declared Rule const
// block, and every entry carries a non-empty Description — so arc lint
// rules can never silently omit a rule --skip is able to reference.
func TestRuleDefinitionsCoverEveryRule(t *testing.T) {
	const wantRules = 18

	it.Then(t).Should(it.Equal(wantRules, len(kernel.RuleDefinitions)))

	for _, def := range kernel.RuleDefinitions {
		it.Then(t).ShouldNot(it.Equal("", def.Description))
	}
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
		kernel.RuleTypeCase,
	}

	seen := map[kernel.Rule]bool{}
	for _, r := range rules {
		it.Then(t).Should(it.True(!seen[r]))
		seen[r] = true
	}
	it.Then(t).Should(it.Equal(13, len(seen)))
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

// contract C1.1: the closed set of twelve is exhaustively verified against
// all 144 positional combinations (3 x 2 x 2 x 12) the old four independent
// word-set checks used to accept — exactly 12 pass, the rest fail.
func TestValidSowaCategoryExhaustive144Combinations(t *testing.T) {
	position1 := []string{"independent", "relative", "mediating"}
	position2 := []string{"physical", "abstract"}
	position3 := []string{"continuant", "occurrent"}
	leaf := []string{
		"object", "process", "schema", "script",
		"juncture", "participation", "description", "history",
		"structure", "situation", "reason", "purpose",
	}

	total, passing := 0, 0
	for _, p1 := range position1 {
		for _, p2 := range position2 {
			for _, p3 := range position3 {
				for _, lf := range leaf {
					total++
					ok, _ := kernel.ValidSowaCategory([]string{p1, p2, p3, lf})
					if ok {
						passing++
					}
				}
			}
		}
	}

	it.Then(t).
		Should(it.Equal(144, total)).
		Should(it.Equal(12, passing))
}

// contract C1.2: a structurally-valid-but-illegal tuple (every word belongs
// to the right word-group, but the four together are not one of the twelve
// legal rows) is rejected naming both the rejected tuple and the closest
// legal row sharing the longest leading-word prefix.
func TestValidSowaCategorySuggestsClosestLegalCombination(t *testing.T) {
	ok, reason := kernel.ValidSowaCategory([]string{"independent", "physical", "continuant", "purpose"})

	it.Then(t).Should(it.True(!ok))
	it.Then(t).
		Should(it.String(reason).Contain("independent, physical, continuant, purpose")).
		Should(it.String(reason).Contain("independent, physical, continuant, object"))
}
