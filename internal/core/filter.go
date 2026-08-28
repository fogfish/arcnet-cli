//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Matcher constrains one position (Source, Predicate, or Target) of a
// Statement. A zero-value Matcher is a wildcard: it matches every string.
// Otherwise it matches s iff s case-insensitively equals any Values entry,
// or s matches any Patterns entry — the two slices are OR'd together, and
// either or both may be populated (research.md D1).
type Matcher struct {
	Values   []string
	Patterns []*regexp.Regexp
}

// IsWildcard reports whether m constrains nothing (both slices empty).
func (m Matcher) IsWildcard() bool {
	return len(m.Values) == 0 && len(m.Patterns) == 0
}

// Match reports whether s satisfies m: wildcard, a case-insensitive Values
// match, or a Patterns match.
func (m Matcher) Match(s string) bool {
	if m.IsWildcard() {
		return true
	}
	for _, v := range m.Values {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	for _, p := range m.Patterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}

// Statement is one clause of a Filter: independently-wildcardable
// constraints on the (Source, Predicate, Target) triple of a fact.
type Statement struct {
	Source    Matcher
	Predicate Matcher
	Target    Matcher
}

// IsTraversalConstraint reports whether s constrains only Predicate
// (Source and Target both wildcard) — research.md D3. Such a statement
// scopes which structural connections BFS follows, rather than narrowing
// which already-reached nodes survive into a flat result.
func (s Statement) IsTraversalConstraint() bool {
	return s.Source.IsWildcard() && !s.Predicate.IsWildcard() && s.Target.IsWildcard()
}

// match reports whether s is satisfied by the (source, predicate, target)
// triple.
func (s Statement) match(source, predicate, target string) bool {
	return s.Source.Match(source) && s.Predicate.Match(predicate) && s.Target.Match(target)
}

// Filter is the optional, composable node-selection criteria shared by
// every VISION.md Filtering-section command and by arc serve's MCP tools.
// A zero-value Filter{} matches every node.
type Filter struct {
	Statements []Statement
}

// Match reports whether node satisfies every statement in f (AND across
// statements; each statement independently satisfied by any one of the
// node's own attribute or edge facts — spec FR-005). Match mutates neither
// f nor node.
func (f Filter) Match(node Node) bool {
	for _, s := range f.Statements {
		if !statementSatisfiedBy(s, node) {
			return false
		}
	}
	return true
}

// statementSatisfiedBy reports whether any fact carried by node — the
// synthesized type fact (research.md D2), an attribute fact, or an edge
// fact — satisfies s.
func statementSatisfiedBy(s Statement, node Node) bool {
	if s.match(node.ID, "type", node.Type) {
		return true
	}
	for name, preds := range node.Attrs {
		for _, p := range preds {
			if p.Value == nil {
				continue
			}
			if s.match(node.ID, name, toString(p.Value)) {
				return true
			}
		}
	}
	for _, e := range node.Edges {
		if s.match(node.ID, e.Predicate, e.Target) {
			return true
		}
	}
	return false
}

// Fact is one (property, value) pair carried by a node that satisfied a
// Filter statement (specs/028-node-match-filter) — node_match's unit of
// evidence for why a node matched.
type Fact struct {
	Property string
	Value    string
}

// MatchingFacts returns every distinct fact on node — its synthesized type
// fact, an attribute fact, or an edge fact — that satisfies at least one
// statement in f, deduplicated by (Property, Value) and sorted for
// deterministic output (research.md D1/D3, specs/028-node-match-filter).
// MatchingFacts mutates neither f nor node; it does not itself require
// f.Match(node) — callers that want facts only for nodes satisfying every
// statement call Match first (research.md D4).
func (f Filter) MatchingFacts(node Node) []Fact {
	seen := map[Fact]bool{}
	var out []Fact
	add := func(property, value string) {
		fact := Fact{Property: property, Value: value}
		if !seen[fact] {
			seen[fact] = true
			out = append(out, fact)
		}
	}

	for _, s := range f.Statements {
		if s.match(node.ID, "type", node.Type) {
			add("type", node.Type)
		}
		for name, preds := range node.Attrs {
			for _, p := range preds {
				if p.Value == nil {
					continue
				}
				value := toString(p.Value)
				if s.match(node.ID, name, value) {
					add(name, value)
				}
			}
		}
		for _, e := range node.Edges {
			if s.match(node.ID, e.Predicate, e.Target) {
				add(e.Predicate, e.Target)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Property != out[j].Property {
			return out[i].Property < out[j].Property
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// Traversal returns the subset of f.Statements that are traversal
// constraints (research.md D3) — used to gate BFS edge admission.
func (f Filter) Traversal() Filter {
	var out Filter
	for _, s := range f.Statements {
		if s.IsTraversalConstraint() {
			out.Statements = append(out.Statements, s)
		}
	}
	return out
}

// Narrowing returns the subset of f.Statements that are NOT traversal
// constraints — used for flat-inclusion narrowing of already-reached,
// non-seed candidates. Traversal() and Narrowing() partition f.Statements;
// their statement counts sum to len(f.Statements).
func (f Filter) Narrowing() Filter {
	var out Filter
	for _, s := range f.Statements {
		if !s.IsTraversalConstraint() {
			out.Statements = append(out.Statements, s)
		}
	}
	return out
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
