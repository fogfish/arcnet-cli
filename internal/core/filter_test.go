//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"regexp"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestFilterZeroValueMatchesEveryNode(t *testing.T) {
	node := core.Node{Type: "entity", Attrs: map[string][]core.Predicate{"status": {{Value: "mature"}}}}

	it.Then(t).Should(it.True(core.Filter{}.Match(node)))
}

func TestFilterTypesIsOR(t *testing.T) {
	source := core.Node{Type: "source"}
	entity := core.Node{Type: "entity"}
	resource := core.Node{Type: "resource"}
	f := core.Filter{Types: []string{"source", "entity"}}

	it.Then(t).
		Should(it.True(f.Match(source))).
		Should(it.True(f.Match(entity))).
		Should(it.True(!f.Match(resource)))
}

func TestFilterTagsIsAND(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"tags": {{Value: "cryptography"}, {Value: "protocols"}}}}
	f := core.Filter{Tags: []string{"cryptography", "protocols"}}
	fMissing := core.Filter{Tags: []string{"cryptography", "unrelated"}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMissing.Match(node)))
}

func TestFilterAttrsExactMatchCaseInsensitiveScalar(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"status": {{Value: "Mature"}}}}
	f := core.Filter{Attrs: map[string]string{"status": "mature"}}
	fMismatch := core.Filter{Attrs: map[string]string{"status": "backlog"}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMismatch.Match(node)))
}

func TestFilterAttrsExactMatchArrayMembership(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}}}
	f := core.Filter{Attrs: map[string]string{"category": "abstract"}}
	fMismatch := core.Filter{Attrs: map[string]string{"category": "relative"}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMismatch.Match(node)))
}

func TestFilterAttrPatternsRegexpMatchScalarAndArray(t *testing.T) {
	scalarNode := core.Node{Attrs: map[string][]core.Predicate{"title": {{Value: "TLS 1.3: Design and Rationale"}}}}
	arrayNode := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}}}
	f := core.Filter{AttrPatterns: map[string]*regexp.Regexp{"title": regexp.MustCompile(`^TLS 1\.3`)}}
	fArray := core.Filter{AttrPatterns: map[string]*regexp.Regexp{"category": regexp.MustCompile(`^abs`)}}
	fMismatch := core.Filter{AttrPatterns: map[string]*regexp.Regexp{"title": regexp.MustCompile(`^SSL`)}}

	it.Then(t).
		Should(it.True(f.Match(scalarNode))).
		Should(it.True(fArray.Match(arrayNode))).
		Should(it.True(!fMismatch.Match(scalarNode)))
}

func TestFilterCombinedGroupsAreANDed(t *testing.T) {
	node := core.Node{
		Type: "entity",
		Attrs: map[string][]core.Predicate{
			"tags":   {{Value: "cryptography"}},
			"status": {{Value: "mature"}},
		},
	}
	f := core.Filter{
		Types:        []string{"entity"},
		Tags:         []string{"cryptography"},
		Attrs:        map[string]string{"status": "mature"},
		AttrPatterns: map[string]*regexp.Regexp{"status": regexp.MustCompile(`^mat`)},
	}
	fWrongType := core.Filter{Types: []string{"resource"}, Tags: []string{"cryptography"}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fWrongType.Match(node)))
}

func TestFilterMatchingZeroNodes(t *testing.T) {
	node := core.Node{Type: "source"}
	f := core.Filter{Types: []string{"resource"}}

	it.Then(t).Should(it.True(!f.Match(node)))
}

func TestFilterAttrsListValuedSingleValue(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"status": {{Value: "mature"}}}}
	f := core.Filter{Attrs: map[string]string{"status": "mature"}}

	it.Then(t).Should(it.True(f.Match(node)))
}

func TestFilterAttrsListValuedMultipleValuesMatchesAny(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}, {Value: "protocol"}}}}
	f := core.Filter{Attrs: map[string]string{"category": "protocol"}}

	it.Then(t).Should(it.True(f.Match(node)))
}

func TestFilterAttrsListValuedNoMatch(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}}}
	f := core.Filter{Attrs: map[string]string{"category": "relative"}}

	it.Then(t).Should(it.True(!f.Match(node)))
}
