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
	for _, folder := range []string{"sources", "entities", "resources", "timeline/yearly", "timeline/monthly", "_schema/types", "_schema/predicates"} {
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, folder), 0o755)))
	}
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, ".arc"), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".arc", ".gitkeep"), nil, 0o644)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".arc/\n"), 0o644)))

	for path, raw := range appschema.Seed() {
		writeNode(t, dir, path, string(raw))
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(init): empty knowledge graph")
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
	writeNode(t, dir, "sources/"+id+".md", content)
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
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`

const conformantEntity = `---
"@id": Widget
"@type": entity
category: [independent, abstract, occurrent, script]
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
	writeNode(t, dir, "entities/Widget.md", conformantEntity)
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
// with a schema document (e.g. entities/hypothesis.md vs.
// _schema/types/hypothesis.md) is not reported as a basename collision.
func TestLintSchemaBasenameDoesNotCollideWithContentNode(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)
	writeNode(t, dir, "_schema/types/hypothesis.md", "---\n\"@id\": hypothesis\n\"@type\": Class\nmerge: union\n---\n# hypothesis\n\nA domain type registered by this test fixture.\n")
	writeNode(t, dir, "entities/hypothesis.md", "---\n\"@id\": hypothesis\n\"@type\": entity\ncategory: [independent, abstract, occurrent, script]\n---\n# hypothesis\n\nA namesake entity, unrelated to the schema kind of the same name.\n\n## MentionedIn\n- mentionedIn:: [[foo-2026-x]]\n")
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
"@type": entity
title: Widget
category: [independent, abstract, occurrent]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- mentions:: [[Nonexistent Node]]
`
	writeNode(t, dir, "entities/Widget.md", broken)
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
"@type": entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- mentions:: [[Not A Real Node]]
`
	writeNode(t, dir, "entities/Widget.md", broken)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("entities/Widget.md:")).
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

	widget := readFile(t, filepath.Join(dir, "entities", "Widget.md"))
	writeNode(t, dir, "resources/Widget.md", widget)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[uniqueBasename]")).
		Should(it.String(out).Contain(`basename "Widget" is used by more than one file`)).
		Should(it.String(out).Contain("entities/Widget.md")).
		Should(it.String(out).Contain("resources/Widget.md")).
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
	writeNode(t, dir, "entities/Broken.md", conflicted)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("entities/Broken.md:1")).
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
	writeNode(t, dir, "entities/Legacy.md", legacy)
	chdir(t, dir)

	before := runGit(t, dir, "status", "--porcelain")

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("entities/Legacy.md")).
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

	missingID := "---\n\"@type\": entity\ntitle: Legacy\ncategory: [independent]\n---\n# Legacy\n\nAn entity missing its mandatory \"@id\" field.\n"
	writeNode(t, dir, "entities/Legacy.md", missingID)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("entities/Legacy.md")).
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
// just for "source" nodes via the narrower, pre-existing sourceCitekey
// rule) — any node type demonstrates the behavior.
func TestLintIdMismatchedBasenameReportsFrontMatterViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	mismatched := `---
"@id": wrong-id
"@type": source
title: "A Mismatched Source"
published: "2026-04-12"
---
# wrong-id

A source whose "@id" does not equal its own filename basename.
`
	writeNode(t, dir, "sources/mismatched-2026-x.md", mismatched)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("sources/mismatched-2026-x.md")).
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
"@type": entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- mentions:: [[Not A Real Node]]
`
	writeNode(t, dir, "entities/Widget.md", broken)
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
		Should(it.String(verboseOut).Contain("sources/foo-2026-x.md")).
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
	entityDoc := readFile(t, filepath.Join(dir, "_schema", "types", "entity.md"))
	writeNode(t, dir, "_schema/types/entity.md", strings.ReplaceAll(entityDoc, "merge: union", "merge: bogus"))
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).Should(it.Error(out, err).Contain("_schema/types/entity.md"))
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
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document.
`
	writeNode(t, dir, "sources/foo-2026-x.md", missingMentions)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("sources/foo-2026-x.md")).
		Should(it.String(out).Contain("[typeRequires]")).
		Should(it.String(out).Contain(`type "source" requires predicate "mentions", but this node does not carry it`))
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
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
status: read
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "sources/foo-2026-x.md", extraPredicate)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("[typeOptional]")).
		Should(it.String(out).Contain(`predicate "status" is not permitted by type "source" (not listed under its Requires or Optional)`))
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
	writeNode(t, dir, "entities/Legacy.md", missingType)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("entities/Legacy.md")).
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
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document.

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "sources/foo-2026-x.md", unquotedID)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("sources/foo-2026-x.md")).
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
	writeNode(t, dir, "_schema/predicates/citesAsExample.md", `---
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
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document. [citesAsExample:: [[Widget]]]

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "sources/foo-2026-x.md", citing)
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
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document. [bogusCite:: [[Widget]]]

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "sources/foo-2026-x.md", citing)
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

	entries, err := os.ReadDir(filepath.Join(dir, "_schema", "predicates"))
	it.Then(t).Should(it.Nil(err))
	for _, e := range entries {
		path := filepath.Join(dir, "_schema", "predicates", e.Name())
		content := readFile(t, path)
		if !strings.Contains(content, "cito:") {
			continue
		}
		it.Then(t).Should(it.Nil(os.WriteFile(path, []byte(strings.ReplaceAll(content, "cito:", "test:")), 0o644)))
	}
	commitAll(t, dir, "seed: strip cito: alignment from every predicate")

	citing := `---
"@id": foo-2026-x
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document. [cites:: [[Widget]]]

## Mentions
- mentions:: [[Widget]]
`
	writeNode(t, dir, "sources/foo-2026-x.md", citing)
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
	writeNode(t, dir, "_schema/predicates/highlight.md", `---
"@id": highlight
"@type": Property
merge: union
role: text
---
# highlight

A short highlighted note about the entity, always written as prose.
`)
	writeNode(t, dir, "_schema/types/entity.md", `---
"@id": entity
"@type": Class
merge: union
---
# entity

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
"@type": entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
- highlight:: [[foo-2026-x]]
`
	writeNode(t, dir, "entities/Widget.md", misplaced)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.Nil(err)).
		Should(it.String(out).Contain("entities/Widget.md")).
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
"@type": entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
- totallyUnknownPred:: [[foo-2026-x]]
`
	writeNode(t, dir, "entities/Widget.md", unregistered)
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
"@type": entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

Additional prose about the entity.

- conformsTo:: [[foo-2026-x]]

## MentionedIn
- mentionedIn:: [[foo-2026-x]]
`
	writeNode(t, dir, "entities/Widget.md", widget)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeOptional"))
}

// arc lint
// BUG-001 regression: a resource node using a §10.5 semantic predicate
// produces no [typeOptional] violation, per this bugfix's explicit scope
// decision that semantic predicates are usable on entity or resource nodes.
func TestLintResourceSemanticPredicateNoTypeOptionalViolation(t *testing.T) {
	dir := t.TempDir()
	buildConformantGraph(t, dir)

	resource := `---
"@id": RFC 8446
"@type": resource
ref: standard
---
# RFC 8446

A normative specification.

- conformsTo:: [[foo-2026-x]]
`
	writeNode(t, dir, "resources/RFC 8446.md", resource)
	chdir(t, dir)

	out, err := sut(NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.String(out).Contain("typeOptional"))
}
