//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package fsys_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/it/v2"
)

// observedFoldsCase reports what the directory ACTUALLY does with letter
// case, established independently of the probe under test so this file
// asserts agreement with reality rather than with a hardcoded expectation.
// It is what lets the same test be meaningful on a case-insensitive
// developer machine (APFS) and a case-sensitive CI volume (ext4) without
// either skipping.
func observedFoldsCase(t *testing.T, dir string) bool {
	t.Helper()

	probe := filepath.Join(dir, "arc-observed-case.tmp")
	it.Then(t).Should(it.Nil(os.WriteFile(probe, []byte("x"), 0o600)))
	defer os.Remove(probe)

	_, err := os.Stat(filepath.Join(dir, "ARC-OBSERVED-CASE.TMP"))
	return err == nil
}

func TestLocalFoldsCaseMatchesTheVolumesRealBehavior(t *testing.T) {
	root := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(root, ".arc"), 0o755)))

	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(
		it.Equal(fsys.FoldsCase(store), observedFoldsCase(t, root)),
	)
}

func TestLocalFoldsCaseLeavesNoProbeFileBehind(t *testing.T) {
	root := t.TempDir()
	arc := filepath.Join(root, ".arc")
	it.Then(t).Should(it.Nil(os.MkdirAll(arc, 0o755)))

	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))
	fsys.FoldsCase(store)

	for _, dir := range []string{root, arc} {
		entries, err := os.ReadDir(dir)
		it.Then(t).Should(it.Nil(err))
		for _, entry := range entries {
			it.Then(t).Should(it.Equal(entry.Name() == ".arc", true))
		}
	}
}

// A root with no .arc/ (one mounted before "arc init" ran, or a patch's own
// containing directory) still answers, falling back to probing the root.
func TestLocalFoldsCaseWithoutArcDirectory(t *testing.T) {
	root := t.TempDir()

	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(
		it.Equal(fsys.FoldsCase(store), observedFoldsCase(t, root)),
	)
}

// A Store that cannot answer the question is case-sensitive — exact
// comparison, the behavior every caller had before FR-026.
func TestFoldsCaseIsFalseForStoreWithoutTheCapability(t *testing.T) {
	it.Then(t).Should(
		it.Equal(fsys.FoldsCase(fstest.MapFS{"Entity/LightStep.md": {}}), false),
	)
}

func TestResolveNameReturnsTheExactNameWhenItExists(t *testing.T) {
	store := fstest.MapFS{"Entity/LightStep.md": {Data: []byte("x")}}

	actual, found, err := fsys.ResolveName(store, "Entity/LightStep.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, true),
		it.Equal(actual, "Entity/LightStep.md"),
	)
}

// The case-sensitive branch: a case variant is simply absent, so the caller
// goes on to create a second, genuinely distinct node (spec FR-005/FR-027).
func TestResolveNameDoesNotFoldForACaseSensitiveStore(t *testing.T) {
	store := fstest.MapFS{"Entity/LightStep.md": {Data: []byte("x")}}

	_, found, err := fsys.ResolveName(store, "Entity/Lightstep.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, false),
	)
}

func TestResolveNameReportsMissingParentAsNotFound(t *testing.T) {
	store := fstest.MapFS{"Entity/LightStep.md": {Data: []byte("x")}}

	_, found, err := fsys.ResolveName(store, "Nothing/Here.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, false),
	)
}

// foldingFS is a case-folding store: it answers CaseFolder, which is how a
// test injects the probe's result instead of depending on the disk the
// tests happen to run on (spec SC-012).
type foldingFS struct{ fstest.MapFS }

func (foldingFS) FoldsCase() bool { return true }

func TestResolveNameFoldsToTheOnDiskSpelling(t *testing.T) {
	store := foldingFS{fstest.MapFS{"Entity/LightStep.md": {Data: []byte("x")}}}

	actual, found, err := fsys.ResolveName(store, "Entity/Lightstep.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, true),
		it.Equal(actual, "Entity/LightStep.md"),
	)
}

func TestResolveNameFoldsAtTheRoot(t *testing.T) {
	store := foldingFS{fstest.MapFS{"README.md": {Data: []byte("x")}}}

	actual, found, err := fsys.ResolveName(store, "readme.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, true),
		it.Equal(actual, "README.md"),
	)
}

// An exact match wins over a case-variant sibling even where the location
// folds — it is the file the caller literally named.
func TestResolveNamePrefersTheExactMatch(t *testing.T) {
	store := foldingFS{fstest.MapFS{
		"Entity/LightStep.md": {Data: []byte("x")},
		"Entity/Lightstep.md": {Data: []byte("y")},
	}}

	actual, found, err := fsys.ResolveName(store, "Entity/Lightstep.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, true),
		it.Equal(actual, "Entity/Lightstep.md"),
	)
}

// stubDirFS enumerates nothing, the way an in-memory fake often does.
// Exact-match existence must still be answered — via Open — or every caller
// would suddenly see an existing file as missing (this is what regressed
// TestApplyMergesExistingNode while BUG-008 was being implemented).
type stubDirFS struct{ fstest.MapFS }

func (stubDirFS) ReadDir(string) ([]fs.DirEntry, error) { return nil, nil }

func TestResolveNameFallsBackToOpenWhenTheStoreCannotEnumerate(t *testing.T) {
	store := stubDirFS{fstest.MapFS{"Entity/LightStep.md": {Data: []byte("x")}}}

	actual, found, err := fsys.ResolveName(store, "Entity/LightStep.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, true),
		it.Equal(actual, "Entity/LightStep.md"),
	)

	_, found, err = fsys.ResolveName(store, "Entity/Missing.md")
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(found, false),
	)
}
