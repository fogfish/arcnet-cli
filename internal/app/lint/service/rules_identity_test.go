//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestCheckSourceCitekeyMatches(t *testing.T) {
	node := core.Node{Type: "Source", ID: "foo-2026-x"}
	out := checkSourceCitekey(node, "sources/foo-2026-x.md", "foo-2026-x", []byte("---\nid: foo-2026-x\n---\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckSourceCitekeyMismatch(t *testing.T) {
	node := core.Node{Type: "Source", ID: "A Test Document"}
	out := checkSourceCitekey(node, "sources/foo-2026-x.md", "foo-2026-x", []byte("---\ntitle: A Test Document\n---\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleSourceCitekey, out[0].Rule))
}

func TestCheckSourceCitekeyNonSourceExempt(t *testing.T) {
	node := core.Node{Type: "Entity", ID: "Widget"}
	out := checkSourceCitekey(node, "entities/Foo.md", "Foo", []byte("---\n---\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckEntityCategoryValid(t *testing.T) {
	node := core.Node{Type: "Entity", Attrs: map[string][]core.Predicate{
		"category": {{Value: "independent"}, {Value: "abstract"}, {Value: "occurrent"}, {Value: "script"}},
	}}
	out := checkEntityCategory(node, "entities/x.md", []byte("---\ncategory: [independent, abstract, occurrent, script]\n---\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckEntityCategoryMissing(t *testing.T) {
	node := core.Node{Type: "Entity", Attrs: map[string][]core.Predicate{}}
	out := checkEntityCategory(node, "entities/x.md", []byte("---\n---\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.String(out[0].Message).Contain("missing"))
}

func TestCheckEntityCategoryWrongLength(t *testing.T) {
	node := core.Node{Type: "Entity", Attrs: map[string][]core.Predicate{
		"category": {{Value: "independent"}, {Value: "abstract"}, {Value: "occurrent"}},
	}}
	out := checkEntityCategory(node, "entities/x.md", []byte("---\ncategory: [independent, abstract, occurrent]\n---\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.String(out[0].Message).Contain("found 3"))
}

func TestCheckEntityCategoryBadWord(t *testing.T) {
	node := core.Node{Type: "Entity", Attrs: map[string][]core.Predicate{
		"category": {{Value: "bogus"}, {Value: "abstract"}, {Value: "occurrent"}, {Value: "script"}},
	}}
	out := checkEntityCategory(node, "entities/x.md", []byte("---\ncategory: [bogus, abstract, occurrent, script]\n---\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleEntityCategory, out[0].Rule))
}

func TestCheckEntityCategoryNonEntityExempt(t *testing.T) {
	node := core.Node{Type: "Resource", Attrs: map[string][]core.Predicate{}}
	out := checkEntityCategory(node, "resources/x.md", []byte("---\n---\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}
