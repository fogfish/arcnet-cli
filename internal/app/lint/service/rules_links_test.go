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

var basenames = map[string][]string{
	"foo-2026-x": {"sources/foo-2026-x.md"},
	"Widget":     {"entities/Widget.md"},
}

func TestCheckLinksResolveAllResolve(t *testing.T) {
	node := core.Node{
		Edges: []core.Link{{Predicate: "mentions", Target: "Widget"}},
	}
	out := checkLinksResolve(node, "sources/foo-2026-x.md", []byte("- mentions:: [[Widget]]\n"), basenames)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckLinksResolveUnresolvedTarget(t *testing.T) {
	node := core.Node{
		Edges: []core.Link{{Predicate: "mentions", Target: "Nonexistent Node"}},
	}
	raw := []byte("- mentions:: [[Nonexistent Node]]\n")
	out := checkLinksResolve(node, "entities/x.md", raw, basenames)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleLinkResolves, out[0].Rule)).
		Should(it.String(out[0].Message).Contain(`"Nonexistent Node"`)).
		Should(it.Equal(1, out[0].Line))
}

func TestCheckLinksResolveDedupSameTargetTwice(t *testing.T) {
	node := core.Node{
		Edges: []core.Link{
			{Target: "Missing"},
			{Target: "Missing"},
		},
	}
	out := checkLinksResolve(node, "entities/x.md", []byte("- [[Missing]]\n"), basenames)
	it.Then(t).Should(it.Equal(1, len(out)))
}

func TestCheckDerivedProvenanceSourceExempt(t *testing.T) {
	node := core.Node{Kind: "source"}
	out := checkDerivedProvenance(node, "sources/x.md", map[string]core.Kind{})
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckDerivedProvenanceLinksToSourcePasses(t *testing.T) {
	node := core.Node{Kind: "entity", Edges: []core.Link{{Target: "foo-2026-x"}}}
	kindIndex := map[string]core.Kind{"foo-2026-x": "source"}
	out := checkDerivedProvenance(node, "entities/x.md", kindIndex)
	it.Then(t).Should(it.Equal(0, len(out)))
}

// BUG-001: a timeline-kind node is the tool's own chronological index
// over many documents, never content distilled from one document, so it
// is exempt from checkDerivedProvenance regardless of its links.
func TestCheckDerivedProvenanceTimelineExempt(t *testing.T) {
	node := core.Node{Kind: "timeline", Edges: []core.Link{{Target: "Other Entity"}}}
	kindIndex := map[string]core.Kind{"Other Entity": "entity"}
	out := checkDerivedProvenance(node, "timeline/yearly/2026.md", kindIndex)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckDerivedProvenanceNoSourceLinkFails(t *testing.T) {
	node := core.Node{Kind: "entity", Edges: []core.Link{{Target: "Other Entity"}}}
	kindIndex := map[string]core.Kind{"Other Entity": "entity"}
	out := checkDerivedProvenance(node, "entities/x.md", kindIndex)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleDerivedProvenance, out[0].Rule))
}
