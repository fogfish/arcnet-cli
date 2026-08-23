//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
)

// ---------------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — planUpgrade (contract C3.2/C3.3)
//
// planUpgrade is the whole reason arc upgrade can run at all on a graph the
// tightened merge menu refuses to load: it is a BYTE comparison, never a
// decode. These tests exercise it directly, with a seed built by hand — a
// plan diffed against schema.Seed()'s real output would agree with itself
// and could not distinguish "unchanged" from "not compared".
// ---------------------------------------------------------------------------

const seededPropertyPath = "_schema/Property/scoreZ.md"
const seededClassPath = "_schema/Class/Source.md"

func newUpgradeStore(files map[string]string) *upgradeFakeStore {
	return newUpgradeFakeStore(files)
}

func seedFixture() map[string][]byte {
	return map[string][]byte{
		seededPropertyPath: []byte("corrected scoreZ\n"),
		seededClassPath:    []byte("corrected Source\n"),
	}
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// A graph already carrying this release's bytes yields an empty plan — no
// write, and (via Upgrade) no commit (FR-021, FR-024, contract C3.4).
func TestPlanUpgradeAlreadyCurrentIsEmpty(t *testing.T) {
	store := newUpgradeStore(map[string]string{
		seededPropertyPath: "corrected scoreZ\n",
		seededClassPath:    "corrected Source\n",
	})

	plan, err := planUpgrade(store, seedFixture(), nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(plan.empty())).
		Should(it.Equal(0, len(plan.writes)))
}

// A hand-edited built-in document IS replaced. The built-in vocabulary is
// not a starting point to be customized in place (contract C3.3).
func TestPlanUpgradeHandEditedBuiltInIsReplaced(t *testing.T) {
	store := newUpgradeStore(map[string]string{
		seededPropertyPath: "merge: validatedOverwrite\n",
		seededClassPath:    "corrected Source\n",
	})

	plan, err := planUpgrade(store, seedFixture(), nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Seq(plan.replaced).Equal(seededPropertyPath)).
		Should(it.Equal(0, len(plan.added))).
		Should(it.Equal(0, len(plan.removed))).
		// The original is retained so a later failure can restore it
		// verbatim (contract C3.8).
		Should(it.Equal("merge: validatedOverwrite\n", string(plan.originals[seededPropertyPath])))
}

// A built-in document this release seeds but the graph does not have yet is
// an addition, and carries no original to restore.
func TestPlanUpgradeMissingBuiltInIsAdded(t *testing.T) {
	store := newUpgradeStore(map[string]string{
		seededClassPath: "corrected Source\n",
	})

	plan, err := planUpgrade(store, seedFixture(), nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Seq(plan.added).Equal(seededPropertyPath)).
		Should(it.Equal(0, len(plan.replaced))).
		Should(it.Equal(0, len(plan.originals)))
}

// An author's own _schema/ document is not part of the built-in set, so the
// plan never mentions it and applyUpgrade never touches it (FR-020).
func TestPlanUpgradeAuthorExtendedDocumentIsUntouched(t *testing.T) {
	const authored = "_schema/Property/aporia.md"
	store := newUpgradeStore(map[string]string{
		seededPropertyPath: "corrected scoreZ\n",
		seededClassPath:    "corrected Source\n",
		authored:           "an author's own predicate\n",
	})

	plan, err := planUpgrade(store, seedFixture(), nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(plan.empty()))

	_, mentioned := plan.writes[authored]
	it.Then(t).Should(it.True(!mentioned))

	it.Then(t).Should(it.Nil(applyUpgrade(store, plan)))
	it.Then(t).Should(it.Equal("an author's own predicate\n", store.files[authored]))
}

// A retired built-in still on disk is removed; one this release still seeds
// is never treated as retired even if it appears in the list.
func TestPlanUpgradeRetiredBuiltInIsRemoved(t *testing.T) {
	const retiredPath = "_schema/Property/scoreLegacy.md"
	store := newUpgradeStore(map[string]string{
		seededPropertyPath: "corrected scoreZ\n",
		seededClassPath:    "corrected Source\n",
		retiredPath:        "a predicate an older release seeded\n",
	})

	plan, err := planUpgrade(store, seedFixture(), []string{retiredPath, seededClassPath})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Seq(plan.removed).Equal(retiredPath)).
		Should(it.True(plan.empty() == false))

	it.Then(t).Should(it.Nil(applyUpgrade(store, plan)))
	_, present := store.files[retiredPath]
	it.Then(t).Should(it.True(!present))
}

// A retired name the graph never had is silently skipped, so an entry in
// RetiredBuiltIns can outlive every graph that carried it.
func TestPlanUpgradeAbsentRetiredBuiltInIsSkipped(t *testing.T) {
	store := newUpgradeStore(map[string]string{
		seededPropertyPath: "corrected scoreZ\n",
		seededClassPath:    "corrected Source\n",
	})

	plan, err := planUpgrade(store, seedFixture(), []string{"_schema/Property/neverExisted.md"})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(plan.empty()))
}

// The plan's three lists are sorted, so a --json consumer and a human
// reader both see a stable order across runs.
func TestPlanUpgradeListsAreSorted(t *testing.T) {
	seed := map[string][]byte{
		"_schema/Property/zeta.md":  []byte("z\n"),
		"_schema/Property/alpha.md": []byte("a\n"),
		"_schema/Class/Mu.md":       []byte("m\n"),
	}
	store := newUpgradeStore(map[string]string{})

	plan, err := planUpgrade(store, seed, nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(plan.added).Equal(sortedCopy(plan.added)...))
	it.Then(t).Should(it.Equal(3, len(plan.added)))
}

// rollbackUpgrade puts the graph back exactly as it was: overwritten and
// deleted documents restored verbatim, newly added ones removed
// (contract C3.8).
func TestRollbackUpgradeRestoresOriginalState(t *testing.T) {
	const retiredPath = "_schema/Property/scoreLegacy.md"
	before := map[string]string{
		seededPropertyPath: "merge: validatedOverwrite\n",
		retiredPath:        "a predicate an older release seeded\n",
	}
	store := newUpgradeStore(map[string]string{
		seededPropertyPath: before[seededPropertyPath],
		retiredPath:        before[retiredPath],
	})

	plan, err := planUpgrade(store, seedFixture(), []string{retiredPath})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Nil(applyUpgrade(store, plan)))

	// Mid-flight state: replaced, added, removed.
	it.Then(t).Should(it.Equal("corrected scoreZ\n", store.files[seededPropertyPath]))

	rollbackUpgrade(store, plan)

	it.Then(t).
		Should(it.Equal(before[seededPropertyPath], store.files[seededPropertyPath])).
		Should(it.Equal(before[retiredPath], store.files[retiredPath]))
	_, addedSurvives := store.files[seededClassPath]
	it.Then(t).Should(it.True(!addedSurvives))
}

// ---------------------------------------------------------------------------
// upgradeFakeStore — an in-memory fsys.Store that can actually be READ.
//
// init_test.go's fakeStore answers every Open with fs.ErrNotExist, which is
// enough for Init (it only ever writes) but useless here: planUpgrade's
// whole job is comparing on-disk bytes against the seed.
// ---------------------------------------------------------------------------

type upgradeFakeStore struct {
	files map[string]string
}

func newUpgradeFakeStore(files map[string]string) *upgradeFakeStore {
	out := map[string]string{}
	for k, v := range files {
		out[k] = v
	}
	return &upgradeFakeStore{files: out}
}

type upgradeFakeInfo struct {
	name string
	dir  bool
}

func (i upgradeFakeInfo) Name() string               { return path.Base(i.name) }
func (i upgradeFakeInfo) Size() int64                { return 0 }
func (i upgradeFakeInfo) Mode() fs.FileMode          { return 0 }
func (i upgradeFakeInfo) ModTime() time.Time         { return time.Time{} }
func (i upgradeFakeInfo) IsDir() bool                { return i.dir }
func (i upgradeFakeInfo) Sys() any                   { return nil }
func (i upgradeFakeInfo) Type() fs.FileMode          { return 0 }
func (i upgradeFakeInfo) Info() (fs.FileInfo, error) { return i, nil }

type upgradeFakeOpenFile struct {
	*strings.Reader
	name string
}

func (f upgradeFakeOpenFile) Close() error               { return nil }
func (f upgradeFakeOpenFile) Stat() (fs.FileInfo, error) { return upgradeFakeInfo{name: f.name}, nil }

type upgradeFakeWriteFile struct {
	name    string
	store   *upgradeFakeStore
	content []byte
}

func (f *upgradeFakeWriteFile) Write(p []byte) (int, error) {
	f.content = append(f.content, p...)
	return len(p), nil
}

func (f *upgradeFakeWriteFile) Close() error {
	f.store.files[f.name] = string(f.content)
	return nil
}

func (f *upgradeFakeWriteFile) Discard() error             { return nil }
func (f *upgradeFakeWriteFile) Stat() (fs.FileInfo, error) { return upgradeFakeInfo{name: f.name}, nil }

func (s *upgradeFakeStore) Open(name string) (fs.File, error) {
	content, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return upgradeFakeOpenFile{Reader: strings.NewReader(content), name: name}, nil
}

func (s *upgradeFakeStore) Stat(name string) (fs.FileInfo, error) {
	if _, ok := s.files[name]; ok {
		return upgradeFakeInfo{name: name}, nil
	}
	for p := range s.files {
		if strings.HasPrefix(p, name+"/") {
			return upgradeFakeInfo{name: name, dir: true}, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (s *upgradeFakeStore) ReadDir(name string) ([]fs.DirEntry, error) {
	prefix := ""
	if name != "." {
		prefix = name + "/"
	}

	seen := map[string]bool{}
	var out []fs.DirEntry
	for p := range s.files {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		rest := strings.TrimPrefix(p, prefix)
		if rest == "" {
			continue
		}
		head, _, isDir := strings.Cut(rest, "/")
		if seen[head] {
			continue
		}
		seen[head] = true
		out = append(out, upgradeFakeInfo{name: prefix + head, dir: isDir})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func (s *upgradeFakeStore) Create(name string) (fsys.File, error) {
	return &upgradeFakeWriteFile{name: name, store: s}, nil
}

func (s *upgradeFakeStore) Remove(name string) error {
	delete(s.files, name)
	return nil
}
