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
	"strings"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/port"
	"github.com/fogfish/arcnet-cli/internal/core"
)

const (
	initCommitMessage = "graph(init): empty knowledge graph"
	arcStateMarker    = ".arc/.gitkeep"
	gitignorePath     = ".gitignore"
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

// Init bootstraps a new, empty knowledge graph at dir: the canonical folder
// layout, _meta/ registry stubs, .arc/ state directory, .arc/config.yml
// seeded with configSeed, .gitignore, and exactly one git commit. Any
// failure after the target has been resolved leaves no partial graph state
// behind (FR-013).
func Init(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, dir string, configSeed []byte) (kernel.InitResult, error) {
	created, err := resolveLocalRoot(dir)
	if err != nil {
		return kernel.InitResult{}, err
	}

	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.InitResult{}, err
	}

	if err := guardNotAlreadyInitialized(store, dir); err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, err
	}

	if err := guardTargetEmpty(store, dir); err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, err
	}

	if err := vcs.IsAvailable(ctx); err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, ErrGitUnavailable.With(err)
	}

	layout := kernel.DefaultLayout
	layout.MetaStubs = make(map[string]string, len(kernel.DefaultLayout.MetaStubs)+1)
	for k, v := range kernel.DefaultLayout.MetaStubs {
		layout.MetaStubs[k] = v
	}
	layout.MetaStubs[core.ConfigPath] = string(configSeed)

	if err := writeLayout(store, layout); err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, err
	}

	if err := vcs.Init(ctx, dir); err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, err
	}

	if err := vcs.StageAll(ctx, dir); err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, err
	}

	hash, err := vcs.Commit(ctx, dir, initCommitMessage)
	if err != nil {
		rollback(store, dir, created)
		return kernel.InitResult{}, err
	}

	return kernel.InitResult{
		Root:           kernel.GraphRoot{Root: dir},
		CommitHash:     hash,
		FoldersCreated: kernel.DefaultLayout.Folders,
	}, nil
}

func guardNotAlreadyInitialized(store fsys.Store, dir string) error {
	_, err := store.Stat(".arc")
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

func writeLayout(store fsys.Store, layout kernel.ArcNetCoreLayout) error {
	for _, folder := range layout.Folders {
		if hasStub(layout, folder) {
			continue
		}
		if err := writeFile(store, folder+"/.gitkeep", ""); err != nil {
			return err
		}
	}

	for path, content := range layout.MetaStubs {
		if err := writeFile(store, path, content); err != nil {
			return err
		}
	}

	if err := writeFile(store, arcStateMarker, ""); err != nil {
		return err
	}

	return writeFile(store, gitignorePath, ".arc/\n")
}

func hasStub(layout kernel.ArcNetCoreLayout, folder string) bool {
	for path := range layout.MetaStubs {
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

// rollback undoes a failed initialization. When created is true, the whole
// directory was this run's own creation, so one RemoveLocalRoot call undoes
// everything. Otherwise, only the fixed, statically-known set of paths
// ArcNetCoreLayout describes is removed, tolerating not-found errors for
// any that were never reached (research.md D4).
func rollback(store fsys.Store, dir string, created bool) {
	if created {
		_ = removeLocalRoot(dir)
		return
	}

	for _, folder := range kernel.DefaultLayout.Folders {
		_ = store.Remove(folder + "/.gitkeep")
	}
	for path := range kernel.DefaultLayout.MetaStubs {
		_ = store.Remove(path)
	}
	_ = store.Remove(core.ConfigPath)
	_ = store.Remove(arcStateMarker)
	_ = store.Remove(gitignorePath)
}
