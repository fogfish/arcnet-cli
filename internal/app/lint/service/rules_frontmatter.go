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
	"strings"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// checkUniqueBasenames reports one RuleUniqueBasename violation per
// basename shared by more than one file (research.md D4) — a graph-
// spanning violation with no single owning node (no line number applies),
// naming every colliding path.
func checkUniqueBasenames(index map[string][]string) []kernel.Violation {
	basenames := make([]string, 0, len(index))
	for b := range index {
		basenames = append(basenames, b)
	}
	sort.Strings(basenames)

	var out []kernel.Violation
	for _, b := range basenames {
		paths := index[b]
		if len(paths) <= 1 {
			continue
		}
		sorted := append([]string(nil), paths...)
		sort.Strings(sorted)
		out = append(out, kernel.Violation{
			Rule:         kernel.RuleUniqueBasename,
			Message:      fmt.Sprintf("basename %q is used by more than one file: %s", b, strings.Join(sorted, ", ")),
			RelatedPaths: sorted,
		})
	}
	return out
}

// checkUnrecognizedKind reports a RuleUnrecognizedKind violation when
// node's Type is absent from the resolved index (research.md D11, spec
// FR-011/FR-018).
func checkUnrecognizedKind(node core.Node, path string, index core.Index) []kernel.Violation {
	if _, ok := index.Types[node.Type]; ok {
		return nil
	}
	return []kernel.Violation{{
		Rule:    kernel.RuleUnrecognizedKind,
		Path:    path,
		Line:    0,
		Message: fmt.Sprintf("kind %q is not recognized by this graph's configuration", node.Type),
	}}
}

// checkBareIdentityKeys reports a RuleIdentityQuoting violation for each of
// "@id"/"@type" written as a bare (unquoted) YAML key. Called only from
// service.Lint's core.ParseNode-error branch, before the generic
// RuleFrontMatter violation is recorded: a leading "@" is a reserved YAML
// plain-scalar indicator character, so a bare "@id"/"@type" key does not
// silently decode — it makes the whole document invalid YAML, and
// core.ParseNode fails outright (research.md D1's own premise, that this
// defect is merely "invisible post-parse", does not hold — verified against
// the real gopkg.in/yaml.v3 parser: it is a hard syntax error). This gives a
// specific, actionable message in place of the raw YAML lexer error for this
// one well-understood root cause; every other parse failure (a genuinely
// missing "@id"/"@type" — already a clear, distinct message via
// core.identityFields — the legacy "kind" field, or any other malformed
// YAML) is unambiguously different (no bare "@id"/"@type" line is present)
// and keeps reporting through the existing RuleFrontMatter path unchanged.
func checkBareIdentityKeys(path string, raw []byte) []kernel.Violation {
	var out []kernel.Violation
	for _, key := range []string{"@id", "@type"} {
		line := locateUnquotedIdentityKey(raw, key)
		if line == 0 {
			continue
		}
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleIdentityQuoting,
			Path:    path,
			Line:    line,
			Message: fmt.Sprintf("%q must be a quoted YAML string key, found it unquoted", key),
		})
	}
	return out
}

// checkIdentityKeyQuoting reports a RuleIdentityQuoting violation for each of
// "@id"/"@type" that is either missing or present but written as a bare
// (unquoted) YAML key (spec FR-004/FR-005, research.md D1). Only ever called
// for nodes that reached service.Lint's post-parse checks: a node whose
// front matter fails to parse at all (including a genuinely absent "@id"/
// "@type", or a bare key — see checkBareIdentityKeys) already produces a
// violation and never reaches this point (service.Lint's existing
// continue-on-parse-failure control flow), so neither branch below is
// reachable through the real pipeline — this function exists so its
// contract is complete and unit-testable in isolation.
func checkIdentityKeyQuoting(node core.Node, path string, raw []byte) []kernel.Violation {
	var out []kernel.Violation
	for _, key := range []string{"@id", "@type"} {
		if line := locateUnquotedIdentityKey(raw, key); line > 0 {
			out = append(out, kernel.Violation{
				Rule:    kernel.RuleIdentityQuoting,
				Path:    path,
				Line:    line,
				Message: fmt.Sprintf("%q must be a quoted YAML string key, found it unquoted", key),
			})
			continue
		}
		if locateIdentityKey(raw, key) > 0 {
			continue
		}
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleIdentityQuoting,
			Path:    path,
			Line:    locateFrontMatterDelimiter(raw),
			Message: fmt.Sprintf("front matter is missing the mandatory %q key", key),
		})
	}
	return out
}
