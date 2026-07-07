//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
)

const subgraphEntityTLS = `---
"@id": Transport Layer Security
"@type": entity
category: form structure attribute process
---
# Transport Layer Security

TLS is the successor to SSL.

- [[rescorla-2026-tls13]]
`

const subgraphSourceTLS13 = `---
"@id": rescorla-2026-tls13
"@type": source
title: TLS 1.3
---
# rescorla-2026-tls13

TLS 1.3 is the latest version of the Transport Layer Security protocol.
`

const subgraphEntitySSL = `---
"@id": SSL
"@type": entity
---
# SSL

- [[Transport Layer Security]]
`

const subgraphIsolatedNote = `---
"@id": Isolated Note
"@type": entity
---
# Isolated Note

A note with no connections to anything else.
`

// seedSubgraphFixture writes a small graph used across most US1/US2
// scenarios: TLS (entity) links out to rescorla-2026-tls13 (source), and
// SSL (entity) links into TLS — so TLS's direct pool is {rescorla-...} and
// its backlink pool is {SSL}.
func seedSubgraphFixture(t *testing.T, dir string) {
	t.Helper()
	writeGrepNode(t, dir, "entities/Transport Layer Security.md", subgraphEntityTLS)
	writeGrepNode(t, dir, "sources/rescorla-2026-tls13.md", subgraphSourceTLS13)
	writeGrepNode(t, dir, "entities/SSL.md", subgraphEntitySSL)
}

// arc subgraph "Transport Layer Security"
// Scenario 1 from spec.md US1: the seed plus every directly connected node
// (in either direction) is included, grouped by kind, front-matter/body
// preserved verbatim.
func TestSubgraphDefaultDepthIncludesSeedAndDirectConnectionsGroupedByKind(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"Transport Layer Security"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("# Entity")).
		Should(it.String(out).Contain("## Transport Layer Security")).
		Should(it.String(out).Contain("## SSL")).
		Should(it.String(out).Contain("# Source")).
		Should(it.String(out).Contain("## rescorla-2026-tls13")).
		Should(it.String(out).Contain("category: form structure attribute process")).
		Should(it.String(out).Contain("TLS is the successor to SSL.")).
		Should(it.String(out).Contain("TLS 1.3 is the latest version of the Transport Layer Security protocol."))
}

// arc subgraph "Isolated Note"
// Scenario 2 from spec.md US1: a seed with no connections yields a
// one-node document, no error.
func TestSubgraphSeedWithNoConnectionsYieldsOneNodeDocument(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/Isolated Note.md", subgraphIsolatedNote)
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"Isolated Note"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## Isolated Note")).
		ShouldNot(it.String(out).Contain("# Source"))
}

// arc subgraph "Transport Layer Security" > out.md && arc apply out.md
// Scenario 3 from spec.md US1: the extracted output is accepted by arc
// apply without a structural parsing failure.
func TestSubgraphExtractedOutputReingestsViaApplyWithoutStructuralError(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"Transport Layer Security"})
	it.Then(t).ShouldNot(it.Error(out, err))

	patchPath := filepath.Join(dir, "extracted.md")
	it.Then(t).Should(it.Nil(os.WriteFile(patchPath, []byte(out), 0o644)))

	applyOut, applyErr := sut(NewApplyCmd(), []string{patchPath})
	it.Then(t).ShouldNot(it.Error(applyOut, applyErr))
}

// arc subgraph "No Such Node"
// Scenario 4 from spec.md US1 / edge case FR-011: an unknown basename
// refuses with a clear error and no output.
func TestSubgraphUnknownBasenameRefusesWithClearErrorNoOutput(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"No Such Node"})

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).Should(it.Error(out, err).Contain("no node found"))
}

const subgraphChainA = `---
"@id": ChainA
"@type": entity
---
# ChainA

- [[ChainB]]
`
const subgraphChainB = `---
"@id": ChainB
"@type": entity
---
# ChainB

- [[ChainC]]
`
const subgraphChainC = `---
"@id": ChainC
"@type": entity
---
# ChainC

- [[ChainD]]
`
const subgraphChainD = `---
"@id": ChainD
"@type": entity
---
# ChainD
`

func seedSubgraphChainFixture(t *testing.T, dir string) {
	t.Helper()
	writeGrepNode(t, dir, "entities/ChainA.md", subgraphChainA)
	writeGrepNode(t, dir, "entities/ChainB.md", subgraphChainB)
	writeGrepNode(t, dir, "entities/ChainC.md", subgraphChainC)
	writeGrepNode(t, dir, "entities/ChainD.md", subgraphChainD)
}

// arc subgraph ChainA --depth 2
// Scenario 1 from spec.md US2: depth 2 includes everything within 2 hops,
// excludes anything farther.
func TestSubgraphDepthTwoIncludesWithinTwoHopsExcludesFarther(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphChainFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "2")))
	out, err := sut(cmd, []string{"ChainA"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## ChainA")).
		Should(it.String(out).Contain("## ChainB")).
		Should(it.String(out).Contain("## ChainC")).
		ShouldNot(it.String(out).Contain("## ChainD"))
}

// arc subgraph ChainA --depth 0
// Scenario 2 from spec.md US2: depth 0 yields the seed alone.
func TestSubgraphDepthZeroYieldsSeedAloneCLI(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphChainFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "0")))
	out, err := sut(cmd, []string{"ChainA"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## ChainA")).
		ShouldNot(it.String(out).Contain("## ChainB"))
}

// arc subgraph ChainA
// Scenario 3 from spec.md US2: omitting --depth behaves as --depth 1.
func TestSubgraphOmittingDepthBehavesAsDepthOne(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphChainFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"ChainA"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## ChainA")).
		Should(it.String(out).Contain("## ChainB")).
		ShouldNot(it.String(out).Contain("## ChainC"))
}

const subgraphDiamondA = `---
"@id": DiamondA
"@type": entity
---
# DiamondA

- [[DiamondB]]
- [[DiamondC]]
`
const subgraphDiamondB = `---
"@id": DiamondB
"@type": entity
---
# DiamondB

- [[DiamondD]]
`
const subgraphDiamondC = `---
"@id": DiamondC
"@type": entity
---
# DiamondC

- [[DiamondD]]
`
const subgraphDiamondD = `---
"@id": DiamondD
"@type": entity
---
# DiamondD
`

// arc subgraph DiamondA --depth 2
// Scenario 4 from spec.md US2: a node reachable by more than one path of
// different lengths appears exactly once.
func TestSubgraphMultiPathNodeAppearsExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/DiamondA.md", subgraphDiamondA)
	writeGrepNode(t, dir, "entities/DiamondB.md", subgraphDiamondB)
	writeGrepNode(t, dir, "entities/DiamondC.md", subgraphDiamondC)
	writeGrepNode(t, dir, "entities/DiamondD.md", subgraphDiamondD)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "2")))
	out, err := sut(cmd, []string{"DiamondA"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.Equal(1, strings.Count(out, "## DiamondD")))
}

const subgraphResourceRFC8446 = `---
"@id": RFC 8446
"@type": resource
tags: [cryptography]
status: draft
---
# RFC 8446

The TLS 1.3 RFC.
`

const subgraphSourceOther = `---
"@id": other-2026
"@type": source
tags: [cryptography]
status: draft
---
# other-2026

An unrelated draft source.
`

const subgraphEntityTLSWithTwoTargets = `---
"@id": Transport Layer Security
"@type": entity
---
# Transport Layer Security

TLS is the successor to SSL.

- [[rescorla-2026-tls13]]
- [[RFC 8446]]
- [[other-2026]]
`

func seedSubgraphFilterFixture(t *testing.T, dir string) {
	t.Helper()
	writeGrepNode(t, dir, "entities/Transport Layer Security.md", subgraphEntityTLSWithTwoTargets)
	writeGrepNode(t, dir, "sources/rescorla-2026-tls13.md", `---
"@id": rescorla-2026-tls13
"@type": source
tags: [cryptography]
status: mature
---
# rescorla-2026-tls13

TLS 1.3 design rationale.
`)
	writeGrepNode(t, dir, "resources/RFC 8446.md", subgraphResourceRFC8446)
	writeGrepNode(t, dir, "sources/other-2026.md", subgraphSourceOther)
}

// arc subgraph "Transport Layer Security" --kind source
// Scenario 1 from spec.md US3: the seed is still included even though its
// own kind doesn't match the filter.
func TestSubgraphKindFilterStillIncludesSeedDespiteMismatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFilterFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "source")))
	out, err := sut(cmd, []string{"Transport Layer Security"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("## Transport Layer Security"))
}

// arc subgraph "Transport Layer Security" --kind resource
// Scenario 2 from spec.md US3: --kind restricts which reachable nodes are
// added, alongside the always-present seed.
func TestSubgraphKindFilterRestrictsReachableNodes(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFilterFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "resource")))
	out, err := sut(cmd, []string{"Transport Layer Security"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## RFC 8446")).
		ShouldNot(it.String(out).Contain("## rescorla-2026-tls13")).
		ShouldNot(it.String(out).Contain("## other-2026"))
}

// arc subgraph "Transport Layer Security" --kind hypothesis
// Scenario 3 from spec.md US3: a filter matching zero reachable nodes
// still yields the seed alone, no error.
func TestSubgraphFilterMatchingZeroReachableYieldsSeedAloneNoErrorCLI(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFilterFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "hypothesis")))
	out, err := sut(cmd, []string{"Transport Layer Security"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## Transport Layer Security")).
		ShouldNot(it.String(out).Contain("## rescorla-2026-tls13")).
		ShouldNot(it.String(out).Contain("## RFC 8446")).
		ShouldNot(it.String(out).Contain("## other-2026"))
}

// arc subgraph "Transport Layer Security" --kind source --tag cryptography --attr status=mature
// Scenario 4 from spec.md US3: a combined kind+tag+attr filter narrows to
// the exact expected subset.
func TestSubgraphCombinedFilterNarrowsFurtherCLI(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFilterFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "source")))
	it.Then(t).Should(it.Nil(cmd.Flags().Set("tag", "cryptography")))
	it.Then(t).Should(it.Nil(cmd.Flags().Set("attr", "status=mature")))
	out, err := sut(cmd, []string{"Transport Layer Security"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("## rescorla-2026-tls13")).
		ShouldNot(it.String(out).Contain("## other-2026")).
		ShouldNot(it.String(out).Contain("## RFC 8446"))
}

// arc subgraph "Transport Layer Security" --depth -1
// Edge case: a negative --depth refuses with a clear usage error, no
// output (spec FR-012).
func TestSubgraphNegativeDepthRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFixture(t, dir)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "-1")))
	out, err := sut(cmd, []string{"Transport Layer Security"})

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).Should(it.Error(out, err).Contain("non-negative integer"))
}

// arc subgraph "Transport Layer Security" --depth two
// Edge case: a non-integer --depth is rejected by Cobra's own flag parsing
// (spec FR-012).
func TestSubgraphNonIntegerDepthRejectedByCobra(t *testing.T) {
	cmd := NewSubgraphCmd()
	err := cmd.Flags().Set("depth", "two")

	it.Then(t).ShouldNot(it.Nil(err))
}

// Edge case: target not an initialized graph (spec FR-010).
func TestSubgraphTargetNotAGraphRefuses(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"Transport Layer Security"})

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).Should(it.Error(out, err).Contain("initialized graph"))
}

const subgraphDanglingEntity = `---
"@id": Dangling
"@type": entity
---
# Dangling

- [[No Such Target]]
`

// Edge case: a dangling link target is silently excluded, not a hard
// failure (spec FR-006).
func TestSubgraphDanglingLinkTargetExcludedCLI(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/Dangling.md", subgraphDanglingEntity)
	chdir(t, dir)

	out, err := sut(NewSubgraphCmd(), []string{"Dangling"})

	it.Then(t).ShouldNot(it.Error(out, err))
	// The dangling reference is still rendered verbatim inside the seed's
	// own body (its Edges list is preserved as-is, like any other node
	// content) — what FR-006 actually excludes is a *separate* node
	// section for the nonexistent target, since no such node exists to
	// serialize. So the only "##" node heading present must be the seed's
	// own.
	it.Then(t).
		Should(it.Equal(1, strings.Count(out, "##"))).
		Should(it.String(out).Contain("## Dangling"))
}

const subgraphCycleA = `---
"@id": CycleA
"@type": entity
---
# CycleA

- [[CycleB]]
`
const subgraphCycleB = `---
"@id": CycleB
"@type": entity
---
# CycleB

- [[CycleA]]
`

// Edge case: a cycle in the graph does not loop forever or duplicate a
// node (spec FR-004).
func TestSubgraphCycleDoesNotLoopOrDuplicateCLI(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/CycleA.md", subgraphCycleA)
	writeGrepNode(t, dir, "entities/CycleB.md", subgraphCycleB)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "5")))
	out, err := sut(cmd, []string{"CycleA"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.Equal(1, strings.Count(out, "## CycleA"))).
		Should(it.Equal(1, strings.Count(out, "## CycleB")))
}

// spec SC-006: arc subgraph never modifies the graph's files or git
// history.
func TestSubgraphIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSubgraphFixture(t, dir)
	chdir(t, dir)

	before := runGit(t, dir, "status", "--short")
	out, err := sut(NewSubgraphCmd(), []string{"Transport Layer Security"})
	it.Then(t).ShouldNot(it.Error(out, err))
	after := runGit(t, dir, "status", "--short")

	it.Then(t).Should(it.Equal(before, after))
}

// research.md D10/D5: when a configured cap truncates a pool, the stats
// block reports it and a plain diagnostic line is printed to stderr.
func TestSubgraphTruncatedPoolReportsStatsAndStderrNotice(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/Hub.md", `---
"@id": Hub
"@type": entity
---
# Hub
`)
	for _, id := range []string{"P1", "P2", "P3"} {
		writeGrepNode(t, dir, "entities/"+id+".md", "---\n\"@id\": "+id+"\n\"@type\": entity\n---\n# "+id+"\n\n- [[Hub]]\n")
	}
	writeGrepNode(t, dir, ".arc/config.yml", "subgraph:\n  backlinkCap: 2\n")
	chdir(t, dir)

	stdout, stderr, err := sutCaptureStderr(t, NewSubgraphCmd(), []string{"Hub"})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.String(stdout).Contain("backlinkTruncated: true")).
		Should(it.String(stderr).Contain("truncated"))
}

const subgraphEntityWithPublished = `---
"@id": PublishedThing
"@type": entity
published: "2026-04-12"
---
# PublishedThing

A published thing.
`

// arc subgraph PublishedThing --depth 0
// Scenario 3 from spec.md US3 / FR-011: a node's own published value
// survives arc subgraph's extraction unchanged, distinct from the
// synthesized patch manifest's own published (today's extraction date,
// research.md D11).
func TestSubgraphPreservesPublishedValueUnchanged(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	writeGrepNode(t, dir, "entities/PublishedThing.md", subgraphEntityWithPublished)
	chdir(t, dir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "0")))
	out, err := sut(cmd, []string{"PublishedThing"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain(`published: "2026-04-12"`))
}

// BUG-001 regression: extracting with --stubs and applying the result into
// a freshly initialized, otherwise empty graph must never leave a
// dangling structural reference behind (spec SC-008: "zero unresolved-
// link violations") — verified by running arc lint against the target
// graph afterward. This is the exact apply-into-a-different-graph flow
// that surfaced the original bug: the seed's own edge to a boundary-
// excluded target (isolated here via --depth 0) is preserved verbatim in
// its rendered body, so without --stubs the target graph would end up
// with a RuleLinkResolves violation ("... does not exist") once linted.
// The assertion is scoped to that specific rule, not a fully clean lint
// run — other, pre-existing lint rules unrelated to referential integrity
// (e.g. citekey/commit-trailer tracing for a synthetic subgraph document)
// are out of this bugfix's scope.
func TestSubgraphStubsRegressionAppliedIntoEmptyGraphHasNoUnresolvedLinks(t *testing.T) {
	srcDir := t.TempDir()
	initGraph(t, srcDir)
	seedSubgraphFixture(t, srcDir)
	chdir(t, srcDir)

	cmd := NewSubgraphCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("depth", "0")))
	it.Then(t).Should(it.Nil(cmd.Flags().Set("stubs", "true")))
	out, err := sut(cmd, []string{"Transport Layer Security"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("## rescorla-2026-tls13"))

	// The patch file lives outside the target graph's own tree — a
	// sibling document, never a node the graph itself would enumerate
	// (mirrors quickstart.md's own /tmp/subgraph.md convention).
	patchDir := t.TempDir()
	patchPath := filepath.Join(patchDir, "extracted.md")
	it.Then(t).Should(it.Nil(os.WriteFile(patchPath, []byte(out), 0o644)))

	targetDir := t.TempDir()
	initGraph(t, targetDir)
	chdir(t, targetDir)

	applyOut, applyErr := sut(NewApplyCmd(), []string{patchPath})
	it.Then(t).ShouldNot(it.Error(applyOut, applyErr))

	lintOut, _ := sut(lint.NewLintCmd(), nil)
	it.Then(t).ShouldNot(it.String(lintOut).Contain("does not exist"))
}
