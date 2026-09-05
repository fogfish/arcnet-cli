//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package ctrl

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// TestMain sets a fake git identity for the whole test binary. arc init
// shells out to a real `git commit`, which fails with "Author identity
// unknown" on any machine (including CI runners) that has no global
// user.name/user.email configured — the tool itself intentionally does not
// configure git identity (spec.md Assumptions), so the tests must supply
// their own, hermetically, rather than depend on the environment's global
// git config.
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

// sutCaptureStderr wraps sut, additionally capturing os.Stderr — needed to
// assert BUG-001's default-mode conciseness (no per-step git progress on
// stderr unless --verbose is set).
func sutCaptureStderr(t *testing.T, cmd *cobra.Command, args []string) (stdout, stderr string, err error) {
	t.Helper()
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	stdout, err = sut(cmd, args)

	w.Close()
	os.Stderr = origStderr
	stderr = <-ch
	return stdout, stderr, err
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Nil(os.Chdir(dir)))
	t.Cleanup(func() { os.Chdir(original) })
}

func assertIsDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(info.IsDir()))
}

func assertIsFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!info.IsDir()))
}

func readGraphFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	it.Then(t).Should(it.Nil(err))
	return string(content)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	it.Then(t).Should(it.Nil(err))
	return string(out)
}

// arc init
// Scenario 1 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectoryCreatesLayout(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewInitCmd(), []string{})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain(dir))

	for _, folder := range []string{"Source", "Entity", "Resource", filepath.Join("timeline", "yearly"), filepath.Join("timeline", "monthly"), filepath.Join("_schema", "Class"), filepath.Join("_schema", "Property")} {
		assertIsDir(t, filepath.Join(dir, folder))
	}
	assertIsFile(t, filepath.Join(dir, "_schema", "Class", "Entity.md"))
	assertIsFile(t, filepath.Join(dir, "_schema", "Property", "related.md"))
	_, metaErr := os.Stat(filepath.Join(dir, "_meta"))
	it.Then(t).Should(it.True(os.IsNotExist(metaErr)))
	assertIsDir(t, filepath.Join(dir, ".arc"))

	// specs/031 research.md D4: the exclusion rule moved inside .arc/, and
	// no .gitignore is written at the graph root any more — in either mode.
	// SC-008's observable outcomes (local state untracked, clean working
	// tree, one commit) are asserted by the tests below and are unchanged.
	gitignore, rerr := os.ReadFile(filepath.Join(dir, ".arc", ".gitignore"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal("*\n", string(gitignore)))

	_, rootIgnoreErr := os.Stat(filepath.Join(dir, ".gitignore"))
	it.Then(t).Should(it.True(os.IsNotExist(rootIgnoreErr)))
}

// arc init
// Scenario 2 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectorySingleCommit(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	log := strings.TrimSpace(gitOutput(t, dir, "log", "--oneline"))
	lines := strings.Split(log, "\n")

	it.Then(t).
		Should(it.Equal(1, len(lines))).
		Should(it.String(log).Contain("graph(init): empty knowledge graph"))
}

// arc init
// Scenario 3 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectoryCleanWorkingTree(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	status := strings.TrimSpace(gitOutput(t, dir, "status", "--short"))
	it.Then(t).Should(it.Equal("", status))

	tracked := gitOutput(t, dir, "ls-files")
	it.Then(t).ShouldNot(it.String(tracked).Contain(".arc/"))
}

// arc init
// Scenario 4 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectoryFoldersInHistory(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	tracked := gitOutput(t, dir, "ls-files")
	it.Then(t).
		Should(it.String(tracked).Contain("Source/.gitkeep")).
		Should(it.String(tracked).Contain("Entity/.gitkeep")).
		Should(it.String(tracked).Contain("Resource/.gitkeep")).
		Should(it.String(tracked).Contain("Reference/.gitkeep")).
		Should(it.String(tracked).Contain("timeline/yearly/.gitkeep")).
		Should(it.String(tracked).Contain("timeline/monthly/.gitkeep")).
		Should(it.String(tracked).Contain("_schema/Class/Entity.md")).
		Should(it.String(tracked).Contain("_schema/Property/related.md"))
}

// arc init
// spec.md US1 Acceptance Scenarios 1-2: every core predicate/type is
// seeded as a real, machine-readable document — role/merge (plus
// label/aligned where declared) and a description for every predicate;
// required/optional and a description for every type — not an
// existence-only stub.
func TestInitSeedsAllCoreKindsAndPredicates(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	for name, def := range kernel.CoreTypeDefs {
		path := filepath.Join(dir, "_schema", "Class", name+".md")
		assertIsFile(t, path)

		content, rerr := os.ReadFile(path)
		it.Then(t).Should(it.Nil(rerr))
		it.Then(t).
			Should(it.String(string(content)).Contain(`"@type": Class`)).
			// FR-005: a seeded Class document declares no merge at all.
			ShouldNot(it.String(string(content)).Contain("merge:"))
		for _, required := range def.Required {
			it.Then(t).Should(it.String(string(content)).Contain("required:: [[" + required + "]]"))
		}
	}

	for name, def := range kernel.CorePredicateDefs {
		path := filepath.Join(dir, "_schema", "Property", name+".md")
		assertIsFile(t, path)

		content, rerr := os.ReadFile(path)
		it.Then(t).Should(it.Nil(rerr))
		it.Then(t).
			Should(it.String(string(content)).Contain(`"@type": Property`)).
			Should(it.String(string(content)).Contain("role: " + def.Role)).
			Should(it.String(string(content)).Contain("merge: " + string(def.Merge)))
	}
}

// arc init
// spec 017 US1 Acceptance Scenario 1: a freshly initialized graph seeds
// _schema/Class/Node.md (Required: published/created; Optional:
// tags/text/updated/scoreZ/scoreC), and source/entity/resource/timeline's
// own seeded documents each carry an explicit subClassOf:: [[Node]] edge
// (aligned to rdfs:subClassOf).
func TestInitSeedsNodeTypeAndWiresContentTypesToIt(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	nodePath := filepath.Join(dir, "_schema", "Class", "Node.md")
	assertIsFile(t, nodePath)

	nodeContent, rerr := os.ReadFile(nodePath)
	it.Then(t).Should(it.Nil(rerr))
	// specs/023 FR-007: §11.1's universal base requires NOTHING;
	// published/created are merely declarable on every type.
	it.Then(t).ShouldNot(it.String(string(nodeContent)).Contain("required:: [["))
	for _, optional := range []string{"published", "created", "tags", "text", "updated", "scoreZ", "scoreC"} {
		it.Then(t).Should(it.String(string(nodeContent)).Contain("optional:: [[" + optional + "]]"))
	}

	for _, name := range []string{"Source", "Entity", "Resource", "Timeline"} {
		content, rerr := os.ReadFile(filepath.Join(dir, "_schema", "Class", name+".md"))
		it.Then(t).Should(it.Nil(rerr))
		it.Then(t).Should(it.String(string(content)).Contain("subClassOf:: [[Node]]"))
	}
}

// arc init
// spec.md US1 Acceptance Scenario 3: no _schema/nodes/ folder exists.
func TestInitNoSchemaNodesFolderExists(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	_, statErr := os.Stat(filepath.Join(dir, "_schema", "nodes"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init
// Scenario 5 from spec.md US1: initialization succeeds with no network
// access at all — there is no fetch to fail (research.md D5)
func TestInitSucceedsWithNoNetworkAccess(t *testing.T) {
	dir := t.TempDir()
	emptyProxy := "http://localhost:1"
	t.Setenv("HTTP_PROXY", emptyProxy)
	t.Setenv("HTTPS_PROXY", emptyProxy)

	out, err := sut(NewInitCmd(), []string{dir})
	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "Class", "Entity.md"))
}

// arc init <target-file>
// FR-010 edge case from specs/002-arc-init/spec.md
func TestInitTargetIsFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "not-a-dir")
	it.Then(t).Should(it.Nil(os.WriteFile(target, []byte("x"), 0o644)))

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).Should(it.Error(out, err).Contain("directory"))

	content, rerr := os.ReadFile(target)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal("x", string(content)))
}

// PATH=<empty> arc init <dir>
// FR-011 edge case from specs/002-arc-init/spec.md
func TestInitGitUnavailable(t *testing.T) {
	dir := t.TempDir()
	emptyPath := t.TempDir()
	t.Setenv("PATH", emptyPath)

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).Should(it.Error(out, err).Contain("git"))

	entries, rerr := os.ReadDir(dir)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc init <non-empty-dir>
// FR-015 edge case from specs/002-arc-init/spec.md
func TestInitTargetNonEmpty(t *testing.T) {
	dir := t.TempDir()
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("x"), 0o644)))

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).Should(it.Error(out, err).Contain("empty"))

	entries, rerr := os.ReadDir(dir)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(1, len(entries)))
}

// arc init --json <dir>
// --json output contract from specs/002-arc-init/contracts/cli-contract.md
func TestInitJSONOutput(t *testing.T) {
	dir := t.TempDir()
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(out, err))

	var payload struct {
		Path           string   `json:"path"`
		Commit         string   `json:"commit"`
		FoldersCreated []string `json:"foldersCreated"`
	}
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &payload)))
	it.Then(t).
		Should(it.Equal(dir, payload.Path)).
		ShouldNot(it.Equal("", payload.Commit)).
		Should(it.LessOrEqual(len(payload.Commit), 12)).
		Should(it.Seq(payload.FoldersCreated).Contain("Source", "Entity", "Resource", "_schema/Class", "_schema/Property"))
}

// arc init <dir>
// FR-016 from specs/002-arc-init/spec.md — default output is a single
// concise line; per-step git progress is opt-in via --verbose (BUG-001)
func TestInitDefaultModeIsConciseSingleLine(t *testing.T) {
	dir := t.TempDir()

	stdout, stderr, err := sutCaptureStderr(t, NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.Equal(1, strings.Count(stdout, "\n"))).
		Should(it.Equal("", stderr))

	commit := strings.TrimSpace(strings.Split(strings.Split(stdout, "commit ")[1], ")")[0])
	fullHash := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))
	it.Then(t).
		Should(it.True(len(commit) <= 12)).
		Should(it.True(strings.HasPrefix(fullHash, commit)))
}

// arc init --verbose <dir>
// --verbose progress contract from specs/002-arc-init/contracts/cli-contract.md (BUG-001)
func TestInitVerboseModeShowsGitProgress(t *testing.T) {
	dir := t.TempDir()
	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	stdout, stderr, err := sutCaptureStderr(t, NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.String(stderr).Contain("Checking git availability")).
		Should(it.String(stderr).Contain("Preparing git repository")).
		Should(it.String(stderr).Contain("Committing empty graph"))
}

// arc init <non-existent-dir>
// Scenario 1 from specs/002-arc-init/spec.md, US2
func TestInitNamedDirectoryCreatesLayout(t *testing.T) {
	base := t.TempDir()
	cwd := filepath.Join(base, "cwd")
	it.Then(t).Should(it.Nil(os.MkdirAll(cwd, 0o755)))
	chdir(t, cwd)

	target := filepath.Join(base, "graph")

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsDir(t, filepath.Join(target, "Source"))
	assertIsDir(t, filepath.Join(target, ".arc"))

	entries, rerr := os.ReadDir(cwd)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc init <dir>
// Scenario 2 from specs/002-arc-init/spec.md, US2
func TestInitNamedDirectoryReportsPath(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "graph")

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain(target))
}

// arc init <already-a-graph>
// Scenario 1 from specs/002-arc-init/spec.md, US3 (FR-014)
func TestInitRefusesReInitialization(t *testing.T) {
	dir := t.TempDir()
	_, err := sut(NewInitCmd(), []string{dir})
	it.Then(t).Should(it.Nil(err))

	before := gitOutput(t, dir, "log", "--oneline")

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).Should(it.Error(out, err).Contain("already"))

	after := gitOutput(t, dir, "log", "--oneline")
	it.Then(t).Should(it.Equal(before, after))
}

// arc init
// spec.md US2 Acceptance Scenarios 1-2: every class name arc init seeds
// under _schema/Class/ begins with an uppercase letter, and no two seeded
// class names differ only by casing.
func TestInitSeededSchemaTypesAreAllCamelCase(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	entries, rerr := os.ReadDir(filepath.Join(dir, "_schema", "Class"))
	it.Then(t).Should(it.Nil(rerr))

	seen := map[string]bool{}
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".md")
		if name == "" {
			continue
		}
		r := []rune(name)
		it.Then(t).Should(it.True(unicode.IsUpper(r[0])))

		lower := strings.ToLower(name)
		it.Then(t).Should(it.True(!seen[lower]))
		seen[lower] = true
	}
}

// ---------------------------------------------------------------------------
// specs/022-reference-type-folders — ARCNET-CORE v0.11
// ---------------------------------------------------------------------------

// schemaBullets returns every target of a `- <predicate>:: [[<target>]]`
// bullet in a rendered schema document, in document order. It reads the
// rendered bytes rather than re-deriving from kernel.CoreTypeDefs, so the
// assertion covers seeding as well as the table.
func schemaBullets(t *testing.T, content, predicate string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^- ` + regexp.QuoteMeta(predicate) + `:: \[\[([^\]]+)\]\]`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		out = append(out, m[1])
	}
	return out
}

// walkDirNames collects every directory under root as a slash-separated
// path relative to root, reading the names the filesystem actually reports
// rather than stat-ing a path the test constructed. Contract C7: os.Stat on
// APFS answers yes to "Source", "source", and "SOURCE" alike, so a
// stat-based assertion cannot detect a case defect. ReadDir returns the
// stored name, so a string comparison against it can.
func walkDirNames(t *testing.T, root string, skip map[string]bool) []string {
	t.Helper()
	var out []string

	var walk func(dir, prefix string)
	walk = func(dir, prefix string) {
		entries, err := os.ReadDir(dir)
		it.Then(t).Should(it.Nil(err))
		for _, e := range entries {
			if !e.IsDir() || skip[e.Name()] {
				continue
			}
			rel := e.Name()
			if prefix != "" {
				rel = prefix + "/" + e.Name()
			}
			out = append(out, rel)
			walk(filepath.Join(dir, e.Name()), rel)
		}
	}
	walk(root, "")

	sort.Strings(out)
	return out
}

// arc init
// BUG-001 (T073) / spec.md US6 Acceptance Scenarios 1 and 4: a freshly
// initialized graph's seeded Entity type requires "text" (CORE 0.12 retires
// "definition"), no "_schema/Property/definition.md" exists, and Entity's
// Optional list no longer declares "notes".
func TestInitSeedsEntityWithTextNotDefinition(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	content := readGraphFile(t, filepath.Join(dir, "_schema", "Class", "Entity.md"))
	it.Then(t).
		Should(it.Seq(schemaBullets(t, content, "required")).Contain("text")).
		ShouldNot(it.Seq(schemaBullets(t, content, "required")).Contain("definition")).
		ShouldNot(it.Seq(schemaBullets(t, content, "optional")).Contain("notes"))

	_, statErr := os.Stat(filepath.Join(dir, "_schema", "Property", "definition.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init
// BUG-001 (T073) / spec.md US6 Acceptance Scenario 6: a freshly initialized
// graph's seeded Timeline type no longer declares "granularity" — retired
// outright under CORE 0.12, not merely optional (FR-035).
func TestInitSeedsTimelineWithoutGranularity(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	content := readGraphFile(t, filepath.Join(dir, "_schema", "Class", "Timeline.md"))
	it.Then(t).
		Should(it.Seq(schemaBullets(t, content, "required")).Equal("cites")).
		ShouldNot(it.Seq(schemaBullets(t, content, "optional")).Contain("granularity")).
		Should(it.Seq(schemaBullets(t, content, "optional")).Contain("period"))

	_, statErr := os.Stat(filepath.Join(dir, "_schema", "Property", "granularity.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init
// spec.md US1 Acceptance Scenario 1: a freshly initialized graph's seeded
// Resource type node requires exactly text, tags, and mentionedIn, and
// offers exactly notes (CORE §11.4 v0.11, contract C2).
func TestInitSeedsResourceWithIngestedFragmentContract(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	content := readGraphFile(t, filepath.Join(dir, "_schema", "Class", "Resource.md"))

	it.Then(t).
		Should(it.Seq(schemaBullets(t, content, "required")).Equal("text", "tags", "mentionedIn")).
		// "indexed" is not a CORE predicate but the arc extension arc apply
		// stamps on every node it creates, so a Resource that did not
		// permit it would fail its own conformance check on a graph the
		// tool itself wrote. §11.4's own sole optional "notes" is retired
		// outright under CORE 0.12 (BUG-001/FR-033) — "indexed" alone
		// remains.
		Should(it.Seq(schemaBullets(t, content, "optional")).Equal("indexed")).
		Should(it.Seq(schemaBullets(t, content, "subClassOf")).Equal("Node"))
}

// arc init
// spec.md US1 Acceptance Scenario 2: the corrected Resource no longer
// offers any of the external-work predicates that moved to Reference, in
// either list (contract C2).
func TestInitSeedsResourceWithoutExternalWorkPredicates(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	content := readGraphFile(t, filepath.Join(dir, "_schema", "Class", "Resource.md"))
	declared := append(schemaBullets(t, content, "required"), schemaBullets(t, content, "optional")...)

	for _, retired := range []string{"ref", "relevance", "url", "authors", "year", "doi", "status", "isCitedBy"} {
		for _, got := range declared {
			it.Then(t).Should(it.True(got != retired))
		}
	}
}

// arc init
// spec.md US1 Acceptance Scenario 3: a Reference type node exists, carries
// the CORE §11.6 predicate lists, and declares Node as its base type
// exactly as the other four content types do (contract C3).
func TestInitSeedsReferenceType(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	content := readGraphFile(t, filepath.Join(dir, "_schema", "Class", "Reference.md"))

	it.Then(t).
		Should(it.String(content).Contain(`"@type": Class`)).
		// specs/023 FR-028 (corrected, BUG-002) supersedes spec 022's
		// Clarification: §11.6 v0.11 requires "title" alone (plan.md F3).
		// "authors"/"year"/"ref"/"status"/"notes"/"relevance" are all
		// retired outright under CORE 0.12 (BUG-001/FR-031/FR-032/FR-034/
		// FR-033, BUG-002/FR-038) — "author"/"published" replace the first
		// two, "text" replaces "relevance" (the same shared generic
		// predicate every other type not otherwise named uses), the rest
		// have no replacement.
		Should(it.Seq(schemaBullets(t, content, "required")).Equal("title")).
		Should(it.Seq(schemaBullets(t, content, "optional")).Equal("text", "url", "author", "published", "doi", "isCitedBy", "indexed")).
		Should(it.String(content).Contain("subClassOf:: [[Node]]"))

	// FR-006: every predicate Reference declares must itself be seeded,
	// so the graph lints clean rather than citing an unregistered name.
	for _, predicate := range append(schemaBullets(t, content, "required"), schemaBullets(t, content, "optional")...) {
		assertIsFile(t, filepath.Join(dir, "_schema", "Property", predicate+".md"))
	}
}

// arc init
// spec.md US1 Acceptance Scenario 5: a freshly initialized graph lints
// clean — the seeded schema is self-consistent and every seeded type's
// predicates are themselves registered.
func TestInitSeededGraphLintsClean(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	out, lintErr := sut(lint.NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(lintErr)).
		Should(it.String(out).Contain("0 failing"))
}

// arc init
// spec.md US3 Acceptance Scenario 1: init creates exactly the eight folders
// of contract C3 and none of the six retired names. Asserted as exact
// string set equality over the names the filesystem reports (contract C7),
// never as a per-path os.Stat.
func TestInitCreatesExactlyTheTypeNamedFolders(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	got := walkDirNames(t, dir, map[string]bool{".git": true, ".arc": true})

	it.Then(t).Should(it.Seq(got).Equal(
		"Entity",
		"Reference",
		"Resource",
		"Source",
		"_schema",
		"_schema/Class",
		"_schema/Property",
		"timeline",
		"timeline/monthly",
		"timeline/yearly",
	))

	for _, retired := range []string{"sources", "entities", "resources", "references", "_schema/predicates", "_schema/types"} {
		for _, name := range got {
			it.Then(t).Should(it.True(name != retired))
		}
	}
}

// ---------------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — User Story 2
//
// A newly initialized graph declares only conformant merge behaviour: every
// seeded predicate draws its merge from CORE §9.3's closed set of six, and
// no seeded Class document carries a merge declaration at all.
// ---------------------------------------------------------------------------

// conformantMergeOps is CORE §9.3's closed set of six, written out as a
// literal rather than derived from internal/core — a test that enumerated
// the constants would agree with the code by construction and could never
// catch a seventh being reintroduced (contract C1.1).
var conformantMergeOps = []string{
	"immutable", "union", "firstWriteWin", "fillIfEmpty", "lastWriteWin", "append",
}

// frontMatterField returns the value of a top-level front-matter key in a
// rendered schema document, and whether the key is present at all.
func frontMatterField(content, key string) (string, bool) {
	m := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `: (.+)$`).FindStringSubmatch(content)
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

func schemaDocuments(t *testing.T, dir, folder string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "_schema", folder))
	it.Then(t).Should(it.Nil(err))

	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out[e.Name()] = readGraphFile(t, filepath.Join(dir, "_schema", folder, e.Name()))
	}
	return out
}

// arc init
// Scenario 1 from spec.md US2 (023): every seeded predicate definition
// declares a merge drawn from exactly the six legal values (FR-001,
// SC-002; contract C2.2a).
func TestInitSeedsOnlyConformantPredicateMergeValues(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	documents := schemaDocuments(t, dir, "Property")
	it.Then(t).Should(it.True(len(documents) > 0))

	for name, content := range documents {
		merge, ok := frontMatterField(content, "merge")
		it.Then(t).Should(it.True(ok))

		legal := false
		for _, op := range conformantMergeOps {
			if merge == op {
				legal = true
			}
		}
		if !legal {
			t.Errorf("_schema/Property/%s declares merge %q, outside the closed set %v", name, merge, conformantMergeOps)
		}
	}
}

// arc init
// Scenario 2 from spec.md US2 (023): no seeded type definition carries a
// merge declaration — CORE §9.3 retired type-level merge (FR-005, SC-002;
// contract C2.2b).
func TestInitSeedsNoTypeLevelMerge(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	documents := schemaDocuments(t, dir, "Class")
	it.Then(t).Should(it.True(len(documents) > 0))

	for name, content := range documents {
		if _, ok := frontMatterField(content, "merge"); ok {
			t.Errorf("_schema/Class/%s carries a merge declaration; CORE §9.3 retired type-level merge", name)
		}
	}
}

// arc init
// Scenario 3 from spec.md US2 (023): the two analytics score predicates
// declare a conformant merge and neither declares the retired seventh one
// (FR-003; research.md D7).
func TestInitSeedsConformantScorePredicates(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	for _, name := range []string{"scoreZ", "scoreC"} {
		content := readGraphFile(t, filepath.Join(dir, "_schema", "Property", name+".md"))
		merge, ok := frontMatterField(content, "merge")

		it.Then(t).
			Should(it.True(ok)).
			Should(it.Equal("lastWriteWin", merge)).
			ShouldNot(it.String(content).Contain("validatedOverwrite"))
	}
}

// arc init, then hand-edit _schema/Property/scoreZ.md, then arc lint
// Scenario 4 from spec.md US2 (023): a hand-written predicate definition
// declaring an out-of-menu merge value is rejected by any command that
// reads the vocabulary, naming the offending document and field (FR-002).
func TestInitOutOfMenuMergeValueIsRejected(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	path := filepath.Join(dir, "_schema", "Property", "scoreZ.md")
	content := readGraphFile(t, path)
	content = strings.Replace(content, "merge: lastWriteWin", "merge: validatedOverwrite", 1)
	it.Then(t).Should(it.Nil(os.WriteFile(path, []byte(content), 0o644)))

	_, lintErr := sut(lint.NewLintCmd(), nil)
	if lintErr == nil {
		t.Fatal("reading a graph whose scoreZ.md declares an out-of-menu merge value must fail")
	}

	message := lintErr.Error()
	it.Then(t).
		Should(it.String(message).Contain("scoreZ.md")).
		Should(it.String(message).Contain("merge"))
}

// arc init, then hand-add a legacy merge attribute to a Class document
// Scenario 5 from spec.md US2 (023): a graph whose type definitions predate
// this feature and still carry a merge declaration keeps loading, the stale
// declaration is ignored, and no diagnostic is produced (FR-006; contract
// C1.3).
func TestInitLegacyTypeLevelMergeToleratedWithoutDiagnostic(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	for _, name := range []string{"Source", "Entity", "Timeline"} {
		path := filepath.Join(dir, "_schema", "Class", name+".md")
		content := readGraphFile(t, path)
		content = strings.Replace(content, `"@type": Class`, `"@type": Class`+"\nmerge: union", 1)
		it.Then(t).Should(it.Nil(os.WriteFile(path, []byte(content), 0o644)))
	}

	out, lintErr := sut(lint.NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(lintErr)).
		Should(it.String(out).Contain("0 failing")).
		ShouldNot(it.String(out).Contain("merge"))
}

// arc init, then arc lint
// Scenario 6 from spec.md US2 (023): a freshly initialized graph lints
// clean (SC-003). Distinct from TestInitSeededGraphLintsClean, which
// records the same guarantee for spec 022 — kept separate so a regression
// is attributed to the feature that broke it.
func TestInitConformantGraphLintsClean(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	out, lintErr := sut(lint.NewLintCmd(), nil)

	it.Then(t).
		Should(it.Nil(lintErr)).
		Should(it.String(out).Contain("0 failing"))
}

// ---------------------------------------------------------------------------
// specs/031-init-existing-git-repo — graph initialization inside an existing
// git repository.
//
// User Story 1 (refuse to nest a repository inside a repository), User Story 2
// (add a graph to an existing project) and User Story 3 (preserve the host
// project's existing files). quickstart.md S2-S8.
// ---------------------------------------------------------------------------

// newHostRepo creates a temp git repository with one commit — the "existing
// project" every scenario below starts from. Identity comes from TestMain's
// GIT_AUTHOR_*/GIT_COMMITTER_* environment, so the fixture needs no git
// config of its own.
func newHostRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitOutput(t, repo, "init", "-q")
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(repo, "README.md"), []byte("hi\n"), 0o644)))
	gitOutput(t, repo, "add", "-A")
	gitOutput(t, repo, "commit", "-qm", "host: initial")
	return repo
}

// initCmdSkipGit builds the init command with --skip-git-init already set.
// sut invokes RunE directly rather than going through Cobra's argument
// parsing, so a command-local flag has to be set on the command itself.
func initCmdSkipGit(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := NewInitCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("skip-git-init", "true")))
	return cmd
}

// errorMessage asserts the command failed and returns the message, so a
// message assertion never dereferences a nil error when the guard under
// test is not yet in place.
func errorMessage(t *testing.T, out string, err error) string {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a failure, got success: %s", out)
	}
	return err.Error()
}

// commitPaths returns every path the repository's HEAD commit touched,
// repository-relative.
func commitPaths(t *testing.T, repo string) []string {
	t.Helper()
	out := strings.TrimSpace(gitOutput(t, repo, "show", "--name-only", "--format=", "HEAD"))
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// arc init <repo-root>
// spec.md US1 Acceptance Scenario 1 (FR-001, FR-004): initialization at the
// root of an existing repository is refused.
func TestInitRefusesAtRepositoryRoot(t *testing.T) {
	repo := newHostRepo(t)

	out, err := sut(NewInitCmd(), []string{repo})

	it.Then(t).Should(it.Error(out, err))
	_, statErr := os.Stat(filepath.Join(repo, "Source"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	it.Then(t).Should(it.Equal("", strings.TrimSpace(gitOutput(t, repo, "status", "--porcelain"))))
}

// arc init <repo>/<existing-subfolder>
// spec.md US1 Acceptance Scenario 2 (FR-002, FR-003): initialization in a
// subfolder of an existing repository is refused just as at its root — the
// enclosing repository is found by upward search.
func TestInitRefusesInsideRepositorySubfolder(t *testing.T) {
	repo := newHostRepo(t)
	sub := filepath.Join(repo, "sub")
	it.Then(t).Should(it.Nil(os.MkdirAll(sub, 0o755)))

	out, err := sut(NewInitCmd(), []string{sub})

	it.Then(t).Should(it.Error(out, err))
	_, statErr := os.Stat(filepath.Join(sub, ".git"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	entries, rerr := os.ReadDir(sub)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc init <repo>/<non-existent-subfolder>
// spec.md US1 Acceptance Scenario 3 (FR-005): the refusal happens before the
// target directory is created, so nothing is left on disk to clean up.
func TestInitRefusesWithoutCreatingTargetDirectory(t *testing.T) {
	repo := newHostRepo(t)
	target := filepath.Join(repo, "not-yet")

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).Should(it.Error(out, err))
	_, statErr := os.Stat(target)
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init <repo>
// spec.md US1 Acceptance Scenario 4 (FR-006): the refusal names the enclosing
// repository root and the flag that overrides it.
func TestInitRefusalNamesRepositoryAndFlag(t *testing.T) {
	repo := newHostRepo(t)

	out, err := sut(NewInitCmd(), []string{filepath.Join(repo, "sub")})

	message := errorMessage(t, out, err)
	it.Then(t).
		Should(it.String(message).Contain(repo)).
		Should(it.String(message).Contain("--skip-git-init"))
}

// arc init <dir>
// spec.md US1 Acceptance Scenario 5 (FR-007): outside any repository the
// default path is untouched — a repository is created and the graph committed.
func TestInitOutsideRepositoryStillCreatesRepository(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "graph")

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsDir(t, filepath.Join(dir, ".git"))
}

// arc init --skip-git-init <repo-root>
// spec.md US2 Acceptance Scenario 1 (FR-009): the graph is created in place at
// a populated repository root, with no repository of its own.
func TestInitSkipGitCreatesGraphAtRepositoryRoot(t *testing.T) {
	repo := newHostRepo(t)

	out, err := sut(initCmdSkipGit(t), []string{repo})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsDir(t, filepath.Join(repo, "Source"))
	assertIsDir(t, filepath.Join(repo, ".arc"))
	it.Then(t).Should(it.Equal("hi\n", readGraphFile(t, filepath.Join(repo, "README.md"))))
}

// arc init --skip-git-init <repo>/notes
// spec.md US2 Acceptance Scenario 2 (FR-013): exactly one new commit lands in
// the host repository, and it contains only what initialization itself wrote.
func TestInitSkipGitAddsExactlyOneScopedCommit(t *testing.T) {
	repo := newHostRepo(t)
	before := len(strings.Split(strings.TrimSpace(gitOutput(t, repo, "log", "--oneline")), "\n"))

	_, err := sut(initCmdSkipGit(t), []string{filepath.Join(repo, "notes")})
	it.Then(t).Should(it.Nil(err))

	after := len(strings.Split(strings.TrimSpace(gitOutput(t, repo, "log", "--oneline")), "\n"))
	it.Then(t).Should(it.Equal(before+1, after))

	paths := commitPaths(t, repo)
	it.Then(t).Should(it.True(len(paths) > 0))
	for _, path := range paths {
		it.Then(t).Should(it.String(path).HavePrefix("notes/"))
	}
}

// arc init --skip-git-init <repo>/notes
// spec.md US2 Acceptance Scenario 3 (FR-014): unrelated modified, staged and
// untracked files the user had in flight are all exactly as they were — the
// case a plain `git commit -m` would sweep in.
func TestInitSkipGitLeavesUnrelatedChangesUntouched(t *testing.T) {
	repo := newHostRepo(t)
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(repo, "README.md"), []byte("hi\ndirty\n"), 0o644)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(repo, "staged.txt"), []byte("staged\n"), 0o644)))
	gitOutput(t, repo, "add", "staged.txt")
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(repo, "loose.txt"), []byte("loose\n"), 0o644)))

	_, err := sut(initCmdSkipGit(t), []string{filepath.Join(repo, "notes")})
	it.Then(t).Should(it.Nil(err))

	status := gitOutput(t, repo, "status", "--porcelain")
	it.Then(t).
		Should(it.String(status).Contain(" M README.md")).
		Should(it.String(status).Contain("A  staged.txt")).
		Should(it.String(status).Contain("?? loose.txt"))

	for _, path := range commitPaths(t, repo) {
		it.Then(t).
			Should(it.True(path != "README.md")).
			Should(it.True(path != "staged.txt")).
			Should(it.True(path != "loose.txt"))
	}
}

// arc init --skip-git-init <repo>/a/b/notes
// spec.md US2 Acceptance Scenario 4 (FR-002): a nested, not-yet-existing
// subfolder is initialized into the same enclosing repository.
func TestInitSkipGitIntoNestedSubfolder(t *testing.T) {
	repo := newHostRepo(t)
	target := filepath.Join(repo, "a", "b", "notes")

	out, err := sut(initCmdSkipGit(t), []string{target})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsDir(t, filepath.Join(target, "Source"))
	_, statErr := os.Stat(filepath.Join(target, ".git"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	for _, path := range commitPaths(t, repo) {
		it.Then(t).Should(it.String(path).HavePrefix("a/b/notes/"))
	}
}

// arc init --skip-git-init <already-a-graph>
// spec.md US2 Acceptance Scenario 5 (FR-011): re-initialization is refused in
// this mode too, with today's message.
func TestInitSkipGitRefusesAlreadyInitialized(t *testing.T) {
	repo := newHostRepo(t)
	target := filepath.Join(repo, "notes")
	_, err := sut(initCmdSkipGit(t), []string{target})
	it.Then(t).Should(it.Nil(err))

	before := gitOutput(t, repo, "log", "--oneline")

	out, err := sut(initCmdSkipGit(t), []string{target})

	it.Then(t).Should(it.Error(out, err).Contain("already"))
	it.Then(t).Should(it.Equal(before, gitOutput(t, repo, "log", "--oneline")))
}

// arc init --skip-git-init <dir-outside-any-repository>
// spec.md US2 Acceptance Scenario 6 (FR-012): the flag has nothing to attach
// the graph to, and there is no silent fallback to creating a repository.
func TestInitSkipGitWithoutEnclosingRepository(t *testing.T) {
	target := filepath.Join(t.TempDir(), "graph")

	out, err := sut(initCmdSkipGit(t), []string{target})

	it.Then(t).Should(it.Error(out, err))
	_, statErr := os.Stat(target)
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init --json [--skip-git-init] <dir>
// spec.md US2 (FR-027, FR-028; quickstart S7): `repository` is present and
// non-empty in both modes, equals `path` exactly when the graph root is the
// repository root, and the three pre-existing fields are unchanged.
func TestInitJSONReportsRepositoryInBothModes(t *testing.T) {
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	var payload struct {
		Path           string   `json:"path"`
		Commit         string   `json:"commit"`
		FoldersCreated []string `json:"foldersCreated"`
		Repository     string   `json:"repository"`
	}

	standalone := filepath.Join(t.TempDir(), "graph")
	out, err := sut(NewInitCmd(), []string{standalone})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &payload)))
	it.Then(t).
		Should(it.Equal(standalone, payload.Path)).
		Should(it.Equal(standalone, payload.Repository)).
		ShouldNot(it.Equal("", payload.Commit)).
		Should(it.Seq(payload.FoldersCreated).Contain("Source", "Entity"))

	repo := newHostRepo(t)
	nested := filepath.Join(repo, "notes")
	out, err = sut(initCmdSkipGit(t), []string{nested})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &payload)))
	it.Then(t).
		Should(it.Equal(nested, payload.Path)).
		Should(it.Equal(repo, payload.Repository)).
		ShouldNot(it.Equal(payload.Path, payload.Repository))
}

// arc init --skip-git-init <repo>/notes
// spec.md US2 (FR-026): the human line names the existing repository the
// graph was added to, and stays a single line (BUG-001).
func TestInitSkipGitHumanLineNamesRepository(t *testing.T) {
	repo := newHostRepo(t)

	stdout, _, err := sutCaptureStderr(t, initCmdSkipGit(t), []string{filepath.Join(repo, "notes")})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.Equal(1, strings.Count(stdout, "\n"))).
		Should(it.String(stdout).Contain("existing repository")).
		Should(it.String(stdout).Contain(repo))

	// The same must hold when the graph root IS the repository root, where
	// `repository` equals `path` and a comparison alone could not tell the
	// two modes apart.
	atRoot := newHostRepo(t)
	stdout, _, err = sutCaptureStderr(t, initCmdSkipGit(t), []string{atRoot})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.Equal(1, strings.Count(stdout, "\n"))).
		Should(it.String(stdout).Contain("existing repository"))

	// ...and must NOT appear for a standalone init, which created the
	// repository it reports.
	standalone := filepath.Join(t.TempDir(), "graph")
	stdout, _, err = sutCaptureStderr(t, NewInitCmd(), []string{standalone})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).ShouldNot(it.String(stdout).Contain("existing repository"))
}

// arc init --skip-git-init <repo>/notes
// spec.md US3 Acceptance Scenario 1 (FR-017): the host project's own ignore
// file is byte-for-byte unchanged and absent from the commit.
func TestInitSkipGitNeverTouchesHostIgnoreFile(t *testing.T) {
	repo := newHostRepo(t)
	hostIgnore := filepath.Join(repo, ".gitignore")
	it.Then(t).Should(it.Nil(os.WriteFile(hostIgnore, []byte("node_modules/\n"), 0o644)))
	gitOutput(t, repo, "add", "-A")
	gitOutput(t, repo, "commit", "-qm", "host: ignore rules")

	_, err := sut(initCmdSkipGit(t), []string{filepath.Join(repo, "notes")})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Equal("node_modules/\n", readGraphFile(t, hostIgnore)))
	for _, path := range commitPaths(t, repo) {
		it.Then(t).Should(it.True(path != ".gitignore"))
	}
	_, statErr := os.Stat(filepath.Join(repo, "notes", ".gitignore"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init --skip-git-init <repo>/notes
// spec.md US3 Acceptance Scenario 2 (FR-016): the graph's local state is
// excluded from version control by .arc/.gitignore alone.
func TestInitSkipGitExcludesLocalStateViaArcIgnore(t *testing.T) {
	repo := newHostRepo(t)

	_, err := sut(initCmdSkipGit(t), []string{filepath.Join(repo, "notes")})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Equal("*\n", readGraphFile(t, filepath.Join(repo, "notes", ".arc", ".gitignore"))))
	it.Then(t).
		ShouldNot(it.String(gitOutput(t, repo, "ls-files")).Contain("notes/.arc")).
		Should(it.Equal("", strings.TrimSpace(gitOutput(t, repo, "status", "--porcelain"))))
}

// arc init --skip-git-init <repo>
// spec.md US3 Acceptance Scenario 3 (FR-015): a file colliding with a path the
// layout would write is refused before anything is written.
func TestInitSkipGitRefusesFileCollision(t *testing.T) {
	repo := newHostRepo(t)
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(repo, "_schema", "Class"), 0o755)))
	collide := filepath.Join(repo, "_schema", "Class", "Entity.md")
	it.Then(t).Should(it.Nil(os.WriteFile(collide, []byte("mine\n"), 0o644)))

	out, err := sut(initCmdSkipGit(t), []string{repo})

	it.Then(t).Should(it.Error(out, err))
	it.Then(t).Should(it.Equal("mine\n", readGraphFile(t, collide)))
	_, statErr := os.Stat(filepath.Join(repo, "Source"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init --skip-git-init <repo>
// spec.md US3 Acceptance Scenario 4 (FR-015, FR-031): an existing canonical
// folder is refused whether or not it is empty, and the message names the
// conflicting path and the subfolder recovery.
func TestInitSkipGitRefusesFolderCollision(t *testing.T) {
	for _, populated := range []bool{true, false} {
		repo := newHostRepo(t)
		source := filepath.Join(repo, "Source")
		it.Then(t).Should(it.Nil(os.MkdirAll(source, 0o755)))
		if populated {
			it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(source, "mine.txt"), []byte("mine\n"), 0o644)))
		}

		out, err := sut(initCmdSkipGit(t), []string{repo})

		message := errorMessage(t, out, err)
		it.Then(t).
			Should(it.String(message).Contain("Source")).
			Should(it.String(message).Contain("subfolder"))
		if populated {
			it.Then(t).Should(it.Equal("mine\n", readGraphFile(t, filepath.Join(source, "mine.txt"))))
		}
		_, statErr := os.Stat(filepath.Join(repo, "Entity"))
		it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	}
}

// arc init --skip-git-init <repo>/graph
// spec.md US3 Acceptance Scenario 5 (FR-031): the subfolder the collision
// message names is a working recovery.
func TestInitSkipGitSubfolderRecoveryAfterCollision(t *testing.T) {
	repo := newHostRepo(t)
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(repo, "Source"), 0o755)))

	_, err := sut(initCmdSkipGit(t), []string{repo})
	it.Then(t).ShouldNot(it.Nil(err))

	out, err := sut(initCmdSkipGit(t), []string{filepath.Join(repo, "graph")})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsDir(t, filepath.Join(repo, "graph", "Source"))
}

// arc init --skip-git-init <repo>/private/graph
// spec.md US3 (FR-020, quickstart S8): a target the host project's own ignore
// rules exclude is refused upfront, not with a late "nothing to commit".
func TestInitSkipGitRefusesIgnoredTarget(t *testing.T) {
	repo := newHostRepo(t)
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("private/\n"), 0o644)))
	gitOutput(t, repo, "add", "-A")
	gitOutput(t, repo, "commit", "-qm", "host: ignore private")

	out, err := sut(initCmdSkipGit(t), []string{filepath.Join(repo, "private", "graph")})

	it.Then(t).Should(it.String(errorMessage(t, out, err)).Contain("ignore"))
	_, statErr := os.Stat(filepath.Join(repo, "private"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init --skip-git-init <repo>/notes, with staging forced to fail
// spec.md US3 Acceptance Scenario 6 (FR-018, FR-019): a failure after the
// layout is written removes exactly what this run wrote and leaves every
// pre-existing file — and the pre-existing directory itself — intact.
func TestInitSkipGitRollbackLeavesPreExistingFilesIntact(t *testing.T) {
	repo := newHostRepo(t)
	target := filepath.Join(repo, "notes")
	it.Then(t).Should(it.Nil(os.MkdirAll(target, 0o755)))
	keep := filepath.Join(target, "keep.txt")
	it.Then(t).Should(it.Nil(os.WriteFile(keep, []byte("keep\n"), 0o644)))

	// an index lock makes `git add` fail after the layout is on disk —
	// the only way to reach rollback from outside the process.
	lock := filepath.Join(repo, ".git", "index.lock")
	it.Then(t).Should(it.Nil(os.WriteFile(lock, nil, 0o644)))
	t.Cleanup(func() { os.Remove(lock) })

	out, err := sut(initCmdSkipGit(t), []string{target})

	it.Then(t).Should(it.Error(out, err))
	it.Then(t).Should(it.Equal("keep\n", readGraphFile(t, keep)))
	assertIsDir(t, target)
	for _, path := range []string{"Source/.gitkeep", "_schema/Class/Entity.md", ".arc/.gitignore"} {
		_, statErr := os.Stat(filepath.Join(target, filepath.FromSlash(path)))
		it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	}
}
