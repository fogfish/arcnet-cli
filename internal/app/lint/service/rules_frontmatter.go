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
// node's Kind is absent from the resolved rules (research.md D11, spec
// FR-011/FR-018).
func checkUnrecognizedKind(node core.Node, path string, rules core.MergeRuleSet) []kernel.Violation {
	if _, ok := rules.Lookup(node.Kind); ok {
		return nil
	}
	return []kernel.Violation{{
		Rule:    kernel.RuleUnrecognizedKind,
		Path:    path,
		Line:    0,
		Message: fmt.Sprintf("kind %q is not recognized by this graph's configuration", node.Kind),
	}}
}
