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

	"github.com/fogfish/arcnet-cli/internal/core"
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
	// Kind is empty when unparseable.
	Kind core.Kind `json:"kind"`
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
}

// NewLintResult derives Violations/Passing/Failing from nodes (in walk
// order) plus any graph-spanning violations with no single owning node
// (e.g. RuleUniqueBasename) — graphSpanning entries are listed first,
// matching the Human renderer's expected order.
func NewLintResult(root string, nodes []NodeStatus, graphSpanning ...Violation) LintResult {
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
	}
}

// sowaPosition1/2/3/Leaf are CORE §9.2.1's fixed four-word Sowa category
// vocabulary, decoded positionally (research.md D7) — this codebase's own
// already-established on-disk convention (a literal four-element word
// array), not the compact "xyz:leaf" code CORE's prose merely uses to
// explain the code's meaning.
var (
	sowaPosition1 = map[string]bool{"independent": true, "relative": true, "mediating": true}
	sowaPosition2 = map[string]bool{"physical": true, "abstract": true}
	sowaPosition3 = map[string]bool{"continuant": true, "occurrent": true}
	sowaLeaf      = map[string]bool{
		"object": true, "process": true, "schema": true, "script": true,
		"juncture": true, "participation": true, "description": true, "history": true,
		"structure": true, "situation": true, "reason": true, "purpose": true,
	}
)

// ValidSowaCategory reports whether words is a valid CORE §9.2.1 four-word
// Sowa category, positionally checked against the fixed word-sets above. ok
// is false with a human-readable reason otherwise.
func ValidSowaCategory(words []string) (ok bool, reason string) {
	if len(words) != 4 {
		return false, fmt.Sprintf("category must decode to exactly four Sowa words, found %d", len(words))
	}
	if !sowaPosition1[words[0]] {
		return false, fmt.Sprintf("%q is not a valid first Sowa word (independent/relative/mediating)", words[0])
	}
	if !sowaPosition2[words[1]] {
		return false, fmt.Sprintf("%q is not a valid second Sowa word (physical/abstract)", words[1])
	}
	if !sowaPosition3[words[2]] {
		return false, fmt.Sprintf("%q is not a valid third Sowa word (continuant/occurrent)", words[2])
	}
	if !sowaLeaf[words[3]] {
		return false, fmt.Sprintf("%q is not a valid fourth (leaf) Sowa word", words[3])
	}
	return true, ""
}
