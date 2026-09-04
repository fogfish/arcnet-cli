//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package service implements the ctrl use-case's business logic.
package service

import (
	"context"
	"errors"
	"io/fs"
	"sort"
	"strings"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/port"
)

const (
	initCommitMessage = "graph(init): empty knowledge graph"
	arcStateDir       = ".arc"
	arcStateMarker    = ".arc/.gitkeep"

	// arcIgnorePath excludes the graph's local state from version control
	// from INSIDE .arc/ rather than from a .gitignore at the graph root.
	// A `*` rule there excludes that directory's entire contents including
	// the rule file itself, so nothing under .arc/ is ever tracked and arc
	// creates no file it does not own — which is what lets a graph be
	// added to a project that already has ignore rules of its own, without
	// arc reading, creating or appending to them (FR-016, FR-017,
	// research.md D4, Constitution XI). Applied identically in both modes:
	// the observable outcome SC-008 protects — local state untracked, a
	// clean working tree after init — is unchanged for a standalone graph.
	arcIgnorePath    = ".arc/.gitignore"
	arcIgnoreContent = "*\n"
)

// errNoCause is passed to faults.SafeN.With for guard conditions that are
// not caused by an underlying Go error, so the rendered message has no
// trailing "%!s(<nil>)" artifact.
var errNoCause = errors.New("")

// resolveLocalRoot/removeLocalRoot are indirected through package-level
// vars (rather than called as fsys.ResolveLocalRoot/fsys.RemoveLocalRoot
// directly) so unit tests can stub root resolution with an in-memory fake
// and exercise guard/rollback logic with no real disk access
// (contracts/fsys-port-contract.md "Test doubles").
var (
	resolveLocalRoot = fsys.ResolveLocalRoot
	removeLocalRoot  = fsys.RemoveLocalRoot
)

// footprint is everything one run of writeLayout put on disk: the
// graph-relative files it wrote, in write order, and the directories it had
// to create to write them. It is consumed by exactly two callers — the
// explicit pathspec staging and committing use (Files, which is what makes
// FR-013/FR-014 true), and rollback, which removes precisely what this run
// created rather than every path the layout COULD have written.
//
// Recording it is load-bearing rather than a refinement: guardTargetEmpty
// no longer applies in --skip-git-init mode (FR-010), so the target is not
// guaranteed to have been empty and a static removal list would delete
// files and folders the tool never created (FR-018, FR-019).
type footprint struct {
	// Files are graph-relative paths, in the order they were written.
	Files []string

	// Dirs are the directories that did not exist before this run,
	// deepest-first so removing them in order never has to recurse. A
	// directory that pre-existed is never listed and therefore never
	// removed (FR-019).
	Dirs []string
}

// Tracked returns the paths that belong in the commit: everything except
// the .arc/ local state, which arcIgnorePath deliberately excludes from
// version control.
//
// The distinction is forced by git, not merely tidy. `git add -A -- .`
// skips ignored paths silently; `git add -- <path>` NAMED an ignored path
// fails outright ("The following paths are ignored by one of your
// .gitignore files"), so the pathspec that makes FR-013/FR-014 true must
// carry only what is meant to be tracked.
func (f footprint) Tracked() []string {
	tracked := make([]string, 0, len(f.Files))
	for _, path := range f.Files {
		if path == arcStateDir || strings.HasPrefix(path, arcStateDir+"/") {
			continue
		}
		tracked = append(tracked, path)
	}
	return tracked
}

// Init bootstraps a new, empty knowledge graph at dir: the canonical folder
// layout, _schema/ seeded with schemaSeed, the .arc/ state directory with
// its own exclusion rule, and exactly one git commit.
//
// opts carries the repository context cmd resolved before calling in
// (kernel.InitOpts). Every guard runs BEFORE resolveLocalRoot, so a refusal
// never touches the filesystem at all (FR-005) rather than relying on
// rollback to undo itself — and so detection works for a target that does
// not exist yet (research.md D2). Any failure at or after writeLayout rolls
// back exactly this run's footprint (FR-018).
func Init(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, dir string, schemaSeed map[string]string, opts kernel.InitOpts) (kernel.InitResult, error) {
	layout := kernel.DefaultLayout
	layout.SeedFiles = make(map[string]string, len(kernel.DefaultLayout.SeedFiles)+len(schemaSeed))
	for k, v := range kernel.DefaultLayout.SeedFiles {
		layout.SeedFiles[k] = v
	}
	for k, v := range schemaSeed {
		layout.SeedFiles[k] = v
	}

	// The target may not exist yet, in which case there is nothing to
	// inspect and every guard that reads it passes vacuously.
	target := existingTarget(mounter, dir)

	// "This is already a graph" comes first and wins over everything else:
	// it is true regardless of repository context, and it is the answer a
	// user re-running arc init needs to see (FR-011, today's message
	// unchanged).
	if target != nil {
		if err := guardNotAlreadyInitialized(target, dir); err != nil {
			return kernel.InitResult{}, err
		}
	}

	if err := guardRepositoryContext(dir, opts); err != nil {
		return kernel.InitResult{}, err
	}

	if target != nil {
		if err := guardTargetLayout(target, dir, layout, opts); err != nil {
			return kernel.InitResult{}, err
		}
	}

	if err := vcs.IsAvailable(ctx); err != nil {
		return kernel.InitResult{}, ErrGitUnavailable.With(err)
	}

	created, err := resolveLocalRoot(dir)
	if err != nil {
		return kernel.InitResult{}, err
	}

	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.InitResult{}, err
	}

	written, err := writeLayout(store, layout)
	if err != nil {
		rollback(store, dir, created, written)
		return kernel.InitResult{}, err
	}

	// --skip-git-init adds the graph to the repository that already
	// encloses it; there is deliberately no repository to create, and no
	// "Preparing git repository" step to report (FR-009).
	if !opts.SkipGitInit {
		if err := vcs.Init(ctx, dir); err != nil {
			rollback(store, dir, created, written)
			return kernel.InitResult{}, err
		}
	}

	// Staging and committing run at the graph root with graph-relative
	// paths in BOTH modes: git resolves a pathspec against the directory it
	// runs in and walks up to the enclosing repository on its own, so one
	// call shape is correct whether the graph root is the repository root
	// or a subfolder of one (contracts/vcs-adapter-contract.md A3/A4).
	if err := vcs.StagePaths(ctx, dir, written.Tracked()); err != nil {
		rollback(store, dir, created, written)
		return kernel.InitResult{}, err
	}

	hash, err := vcs.CommitPaths(ctx, dir, initCommitMessage, written.Tracked())
	if err != nil {
		rollback(store, dir, created, written)
		return kernel.InitResult{}, err
	}

	repository := dir
	if opts.SkipGitInit {
		repository = opts.ParentRepo
	}

	return kernel.InitResult{
		Root:           kernel.GraphRoot{Root: dir},
		CommitHash:     hash,
		FoldersCreated: kernel.DefaultLayout.Folders,
		Repository:     repository,
	}, nil
}

// guardRepositoryContext applies the two rules that depend only on the
// repository facts cmd resolved — R1 and R2 of data-model.md — plus R3,
// which asks whether the host project would exclude the graph from version
// control. None of them reads the filesystem, so all three can and must
// refuse before anything exists to clean up.
func guardRepositoryContext(dir string, opts kernel.InitOpts) error {
	switch {
	// R1 (FR-004, FR-006): nesting a repository inside a repository is the
	// harmful default this feature exists to stop.
	case !opts.SkipGitInit && opts.ParentRepo != "":
		return ErrInsideRepository.With(errNoCause, dir, opts.ParentRepo)

	// R2 (FR-012): the flag has nothing to attach the graph to. There is
	// deliberately no fallback to creating a repository — that would
	// silently do the thing the user asked not to do.
	case opts.SkipGitInit && opts.ParentRepo == "":
		return ErrNoParentRepository.With(errNoCause, dir)

	// R3 (FR-020): without this the failure still happens, but late and
	// incomprehensibly — the layout is written, `git add` silently matches
	// nothing, and the commit fails with "nothing to commit".
	case opts.SkipGitInit && opts.TargetIgnored:
		return ErrTargetIgnored.With(errNoCause, dir)
	}

	return nil
}

// existingTarget mounts dir when it already exists as a directory, and
// returns nil otherwise, so the guards that inspect the target can be
// skipped entirely when there is nothing there yet. Mounting an absent root
// is safe — fsys.Local performs no existence check — but asking it
// questions is not, so the probe is explicit. A target that exists and is
// NOT a directory is left to resolveLocalRoot, which owns that diagnosis
// (fsys.ErrRootNotDirectory).
func existingTarget(mounter fsys.Mounter, dir string) fsys.Store {
	store, err := mounter.Mount(dir)
	if err != nil {
		return nil
	}
	if info, err := store.Stat("."); err != nil || !info.IsDir() {
		return nil
	}
	return store
}

// guardTargetLayout applies the rules about what the target already holds.
// It is skipped entirely when the target does not exist.
func guardTargetLayout(store fsys.Store, dir string, layout kernel.ArcNetCoreLayout, opts kernel.InitOpts) error {
	if err := guardNoLayoutCollision(store, layout); err != nil {
		return err
	}

	// FR-010: --skip-git-init exists precisely to add a graph to a
	// directory that already holds the user's own files, so the
	// emptiness requirement does not apply to it. It is unchanged for the
	// standalone path (FR-007).
	if opts.SkipGitInit {
		return nil
	}

	return guardTargetEmpty(store, dir)
}

func guardNotAlreadyInitialized(store fsys.Store, dir string) error {
	_, err := store.Stat(arcStateDir)
	switch {
	case err == nil:
		return ErrAlreadyInitialized.With(errNoCause, dir)
	case errors.Is(err, fs.ErrNotExist):
		return nil
	default:
		return err
	}
}

func guardTargetEmpty(store fsys.Store, dir string) error {
	entries, err := store.ReadDir(".")
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return ErrTargetNotEmpty.With(errNoCause, dir)
	}
	return nil
}

// guardNoLayoutCollision refuses, before any write, a target where the
// layout would overwrite or adopt something the user already owns (FR-015).
//
// Both classes collide on mere existence, and that is the point. A
// canonical FOLDER is refused whether or not it holds anything: an empty
// Source/ is no safer than a full one, because what initialization would
// take over is the folder's meaning, not its contents (spec clarification
// 2). A layout FILE is refused because writing it would overwrite the
// user's own. Either failure names the conflicting path and the subfolder
// recovery (FR-031).
func guardNoLayoutCollision(store fsys.Store, layout kernel.ArcNetCoreLayout) error {
	// Folders first: a folder collision is the one a user is most likely
	// to hit and the one whose recovery advice is most useful, and naming
	// "Source" reads better than naming "Source/.gitkeep".
	for _, path := range append(append([]string{}, layout.Folders...), layoutPaths(layout)...) {
		_, err := store.Stat(path)
		switch {
		case err == nil:
			return ErrLayoutCollision.With(errNoCause, path)
		case errors.Is(err, fs.ErrNotExist):
			continue
		default:
			return err
		}
	}

	return nil
}

// layoutPaths enumerates, in write order, every file path writeLayout will
// create. Both writeLayout and the collision guard derive their work from
// it, so the guard can never check a different set than the writer writes.
func layoutPaths(layout kernel.ArcNetCoreLayout) []string {
	paths := make([]string, 0, len(layout.Folders)+len(layout.SeedFiles)+2)

	for _, folder := range layout.Folders {
		if hasStub(layout, folder) {
			continue
		}
		paths = append(paths, folder+"/.gitkeep")
	}

	seeded := make([]string, 0, len(layout.SeedFiles))
	for path := range layout.SeedFiles {
		seeded = append(seeded, path)
	}
	sort.Strings(seeded)
	paths = append(paths, seeded...)

	return append(paths, arcStateMarker, arcIgnorePath)
}

// writeLayout writes every path layoutPaths describes and returns the
// footprint it actually put on disk — what the commit's pathspec and any
// rollback are both expressed in terms of. A partial write returns what got
// as far as disk, so rollback removes exactly that and no more.
//
// The directories are surveyed BEFORE the first write, since fsys.Store
// creates missing parents implicitly on Create: afterwards there is no way
// left to tell a directory this run made from one the user already had.
func writeLayout(store fsys.Store, layout kernel.ArcNetCoreLayout) (footprint, error) {
	paths := layoutPaths(layout)
	written := footprint{
		Files: make([]string, 0, len(paths)),
		Dirs:  absentDirs(store, paths),
	}

	for _, path := range paths {
		if err := writeFile(store, path, contentFor(layout, path)); err != nil {
			return written, err
		}
		written.Files = append(written.Files, path)
	}

	return written, nil
}

// absentDirs returns every directory that paths need but that does not
// exist yet, deepest-first — the set this run is about to create and is
// therefore entitled to remove again on failure.
func absentDirs(store fsys.Store, paths []string) []string {
	absent := map[string]bool{}

	for _, path := range paths {
		for dir := parentDir(path); dir != ""; dir = parentDir(dir) {
			if absent[dir] {
				break
			}
			if _, err := store.Stat(dir); err == nil {
				break
			}
			absent[dir] = true
		}
	}

	dirs := make([]string, 0, len(absent))
	for dir := range absent {
		dirs = append(dirs, dir)
	}
	// deepest-first: more separators sorts earlier, ties broken by name so
	// the order is stable and testable.
	sort.Slice(dirs, func(i, j int) bool {
		di, dj := strings.Count(dirs[i], "/"), strings.Count(dirs[j], "/")
		if di != dj {
			return di > dj
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

func parentDir(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return ""
	}
	return path[:i]
}

func contentFor(layout kernel.ArcNetCoreLayout, path string) string {
	if content, ok := layout.SeedFiles[path]; ok {
		return content
	}
	if path == arcIgnorePath {
		return arcIgnoreContent
	}
	return ""
}

func hasStub(layout kernel.ArcNetCoreLayout, folder string) bool {
	for path := range layout.SeedFiles {
		if strings.HasPrefix(path, folder+"/") {
			return true
		}
	}
	return false
}

func writeFile(store fsys.Store, path, content string) error {
	f, err := store.Create(path)
	if err != nil {
		return ErrLayoutWrite.With(err, path)
	}

	if _, err := f.Write([]byte(content)); err != nil {
		_ = f.Discard()
		return ErrLayoutWrite.With(err, path)
	}

	if err := f.Close(); err != nil {
		return ErrLayoutWrite.With(err, path)
	}

	return nil
}

// rollback undoes a failed initialization. When created is true the whole
// directory was this run's own creation, so one RemoveLocalRoot call undoes
// everything. Otherwise the graph root pre-existed and is not this run's to
// delete (FR-019): only the recorded footprint is removed, file by file, so
// nothing the user already owned is touched (FR-018).
func rollback(store fsys.Store, dir string, created bool, written footprint) {
	if created {
		_ = removeLocalRoot(dir)
		return
	}

	for _, path := range written.Files {
		_ = store.Remove(path)
	}
	for _, path := range written.Dirs {
		_ = store.Remove(path)
	}
}
