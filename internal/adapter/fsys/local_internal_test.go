//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package fsys

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/it/v2"
)

// specs/034-serve-dir-public-state research D6: the case-folding probe
// writes into .arc/cache/ — the only part of .arc/ still excluded from
// version control — rather than into .arc/, which is tracked now and would
// show a leaked probe file in git status.
func TestProbeDirPrefersArcCache(t *testing.T) {
	root := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(root, ".arc", "cache"), 0o755)))

	it.Then(t).Should(it.Equal(filepath.Join(root, ".arc", "cache"), probeDir(root)))
}

// The clone case (data-model I6). A fresh clone has .arc/ but no
// .arc/cache/, because that directory's only file ignores itself. Without
// this fallback os.CreateTemp would fail and probeFoldsCase would return
// its safeDefault of case-insensitive — silently wrong on Linux, in every
// clone.
func TestProbeDirFallsBackToArcWhenCacheAbsent(t *testing.T) {
	root := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(root, ".arc"), 0o755)))

	it.Then(t).Should(it.Equal(filepath.Join(root, ".arc"), probeDir(root)))
}

// A root mounted before "arc init" has run — and a patch's own containing
// directory, which is not a graph at all.
func TestProbeDirFallsBackToRootWhenNoArcState(t *testing.T) {
	root := t.TempDir()

	it.Then(t).Should(it.Equal(root, probeDir(root)))
}

// .arc/cache occupied by a FILE is not a usable probe location, so the
// chain must step past it rather than hand os.CreateTemp a non-directory.
func TestProbeDirSkipsCachePathThatIsNotADirectory(t *testing.T) {
	root := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(root, ".arc"), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(root, ".arc", "cache"), nil, 0o644)))

	it.Then(t).Should(it.Equal(filepath.Join(root, ".arc"), probeDir(root)))
}

// probeFoldsCase itself must answer the same for a graph with a cache, a
// clone without one, and a bare directory — the fallback chain exists so
// the answer never degrades to safeDefault just because the preferred
// location is missing.
func TestProbeFoldsCaseAgreesAcrossEveryProbeLocation(t *testing.T) {
	bare := t.TempDir()
	truth := probeFoldsCase(bare)

	clone := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(clone, ".arc"), 0o755)))

	graph := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(graph, ".arc", "cache"), 0o755)))

	it.Then(t).
		Should(it.Equal(truth, probeFoldsCase(clone))).
		Should(it.Equal(truth, probeFoldsCase(graph)))
}

// The probe leaves nothing behind — the property that made .arc/ an
// attractive home for it in the first place, now that a leak there would
// be committable.
func TestProbeFoldsCaseLeavesNoFileBehind(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, ".arc", "cache")
	it.Then(t).Should(it.Nil(os.MkdirAll(cache, 0o755)))

	probeFoldsCase(root)

	entries, err := os.ReadDir(cache)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(entries)))
}
