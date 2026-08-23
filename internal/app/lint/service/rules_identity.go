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

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// checkSourceCitekey reports a RuleSourceCitekey violation when a source
// node's ID does not equal its file's actual on-disk basename
// (research.md D6).
func checkSourceCitekey(node core.Node, path, basename string, raw []byte) []kernel.Violation {
	if node.Type != "Source" || node.ID == basename {
		return nil
	}
	return []kernel.Violation{{
		Rule:    kernel.RuleSourceCitekey,
		Path:    path,
		Line:    locateFrontMatterField(raw, "id"),
		Message: fmt.Sprintf("source id %q does not match its filename basename %q", node.ID, basename),
	}}
}

// checkIdentityCharset reports a RuleIdentityCharset violation when node's
// own identity contains one or more ARCNET-CORE §7.1 forbidden filesystem
// characters (contract C2.2, data-model.md §2/§4), naming every offending
// character and its 1-indexed rune position, not only the first. The
// violation's line is located via the quoted "@id" front-matter key
// (research.md D4), matching lint.go's own "@id"-basename-mismatch check
// rather than this file's bare-"id" convention used elsewhere.
func checkIdentityCharset(node core.Node, path string, raw []byte) []kernel.Violation {
	pairs := core.ScanIdentityCharset(node.ID)
	if len(pairs) == 0 {
		return nil
	}
	return []kernel.Violation{{
		Rule:    kernel.RuleIdentityCharset,
		Path:    path,
		Line:    locateFrontMatterField(raw, `"@id"`),
		Message: fmt.Sprintf("identity %q %s", node.ID, core.FormatIdentityCharsetViolation(pairs)),
	}}
}

// checkEntityCategory reports a RuleEntityCategory violation when an
// entity node's category attribute is missing, is not a four-element
// sequence, or fails the fixed positional Sowa word-sets (research.md D7).
func checkEntityCategory(node core.Node, path string, raw []byte) []kernel.Violation {
	if node.Type != "Entity" {
		return nil
	}

	line := locateFrontMatterField(raw, "category")

	violation := func(msg string) []kernel.Violation {
		return []kernel.Violation{{Rule: kernel.RuleEntityCategory, Path: path, Line: line, Message: msg}}
	}

	items, ok := node.Attrs["category"]
	if !ok {
		return violation("category field is missing")
	}

	words := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.Value.(string)
		if !ok {
			return violation("category words must all be strings")
		}
		words = append(words, s)
	}

	if ok, reason := kernel.ValidSowaCategory(words); !ok {
		return violation(reason)
	}
	return nil
}
