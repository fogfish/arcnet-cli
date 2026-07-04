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

	"github.com/fogfish/arcnet-cli/internal/bios"
)

// writeGrepNode writes a node file directly, with no git commit — arc grep
// never touches git history, so fixtures for it need no commit of their
// own (unlike apply_test.go's seedNode, which seeds pre-existing state for
// a merge scenario under test).
func writeGrepNode(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
}

const grepSourceTLS13 = `---
kind: source
id: rescorla-2026-tls13
tags: [cryptography]
status: mature
---
# rescorla-2026-tls13

TLS 1.3 is the latest version of the Transport Layer Security protocol.

This protocol replaces earlier, now-deprecated versions.
`

const grepEntityTLS = `---
kind: entity
title: Transport Layer Security
tags: [cryptography]
status: mature
---
# Transport Layer Security

TLS is the successor to SSL.
`

const grepEntityBacklog = `---
kind: entity
title: Another Entity
tags: [cryptography]
status: backlog
---
# Another Entity

TLS appears here too, in a backlog entity.
`

const grepResourceUnrelatedTag = `---
kind: resource
title: Unrelated Note
tags: [other]
status: draft
---
# Unrelated Note

TLS is mentioned here without the cryptography tag.
`

// seedGrepFixture writes a 4-node graph used across most scenarios below:
// unfiltered "TLS" matches exactly 4 nodes (source, 2 entities, 1
// resource), "--tag cryptography" matches exactly 3 (excludes the
// resource), "--kind entity" matches exactly 2, and "--kind entity --attr
// status=mature" narrows to exactly 1.
func seedGrepFixture(t *testing.T, dir string) {
	t.Helper()
	writeGrepNode(t, dir, "sources/rescorla-2026-tls13.md", grepSourceTLS13)
	writeGrepNode(t, dir, "entities/Transport Layer Security.md", grepEntityTLS)
	writeGrepNode(t, dir, "entities/Another Entity.md", grepEntityBacklog)
	writeGrepNode(t, dir, "resources/Unrelated Note.md", grepResourceUnrelatedTag)
}

// arc grep TLS
// Scenario 1 from spec.md US1: every occurrence is reported with kind/id/line.
func TestGrepReportsEveryOccurrenceAcrossWholeGraph(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(4, len(lines)))
	it.Then(t).
		Should(it.String(out).Contain("source  rescorla-2026-tls13  9")).
		Should(it.String(out).Contain("entity  Transport Layer Security  9")).
		Should(it.String(out).Contain("entity  Another Entity  9")).
		Should(it.String(out).Contain("resource  Unrelated Note  9"))
}

// arc grep NoSuchTermAnywhere
// Scenario 2 from spec.md US1: no matches -> no output, non-zero exit.
func TestGrepNoMatchesProducesNoOutputAndNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"NoSuchTermAnywhere"})

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).ShouldNot(it.Nil(err))
}

// arc grep protocol
// Scenario 3 from spec.md US1: a node matching on more than one line
// reports each line separately, in order.
func TestGrepNodeMatchingMultipleLinesReportsEachLineSeparately(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"protocol"})

	it.Then(t).ShouldNot(it.Error(out, err))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(2, len(lines)))
	it.Then(t).
		Should(it.String(lines[0]).Contain("source  rescorla-2026-tls13  9")).
		Should(it.String(lines[1]).Contain("source  rescorla-2026-tls13  11"))
}

// arc grep --kind entity TLS
// Scenario 1 from spec.md US2: --kind restricts to that kind.
func TestGrepKindFilterRestrictsToThatKind(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	cmd := NewGrepCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "entity")))
	out, err := sut(cmd, []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(2, len(lines)))
	it.Then(t).
		Should(it.String(out).Contain("entity  Transport Layer Security")).
		Should(it.String(out).Contain("entity  Another Entity")).
		ShouldNot(it.String(out).Contain("source")).
		ShouldNot(it.String(out).Contain("resource"))
}

// arc grep --tag cryptography TLS
// Scenario 2 from spec.md US2: --tag restricts to nodes carrying that tag.
func TestGrepTagFilterRestrictsToNodesCarryingTag(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	cmd := NewGrepCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("tag", "cryptography")))
	out, err := sut(cmd, []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(3, len(lines)))
	it.Then(t).ShouldNot(it.String(out).Contain("Unrelated Note"))
}

// arc grep --kind entity --attr status=mature TLS
// Scenario 3 from spec.md US2: combined kind+attribute narrows further.
func TestGrepCombinedKindAndAttrFilterNarrowsFurther(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	cmd := NewGrepCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "entity")))
	it.Then(t).Should(it.Nil(cmd.Flags().Set("attr", "status=mature")))
	out, err := sut(cmd, []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(1, len(lines)))
	it.Then(t).Should(it.String(out).Contain("entity  Transport Layer Security"))
}

// arc grep --kind hypothesis TLS
// Scenario 4 from spec.md US2: a filter matching zero nodes behaves like a
// pattern matching nothing — no output, non-zero exit, no error.
func TestGrepFilterMatchingZeroNodesProducesNoOutputAndNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	cmd := NewGrepCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "hypothesis")))
	out, err := sut(cmd, []string{"TLS"})

	it.Then(t).Should(it.Equal("", out))
	it.Then(t).ShouldNot(it.Nil(err))
}

// arc grep TLS | wc -l
// Scenario 1 from spec.md US3: piped through a line-counting tool yields
// the exact match count, no header/footer/summary lines.
func TestGrepOutputPipedThroughLineCounterYieldsExactCount(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(4, len(lines)))
	for _, l := range lines {
		it.Then(t).
			ShouldNot(it.String(l).Contain("checked")).
			ShouldNot(it.String(l).Contain("matches found"))
	}
}

// arc grep --kind source TLS | awk '{print $1, $2, $3}'
// Scenario 2 from spec.md US3: a field-extraction tool splits kind/id/line
// cleanly, remainder is the matched text. Restricted to the source node
// here, whose id has no embedded space, since a title-derived id (e.g. an
// entity's "Transport Layer Security") legitimately can contain spaces —
// whitespace-splitting a single-token id/kind pair is the well-defined
// case this scenario exercises.
func TestGrepOutputFieldsSplitCleanlyByWhitespace(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	cmd := NewGrepCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("kind", "source")))
	out, err := sut(cmd, []string{"TLS"})
	it.Then(t).ShouldNot(it.Error(out, err))

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	it.Then(t).Should(it.Equal(1, len(lines)))
	fields := strings.Fields(lines[0])
	it.Then(t).
		Should(it.Equal("source", fields[0])).
		Should(it.Equal("rescorla-2026-tls13", fields[1])).
		Should(it.Equal("9", fields[2])).
		Should(it.True(strings.HasPrefix(strings.Join(fields[3:], " "), "TLS")))
}

// Scenario 3 from spec.md US3: one output row is always exactly one line —
// no embedded newline can split a single match across two output lines.
func TestGrepEachMatchIsExactlyOneOutputLine(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.True(strings.HasSuffix(out, "\n")))
	nonEmpty := 0
	for _, l := range strings.Split(out, "\n") {
		if l != "" {
			nonEmpty++
		}
	}
	it.Then(t).Should(it.Equal(4, nonEmpty))
}

// arc grep "[TLS"
// Edge case: an invalid <pattern> regexp refuses with a clear error, no scan.
func TestGrepInvalidPatternRefusesWithoutScanning(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"[TLS"})

	it.Then(t).Should(it.Error(out, err).Contain("not a valid pattern"))
}

// Edge case: target not an initialized graph.
func TestGrepTargetNotAGraphRefuses(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"TLS"})

	it.Then(t).Should(it.Error(out, err).Contain("initialized graph"))
}

// Edge case: an unreadable/unparseable node file is excluded and does not
// abort the run — the rest of the graph is still scanned and the
// unreadable path is surfaced in --json output.
func TestGrepUnreadableNodeExcludedRunContinues(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	writeGrepNode(t, dir, "sources/broken.md", "not: [valid, front matter\nTLS mentioned here too\n")
	chdir(t, dir)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewGrepCmd(), []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain(`"unreadable"`)).
		Should(it.String(out).Contain("sources/broken.md")).
		Should(it.String(out).Contain(`"pattern": "TLS"`))
}

// arc grep --color --verbose <long-line pattern>
// Quickstart Scenario 4: --verbose shows the full untruncated line, even on
// a color terminal where default mode ellipsis-fits a long line.
func TestGrepVerboseShowsFullLineColorModeTruncatesDefault(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)

	longLine := "TLS 1.3 removes support for static RSA key exchange, replacing it with ephemeral Diffie-Hellman key agreement for every handshake, a change motivated entirely by forward secrecy."
	content := "---\nkind: source\nid: longline-2026-doc\n---\n# longline-2026-doc\n\n" + longLine + "\n"
	writeGrepNode(t, dir, "sources/longline-2026-doc.md", content)
	// Pinned explicitly rather than relying on the built-in default, so
	// this test stays correct regardless of that default's own value —
	// only that *some* configured width shorter than longLine triggers
	// truncation is under test here.
	writeGrepNode(t, dir, ".arc/config.yml", "grep:\n  maxLineWidth: 80\n")
	chdir(t, dir)

	bios.SCHEMA = bios.SCHEMA_COLOR
	t.Cleanup(func() { bios.SCHEMA = bios.SCHEMA_PLAIN })

	out, err := sut(NewGrepCmd(), []string{"TLS"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("…"))
	it.Then(t).ShouldNot(it.String(out).Contain(longLine))

	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	verboseOut, err := sut(NewGrepCmd(), []string{"TLS"})
	it.Then(t).ShouldNot(it.Error(verboseOut, err))
	it.Then(t).Should(it.String(verboseOut).Contain(longLine))
}

// T048: SCHEMA_PLAIN output (simulating piped/non-TTY) is always the full,
// untruncated, unstyled line, in every mode — truncation/highlight never
// fires off a color terminal (spec FR-006/FR-007, research.md D11).
func TestGrepPlainModeNeverTruncatesEvenLongLine(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)

	longLine := "TLS 1.3 removes support for static RSA key exchange, replacing it with ephemeral Diffie-Hellman key agreement for every handshake, a change motivated entirely by forward secrecy."
	content := "---\nkind: source\nid: longline-2026-doc\n---\n# longline-2026-doc\n\n" + longLine + "\n"
	writeGrepNode(t, dir, "sources/longline-2026-doc.md", content)
	chdir(t, dir)

	out, err := sut(NewGrepCmd(), []string{"TLS"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain(longLine)).
		ShouldNot(it.String(out).Contain("…"))
}

// spec SC-006: arc grep never modifies the graph's files or git history.
func TestGrepIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedGrepFixture(t, dir)
	chdir(t, dir)

	before := runGit(t, dir, "status", "--short")
	out, err := sut(NewGrepCmd(), []string{"TLS"})
	it.Then(t).ShouldNot(it.Error(out, err))
	after := runGit(t, dir, "status", "--short")

	it.Then(t).Should(it.Equal(before, after))
}
