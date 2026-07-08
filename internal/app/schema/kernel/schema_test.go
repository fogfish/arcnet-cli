//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel_test

import (
	"regexp"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var camelCasePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

var validRoles = map[string]bool{"meta": true, "text": true, "href": true, "edge": true, "link": true}

func TestCorePredicateDefsContainsFullCoreVocabulary(t *testing.T) {
	names := []string{
		"tags", "text",
		"published", "created", "updated",
		"mentions", "mentionedIn",
		"broader", "narrower", "isPartOf", "hasPart", "requires", "replaces", "isReplacedBy", "conformsTo", "related",
		"cites", "citesAsEvidence", "citesAsAuthority", "supports", "confirms", "extends", "critiques", "disputes", "refutes", "isCitedBy",
		"title", "abstract", "authors", "url", "doi", "category", "aliases", "definition", "notes", "ref", "year", "status", "relevance", "granularity", "entries", "heading",
		"role", "merge", "label", "aligned", "description", "required", "optional",
	}

	it.Then(t).Should(it.Equal(len(names), len(kernel.CorePredicateDefs)))

	for _, name := range names {
		def, ok := kernel.CorePredicateDefs[name]
		it.Then(t).Should(it.True(ok))
		it.Then(t).
			Should(it.True(validRoles[def.Role])).
			ShouldNot(it.Equal("", string(def.Merge))).
			ShouldNot(it.Equal("", def.Description))
	}
}

func TestCorePredicateDefNamesAreCamelCase(t *testing.T) {
	for name := range kernel.CorePredicateDefs {
		it.Then(t).Should(it.True(camelCasePattern.MatchString(name)))
	}
}

func TestCoreTypeDefsContainsCoreTypesAndSchemaTypesThemselves(t *testing.T) {
	it.Then(t).Should(it.Equal(6, len(kernel.CoreTypeDefs)))

	for _, name := range []string{"source", "entity", "resource", "timeline", "Property", "Class"} {
		def, ok := kernel.CoreTypeDefs[name]
		it.Then(t).Should(it.True(ok))
		it.Then(t).ShouldNot(it.Equal("", def.Description))
	}
}

func TestCoreTypeDefsRequiredListsMatchCoreSection11(t *testing.T) {
	source := kernel.CoreTypeDefs["source"]
	it.Then(t).Should(it.Seq(source.Required).Equal("title", "published", "abstract", "mentions"))

	entity := kernel.CoreTypeDefs["entity"]
	it.Then(t).Should(it.Seq(entity.Required).Equal("category", "definition", "mentionedIn"))

	resource := kernel.CoreTypeDefs["resource"]
	it.Then(t).Should(it.Seq(resource.Required).Equal("ref", "relevance"))

	timeline := kernel.CoreTypeDefs["timeline"]
	it.Then(t).Should(it.Seq(timeline.Required).Equal("granularity", "entries"))
}

// BUG-001 / spec.md FR-018: every role:"text" predicate in the built-in
// vocabulary seeds MergeAppend — role alone must be enough to predict
// dispatch, without reading each predicate's individual assignment.
func TestCorePredicateDefsTextRoleSeedsAppend(t *testing.T) {
	for _, def := range kernel.CorePredicateDefs {
		if def.Role != "text" {
			continue
		}
		it.Then(t).Should(it.Equal(core.MergeAppend, def.Merge))
	}
}
