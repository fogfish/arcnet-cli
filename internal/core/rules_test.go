//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestPredicateDefFieldConstruction(t *testing.T) {
	def := core.PredicateDef{
		Role:        "edge",
		Merge:       core.MergeUnion,
		Label:       "Is Part Of",
		Aligned:     "dcterms:isPartOf",
		Description: "Asserts a whole-part relationship.",
	}

	it.Then(t).
		Should(it.Equal("edge", def.Role)).
		Should(it.Equal(core.MergeUnion, def.Merge)).
		Should(it.Equal("Is Part Of", def.Label)).
		Should(it.Equal("dcterms:isPartOf", def.Aligned)).
		Should(it.Equal("Asserts a whole-part relationship.", def.Description))
}

func TestTypeDefFieldConstruction(t *testing.T) {
	def := core.TypeDef{
		Merge:       core.MergeUnion,
		Required:    []string{"category", "definition"},
		Optional:    []string{"aliases"},
		Description: "A subject occurring in sources.",
	}

	it.Then(t).
		Should(it.Equal(core.MergeUnion, def.Merge)).
		Should(it.Seq(def.Required).Equal("category", "definition")).
		Should(it.Seq(def.Optional).Equal("aliases")).
		Should(it.Equal("A subject occurring in sources.", def.Description))
}

func TestIndexPredicatesLookup(t *testing.T) {
	index := core.Index{
		Predicates: map[string]core.PredicateDef{
			"isPartOf": {Role: "edge", Merge: core.MergeUnion, Description: "..."},
		},
	}

	def, ok := index.Predicates["isPartOf"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal("edge", def.Role))

	_, ok = index.Predicates["hasPart"]
	it.Then(t).Should(it.True(!ok))
}

func TestIndexTypesLookup(t *testing.T) {
	index := core.Index{
		Types: map[string]core.TypeDef{
			"Entity": {Merge: core.MergeUnion, Description: "..."},
		},
	}

	def, ok := index.Types["Entity"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnion, def.Merge))

	_, ok = index.Types["hypothesis"]
	it.Then(t).Should(it.True(!ok))
}
