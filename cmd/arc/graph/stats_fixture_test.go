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
	"sort"
	"testing"

	"github.com/fogfish/it/v2"
)

// The six arc stats fixtures (specs/032-arc-stats quickstart.md). Each is
// built here rather than checked in under testdata/ for two reasons the
// on-disk form cannot satisfy: an empty graph needs empty _schema/Class and
// _schema/Property directories, which git cannot track and which a .gitkeep
// placeholder would break (schema.Resolve parses every file it finds
// there), and a hand-counted-composition README.md placed beside a fixture
// root would itself be counted as a foreign file, changing the very figures
// it documents. The composition tables below are the "independently known
// values" SC-002 requires — every assertion in stats_test.go compares
// against a number read off these tables, never against the
// implementation's own output.

// writeStatsFiles writes files (relative slash-separated path → content)
// beneath dir, creating parent directories as needed.
func writeStatsFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		full := filepath.Join(dir, filepath.FromSlash(path))
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
		it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(files[path]), 0o644)))
	}
}

// statsGraphRoot creates an initialized-but-bare graph root: .arc/ plus the
// two schema directories every graph carries, and nothing else.
func statsGraphRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, folder := range []string{".arc", "_schema/Class", "_schema/Property"} {
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, folder), 0o755)))
	}
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".arc", ".gitkeep"), nil, 0o644)))
	return dir
}

// classDoc renders a minimal, schema.Resolve-valid Class document.
func classDoc(id string) string {
	return "---\n\"@id\": \"" + id + "\"\n\"@type\": Class\n---\n# " + id +
		"\n\nThe " + id + " type, declared by this fixture.\n"
}

// propertyDoc renders a minimal, schema.Resolve-valid Property document.
func propertyDoc(id, role, merge string) string {
	return "---\n\"@id\": \"" + id + "\"\n\"@type\": Property\nrole: " + role +
		"\nmerge: " + merge + "\n---\n# " + id +
		"\n\nThe " + id + " predicate, declared by this fixture.\n"
}

// knownSchema is the declared vocabulary of the known/ and moved/
// fixtures: 4 Class documents, 4 Property documents. Node.md is not
// decoration — schema.Resolve makes Node every other type's implicit
// rdfs:subClassOf base and refuses a graph whose _schema/Class/ omits it.
func knownSchema() map[string]string {
	return map[string]string{
		"_schema/Class/Node.md":           classDoc("Node"),
		"_schema/Class/Source.md":         classDoc("Source"),
		"_schema/Class/Entity.md":         classDoc("Entity"),
		"_schema/Class/Timeline.md":       classDoc("Timeline"),
		"_schema/Property/cites.md":       propertyDoc("cites", "edge", "union"),
		"_schema/Property/mentions.md":    propertyDoc("mentions", "edge", "union"),
		"_schema/Property/tags.md":        propertyDoc("tags", "meta", "union"),
		"_schema/Property/granularity.md": propertyDoc("granularity", "meta", "lastWriteWin"),
	}
}

const (
	knownAlpha = `---
"@id": "alpha-2024"
"@type": Source
published: 2024-03-01
tags: [crypto]
---
# alpha-2024

Discusses [[Entity One]] at length.

- cites:: [[Entity One]]
- cites:: [[Entity One]]
- mentions:: [[Entity Two]]
`

	knownBeta = `---
"@id": "beta-2025"
"@type": Source
published: 2025-06-15
tags: [crypto, tls]
---
# beta-2025

Follows on from the earlier work.

- cites:: [[Entity Two]]
`

	knownEntityOne = `---
"@id": "Entity One"
"@type": Entity
---
# Entity One

A concept several sources reach for.
`

	knownEntityTwo = `---
"@id": "Entity Two"
"@type": Entity
status: draft
---
# Entity Two

Closely related to [[Entity One]].

- mentions:: [[Entity One]]
`

	knownEntityThree = `---
"@id": "Entity Three"
"@type": Entity
---
# Entity Three
`

	knownYear2024 = `---
"@id": "2024"
"@type": Timeline
granularity: yearly
---
# 2024

- cites:: [[alpha-2024]]
`

	knownYear2025 = `---
"@id": "2025"
"@type": Timeline
granularity: yearly
---
# 2025

- cites:: [[beta-2025]]
`

	knownMonth202403 = `---
"@id": "2024-03"
"@type": Timeline
granularity: monthly
---
# 2024-03

- cites:: [[alpha-2024]]
`

	knownMonth202506 = `---
"@id": "2025-06"
"@type": Timeline
granularity: monthly
---
# 2025-06

- cites:: [[beta-2025]]
`
)

// knownNodes maps each known/ content node to its canonical folder. moved/
// holds byte-identical content under different folders, so every reported
// figure must be identical for the two (spec FR-004a, contract C9).
func knownNodes(layout map[string]string) map[string]string {
	return map[string]string{
		layout["alpha"]:  knownAlpha,
		layout["beta"]:   knownBeta,
		layout["one"]:    knownEntityOne,
		layout["two"]:    knownEntityTwo,
		layout["three"]:  knownEntityThree,
		layout["y2024"]:  knownYear2024,
		layout["y2025"]:  knownYear2025,
		layout["m20243"]: knownMonth202403,
		layout["m20256"]: knownMonth202506,
	}
}

var knownLayout = map[string]string{
	"alpha":  "Source/alpha-2024.md",
	"beta":   "Source/beta-2025.md",
	"one":    "Entity/Entity One.md",
	"two":    "Entity/Entity Two.md",
	"three":  "Entity/Entity Three.md",
	"y2024":  "timeline/yearly/2024.md",
	"y2025":  "timeline/yearly/2025.md",
	"m20243": "timeline/monthly/2024-03.md",
	"m20256": "timeline/monthly/2025-06.md",
}

var movedLayout = map[string]string{
	"alpha":  "library/papers/alpha-2024.md",
	"beta":   "library/papers/beta-2025.md",
	"one":    "concepts/Entity One.md",
	"two":    "concepts/nested/Entity Two.md",
	"three":  "Entity Three.md",
	"y2024":  "chrono/2024.md",
	"y2025":  "chrono/2025.md",
	"m20243": "chrono/months/2024-03.md",
	"m20256": "chrono/months/2025-06.md",
}

// fixtureKnown is the primary assertion target. Hand-counted composition:
//
//	census nodes       17 = 9 content + 8 schema (4 Class + 4 Property)
//	byType             Class 4, Property 4, Timeline 4, Entity 3, Source 2
//	edges (occurrences) 9 = alpha 3, beta 1, Entity Two 1, 4 timelines 1 each
//	byPredicate        cites 7, mentions 2
//	brokenLinks         0 — every target resolves
//	ingestion          2024:1, 2025:1
//	ingestionMonthly   2024-03:1, 2025-06:1
//	unreadable/foreign none
//	orphans             1 (Entity Three)
//	stubs               1 (Entity Three)
//	avgOutDegree      1.0 = 9 edges / 9 content nodes
//	medianOutDegree   1   — degrees 0,0,1,1,1,1,1,1,3
//	topReferenced     Entity One 5, Entity Two 2, alpha-2024 2, beta-2025 2
//	schema            classes 4 declared / 3 used (Node is declared but
//	                  never a node's own type), properties 4/4 used,
//	                  undeclared predicates [status]
//	content           inlineRefs 2, attributeValues 8, nodesWithoutPublished 7
func fixtureKnown(t *testing.T) string {
	t.Helper()
	dir := statsGraphRoot(t)
	writeStatsFiles(t, dir, knownSchema())
	writeStatsFiles(t, dir, knownNodes(knownLayout))
	return dir
}

// fixtureMoved holds known/'s exact node set reorganized into folders arc
// never creates. Every figure it reports must equal fixtureKnown's.
func fixtureMoved(t *testing.T) string {
	t.Helper()
	dir := statsGraphRoot(t)
	writeStatsFiles(t, dir, knownSchema())
	writeStatsFiles(t, dir, knownNodes(movedLayout))
	return dir
}

// fixtureEmpty is an initialized graph holding no node at all: .arc/ plus
// two empty schema directories (spec FR-010, contract C8).
func fixtureEmpty(t *testing.T) string {
	t.Helper()
	return statsGraphRoot(t)
}

// fixtureBroken carries unresolved links, some repeated within one node and
// some shared across nodes. Hand-counted composition:
//
//	census nodes    8 = 3 content + 5 schema (3 Class + 2 Property)
//	edges           5 = A 3, B 1, C 1
//	brokenLinks     4 = A {Ghost, Missing One} + B {Ghost, Missing One}
//	unresolvedTargets  Ghost refs 2, Missing One refs 2
//
// A references Ghost twice inline and Missing One twice structurally: both
// collapse to one count per node, which is precisely arc lint's own
// linkResolves rule (contract C5).
func fixtureBroken(t *testing.T) string {
	t.Helper()
	dir := statsGraphRoot(t)
	writeStatsFiles(t, dir, map[string]string{
		"_schema/Class/Node.md":        classDoc("Node"),
		"_schema/Class/Entity.md":      classDoc("Entity"),
		"_schema/Class/Source.md":      classDoc("Source"),
		"_schema/Property/cites.md":    propertyDoc("cites", "edge", "union"),
		"_schema/Property/mentions.md": propertyDoc("mentions", "edge", "union"),
		"Entity/A.md": `---
"@id": "A"
"@type": Entity
---
# A

Mentions [[Ghost]], and then [[Ghost]] once more.

- cites:: [[Missing One]]
- cites:: [[Missing One]]
- mentions:: [[B]]
`,
		"Entity/B.md": `---
"@id": "B"
"@type": Entity
---
# B

Also reaches for [[Ghost]].

- cites:: [[Missing One]]
`,
		"Entity/C.md": `---
"@id": "C"
"@type": Entity
---
# C

Everything here resolves.

- mentions:: [[A]]
`,
	})
	return dir
}

// fixtureMessy mixes a well-formed node, a node file whose front matter is
// malformed, and two ordinary host-project markdown files. Hand-counted:
//
//	census nodes  4 = 1 content (Entity/Good.md) + 3 schema
//	unreadable    ["Entity/Broken.md"]
//	foreign       ["README.md", "docs/design.md"]
//
// The distinction is the point (spec FR-009, research.md D8): a host
// README beside the graph is expected, not damage.
func fixtureMessy(t *testing.T) string {
	t.Helper()
	dir := statsGraphRoot(t)
	writeStatsFiles(t, dir, map[string]string{
		"_schema/Class/Node.md":     classDoc("Node"),
		"_schema/Class/Entity.md":   classDoc("Entity"),
		"_schema/Property/cites.md": propertyDoc("cites", "edge", "union"),
		"Entity/Good.md": `---
"@id": "Good"
"@type": Entity
---
# Good

A perfectly ordinary node.
`,
		"Entity/Broken.md": `---
"@id": "Broken
"@type": [Entity
  bad: : :
---
# Broken
`,
		"README.md":      "# Host project\n\nThis repository holds more than a graph.\n",
		"docs/design.md": "# Design notes\n\nOrdinary host markdown, not a node.\n",
	})
	return dir
}

// fixtureNotGraph is a directory holding markdown but no .arc/ (spec
// FR-011).
func fixtureNotGraph(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeStatsFiles(t, dir, map[string]string{"notes.md": "# Notes\n\nNot a graph.\n"})
	return dir
}
