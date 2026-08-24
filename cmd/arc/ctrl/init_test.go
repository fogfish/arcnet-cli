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

	gitignore, rerr := os.ReadFile(filepath.Join(dir, ".gitignore"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.String(string(gitignore)).Contain(".arc/"))
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
		// specs/023 FR-028 supersedes spec 022's Clarification: §11.6 v0.11
		// requires "title" alone (plan.md F3). "authors"/"year"/"ref"/
		// "status"/"notes" are retired outright under CORE 0.12
		// (BUG-001/FR-031/FR-032/FR-034/FR-033) — "author"/"published"
		// replace the first two, the rest have no replacement.
		Should(it.Seq(schemaBullets(t, content, "required")).Equal("title")).
		Should(it.Seq(schemaBullets(t, content, "optional")).Equal("url", "author", "published", "doi", "isCitedBy", "relevance", "indexed")).
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
