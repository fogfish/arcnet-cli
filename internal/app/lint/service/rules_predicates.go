//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// predicatesPath is where a graph's controlled predicate vocabulary lives
// (CORE §7.3), created as an empty stub by arc init.
const predicatesPath = "_meta/predicates.md"

var predicateBulletPattern = regexp.MustCompile("^- `([^`]+)`")

// parsePredicateRegistry parses _meta/predicates.md as a bullet list, one
// predicate per item, named by its first inline-code span (research.md D9).
// An absent file is not an error — it resolves to an empty registry (every
// predicate unregistered); a genuine read failure is ErrPredicatesUnreadable.
func parsePredicateRegistry(store fsys.Store) (map[string]bool, error) {
	f, err := store.Open(predicatesPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]bool{}, nil
		}
		return nil, ErrPredicatesUnreadable.With(err, predicatesPath)
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, ErrPredicatesUnreadable.With(err, predicatesPath)
	}

	registry := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := predicateBulletPattern.FindStringSubmatch(line); m != nil {
			registry[m[1]] = true
		}
	}
	return registry, nil
}

// citoPredicates is CORE §8's fixed, cito:-aligned citation predicate
// vocabulary (research.md D10).
var citoPredicates = map[string]bool{
	"cites": true, "citesAsEvidence": true, "citesAsAuthority": true,
	"supports": true, "confirms": true, "extends": true,
	"critiques": true, "disputes": true, "refutes": true, "isCitedBy": true,
}

var camelCasePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// predicateOccurrence is one distinct predicate token found in a node's
// Edges/HRefs (structural, inline) or Links (predicate-grouped block key),
// paired with the line it was located at.
type predicateOccurrence struct {
	predicate string
	line      int
}

// predicateOccurrences collects every distinct predicate a node declares:
// every Edges/HRefs entry with a non-empty Predicate, plus every Links
// block key (research.md D9).
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
	for _, key := range sortedLinkKeys(node.Links) {
		block := node.Links[key]
		line := locatePredicateToken(raw, key)
		if line == 0 {
			line = locateBlockLabel(raw, block.Title)
		}
		out = append(out, predicateOccurrence{predicate: key, line: line})
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
func checkPredicateRegistered(node core.Node, path string, raw []byte, registry map[string]bool) []kernel.Violation {
	var out []kernel.Violation
	seen := map[string]bool{}
	for _, occ := range predicateOccurrences(node, raw) {
		if seen[occ.predicate] {
			continue
		}
		seen[occ.predicate] = true
		if !registry[occ.predicate] {
			out = append(out, kernel.Violation{
				Rule:    kernel.RulePredicateRegistered,
				Path:    path,
				Line:    occ.line,
				Message: fmt.Sprintf("predicate %q is not registered in %s", occ.predicate, predicatesPath),
			})
		}
	}
	return out
}

// checkCitationPredicate reports one RuleCitationPredicate violation per
// HRefs entry whose Predicate is non-empty but not one of CORE §8's fixed
// cito:-aligned set (research.md D10/spec FR-009).
func checkCitationPredicate(node core.Node, path string, raw []byte) []kernel.Violation {
	var out []kernel.Violation
	for _, l := range node.HRefs {
		if l.Predicate == "" || citoPredicates[l.Predicate] {
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
