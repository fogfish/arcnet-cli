//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package fsys

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Local is the sole Mounter/Store implementation, wrapping os.DirFS(root)
// for reads and os.Create/os.MkdirAll/os.Remove for writes. It requires
// ResolveLocalRoot(root) to have already succeeded — it performs no
// existence checks or creation of its own.
type Local struct{}

func (Local) Mount(root string) (Store, error) {
	return &local{root: root, fsys: os.DirFS(root)}, nil
}

type local struct {
	root string
	fsys fs.FS

	probeOnce sync.Once
	foldsCase bool
}

// FoldsCase implements CaseFolder by probing this mounted root once, on
// first use, and caching the answer for the rest of the run (spec 003
// FR-026, research.md D11). The probe is lazy rather than done in Mount
// because Mount is also used for a patch's own containing directory — an
// arbitrary place on disk that is not a graph and must not have a
// throwaway file written into it unless something actually asks.
func (l *local) FoldsCase() bool {
	l.probeOnce.Do(func() { l.foldsCase = probeFoldsCase(l.root) })
	return l.foldsCase
}

// probeFoldsCase writes a throwaway file and asks whether the location
// resolves it under the opposite letter case. It prefers .arc/cache/, then
// .arc/, then the root itself.
//
// The preference is about where a probe file that outlives a crash does no
// harm. .arc/cache/ is the graph's machine-local, disposable area, and it
// is the only part of .arc/ still excluded from version control — .arc/
// itself is tracked now, so a leaked probe file there would show up in
// git status and be committable (specs/034-serve-dir-public-state,
// research D6).
//
// The fallback chain is load-bearing, not decorative. A fresh clone has no
// .arc/cache/ at all: its only file ignores itself, so git cannot
// materialise the directory (data-model I6). Without the fallback
// os.CreateTemp would fail there and this function would return its
// safeDefault of case-INSENSITIVE — wrong on Linux, and it would silently
// change node-identity merge behaviour in every clone. Falling back to
// .arc/ keeps the pre-034 behaviour exactly; falling back to the root
// covers a root mounted before "arc init" has run.
//
// Stat is the right tool here and is not the "Stat trap" research.md D11
// warns about: this asks whether a flipped-case name RESOLVES, which is an
// existence question Stat answers truthfully. The trap concerns asking Stat
// what a file is NAMED, where os echoes the caller's own spelling back —
// that question goes through ResolveName's ReadDir instead.
//
// A location that refuses the probe (read-only mount, no permission) is
// reported as case-INSENSITIVE: of the two possible wrong answers, that is
// the one that refuses to fork one subject across two node files, and it
// degrades to a merge plus a warning rather than to silent duplication
// (spec 003 FR-026).
// probeDir picks where probeFoldsCase writes its throwaway file: the
// deepest of .arc/cache/, .arc/ and root that exists as a directory. It is
// separated from the probe itself purely so the preference chain — whose
// fallbacks only matter on a checkout that lacks the preferred directory —
// is directly assertable.
func probeDir(root string) string {
	dir := root
	for _, candidate := range []string{filepath.Join(root, ".arc"), filepath.Join(root, ".arc", "cache")} {
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			dir = candidate
		}
	}
	return dir
}

func probeFoldsCase(root string) bool {
	const safeDefault = true

	dir := probeDir(root)

	probe, err := os.CreateTemp(dir, "arc-case-probe-*.tmp")
	if err != nil {
		return safeDefault
	}
	name := filepath.Base(probe.Name())
	probe.Close()
	defer os.Remove(probe.Name())

	flipped, cased := flipCase(name)
	if !cased {
		return safeDefault
	}

	_, err = os.Stat(filepath.Join(dir, flipped))
	return err == nil
}

func (l *local) Open(name string) (fs.File, error) { return l.fsys.Open(name) }

func (l *local) Stat(name string) (fs.FileInfo, error) { return fs.Stat(l.fsys, name) }

func (l *local) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(l.fsys, name) }

func (l *local) Create(name string) (File, error) {
	path := filepath.Join(l.root, filepath.FromSlash(name))

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, ErrCreate.With(err, name)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, ErrCreate.With(err, name)
	}

	return &localFile{File: f, path: path}, nil
}

func (l *local) Remove(name string) error {
	path := filepath.Join(l.root, filepath.FromSlash(name))
	if err := os.Remove(path); err != nil {
		return ErrRemove.With(err, name)
	}
	return nil
}

type localFile struct {
	*os.File
	path string
}

func (f *localFile) Discard() error {
	f.File.Close()
	return os.Remove(f.path)
}
