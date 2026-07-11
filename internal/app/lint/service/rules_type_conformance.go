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
	"sort"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// occurrenceCategory is one of the four structural positions a predicate
// occurrence can be found in (data-model.md D5), derived from which of
// core.Node's fields produced it.
type occurrenceCategory string

const (
	categoryMeta       occurrenceCategory = "meta"
	categoryText       occurrenceCategory = "text"
	categoryEdgeOrLink occurrenceCategory = "edge-or-link"
	categoryHRef       occurrenceCategory = "href"
)

// occurrence is one located appearance of a predicate on a node.
type occurrence struct {
	category       occurrenceCategory
	line           int
	citationTagged bool
}

// enumerateOccurrences collects, per distinct predicate name a node carries,
// every located occurrence plus its structural category (data-model.md D5) —
// shared by checkTypeRequires/checkTypeOptional/checkPredicateRole so each
// only walks the node once. An HRefs entry with a non-empty Predicate is the
// pre-existing inline citation-tagging convention ("[predicate:: [[Target]]]"
// embedded in prose, research.md D4): it still counts as the predicate being
// present on the node (spec Assumptions), but each such occurrence is marked
// citationTagged so checkPredicateRole can exempt it. An HRefs entry with an
// empty Predicate (a bare "[[Target]]" wikilink) carries no predicate name at
// all and is not a predicate occurrence of anything.
//
// "published" is special-cased: internal/core's parser extracts it into
// Node.Published (a dedicated field), never leaving it in Attrs — without
// this, a source node's own Required "published" predicate would be
// permanently unsatisfiable regardless of the front matter it actually
// carries.
func enumerateOccurrences(node core.Node, raw []byte) map[string][]occurrence {
	out := map[string][]occurrence{}

	add := func(predicate string, occ occurrence) {
		if predicate == "" {
			return
		}
		out[predicate] = append(out[predicate], occ)
	}

	if !node.Published.IsZero() {
		add("published", occurrence{category: categoryMeta, line: locateFrontMatterField(raw, "published")})
	}

	for predicate := range node.Attrs {
		add(predicate, occurrence{category: categoryMeta, line: locateFrontMatterField(raw, predicate)})
	}

	for predicate := range node.Texts {
		add(predicate, occurrence{category: categoryText, line: locateOccurrenceFallback(raw, locateBlockLabel(raw, predicate))})
	}

	for _, l := range node.Edges {
		add(l.Predicate, occurrence{category: categoryEdgeOrLink, line: locateOccurrenceFallback(raw, locatePredicateToken(raw, l.Predicate))})
	}

	for _, l := range node.HRefs {
		if l.Predicate == "" {
			continue
		}
		add(l.Predicate, occurrence{category: categoryHRef, line: locateOccurrenceFallback(raw, locatePredicateToken(raw, l.Predicate)), citationTagged: true})
	}

	return out
}

// locateOccurrenceFallback substitutes the front-matter delimiter line when
// line is 0 (not located), consistent with every other node-level-only
// violation's fallback (research.md D3).
func locateOccurrenceFallback(raw []byte, line int) int {
	if line > 0 {
		return line
	}
	return locateFrontMatterDelimiter(raw)
}

// sortedPredicateNames returns occ's keys sorted alphabetically, so
// multi-violation output is deterministic.
func sortedPredicateNames(occ map[string][]occurrence) []string {
	names := make([]string, 0, len(occ))
	for name := range occ {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// checkTypeRequires reports one RuleTypeRequires violation per predicate
// node's own type's "## Requires" section lists that is absent from the
// node (spec FR-001); skipped entirely when node.Type is not registered
// (spec FR-003 — the existing unrecognizedKind check already covers that
// gap).
func checkTypeRequires(node core.Node, path string, raw []byte, index core.Index) []kernel.Violation {
	def, ok := index.Types[node.Type]
	if !ok {
		return nil
	}

	occ := enumerateOccurrences(node, raw)

	var out []kernel.Violation
	for _, predicate := range def.Required {
		if _, present := occ[predicate]; present {
			continue
		}
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleTypeRequires,
			Path:    path,
			Line:    locateFrontMatterDelimiter(raw),
			Message: fmt.Sprintf("type %q requires predicate %q, but this node does not carry it", node.Type, predicate),
		})
	}
	return out
}

// checkTypeOptional reports one RuleTypeOptional violation per distinct
// predicate the node carries that its own type's schema lists under neither
// "## Requires" nor "## Optional" (spec FR-002); skipped entirely when
// node.Type is not registered (spec FR-003). "@id"/"@type" never appear in
// enumerateOccurrences' output (they are stripped from the manifest before
// Attrs is ever populated, core.identityFields), so they are structurally
// never flagged here.
func checkTypeOptional(node core.Node, path string, raw []byte, index core.Index) []kernel.Violation {
	def, ok := index.Types[node.Type]
	if !ok {
		return nil
	}

	permitted := map[string]bool{}
	for _, p := range def.Required {
		permitted[p] = true
	}
	for _, p := range def.Optional {
		permitted[p] = true
	}

	occ := enumerateOccurrences(node, raw)

	var out []kernel.Violation
	for _, predicate := range sortedPredicateNames(occ) {
		if permitted[predicate] {
			continue
		}
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleTypeOptional,
			Path:    path,
			Line:    occ[predicate][0].line,
			Message: fmt.Sprintf("predicate %q is not permitted by type %q (not listed under its Requires or Optional)", predicate, node.Type),
		})
	}
	return out
}

// validPredicateRoles is CORE §5's fixed five-value role vocabulary
// (research.md D7).
var validPredicateRoles = map[string]bool{
	"meta": true, "text": true, "href": true, "edge": true, "link": true,
}

// roleMatchesCategory reports whether an occurrence found in category
// satisfies a predicate registered with role (data-model.md D5): Edges
// occurrences satisfy either "edge" or "link" (research.md D5 — the
// flat-vs-grouped rendering distinction is not preserved at parse time).
func roleMatchesCategory(role string, category occurrenceCategory) bool {
	switch category {
	case categoryMeta:
		return role == "meta"
	case categoryText:
		return role == "text"
	case categoryHRef:
		return role == "href"
	case categoryEdgeOrLink:
		return role == "edge" || role == "link"
	default:
		return true
	}
}

// categoryLabel renders category for a predicateRole violation's message —
// an edge-or-link occurrence is always described as "edge" (quickstart.md
// Scenario 6), since that is the actual structural shape (a body list entry)
// regardless of whether the mismatched role was "edge" or "link".
func categoryLabel(category occurrenceCategory) string {
	if category == categoryEdgeOrLink {
		return "edge"
	}
	return string(category)
}

// checkPredicateRole reports one RulePredicateRole violation per occurrence
// whose structural position doesn't match its predicate's own schema-
// declared role (spec FR-008). Skipped per-predicate when unregistered (spec
// FR-009 — the existing predicateRegistered check already covers that gap)
// or when its registered Role is empty/unrecognized (research.md D7).
// Skipped per-occurrence when the occurrence is a citation-tagged inline
// HRefs entry (research.md D4).
func checkPredicateRole(node core.Node, path string, raw []byte, index core.Index) []kernel.Violation {
	occ := enumerateOccurrences(node, raw)

	var out []kernel.Violation
	for _, predicate := range sortedPredicateNames(occ) {
		def, ok := index.Predicates[predicate]
		if !ok || !validPredicateRoles[def.Role] {
			continue
		}
		for _, o := range occ[predicate] {
			if o.citationTagged {
				continue
			}
			if roleMatchesCategory(def.Role, o.category) {
				continue
			}
			out = append(out, kernel.Violation{
				Rule:    kernel.RulePredicateRole,
				Path:    path,
				Line:    o.line,
				Message: fmt.Sprintf("predicate %q is registered with role %q, but appears as a %s occurrence", predicate, def.Role, categoryLabel(o.category)),
			})
		}
	}
	return out
}
