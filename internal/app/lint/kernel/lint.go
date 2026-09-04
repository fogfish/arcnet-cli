//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the lint (graph conformance validation) domain's
// value types.
package kernel

import (
	"fmt"
	"strings"
)

// Rule identifies exactly one CORE §14 checklist item, so every Violation
// names precisely which rule fired without a second lookup table.
type Rule string

const (
	RuleFrontMatter         Rule = "frontMatter"
	RuleUniqueBasename      Rule = "uniqueBasename"
	RuleLinkResolves        Rule = "linkResolves"
	RuleSourceCitekey       Rule = "sourceCitekey"
	RuleEntityCategory      Rule = "entityCategory"
	RuleDerivedProvenance   Rule = "derivedProvenance"
	RulePredicateCase       Rule = "predicateCase"
	RulePredicateRegistered Rule = "predicateRegistered"
	RuleCitationPredicate   Rule = "citationPredicate"
	RuleUnrecognizedKind    Rule = "unrecognizedKind"
	RuleIngestCommit        Rule = "ingestCommit"
	RuleMergeConflict       Rule = "mergeConflict"
	RuleTypeRequires        Rule = "typeRequires"
	RuleTypeOptional        Rule = "typeOptional"
	RuleIdentityQuoting     Rule = "identityQuoting"
	RulePredicateRole       Rule = "predicateRole"
	RuleTypeCase            Rule = "typeCase"
	RuleIdentityCharset     Rule = "identityCharset"
)

// Violation is the domain value one failed check produces.
type Violation struct {
	// Rule is the checklist item that failed.
	Rule Rule `json:"rule"`
	// Path is the node file path, relative to the graph root; empty when
	// the violation spans more than one file (RuleUniqueBasename).
	Path string `json:"path"`
	// Line is the 1-based line number within Path; 0 means "not
	// applicable" (spec FR-015).
	Line int `json:"line"`
	// Message is a human-readable detail.
	Message string `json:"message"`
	// RelatedPaths is populated only for violations spanning more than one
	// file; empty otherwise.
	RelatedPaths []string `json:"relatedPaths"`
}

// NodeStatus is one enumerated node's overall outcome, the unit --verbose
// output lists one of per node.
type NodeStatus struct {
	// Path is relative to the graph root.
	Path string `json:"path"`
	// ID is the parsed node identity; empty when RuleFrontMatter itself
	// failed and core.ParseNode never ran.
	ID string `json:"id"`
	// Type is empty when unparseable.
	Type string `json:"type"`
	// Violations is empty when this node passed every applicable check.
	Violations []Violation `json:"violations"`
}

// LintResult is the domain value component.go's Lint returns to
// cmd/arc/lint, rendered by bios.Registry[LintResult].
type LintResult struct {
	// Root is the graph root that was linted.
	Root string `json:"root"`
	// Nodes holds every enumerated node, in walk order.
	Nodes []NodeStatus `json:"nodes"`
	// Violations is a flattened view of every NodeStatus.Violations plus
	// file-spanning violations with no single owning node
	// (RuleUniqueBasename).
	Violations []Violation `json:"violations"`
	// Passing is the count of nodes with zero violations.
	Passing int `json:"passing"`
	// Failing is the count of nodes with at least one violation.
	Failing int `json:"failing"`
	// Foreign holds markdown files under the graph root that carry no
	// graph identity at all — the host project's own README, ADRs, design
	// notes (spec 031 FR-032). They were walked, recognized as not-ours,
	// and skipped: they are counted in neither Passing nor Failing, appear
	// in neither Nodes nor Violations, and never affect exit status. This
	// is a record of what lint declined to check, not a defect list (spec
	// 031 FR-035). Additive to the --json contract, exactly as spec 031
	// FR-028's precedent requires.
	Foreign []string `json:"foreign"`
}

// NewLintResult derives Violations/Passing/Failing from nodes (in walk
// order) plus any graph-spanning violations with no single owning node
// (e.g. RuleUniqueBasename) — graphSpanning entries are listed first,
// matching the Human renderer's expected order.
func NewLintResult(root string, nodes []NodeStatus, graphSpanning ...Violation) LintResult {
	return NewLintResultWithForeign(root, nodes, nil, graphSpanning...)
}

// NewLintResultWithForeign is NewLintResult plus the foreign-file index
// (spec 031 FR-035). It is a separate constructor rather than a fifth
// parameter on NewLintResult so the existing three-argument callers — and
// their tests — keep compiling unchanged.
func NewLintResultWithForeign(root string, nodes []NodeStatus, foreign []string, graphSpanning ...Violation) LintResult {
	violations := make([]Violation, 0, len(graphSpanning))
	violations = append(violations, graphSpanning...)

	passing, failing := 0, 0
	for _, n := range nodes {
		if len(n.Violations) == 0 {
			passing++
			continue
		}
		failing++
		violations = append(violations, n.Violations...)
	}

	return LintResult{
		Root:       root,
		Nodes:      nodes,
		Violations: violations,
		Passing:    passing,
		Failing:    failing,
		Foreign:    foreign,
	}
}

// sowaCategories is ARCNET-CORE §10.2's closed set of twelve legal four-word
// Sowa category combinations (data-model.md §1, research.md D1) — replacing
// the four independent positional word-set checks this codebase used to run,
// which accepted 144 combinations instead of the twelve CORE actually
// defines.
var sowaCategories = [12][4]string{
	{"independent", "physical", "continuant", "object"},
	{"independent", "physical", "occurrent", "process"},
	{"independent", "abstract", "continuant", "schema"},
	{"independent", "abstract", "occurrent", "script"},
	{"relative", "physical", "continuant", "juncture"},
	{"relative", "physical", "occurrent", "participation"},
	{"relative", "abstract", "continuant", "description"},
	{"relative", "abstract", "occurrent", "history"},
	{"mediating", "physical", "continuant", "structure"},
	{"mediating", "physical", "occurrent", "situation"},
	{"mediating", "abstract", "continuant", "reason"},
	{"mediating", "abstract", "occurrent", "purpose"},
}

// ValidSowaCategory reports whether words is one of sowaCategories' twelve
// legal rows, byte-for-byte (contract C1.1). ok is false with a
// human-readable reason otherwise: C1.3's unchanged wrong-length message when
// words does not have exactly four elements, or C1.2's rejected-tuple-plus-
// closest-legal-suggestion message when it does but matches no row.
func ValidSowaCategory(words []string) (ok bool, reason string) {
	if len(words) != 4 {
		return false, fmt.Sprintf("category must decode to exactly four Sowa words, found %d", len(words))
	}

	tuple := [4]string{words[0], words[1], words[2], words[3]}
	for _, row := range sowaCategories {
		if row == tuple {
			return true, ""
		}
	}

	suggestion := closestSowaCategory(tuple)
	return false, fmt.Sprintf(
		"[%s] is not one of the twelve legal Sowa category combinations; the closest legal combination is [%s]",
		strings.Join(words, ", "), strings.Join(suggestion[:], ", "),
	)
}

// closestSowaCategory finds the sowaCategories row sharing the longest
// leading-word prefix with tuple, first table-order match winning any tie
// (research.md D2, contract C1.2).
func closestSowaCategory(tuple [4]string) [4]string {
	best := sowaCategories[0]
	bestScore := -1
	for _, row := range sowaCategories {
		score := 0
		for i := 0; i < 4; i++ {
			if row[i] != tuple[i] {
				break
			}
			score++
		}
		if score > bestScore {
			bestScore = score
			best = row
		}
	}
	return best
}
