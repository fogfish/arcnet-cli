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
	"testing/fstest"

	"github.com/fogfish/it/v2"

	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// entityNode builds a minimal on-disk entity node file, with an optional
// bare-list Edges section (bullet per target, no predicate).
func entityNode(id string, edges ...string) string {
	body := "---\n\"@id\": " + id + "\n\"@type\": Entity\n---\n# " + id + "\n"
	if len(edges) > 0 {
		body += "\n"
		for _, e := range edges {
			body += "- [[" + e + "]]\n"
		}
	}
	return body
}

// predicateEdgeNode builds a minimal on-disk entity node file whose edges
// each carry an explicit predicate (name:target pairs), used by --predicate
// traversal-scoping tests (research.md D5).
func predicateEdgeNode(id string, edges ...[2]string) string {
	body := "---\n\"@id\": " + id + "\n\"@type\": Entity\n---\n# " + id + "\n"
	if len(edges) > 0 {
		body += "\n"
		for _, e := range edges {
			body += "- " + e[0] + ":: [[" + e[1] + "]]\n"
		}
	}
	return body
}

func sourceNodeWithAttrs(id, tags, status string) string {
	return "---\n\"@id\": " + id + "\n\"@type\": Source\ntags: [" + tags + "]\nstatus: " + status + "\n---\n# " + id + "\n"
}

func resourceNodeWithAttrs(id, tags, status string) string {
	return "---\n\"@id\": " + id + "\n\"@type\": Resource\ntags: [" + tags + "]\nstatus: " + status + "\n---\n# " + id + "\n"
}

// filterType/filterTag/filterAttr/filterPredicate build a core.Filter from
// one statement each, mirroring the CLI's own --type/--tag/--attr/
// --predicate lowering (research.md D4/D10) — shared across this package's
// test files.
func filterType(values ...string) core.Filter {
	return core.Filter{Statements: []core.Statement{{
		Predicate: core.Matcher{Values: []string{"type"}},
		Target:    core.Matcher{Values: values},
	}}}
}

func filterTag(value string) core.Filter {
	return core.Filter{Statements: []core.Statement{{
		Predicate: core.Matcher{Values: []string{"tags"}},
		Target:    core.Matcher{Values: []string{value}},
	}}}
}

func filterAttr(name, value string) core.Filter {
	return core.Filter{Statements: []core.Statement{{
		Predicate: core.Matcher{Values: []string{name}},
		Target:    core.Matcher{Values: []string{value}},
	}}}
}

func filterPredicate(values ...string) core.Filter {
	return core.Filter{Statements: []core.Statement{{Predicate: core.Matcher{Values: values}}}}
}

// mergeFilters concatenates every argument's Statements into one Filter —
// used to compose several single-statement filterX helpers the way a
// combined --type/--tag/--attr invocation would (each flag repeat is its
// own statement, ANDed via Filter.Match, research.md D4).
func mergeFilters(filters ...core.Filter) core.Filter {
	var out core.Filter
	for _, f := range filters {
		out.Statements = append(out.Statements, f.Statements...)
	}
	return out
}

func TestSubgraphGuardNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	_, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "TLS", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestSubgraphUnknownSeedReturnsErrSeedNotFound(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md": entityNode("A"),
	})

	_, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "No Such Node", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSeedNotFound)))
}

func TestSubgraphDepthZeroYieldsSeedAlone(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md": entityNode("A", "B"),
		"Entity/B.md": entityNode("B"),
	})

	result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 0, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(1, len(result.Patch.Nodes))).
		Should(it.Equal("A", result.Patch.Nodes[0].ID)).
		Should(it.Equal(0, result.DirectReachable)).
		Should(it.Equal(0, result.BacklinkReachable))
}

func TestSubgraphSeedWithNoConnectionsYieldsOneNodePatch(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/Lonely.md": entityNode("Lonely"),
	})

	result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "Lonely", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(1, len(result.Patch.Nodes)))
}

// TestSubgraphDefaultDepthIncludesEveryDirectlyConnectedNodeGroupedByKind
// exercises User Story 1's central acceptance scenario at the service
// layer: default depth 1, no filter, the seed plus every directly
// connected node (in either direction) is included exactly once.
func TestSubgraphDefaultDepthIncludesEveryDirectlyConnectedNodeGroupedByKind(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md":                 entityNode("TLS", "rescorla-2026-tls13", "RFC 8446"),
		"Entity/SSL.md":                 entityNode("SSL", "TLS"),
		"Source/rescorla-2026-tls13.md": sourceNodeWithAttrs("rescorla-2026-tls13", "cryptography", "mature"),
		"Resource/RFC 8446.md":          resourceNodeWithAttrs("RFC 8446", "cryptography", "draft"),
	})

	result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "TLS", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(4, len(result.Patch.Nodes)))

	ids := map[string]bool{}
	for _, n := range result.Patch.Nodes {
		ids[n.ID] = true
	}
	it.Then(t).
		Should(it.True(ids["TLS"])).
		Should(it.True(ids["SSL"])).
		Should(it.True(ids["rescorla-2026-tls13"])).
		Should(it.True(ids["RFC 8446"]))
	it.Then(t).
		Should(it.Equal(2, result.DirectReachable)).
		Should(it.Equal(1, result.BacklinkReachable))
}

func TestSubgraphCycleDoesNotLoopOrDuplicateNodes(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/CycleA.md": entityNode("CycleA", "CycleB"),
		"Entity/CycleB.md": entityNode("CycleB", "CycleA"),
	})

	result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "CycleA", 5, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(2, len(result.Patch.Nodes)))
}

func TestSubgraphDanglingLinkTargetExcluded(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/Dangling.md": entityNode("Dangling", "No Such Target"),
	})

	result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "Dangling", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(1, len(result.Patch.Nodes))).
		Should(it.Equal(0, result.DirectReachable))
}

func TestSubgraphMultiHopChainRespectsDepthBoundaries(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md": entityNode("A", "B", "F"),
		"Entity/B.md": entityNode("B", "C"),
		"Entity/C.md": entityNode("C", "D", "E"),
		"Entity/D.md": entityNode("D"),
		"Entity/E.md": entityNode("E"),
		"Entity/F.md": entityNode("F", "E"),
	})

	depth0, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 0, configkernel.SubgraphConfig{}, "/graph", false)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(depth0.Patch.Nodes)))

	depth1, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 1, configkernel.SubgraphConfig{}, "/graph", false)
	it.Then(t).Should(it.Nil(err))
	depth1IDs := nodeIDSet(depth1.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(3, len(depth1.Patch.Nodes))).
		Should(it.True(depth1IDs["A"])).
		Should(it.True(depth1IDs["B"])).
		Should(it.True(depth1IDs["F"]))

	// E is reachable at hop 2 (A->F->E) and also at hop 3 (A->B->C->E) —
	// it must appear exactly once, included via its shortest path.
	depth2, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 2, configkernel.SubgraphConfig{}, "/graph", false)
	it.Then(t).Should(it.Nil(err))
	depth2IDs := nodeIDSet(depth2.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(5, len(depth2.Patch.Nodes))).
		Should(it.True(depth2IDs["A"])).
		Should(it.True(depth2IDs["B"])).
		Should(it.True(depth2IDs["F"])).
		Should(it.True(depth2IDs["C"])).
		Should(it.True(depth2IDs["E"])).
		ShouldNot(it.True(depth2IDs["D"]))

	depth3, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 3, configkernel.SubgraphConfig{}, "/graph", false)
	it.Then(t).Should(it.Nil(err))
	depth3IDs := nodeIDSet(depth3.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(6, len(depth3.Patch.Nodes))).
		Should(it.True(depth3IDs["D"]))
}

func nodeIDSet(nodes []core.Node) map[string]bool {
	out := map[string]bool{}
	for _, n := range nodes {
		out[n.ID] = true
	}
	return out
}

func TestSubgraphFilterExcludesNonSeedCandidatesOnlyNeverSeed(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md":                 entityNode("TLS", "rescorla-2026-tls13", "RFC 8446"),
		"Source/rescorla-2026-tls13.md": sourceNodeWithAttrs("rescorla-2026-tls13", "cryptography", "mature"),
		"Resource/RFC 8446.md":          resourceNodeWithAttrs("RFC 8446", "cryptography", "draft"),
	})

	filter := filterType("Source")
	result, err := service.Subgraph(context.Background(), mounter, filter, "TLS", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(2, len(result.Patch.Nodes))).
		Should(it.True(ids["TLS"])).
		Should(it.True(ids["rescorla-2026-tls13"])).
		ShouldNot(it.True(ids["RFC 8446"]))
}

func TestSubgraphFilterMatchingZeroReachableNodesYieldsSeedAloneNoError(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md":                 entityNode("TLS", "rescorla-2026-tls13"),
		"Source/rescorla-2026-tls13.md": sourceNodeWithAttrs("rescorla-2026-tls13", "cryptography", "mature"),
	})

	filter := filterType("Resource")
	result, err := service.Subgraph(context.Background(), mounter, filter, "TLS", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(1, len(result.Patch.Nodes))).
		Should(it.Equal("TLS", result.Patch.Nodes[0].ID))
}

func TestSubgraphCombinedKindTagAttrFilterNarrowsToExactSubset(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md":                 entityNode("TLS", "rescorla-2026-tls13", "other-2026"),
		"Source/rescorla-2026-tls13.md": sourceNodeWithAttrs("rescorla-2026-tls13", "cryptography", "mature"),
		"Source/other-2026.md":          sourceNodeWithAttrs("other-2026", "cryptography", "draft"),
	})

	filter := mergeFilters(filterType("Source"), filterTag("cryptography"), filterAttr("status", "mature"))
	result, err := service.Subgraph(context.Background(), mounter, filter, "TLS", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(2, len(result.Patch.Nodes))).
		Should(it.True(ids["TLS"])).
		Should(it.True(ids["rescorla-2026-tls13"])).
		ShouldNot(it.True(ids["other-2026"]))
}

// TestSubgraphCapRetainsExactlyHighestDegreeCandidates proves SC-007: when
// a pool exceeds its configured cap, the retained nodes are always exactly
// the highest-degree candidates, deterministic across repeated runs.
func TestSubgraphCapRetainsExactlyHighestDegreeCandidates(t *testing.T) {
	files := map[string]string{
		"Entity/Hub.md": entityNode("Hub"),
		"Entity/P1.md":  entityNode("P1", "Hub", "DummyA", "DummyB"),
		"Entity/P2.md":  entityNode("P2", "Hub", "DummyC"),
		"Entity/P3.md":  entityNode("P3", "Hub"),
		"Entity/P4.md":  entityNode("P4", "Hub"),
		"Entity/P5.md":  entityNode("P5", "Hub"),
	}
	mounter := newGrepGraph(files)
	cfg := configkernel.SubgraphConfig{BacklinkCap: 2}

	for i := 0; i < 5; i++ {
		result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "Hub", 1, cfg, "/graph", false)
		it.Then(t).Should(it.Nil(err))

		ids := nodeIDSet(result.Patch.Nodes)
		it.Then(t).
			Should(it.True(result.BacklinkTruncated)).
			Should(it.Equal(5, result.BacklinkReachable)).
			Should(it.Equal(2, result.BacklinkIncluded)).
			Should(it.Equal(3, len(result.Patch.Nodes))). // Hub + P1 + P2
			Should(it.True(ids["Hub"])).
			Should(it.True(ids["P1"])).
			Should(it.True(ids["P2"])).
			ShouldNot(it.True(ids["P3"])).
			ShouldNot(it.True(ids["P4"])).
			ShouldNot(it.True(ids["P5"]))
	}
}

// BUG-001: a boundary target — referenced by an included node but itself
// excluded from the extraction (here, by --depth) — gets a stub node only
// when stubs=true; without it, behavior is unchanged from before the
// bugfix. The stub (C) is never itself expanded: its own edge to D must
// not pull D into the output (no recursive stub expansion).
func TestSubgraphStubsEmittedOnlyWhenRequested(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md": entityNode("A", "B"),
		"Entity/B.md": entityNode("B", "C"),
		"Entity/C.md": entityNode("C", "D"),
		"Entity/D.md": entityNode("D"),
	})

	withoutStubs, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 1, configkernel.SubgraphConfig{}, "/graph", false)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(2, len(withoutStubs.Patch.Nodes))).
		Should(it.Equal(0, withoutStubs.Stubs))

	withStubs, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 1, configkernel.SubgraphConfig{}, "/graph", true)
	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(withStubs.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(3, len(withStubs.Patch.Nodes))).
		Should(it.Equal(1, withStubs.Stubs)).
		Should(it.True(ids["A"])).
		Should(it.True(ids["B"])).
		Should(it.True(ids["C"])).
		ShouldNot(it.True(ids["D"]))
}

// BUG-001: a stub node carries only kind and id — no attributes, text, or
// connections of its own — even when the real node it stands in for has
// plenty of both.
func TestSubgraphStubCarriesOnlyKindAndIDNoOtherContent(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md":                   entityNode("A", "rescorla-2026-tls13"),
		"Source/rescorla-2026-tls13.md": sourceNodeWithAttrs("rescorla-2026-tls13", "cryptography", "mature"),
	})

	// depth 0: the seed's own direct edge to rescorla-2026-tls13 is itself
	// a boundary reference (BFS never runs at all), isolating the stub's
	// shape from any other included-node behavior.
	result, err := service.Subgraph(context.Background(), mounter, core.Filter{}, "A", 0, configkernel.SubgraphConfig{}, "/graph", true)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(result.Patch.Nodes)))

	var stub core.Node
	for _, n := range result.Patch.Nodes {
		if n.ID == "rescorla-2026-tls13" {
			stub = n
		}
	}
	it.Then(t).
		Should(it.Equal("Source", stub.Type)).
		Should(it.Equal(0, len(stub.Attrs))).
		Should(it.Equal(0, len(stub.Texts))).
		Should(it.Equal(0, len(stub.Edges)))
}

// --- User Story 1: predicate-scoped traversal (research.md D3/D5) ---

// TestSubgraphPredicateScopesOutgoingTraversal exercises spec.md US1
// Acceptance Scenario 1 at the service layer: a traversal-constraint
// statement admits only same-predicate outgoing edges.
func TestSubgraphPredicateScopesOutgoingTraversal(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md": predicateEdgeNode("A", [2]string{"cites", "B"}, [2]string{"mentions", "C"}),
		"Entity/B.md": entityNode("B"),
		"Entity/C.md": entityNode("C"),
	})

	result, err := service.Subgraph(context.Background(), mounter, filterPredicate("cites"), "A", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.True(ids["A"])).
		Should(it.True(ids["B"])).
		ShouldNot(it.True(ids["C"]))
}

// TestSubgraphPredicateScopesBacklinkTraversal mirrors the above for the
// reverse (backlink) direction — a node whose only connection to the seed
// is a different-predicate edge is not reached.
func TestSubgraphPredicateScopesBacklinkTraversal(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/Hub.md":  entityNode("Hub"),
		"Entity/Cit.md":  predicateEdgeNode("Cit", [2]string{"cites", "Hub"}),
		"Entity/Ment.md": predicateEdgeNode("Ment", [2]string{"mentions", "Hub"}),
	})

	result, err := service.Subgraph(context.Background(), mounter, filterPredicate("cites"), "Hub", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.True(ids["Hub"])).
		Should(it.True(ids["Cit"])).
		ShouldNot(it.True(ids["Ment"]))
}

// TestSubgraphPredicateReachedNodeSurvivesNarrowingUnconditionally proves
// the one-directional-relation case research.md D3 exists to protect
// (spec.md US1 Acceptance Scenario 5's underlying concern): a
// traversal-only filter (no narrowing statement) must not, by itself,
// re-exclude the very node it scoped traversal to reach — even though that
// node carries no reciprocal "cites" fact of its own.
func TestSubgraphPredicateReachedNodeSurvivesNarrowingUnconditionally(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md": predicateEdgeNode("A", [2]string{"cites", "B"}),
		"Entity/B.md": entityNode("B"), // no reciprocal "cites" fact
	})

	result, err := service.Subgraph(context.Background(), mounter, filterPredicate("cites"), "A", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(nodeIDSet(result.Patch.Nodes)["B"]))
}

// TestSubgraphPredicateCombinedWithNarrowingStatementBothApply: a filter
// combining a traversal constraint and an ordinary narrowing statement
// still narrows via the ordinary statement (research.md D3).
func TestSubgraphPredicateCombinedWithNarrowingStatementBothApply(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/A.md":   predicateEdgeNode("A", [2]string{"cites", "B"}, [2]string{"cites", "C"}),
		"Source/B.md":   sourceNodeWithAttrs("B", "cryptography", "mature"),
		"Resource/C.md": resourceNodeWithAttrs("C", "cryptography", "draft"),
	})

	result, err := service.Subgraph(context.Background(), mounter, mergeFilters(filterPredicate("cites"), filterType("Source")), "A", 1, configkernel.SubgraphConfig{}, "/graph", false)

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.True(ids["A"])).
		Should(it.True(ids["B"])).
		ShouldNot(it.True(ids["C"]))
}
