//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var camelCasePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// predicateOccurrence is one distinct predicate token found in a node's
// Edges/HRefs, paired with the line it was located at.
type predicateOccurrence struct {
	predicate string
	line      int
}

// predicateOccurrences collects every distinct predicate a node declares:
// every Edges/HRefs entry with a non-empty Predicate (research.md D9).
func predicateOccurrences(node core.Node, raw []byte) []predicateOccurrence {
	var out []predicateOccurrence
	for _, l := range node.Edges {
		if l.Predicate != "" {
			out = append(out, predicateOccurrence{predicate: l.Predicate, line: locatePredicateToken(raw, l.Predicate)})
		}
	}
	for _, l := range node.HRefs {
		if l.Predicate != "" {
			out = append(out, predicateOccurrence{predicate: l.Predicate, line: locatePredicateToken(raw, l.Predicate)})
		}
	}
	return out
}

// checkPredicateCase reports one RulePredicateCase violation per distinct
// non-camelCase predicate a node declares (research.md D9/spec FR-007).
func checkPredicateCase(node core.Node, path string, raw []byte) []kernel.Violation {
	var out []kernel.Violation
	seen := map[string]bool{}
	for _, occ := range predicateOccurrences(node, raw) {
		if seen[occ.predicate] {
			continue
		}
		seen[occ.predicate] = true
		if !camelCasePattern.MatchString(occ.predicate) {
			out = append(out, kernel.Violation{
				Rule:    kernel.RulePredicateCase,
				Path:    path,
				Line:    occ.line,
				Message: fmt.Sprintf("predicate %q is not camelCase", occ.predicate),
			})
		}
	}
	return out
}

// checkPredicateRegistered reports one RulePredicateRegistered violation
// per distinct predicate a node declares that is absent from registry
// (research.md D9/spec FR-008).
func checkPredicateRegistered(node core.Node, path string, raw []byte, registry map[string]core.PredicateDef) []kernel.Violation {
	var out []kernel.Violation
	seen := map[string]bool{}
	for _, occ := range predicateOccurrences(node, raw) {
		if seen[occ.predicate] {
			continue
		}
		seen[occ.predicate] = true
		if _, ok := registry[occ.predicate]; !ok {
			out = append(out, kernel.Violation{
				Rule:    kernel.RulePredicateRegistered,
				Path:    path,
				Line:    occ.line,
				Message: fmt.Sprintf("predicate %q is not registered in %s", occ.predicate, "_schema/predicates/"),
			})
		}
	}
	return out
}

// checkCitationPredicate reports one RuleCitationPredicate violation per
// HRefs entry whose Predicate is non-empty but is not registered in registry
// with a "cito:"-prefixed Aligned value (spec FR-006/FR-007, research.md
// D3) — the valid citation-predicate vocabulary is sourced dynamically from
// the graph's own schema, with no built-in fallback list.
func checkCitationPredicate(node core.Node, path string, raw []byte, registry map[string]core.PredicateDef) []kernel.Violation {
	var out []kernel.Violation
	for _, l := range node.HRefs {
		if l.Predicate == "" || strings.HasPrefix(registry[l.Predicate].Aligned, "cito:") {
			continue
		}
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleCitationPredicate,
			Path:    path,
			Line:    locatePredicateToken(raw, l.Predicate),
			Message: fmt.Sprintf("citation predicate %q is not a recognized cito-aligned predicate", l.Predicate),
		})
	}
	return out
}
