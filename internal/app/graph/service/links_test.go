//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
)

// entityNodeWithHRef builds a minimal on-disk entity node file whose leading
// body text carries one inline prose reference (an HRef, not a bullet-list
// Edge) with no explicit predicate — the shape extractInlineLinks turns
// into a Link{Target: target} with an empty Predicate.
func entityNodeWithHRef(id, target string) string {
	return "---\n\"@id\": " + id + "\n\"@type\": Entity\n---\n# " + id + "\n\nSee [[" + target + "]] for details.\n"
}

func TestNodeLinksMixedEdgesAndHRefsReturnsCombinationInOrder(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/a.md": "---\n\"@id\": a\n\"@type\": Entity\n---\n# a\n\nSee [[c]] for details.\n\n- [[b]]\n",
		"Entity/b.md": entityNode("b"),
		"Entity/c.md": entityNode("c"),
	})

	links, err := service.NodeLinks(context.Background(), mounter, "/graph", "a")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(links)))
	it.Then(t).
		Should(it.Equal("b", links[0].Target)).
		Should(it.Equal("c", links[1].Target)).
		Should(it.Equal("", links[1].Predicate))
}

func TestNodeLinksZeroRelationsReturnsEmptySlice(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/a.md": entityNode("a"),
	})

	links, err := service.NodeLinks(context.Background(), mounter, "/graph", "a")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(links)))
}

func TestNodeLinksUnknownIDReturnsErrSeedNotFound(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/a.md": entityNode("a"),
	})

	_, err := service.NodeLinks(context.Background(), mounter, "/graph", "No Such Node")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSeedNotFound)))
}

func TestNodeBacklinksReferencedByMultipleReturnsOneEntryPerRelation(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/a.md":      entityNode("a", "target"),
		"Entity/b.md":      entityNodeWithHRef("b", "target"),
		"Entity/target.md": entityNode("target"),
	})

	backlinks, err := service.NodeBacklinks(context.Background(), mounter, "/graph", "target")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(backlinks)))

	var sources []string
	for _, b := range backlinks {
		sources = append(sources, b.Source)
	}
	it.Then(t).
		Should(it.Seq(sources).Contain("a")).
		Should(it.Seq(sources).Contain("b"))
}

func TestNodeBacklinksZeroReferencingRelationsReturnsEmptySlice(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/target.md": entityNode("target"),
	})

	backlinks, err := service.NodeBacklinks(context.Background(), mounter, "/graph", "target")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(backlinks)))
}

func TestNodeBacklinksUnknownIDReturnsErrSeedNotFound(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/a.md": entityNode("a"),
	})

	_, err := service.NodeBacklinks(context.Background(), mounter, "/graph", "No Such Node")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSeedNotFound)))
}

func TestNodeBacklinksSelfReferencingNodeAppearsInOwnResult(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/a.md": entityNode("a", "a"),
	})

	backlinks, err := service.NodeBacklinks(context.Background(), mounter, "/graph", "a")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(backlinks)))
	it.Then(t).Should(it.Equal("a", backlinks[0].Source))
}
