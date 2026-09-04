//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package lint

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// TestMain sets a fake git identity for the whole test binary — arc lint
// shells out to a real `git log` via internal/adapter/git.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "arc-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "arc-test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "arc-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "arc-test@example.com")
	os.Exit(m.Run())
}

func sut(cmd *cobra.Command, args []string) (string, error) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	err := cmd.RunE(cmd, args)

	w.Close()
	os.Stdout = stdout
	return <-ch, err
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Nil(os.Chdir(dir)))
	t.Cleanup(func() { os.Chdir(original) })
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	it.Then(t).Should(it.Nil(err))
	return string(out)
}

// initGraph builds a minimal, real, git-committed graph root — equivalent
// to arc init's own layout — without depending on cmd/arc/ctrl. _schema/ is
// seeded with appschema.Seed()'s own real output — the full CORE
// vocabulary, matching arc init's actual behavior — so Resolve's fail-fast
// validation never rejects this fixture.
func initGraph(t *testing.T, dir string) {
	t.Helper()
	writeGraphLayout(t, dir)

	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(init): empty knowledge graph")
}

// writeGraphLayout writes the canonical layout and seeded _schema/ without
// touching git — shared by initGraph and the nested-repository fixture
// (BUG-002 T051).
func writeGraphLayout(t *testing.T, dir string) {
	t.Helper()
	for _, folder := range []string{"Source", "Entity", "Resource", "timeline/yearly", "timeline/monthly", "_schema/Class", "_schema/Property"} {
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, folder), 0o755)))
	}
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, ".arc"), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".arc", ".gitkeep"), nil, 0o644)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".arc/\n"), 0o644)))

	for path, raw := range appschema.Seed() {
		writeNode(t, dir, path, string(raw))
	}
}

func writeNode(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
}

// ingestSource writes a source node and commits it with the exact
// "graph(ingest):"/"Source-Id:" shape arc apply itself produces (CORE
// §11.3), so RuleIngestCommit's check finds exactly one matching commit.
func ingestSource(t *testing.T, dir, id, title, content string) {
	t.Helper()
	writeNode(t, dir, "Source/"+id+".md", content)
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(ingest): "+id+" — "+title+"\n\nSource-Id: "+id+"\n")
}

func commitAll(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", message)
}

const conformantSource = `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
created: "2026-04-12"
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`

const conformantEntity = `---
"@id": Widget
"@type": Entity
category: [independent, abstract, occurrent, script]
published: "2026-04-12"
created: "2026-04-12"
---
# Widget

A test entity.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
`

// buildConformantGraph builds a graph satisfying every base CORE §14 rule:
// valid front-matter/kind, unique basenames, resolvable links, source
// citekey identity, entity Sowa category, derived provenance (Widget links
// to the source), registered camelCase predicates, one ingest commit, and
// no merge-conflict markers.
func buildConformantGraph(t *testing.T, dir string) {
	t.Helper()
	initGraph(t, dir)
	ingestSource(t, dir, "foo-2026-x", "A Test Document", conformantSource)
	writeNode(t, dir, "Entity/Widget.md", conformantEntity)
	commitAll(t, dir, "seed: Widget entity")
}

// arc lint
// Scenario 1 from spec.md US1: a fully conformant graph passes cleanly
func TestLintConformantGraphPassesCleanly(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(out).Contain("2 nodes checked, 2 passing, 0 failing"))
}

// arc lint
// spec.md Clarifications Q1/Q3, FR-015: a freshly initialized graph's
// _schema/ documents never appear in arc lint's checked-node count or
// violation list.
func TestLintExcludesSchemaDocumentsFromCheckedCount(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(out).Contain("2 nodes checked, 2 passing, 0 failing")).
		ShouldNot(it.String(out).Contain("_schema"))
}

// arc lint
// spec.md Clarifications Q3: an ordinary content node sharing a basename
// with a schema document (e.g. Entity/hypothesis.md vs.
// _schema/Class/hypothesis.md) is not reported as a basename collision.
func TestLintSchemaBasenameDoesNotCollideWithContentNode(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "_schema/Class/Hypothesis.md", "---\n\"@id\": Hypothesis\n\"@type\": Class\nmerge: union\n---\n# Hypothesis\n\nA domain type registered by this test fixture.\n")
	writeNode(t, dir, "Entity/Hypothesis.md", "---\n\"@id\": Hypothesis\n\"@type\": Entity\ncategory: [independent, abstract, occurrent, script]\npublished: \"2026-04-12\"\ncreated: \"2026-04-12\"\n---\n# Hypothesis\n\nA namesake entity, unrelated to the schema kind of the same name.\n\n## MentionedIn\n- mentionedIn:: [[foo-2026-x]]\n")
	commitAll(t, dir, "seed: hypothesis entity and schema doc")
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("uniqueBasename")).
		Should(it.String(out).Contain("3 nodes checked, 3 passing, 0 failing"))
}

// arc lint
// Scenario 2 from spec.md US1: violations across multiple rules in the
// same file are all reported in one run (FR-013 — never stop at the first)
func TestLintReportsEveryViolationAcrossRulesInSameFile(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	broken := `---
"@id": Widget
"@type": Entity
title: Widget
category: [independent, abstract, occurrent]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- mentions:: [[Nonexistent Node]]
`
	writeNode(t, dir, "Entity/Widget.md", broken)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[linkResolves]")).
		Should(it.String(out).Contain(`target "Nonexistent Node" does not exist`)).
		Should(it.String(out).Contain("[entityCategory]")).
		Should(it.String(out).Contain("found 3")).
		Should(it.String(out).Contain("2 nodes checked, 1 passing, 1 failing"))
}

// arc lint
// Scenario 1 from spec.md US2: a broken link is caught precisely, file and
// line named, everything else stays clean
func TestLintUnresolvedLinkReportedPrecisely(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	broken := `---
"@id": Widget
"@type": Entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- mentions:: [[Not A Real Node]]
`
	writeNode(t, dir, "Entity/Widget.md", broken)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Entity/Widget.md:")).
		Should(it.String(out).Contain("[linkResolves]")).
		Should(it.String(out).Contain(`target "Not A Real Node" does not exist`)).
		Should(it.String(out).Contain("2 nodes checked, 1 passing, 1 failing")).
		ShouldNot(it.String(out).Contain("entityCategory"))
}

// arc lint
// Scenario 2 from spec.md US2: a resolvable link produces no violation
func TestLintResolvableLinkProducesNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("linkResolves"))
}

// arc lint
// Scenario 3 from spec.md US2: two independently created nodes sharing a
// basename are reported, naming both files
func TestLintBasenameCollisionNamesBothFiles(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	widget := readFile(t, filepath.Join(dir, "Entity", "Widget.md"))
	writeNode(t, dir, "Resource/Widget.md", widget)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[uniqueBasename]")).
		Should(it.String(out).Contain(`basename "Widget" is used by more than one file`)).
		Should(it.String(out).Contain("Entity/Widget.md")).
		Should(it.String(out).Contain("Resource/Widget.md")).
		Should(it.String(out).Contain("3 nodes checked"))
}

// arc lint
// Scenario 1 from spec.md US3: an unresolved merge conflict is reported,
// with the first marker's line, and no secondary front-matter noise for
// the same file
func TestLintUnresolvedMergeConflictReportedOnce(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	conflicted := "<<<<<<< HEAD\nkind: entity\n=======\nkind: entity\ncategory: [independent, abstract, occurrent, script]\n>>>>>>> feature-branch\n"
	writeNode(t, dir, "Entity/Broken.md", conflicted)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Entity/Broken.md:1")).
		Should(it.String(out).Contain("[mergeConflict]")).
		Should(it.String(out).Contain("3 nodes checked, 2 passing, 1 failing")).
		ShouldNot(it.String(out).Contain("frontMatter"))
}

// arc lint
// Scenario 2 from spec.md US3: a conflict-marker-free graph reports none
func TestLintNoConflictMarkersReportsNone(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("mergeConflict"))
}

// arc lint
// Acceptance Scenario 1 from spec.md US3: a node file using the legacy
// "kind" front-matter field (with no "@id"/"@type") is rejected as an
// unsupported pre-0.5 format, reported under the frontMatter rule, naming
// the offending file. Also confirms lint made no write to the graph (spec
// FR-012, US3 Acceptance Scenario 4) by comparing `git status --porcelain`
// before and after the run.
func TestLintOldFormatKindFieldReportsFrontMatterViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	legacy := "---\nkind: entity\ntitle: Legacy\ncategory: [independent]\n---\n# Legacy\n\nAn entity written before this feature shipped.\n"
	writeNode(t, dir, "Entity/Legacy.md", legacy)
	chdir(t, dir)

	before := runGit(t, dir, "status", "--porcelain")

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Entity/Legacy.md")).
		Should(it.String(out).Contain("[frontMatter]")).
		Should(it.String(out).Contain(`legacy "kind" field present`))

	after := runGit(t, dir, "status", "--porcelain")
	it.Then(t).Should(it.Equal(before, after))
}

// arc lint
// Acceptance Scenario 2 from spec.md US3: a node file declaring "@type" but
// missing "@id" is rejected rather than falling back to any other field
// (e.g. title), reported under the frontMatter rule, naming the offending
// file.
func TestLintMissingIdReportsFrontMatterViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	missingID := "---\n\"@type\": Entity\ntitle: Legacy\ncategory: [independent]\n---\n# Legacy\n\nAn entity missing its mandatory \"@id\" field.\n"
	writeNode(t, dir, "Entity/Legacy.md", missingID)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Entity/Legacy.md")).
		Should(it.String(out).Contain("[frontMatter]")).
		Should(it.String(out).Contain(`missing mandatory "@id" field`))
}

// arc lint
// Acceptance Scenario 3 from spec.md US3: a node file whose "@id" does not
// equal its own filename basename is rejected rather than the mismatch
// being silently accepted. core.ParseNode itself cannot perform this check
// (it has no filename parameter — see internal/core/markdown.go's ParseNode
// doc comment); arc lint enforces it universally, for every node type, via
// a frontMatter violation in internal/app/lint/service.Lint itself (not
// just for "Source" nodes via the narrower, pre-existing sourceCitekey
// rule) — any node type demonstrates the behavior.
func TestLintIdMismatchedBasenameReportsFrontMatterViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	mismatched := `---
"@id": wrong-id
"@type": Source
title: "A Mismatched Source"
published: "2026-04-12"
---
# wrong-id

A source whose "@id" does not equal its own filename basename.
`
	writeNode(t, dir, "Source/mismatched-2026-x.md", mismatched)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Source/mismatched-2026-x.md")).
		Should(it.String(out).Contain("[frontMatter]")).
		Should(it.String(out).Contain(`"@id" wrong-id does not match this file's basename mismatched-2026-x`))
}

// arc lint --verbose
// User's explicit -v requirement: verbose mode lists every node's status;
// default mode lists only the failing node, same summary line closes both
func TestLintVerboseListsEveryNodeDefaultListsOnlyFailing(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	broken := `---
"@id": Widget
"@type": Entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- mentions:: [[Not A Real Node]]
`
	writeNode(t, dir, "Entity/Widget.md", broken)
	chdir(t, dir)

	defaultOut, err := sut(NewLintCmd(), nil)
	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).
		ShouldNot(it.String(defaultOut).Contain("foo-2026-x.md\n")).
		Should(it.String(defaultOut).Contain("2 nodes checked, 1 passing, 1 failing"))

	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	verboseOut, err := sut(NewLintCmd(), nil)
	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).
		Should(it.String(verboseOut).Contain("Source/foo-2026-x.md")).
		Should(it.String(verboseOut).Contain("[linkResolves]")).
		Should(it.String(verboseOut).Contain("2 nodes checked, 1 passing, 1 failing"))
}

// arc lint
// Edge case from spec.md FR-017: target not an initialized graph
func TestLintTargetNotAGraphRefuses(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).Should(it.Error(out, err).Contain("initialized graph"))
}

// arc lint
// spec.md US2 Acceptance Scenario 5 / FR-014, mirrored read-only: a
// malformed _schema/ document aborts arc lint (which never writes to the
// graph regardless, but must still refuse to run against a broken schema).
func TestLintMalformedSchemaDocumentRefuses(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	// spec 012 FR-020 (Bugfix 018/BUG-001): a Class document's own "merge"
	// field is no longer validated as mandatory, so corrupting it (as this
	// test previously did via "merge: bogus") no longer makes the document
	// malformed — its mandatory descriptive body is corrupted instead.
	entityDoc := readFile(t, filepath.Join(dir, "_schema", "Class", "Entity.md"))
	writeNode(t, dir, "_schema/Class/Entity.md", strings.ReplaceAll(entityDoc, "A node for a subject occurring in sources, typed by Sowa category.", ""))
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).Should(it.Error(out, err).Contain("_schema/Class/Entity.md"))
}

// arc lint --json
// --json output contract from contracts/cli-contract.md
func TestLintJSONOutput(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(out).Contain(`"nodes"`)).
		Should(it.String(out).Contain(`"passing": 2`)).
		Should(it.String(out).Contain(`"failing": 0`))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	it.Then(t).Should(it.Nil(err))
	return string(content)
}

// arc lint
// Scenario 1 from spec.md User Story 1: a node missing a type-required
// predicate is reported under [typeRequires], naming the predicate, type,
// and file.
func TestLintTypeRequiresMissingPredicateReported(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	missingMentions := `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document.
`
	writeNode(t, dir, "Source/foo-2026-x.md", missingMentions)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Source/foo-2026-x.md")).
		Should(it.String(out).Contain("[typeRequires]")).
		Should(it.String(out).Contain(`type "Source" requires predicate "mentions", but this node does not carry it`))
}

// arc lint
// Scenario 2 from spec.md User Story 1: a node carrying every predicate its
// type requires produces no [typeRequires] violation.
func TestLintTypeRequiresAllPresentNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeRequires"))
}

// arc lint
// Scenario 3 from spec.md User Story 1: a node of an unregistered type
// produces no [typeRequires] violation — only the pre-existing
// unrecognizedKind violation covers that gap.
func TestLintTypeRequiresUnregisteredTypeNotDoubleReported(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	unregistered := "---\n\"@id\": A Test Hypothesis\n\"@type\": hypothesis\ntitle: A Test Hypothesis\n---\n# A Test Hypothesis\n\nA conclusion.\n"
	writeNode(t, dir, "hypothesis/A Test Hypothesis.md", unregistered)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[unrecognizedKind]")).
		ShouldNot(it.String(out).Contain("typeRequires"))
}

// arc lint
// Scenario 1 from spec.md User Story 2: a node carrying a predicate its
// type's schema lists under neither Requires nor Optional is reported under
// [typeOptional].
func TestLintTypeOptionalUnlistedPredicateReported(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	extraPredicate := `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
status: read
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "Source/foo-2026-x.md", extraPredicate)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[typeOptional]")).
		Should(it.String(out).Contain(`predicate "status" is not permitted by type "Source" (not listed under its Requires or Optional)`))
}

// arc lint
// Scenario 2 from spec.md User Story 2: a node carrying only predicates
// listed under its type's Requires/Optional sections produces no
// [typeOptional] violation.
func TestLintTypeOptionalAllListedPredicatesNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeOptional"))
}

// arc lint
// Scenario 3 from spec.md User Story 2: the universal identity predicates
// ("@id"/"@type") are never reported as "not permitted" by a type's schema —
// structurally guaranteed, since core.identityFields strips them out of the
// manifest before Attrs is ever populated, so they never enter the
// occurrence set checkTypeOptional walks.
func TestLintIdentityPredicatesNeverReportedAsTypeOptionalViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain(`predicate "@id"`)).
		ShouldNot(it.String(out).Contain(`predicate "@type"`))
}

// arc lint
// Scenario 1 from spec.md User Story 3: an unresolved merge-conflict-free
// node missing "@type" entirely fails to parse at all — the pre-existing
// frontMatter violation already covers this gap (spec Edge Cases: "the new
// missing/unquoted-key check applies to nodes whose front matter is
// otherwise parseable"; core.ParseNode itself rejects a document with no
// "@type" before any identityQuoting-level check could ever run against it).
func TestLintMissingTypeReportsFrontMatterViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	missingType := "---\n\"@id\": Legacy\ntitle: Legacy\ncategory: [independent]\n---\n# Legacy\n\nAn entity missing its mandatory \"@type\" field.\n"
	writeNode(t, dir, "Entity/Legacy.md", missingType)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Entity/Legacy.md")).
		Should(it.String(out).Contain("[frontMatter]")).
		Should(it.String(out).Contain(`missing mandatory "@type" field`))
}

// arc lint
// Scenario 2 from spec.md User Story 3: a node's front matter writing "@id"
// as a bare (unquoted) key is reported under [identityQuoting], naming the
// key and line, while the node still parses correctly (no frontMatter
// violation).
func TestLintUnquotedIdKeyReportsIdentityQuotingViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	unquotedID := `---
@id: foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "Source/foo-2026-x.md", unquotedID)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Source/foo-2026-x.md")).
		Should(it.String(out).Contain("[identityQuoting]")).
		Should(it.String(out).Contain(`"@id" must be a quoted YAML string key, found it unquoted`)).
		ShouldNot(it.String(out).Contain("[frontMatter]"))
}

// arc lint
// Scenario 3 from spec.md User Story 3: a node with both "@id" and "@type"
// present and quoted produces no [identityQuoting] violation.
func TestLintQuotedIdentityKeysNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("identityQuoting"))
}

// registerCitesAsExample writes a domain-registered cito:-aligned citation
// predicate to dir's schema — not one of the old hardcoded citoPredicates
// (research.md D3) — and commits it.
func registerCitesAsExample(t *testing.T, dir string) {
	t.Helper()
	writeNode(t, dir, "_schema/Property/citesAsExample.md", `---
"@id": citesAsExample
"@type": Property
aligned: "cito:citesAsExample"
merge: union
role: edge
---
# citesAsExample

A domain-specific citation relationship, not built into arc itself.
`)
	commitAll(t, dir, "seed: register citesAsExample predicate")
}

// arc lint
// Scenario 1 from spec.md User Story 4: a domain-registered predicate whose
// alignment is a cito: vocabulary term is recognized as a valid citation
// predicate — with zero changes to arc's own source code.
func TestLintDomainRegisteredCitoPredicateAcceptedAsCitation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerCitesAsExample(t, dir)

	citing := `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document. [citesAsExample:: [[Widget]]]

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "Source/foo-2026-x.md", citing)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)
	_ = err

	it.Then(t).
		ShouldNot(it.String(out).Contain("[citationPredicate]"))
}

// arc lint
// Scenario 2 from spec.md User Story 4: an unregistered/non-cito:-aligned
// predicate used as a citation is still reported.
func TestLintUnregisteredCitationPredicateStillReported(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	citing := `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document. [bogusCite:: [[Widget]]]

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "Source/foo-2026-x.md", citing)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[citationPredicate]")).
		Should(it.String(out).Contain(`citation predicate "bogusCite" is not a recognized cito-aligned predicate`))
}

// arc lint
// Scenario 3 from spec.md User Story 4: a graph whose schema registers zero
// cito:-aligned predicates reports every citation usage as a violation —
// there is no built-in fallback vocabulary.
func TestLintZeroCitoAlignedPredicatesRejectsEveryCitation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	entries, err := os.ReadDir(filepath.Join(dir, "_schema", "Property"))
	it.Then(t).Should(it.Nil(err))
	for _, e := range entries {
		path := filepath.Join(dir, "_schema", "Property", e.Name())
		content := readFile(t, path)
		if !strings.Contains(content, "cito:") {
			continue
		}
		it.Then(t).Should(it.Nil(os.WriteFile(path, []byte(strings.ReplaceAll(content, "cito:", "test:")), 0o644)))
	}
	commitAll(t, dir, "seed: strip cito: alignment from every predicate")

	citing := `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document. [cites:: [[Widget]]]

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "Source/foo-2026-x.md", citing)
	chdir(t, dir)

	out, err2 := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err2)).
		Should(it.String(out).Contain("[citationPredicate]")).
		Should(it.String(out).Contain(`citation predicate "cites" is not a recognized cito-aligned predicate`))
}

// registerHighlightPredicate registers a role:"text" predicate and adds it
// to entity's Optional list, so a node can carry it in the wrong structural
// position (an edge bullet instead of prose) for the predicateRole check.
func registerHighlightPredicate(t *testing.T, dir string) {
	t.Helper()
	writeNode(t, dir, "_schema/Property/highlight.md", `---
"@id": highlight
"@type": Property
merge: union
role: text
---
# highlight

A short highlighted note about the entity, always written as prose.
`)
	writeNode(t, dir, "_schema/Class/Entity.md", `---
"@id": Entity
"@type": Class
merge: union
---
# Entity

A node for a subject occurring in sources, typed by Sowa category.

## Requires
- required:: [[category]]
- required:: [[definition]]
- required:: [[mentionedIn]]

## Optional
- optional:: [[aliases]]
- optional:: [[tags]]
- optional:: [[highlight]]
`)
	commitAll(t, dir, "seed: register highlight predicate on entity")
}

// arc lint
// Scenario 1 from spec.md User Story 5: a predicate registered with role
// "text" written as a typed edge bullet is reported under [predicateRole],
// naming the predicate, its declared role, and where it was found.
func TestLintTextRolePredicateAsEdgeReportsPredicateRoleViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerHighlightPredicate(t, dir)

	misplaced := `---
"@id": Widget
"@type": Entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
- highlight:: [[foo-2026-x]]
`
	writeNode(t, dir, "Entity/Widget.md", misplaced)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Entity/Widget.md")).
		Should(it.String(out).Contain("[predicateRole]")).
		Should(it.String(out).Contain(`predicate "highlight" is registered with role "text", but appears as a edge occurrence`))
}

// arc lint
// Scenario 2 from spec.md User Story 5: a node where every predicate's
// usage matches its declared role produces no [predicateRole] violation.
func TestLintPredicateRoleMatchingNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("predicateRole"))
}

// arc lint
// Scenario 3 from spec.md User Story 5: a node using an unregistered
// predicate produces no [predicateRole] violation — only the pre-existing
// predicateRegistered violation covers that gap.
func TestLintUnregisteredPredicateNoPredicateRoleViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	unregistered := `---
"@id": Widget
"@type": Entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
- totallyUnknownPred:: [[foo-2026-x]]
`
	writeNode(t, dir, "Entity/Widget.md", unregistered)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[predicateRegistered]")).
		ShouldNot(it.String(out).Contain("predicateRole"))
}

// arc lint
// BUG-001 regression: an entity node using a §10.5 semantic predicate
// (conformsTo) and notes — exactly per ARCNET-CORE §11.3's own worked
// example — produces no [typeOptional] violation. This is this bug's own
// originally-reported false positive.
func TestLintEntitySemanticPredicateAndNotesNoTypeOptionalViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	widget := `---
"@id": Widget
"@type": Entity
category: [independent, abstract, occurrent, script]
published: "2026-04-12"
created: "2026-04-12"
---
# Widget

A test entity.

Additional prose about the entity.

- conformsTo:: [[foo-2026-x]]

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
`
	writeNode(t, dir, "Entity/Widget.md", widget)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeOptional"))
}

// arc lint
// BUG-001 regression, narrowed by ARCNET-CORE v0.11
// (specs/022-reference-type-folders): BUG-001's scope decision was that
// §10.5 semantic predicates are usable on entity *or resource* nodes. The
// entity half still holds and is asserted above. The resource half does not
// survive v0.11: CORE §11.4 redefines Resource as a fragment of an ingested
// document and lists "notes" as its sole optional, so the structural and
// semantic optionals it used to carry are not carried over
// (data-model.md §1.1, contract C2). §11.6 gives Reference none of them
// either, so under v0.11 semantic predicates are permitted on Entity alone.
//
// This asserts that narrowed contract directly rather than deleting the
// case: a conformsTo on a Resource is now a real [typeOptional] violation,
// and the required text/tags/mentionedIn it omits are real [typeRequires]
// ones. Restoring the old optionals to Resource would silently turn this
// test red, which is the point.
func TestLintResourceSemanticPredicateIsNoLongerPermitted(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	resource := `---
"@id": RFC 8446
"@type": Resource
ref: standard
published: "2026-04-12"
created: "2026-04-12"
---
# RFC 8446

A normative specification.

- conformsTo:: [[foo-2026-x]]
`
	writeNode(t, dir, "Resource/RFC 8446.md", resource)
	chdir(t, dir)

	// The error is the bios.ErrSilent sentinel arc lint returns to carry a
	// non-zero exit code once it has findings; the findings themselves are
	// on stdout, which is what this asserts.
	out, _ := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.String(out).Contain(`predicate "conformsTo" is not permitted by type "Resource"`)).
		Should(it.String(out).Contain(`predicate "ref" is not permitted by type "Resource"`)).
		Should(it.String(out).Contain(`type "Resource" requires predicate "tags"`)).
		Should(it.String(out).Contain(`type "Resource" requires predicate "mentionedIn"`))
}

// arc lint
// specs/022-reference-type-folders US1 Acceptance Scenario 6: the
// external-work shape the retired Resource carried is conformant again once
// the node is typed Reference — which is where those predicates went. The
// node below is the one above, retyped and given the title §11.6 requires;
// its leading prose supplies the third required predicate, relevance.
//
// The isCitedBy backlink does double duty: it is one of Reference's own
// §11.6 optionals, and it is the link to a source node checkDerivedProvenance
// requires of every non-Source, non-Timeline node.
func TestLintReferenceCarriesExternalWorkPredicatesConformantly(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	// "ref"/"status" are retired outright under CORE 0.12 (BUG-001/FR-034) —
	// no longer part of Reference's conformant shape.
	reference := `---
"@id": RFC 8446
"@type": Reference
title: The TLS 1.3 Protocol
published: "2026-04-12"
created: "2026-04-12"
---
# RFC 8446

Why this un-ingested work is worth pointing at.

- isCitedBy:: [[foo-2026-x]]
`
	writeNode(t, dir, "Reference/RFC 8446.md", reference)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(out).Contain("RFC 8446.md"))
}

// arc lint
// spec 017 US1 Acceptance Scenarios 1.2/1.3: a source node missing
// "published" — required only via the implicit Node base, not declared
// directly on "Source" (data-model.md's reshaped-types table) — is flagged
// under [typeRequires] exactly as a directly-required predicate would be,
// and the violation clears once the node carries it.
func TestLintInheritedRequiredPredicateEnforcedLikeDirect(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	missingPublished := `---
"@id": foo-2026-x
"@type": Source
title: "A Test Document"
author: [Test Author]
created: "2026-04-12"
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "Source/foo-2026-x.md", missingPublished)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Source/foo-2026-x.md")).
		Should(it.String(out).Contain("[typeRequires]")).
		Should(it.String(out).Contain(`type "Source" requires predicate "published", but this node does not carry it`))

	restored := strings.Replace(missingPublished, "created: \"2026-04-12\"", "published: \"2026-04-12\"\ncreated: \"2026-04-12\"", 1)
	writeNode(t, dir, "Source/foo-2026-x.md", restored)

	out, err = sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeRequires"))
}

// registerType writes a minimal custom type schema document, optionally
// declaring one or more subClassOf base types. Every registered type
// permits "mentions" so a fixture node of this type can link back to a
// source node without tripping [derivedProvenance] (research.md D8) or
// [typeOptional].
func registerType(t *testing.T, dir, name string, required []string, subClassOf []string) {
	t.Helper()
	var body strings.Builder
	body.WriteString("---\n\"@id\": " + name + "\n\"@type\": Class\nmerge: union\n---\n# " + name + "\n\n" + name + " test type.\n")
	for _, base := range subClassOf {
		body.WriteString("\n- subClassOf:: [[" + base + "]]\n")
	}
	if len(required) > 0 {
		body.WriteString("\n## Requires\n")
		for _, r := range required {
			body.WriteString("- required:: [[" + r + "]]\n")
		}
	}
	body.WriteString("\n## Optional\n- optional:: [[mentions]]\n")
	writeNode(t, dir, "_schema/Class/"+name+".md", body.String())
}

// arc lint
// spec 017 US2 Acceptance Scenarios 2.1-2.3: two independent custom base
// types, each requiring a distinct predicate, combined via multiple
// subClassOf declarations on a third type — arc lint enforces both
// inherited predicates, with no duplicate violation when both bases also
// separately contribute the shared implicit Node base.
func TestLintComposedTypeEnforcesBothBasePredicates(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "Citable", []string{"doi"}, nil)
	registerType(t, dir, "Timestamped", []string{"updated"}, nil)
	registerType(t, dir, "Dataset", nil, []string{"Citable", "Timestamped"})
	commitAll(t, dir, "seed: Citable/Timestamped/Dataset types")

	missingBoth := `---
"@id": widget-dataset
"@type": Dataset
published: "2026-04-12"
created: "2026-04-12"
---
# widget-dataset

A dataset missing both inherited predicates.

- mentions:: [[foo-2026-x]]
`
	writeNode(t, dir, "Resource/widget-dataset.md", missingBoth)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain(`type "Dataset" requires predicate "doi", but this node does not carry it`)).
		Should(it.String(out).Contain(`type "Dataset" requires predicate "updated", but this node does not carry it`))

	complete := `---
"@id": widget-dataset
"@type": Dataset
published: "2026-04-12"
created: "2026-04-12"
doi: "10.1000/example"
updated: "2026-04-12"
---
# widget-dataset

A dataset carrying every inherited predicate from both bases.

- mentions:: [[foo-2026-x]]
`
	writeNode(t, dir, "Resource/widget-dataset.md", complete)

	out, err = sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeRequires"))
}

// arc lint
// spec 017 US3 Acceptance Scenario 3.1: a three-level subClassOf chain
// (Level1 -> Level2 -> Level3) resolves transitively — Level1's effective
// contract requires Level3's own predicate even though Level1 has no direct
// subClassOf relationship to Level3.
func TestLintThreeLevelChainResolvesTransitively(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "Level3", []string{"year"}, nil)
	registerType(t, dir, "Level2", nil, []string{"Level3"})
	registerType(t, dir, "Level1", nil, []string{"Level2"})
	commitAll(t, dir, "seed: three-level chain types")

	missingYear := `---
"@id": bottom-of-chain
"@type": Level1
published: "2026-04-12"
created: "2026-04-12"
---
# bottom-of-chain

Missing the ancestor-required predicate.

- mentions:: [[foo-2026-x]]
`
	writeNode(t, dir, "Resource/bottom-of-chain.md", missingYear)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain(`type "Level1" requires predicate "year", but this node does not carry it`))
}

// arc lint
// spec 017 US3 Acceptance Scenario 3.2: a diamond-shaped hierarchy (two
// branches converging on one common ancestor) counts the ancestor's
// predicate exactly once — one violation, not two, when it's missing.
func TestLintDiamondHierarchyCountsCommonAncestorOnce(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "Apex", []string{"status"}, nil)
	registerType(t, dir, "BranchA", nil, []string{"Apex"})
	registerType(t, dir, "BranchB", nil, []string{"Apex"})
	registerType(t, dir, "Bottom", nil, []string{"BranchA", "BranchB"})
	commitAll(t, dir, "seed: diamond hierarchy types")

	missingStatus := `---
"@id": diamond-bottom
"@type": Bottom
published: "2026-04-12"
created: "2026-04-12"
---
# diamond-bottom

Missing the common ancestor's required predicate.

- mentions:: [[foo-2026-x]]
`
	writeNode(t, dir, "Resource/diamond-bottom.md", missingStatus)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).ShouldNot(it.Nil(err))
	message := `type "Bottom" requires predicate "status", but this node does not carry it`
	it.Then(t).Should(it.Equal(1, strings.Count(out, message)))
}

// arc lint
// spec 017 US4 Acceptance Scenario 4.1: a type declaring subClassOf toward
// an unregistered type name fails schema loading clearly, naming the
// offending type and the unresolved reference — not a silent pass.
func TestLintUnresolvedSubClassOfBaseFailsClearly(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "Orphan", nil, []string{"NoSuchType"})
	commitAll(t, dir, "seed: type with unresolved base")
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).Should(it.Error(out, err).Contain("Orphan"))
}

// arc lint
// spec 017 US4 Acceptance Scenario 4.2: a direct self-reference cycle fails
// schema loading clearly, without hanging or crashing.
func TestLintDirectSelfReferenceCycleFailsClearly(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "SelfRef", nil, []string{"SelfRef"})
	commitAll(t, dir, "seed: self-referencing type")
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).Should(it.Error(out, err).Contain("SelfRef"))
}

// arc lint
// spec 017 US4 Acceptance Scenario 4.2: a longer cycle (two types each
// subClassOf the other) fails schema loading clearly, without hanging or
// crashing while computing either type's effective contract.
func TestLintTwoTypeCycleFailsClearly(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "CycleA", nil, []string{"CycleB"})
	registerType(t, dir, "CycleB", nil, []string{"CycleA"})
	commitAll(t, dir, "seed: two mutually cyclic types")
	chdir(t, dir)

	_, err := sut(NewLintCmd(), nil)

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.True(strings.Contains(err.Error(), "CycleA") || strings.Contains(err.Error(), "CycleB")))
}

// arc lint
// spec.md US3 Acceptance Scenario 1: a schema type definition whose own
// name begins with a lowercase letter produces a typeCase violation.
func TestLintLowercaseSchemaTypeDefinitionProducesTypeCaseViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "widget", nil, nil)
	commitAll(t, dir, "seed: lowercase widget type")
	chdir(t, dir)

	out, _ := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.String(out).Contain("typeCase")).
		Should(it.String(out).Contain("widget"))
}

const lowercaseTypedNode = `---
"@id": Thing
"@type": widget
published: "2026-04-12"
created: "2026-04-12"
---
# Thing

A node whose own class reference is not CamelCase.
`

// arc lint
// spec.md US3 Acceptance Scenario 2: a node whose own "@type" reference
// begins with a lowercase letter produces a typeCase violation.
func TestLintLowercaseNodeTypeReferenceProducesTypeCaseViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	registerType(t, dir, "widget", nil, nil)
	writeNode(t, dir, "widget/Thing.md", lowercaseTypedNode)
	commitAll(t, dir, "seed: lowercase-typed node")
	chdir(t, dir)

	out, _ := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.String(out).Contain("typeCase")).
		Should(it.String(out).Contain("Thing"))
}

// arc lint
// spec.md US3 Acceptance Scenario 3: a schema and graph where every class
// name begins with an uppercase letter reports no typeCase violation.
func TestLintAllCamelCaseGraphReportsNoTypeCaseViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeCase"))
}

// ---------------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — User Story 3
//
// A graph shaped exactly as ARCNET-CORE v0.11 describes lints clean.
//
// Every assertion below runs against the HAND-BUILT fixture at
// internal/app/lint/service/testdata/v011-graph, never against arc init or
// arc apply output. A lint expectation asserted against the tool's own
// output agrees with the tool by construction and passes vacuously
// (CORE-FIX.md §5.7).
// ---------------------------------------------------------------------------

// v011FixtureRoot is resolved at package-variable initialization time —
// before any test calls chdir(t, ...) into a temporary graph, after which a
// relative path no longer resolves. The fixture is shared with
// internal/app/lint/service's own unit tests rather than duplicated here,
// so the two cannot drift apart.
var v011FixtureRoot = func() string {
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "internal", "app", "lint", "service", "testdata", "v011-graph"))
	if err != nil {
		panic(err)
	}
	return abs
}()

// buildV011Graph seeds a real graph (git, .arc/, _schema/ from Seed()) and
// copies the fixture's content nodes into it. _schema/ is never taken from
// the fixture, so the fixture cannot carry a stale copy of the built-in
// vocabulary.
func buildV011Graph(t *testing.T, dir string) {
	t.Helper()
	initGraph(t, dir)

	err := filepath.WalkDir(v011FixtureRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(v011FixtureRoot, path)
		if rerr != nil {
			return rerr
		}
		if rel == "README.md" {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		writeNode(t, dir, filepath.ToSlash(rel), string(raw))
		return nil
	})
	it.Then(t).Should(it.Nil(err))

	// The fixture's Source needs the CORE §11.3 ingest commit shape, so
	// RuleIngestCommit finds exactly one matching commit for it.
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(ingest): kolesnikov-2026-vocabulary — Schema Vocabulary Conformance\n\nSource-Id: kolesnikov-2026-vocabulary\n")
}

// arc lint
// Scenario 1 from spec.md US3 (023): a subject node carrying no publication
// date and no creation timestamp draws no missing-required-predicate
// diagnostic — CORE §11.1's universal base requires neither (FR-007).
func TestLintV011EntityWithoutProvenanceTimestampsIsClean(t *testing.T) {
	dir := t.TempDir()
	buildV011Graph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("Merge Vocabulary"))
}

// arc lint
// Scenario 2 from spec.md US3 (023): a timeline period node carrying only
// its chronological citations draws no missing-required-predicate
// diagnostic — §11.5 requires cites alone (FR-009).
func TestLintV011TimelineWithCitesAloneIsClean(t *testing.T) {
	dir := t.TempDir()
	buildV011Graph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("timeline/yearly/2026.md"))
}

// arc lint
// Scenario 3 from spec.md US3 (023): a timeline period node additionally
// carrying granularity, a period code, and a heading has none of the three
// reported as an undeclared predicate — they stay declared, as Optional.
func TestLintV011TimelineOptionalPredicatesArePermitted(t *testing.T) {
	dir := t.TempDir()
	buildV011Graph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeOptional")).
		ShouldNot(it.String(out).Contain("timeline/monthly/2026-04.md"))
}

// arc lint
// Scenario 4 from spec.md US3 (023): a document node carrying no
// publication date IS reported — a Source requires published by its own
// type, not by inheritance (FR-008). This is the assertion that keeps
// FR-007's relaxation from becoming a blanket exemption.
func TestLintV011SourceMissingPublishedIsReported(t *testing.T) {
	dir := t.TempDir()
	buildV011Graph(t, dir)

	defective := `---
"@id": nakamoto-2026-undated
"@type": Source
title: "A Document With No Publication Date"
---
# nakamoto-2026-undated

A Source deliberately missing the publication date its own type requires.

## Mentions
- mentions:: [[Merge Vocabulary]]
`
	writeNode(t, dir, "Source/nakamoto-2026-undated.md", defective)
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(ingest): nakamoto-2026-undated — A Document With No Publication Date\n\nSource-Id: nakamoto-2026-undated\n")
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("Source/nakamoto-2026-undated.md")).
		Should(it.String(out).Contain("typeRequires")).
		Should(it.String(out).Contain("published"))
}

// arc lint
// Scenario 5 from spec.md US3 (023): a graph shaped exactly as CORE
// describes it reports clean end to end (SC-003, SC-004).
func TestLintV011GraphReportsClean(t *testing.T) {
	dir := t.TempDir()
	buildV011Graph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(out).Contain("5 nodes checked, 5 passing, 0 failing"))
}

// ---------------------------------------------------------------------------
// specs/024-lint-conformance-gaps — User Story 2
//
// arc lint reports every node — content or schema — whose identity contains
// an ARCNET-CORE §7.1 forbidden character, naming each character and its
// 1-indexed position, without modifying any file.
// ---------------------------------------------------------------------------

// identityCharsetEntityFixture builds a minimal, otherwise-conformant Entity
// node carrying id both as its "@id" and its H1 heading — used so a
// forbidden-character identity that IS a real, Unix-legal on-disk basename
// (unlike "/", which can never be a file's actual basename, since
// walkNodeFiles' own directory walk would read it as a path separator
// instead) reaches checkIdentityCharset via the same "@id"==basename path
// every other content node does.
func identityCharsetEntityFixture(id string) string {
	return `---
"@id": "` + id + `"
"@type": Entity
category: [independent, abstract, occurrent, script]
published: "2026-04-12"
created: "2026-04-12"
---
# ` + id + `

A test entity.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
`
}

// arc lint
// Scenario 1 from spec.md US2 (024): a content node whose identity contains
// a forbidden character (here ":", chosen because it — unlike "/" — is a
// real Unix-legal on-disk basename character) is reported, naming the
// character and its 1-indexed position.
func TestLintIdentityCharsetContentNodeViolationNamesCharacterAndPosition(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "Entity/Handshake: Protocol.md", identityCharsetEntityFixture("Handshake: Protocol"))
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[identityCharset]")).
		Should(it.String(out).Contain(`identity "Handshake: Protocol"`)).
		Should(it.String(out).Contain(`":" at position 10`))
}

// arc lint
// Scenario 2 from spec.md US2 (024): a node whose identity carries no
// forbidden character produces no identityCharset violation.
func TestLintIdentityCharsetSafeIdentityNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("identityCharset"))
}

// arc lint
// Scenario 3 from spec.md US2 (024): a Source node's citekey identity
// ("foo-2026-x") produces no identityCharset violation.
func TestLintIdentityCharsetSourceCitekeyNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(out).Contain("2 nodes checked, 2 passing, 0 failing")).
		ShouldNot(it.String(out).Contain("identityCharset"))
}

// arc lint
// Scenario 4 from spec.md US2 (024): a schema node (a type definition) whose
// identity contains a forbidden character reports the same violation as a
// content node.
func TestLintIdentityCharsetSchemaNodeViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "_schema/Class/V1.3.md", "---\n\"@id\": V1.3\n\"@type\": Class\nmerge: union\n---\n# V1.3\n\nA schema type whose own name contains a forbidden character.\n")
	commitAll(t, dir, "seed: schema type with forbidden character")
	chdir(t, dir)

	out, _ := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.String(out).Contain("[identityCharset]")).
		Should(it.String(out).Contain(`identity "V1.3"`)).
		Should(it.String(out).Contain(`"." at position 3`)).
		Should(it.String(out).Contain("_schema/Class/V1.3.md"))
}

// arc lint
// Scenario 5 from spec.md US2 (024): an identity carrying more than one
// forbidden character has every one of them named, along with its own
// position — not just the first.
func TestLintIdentityCharsetMultipleOffendingCharactersAllNamed(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "Entity/v1.3: Handshake.md", identityCharsetEntityFixture("v1.3: Handshake"))
	chdir(t, dir)

	out, _ := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.String(out).Contain("[identityCharset]")).
		Should(it.String(out).Contain(`"." at position 3`)).
		Should(it.String(out).Contain(`":" at position 5`))
}

// arc lint
// Scenario 6 from spec.md US2 (024) / FR-006: running arc lint against a
// graph carrying an identityCharset violation changes no file.
func TestLintIdentityCharsetModifiesNoFile(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "Entity/Handshake: Protocol.md", identityCharsetEntityFixture("Handshake: Protocol"))
	chdir(t, dir)

	before := runGit(t, dir, "status", "--porcelain")

	_, _ = sut(NewLintCmd(), nil)

	after := runGit(t, dir, "status", "--porcelain")
	it.Then(t).Should(it.Equal(before, after))
}

// ---------------------------------------------------------------------------
// specs/024-lint-conformance-gaps — User Story 3
//
// arc lint validates an Entity's four-word category as a whole combination
// against the twelve legal ARCNET-CORE §10.2 rows, suggesting the closest
// legal row on rejection.
// ---------------------------------------------------------------------------

const sowaMixedCategoryEntity = `---
"@id": Free Will
"@type": Entity
category: [independent, physical, continuant, purpose]
published: "2026-04-12"
created: "2026-04-12"
---
# Free Will

A test entity with a mixed-taxonomy category.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
`

// arc lint
// Scenario 1 from spec.md US3 (024): a legal Sowa combination
// ([independent, abstract, occurrent, script]) produces no category
// violation.
func TestLintSowaCategoryLegalCombinationNoViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("entityCategory"))
}

// arc lint
// Scenario 2 from spec.md US3 (024): a structurally-valid-but-illegal
// combination — every word individually belongs to the right word-group, but
// the four together are not one of the twelve legal rows — is rejected,
// naming the rejected combination and suggesting the closest legal one.
func TestLintSowaCategoryIllegalCombinationSuggestsClosestLegalRow(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "Entity/Free Will.md", sowaMixedCategoryEntity)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[entityCategory]")).
		Should(it.String(out).Contain("independent, physical, continuant, purpose")).
		Should(it.String(out).Contain("independent, physical, continuant, object"))
}

// arc lint
// Scenario 3 from spec.md US3 (024): a category with the wrong number of
// words keeps the existing, distinct wrong-length message, unaffected by the
// twelve-row comparison.
func TestLintSowaCategoryWrongLengthKeepsExistingMessage(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	wrongLength := `---
"@id": Free Will
"@type": Entity
category: [independent, abstract, occurrent]
published: "2026-04-12"
created: "2026-04-12"
---
# Free Will

A test entity with a wrong-length category.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
`
	writeNode(t, dir, "Entity/Free Will.md", wrongLength)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[entityCategory]")).
		Should(it.String(out).Contain("category must decode to exactly four Sowa words, found 3"))
}

// arc lint
// Scenario 4 from spec.md US3 (024): running arc lint against a graph
// carrying a Sowa category violation changes no file.
func TestLintSowaCategoryModifiesNoFile(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "Entity/Free Will.md", sowaMixedCategoryEntity)
	chdir(t, dir)

	before := runGit(t, dir, "status", "--porcelain")

	_, _ = sut(NewLintCmd(), nil)

	after := runGit(t, dir, "status", "--porcelain")
	it.Then(t).Should(it.Equal(before, after))
}

// --- BUG-002 T051: FR-010 against a graph nested in a larger repository ----

// spec.md 016 FR-021 / 004 FR-010: CommitsMatching must count only ingest
// commits touching this graph. Before BUG-002's fix `git log --all --grep`
// carried no pathspec, so a sibling graph sharing the repository that
// ingested the same citekey was counted too, and this graph's own source
// node was wrongly reported as having 2 matching commits.
func TestLintIngestCommitIgnoresSiblingGraphInSameRepository(t *testing.T) {
	repo := t.TempDir()
	graph := filepath.Join(repo, "graph")
	other := filepath.Join(repo, "other")

	writeGraphLayout(t, graph)
	writeGraphLayout(t, other)
	runGit(t, repo, "init")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(init): empty knowledge graph")

	// the sibling graph ingests the same citekey, in the same repository
	writeNode(t, other, "Source/foo-2026-x.md", conformantSource)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(ingest): foo-2026-x — sibling graph\n\nSource-Id: foo-2026-x\n")

	// this graph ingests it exactly once
	writeNode(t, graph, "Source/foo-2026-x.md", conformantSource)
	writeNode(t, graph, "Entity/Widget.md", conformantEntity)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(ingest): foo-2026-x — this graph\n\nSource-Id: foo-2026-x\n")

	chdir(t, graph)
	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(out).Contain("2 nodes checked, 2 passing, 0 failing")).
		Should(it.True(!strings.Contains(out, "matching commits found")))
}
