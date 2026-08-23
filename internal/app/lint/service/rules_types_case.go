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
	"sort"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	schemakernel "github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var typeCasePattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// checkNodeTypeCase reports a RuleTypeCase violation when node's own Type
// does not begin with an uppercase letter (spec 019 FR-007), mirroring
// checkPredicateCase's shape, inverted onto the type axis.
func checkNodeTypeCase(node core.Node, path string) []kernel.Violation {
	if typeCasePattern.MatchString(node.Type) {
		return nil
	}
	return []kernel.Violation{{
		Rule:    kernel.RuleTypeCase,
		Path:    path,
		Line:    0,
		Message: fmt.Sprintf("type %q is not CamelCase", node.Type),
	}}
}

// checkSchemaIdentityCharset reports one graph-spanning RuleIdentityCharset
// violation per index.Types key, then per index.Predicates key, that
// contains an ARCNET-CORE §7.1 forbidden filesystem character (contract
// C2.2, data-model.md §2/§3, research.md D3) — mirroring
// checkSchemaTypeCase's graph-spanning shape: no real file is opened, the
// identity string checked is the map key itself, and Path/Line are
// synthesized exactly as every other graph-spanning rule already does.
// Types are iterated before Predicates, both in sorted-key order, for
// deterministic output.
func checkSchemaIdentityCharset(index core.Index) []kernel.Violation {
	var out []kernel.Violation

	typeNames := make([]string, 0, len(index.Types))
	for name := range index.Types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)
	for _, name := range typeNames {
		out = append(out, schemaIdentityCharsetViolations(name, schemakernel.TypesDir)...)
	}

	predicateNames := make([]string, 0, len(index.Predicates))
	for name := range index.Predicates {
		predicateNames = append(predicateNames, name)
	}
	sort.Strings(predicateNames)
	for _, name := range predicateNames {
		out = append(out, schemaIdentityCharsetViolations(name, schemakernel.PredicatesDir)...)
	}

	return out
}

func schemaIdentityCharsetViolations(name, dir string) []kernel.Violation {
	pairs := core.ScanIdentityCharset(name)
	if len(pairs) == 0 {
		return nil
	}
	return []kernel.Violation{{
		Rule:    kernel.RuleIdentityCharset,
		Path:    dir + "/" + name + ".md",
		Line:    0,
		Message: fmt.Sprintf("identity %q %s", name, core.FormatIdentityCharsetViolation(pairs)),
	}}
}

// checkSchemaTypeCase reports one graph-spanning RuleTypeCase violation per
// index.Types key that does not begin with an uppercase letter (spec 019
// FR-006), iterated in sorted-key order for deterministic output —
// mirroring checkUniqueBasenames' graph-spanning shape.
func checkSchemaTypeCase(index core.Index) []kernel.Violation {
	names := make([]string, 0, len(index.Types))
	for name := range index.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []kernel.Violation
	for _, name := range names {
		if typeCasePattern.MatchString(name) {
			continue
		}
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleTypeCase,
			Path:    schemakernel.TypesDir + "/" + name + ".md",
			Line:    0,
			Message: fmt.Sprintf("type %q is not CamelCase", name),
		})
	}
	return out
}
