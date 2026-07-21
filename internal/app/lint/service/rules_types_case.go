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
