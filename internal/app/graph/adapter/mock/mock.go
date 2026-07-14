//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package mock is an in-memory fake implementing port.VCS with
// configurable return values/errors and a call log, for
// internal/app/graph/service unit tests.
package mock

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
)

type VCS struct {
	Tracked      map[string]bool
	IsTrackedErr error
	StageAllErr  error
	CommitHash   string
	CommitErr    error

	// CommitsMatchingFn/ChangedPathsFn/CommitsTouchingFn/RevertCommitFn/
	// BlameFn/ShowFileFn back arc revert's six new port.VCS methods
	// (research.md D11) — each defaults to a zero-value, nil-error
	// response when unset, letting a test configure only the primitives
	// its scenario actually exercises.
	CommitsMatchingFn func(dir, needle string) ([]string, error)
	ChangedPathsFn    func(dir, hash string) ([]string, error)
	CommitsTouchingFn func(dir, path string) ([]string, error)
	RevertCommitFn    func(dir, hash string) (string, error)
	BlameFn           func(dir, path string) ([]port.BlameLine, error)
	ShowFileFn        func(dir, hash, path string) ([]byte, error)

	Calls []string
}

func (m *VCS) IsTracked(ctx context.Context, dir, path string) (bool, error) {
	m.Calls = append(m.Calls, "IsTracked:"+dir+":"+path)
	if m.IsTrackedErr != nil {
		return false, m.IsTrackedErr
	}
	return m.Tracked[path], nil
}

func (m *VCS) StageAll(ctx context.Context, dir string) error {
	m.Calls = append(m.Calls, "StageAll:"+dir)
	return m.StageAllErr
}

func (m *VCS) Commit(ctx context.Context, dir, message string) (string, error) {
	m.Calls = append(m.Calls, "Commit:"+dir+":"+message)
	if m.CommitErr != nil {
		return "", m.CommitErr
	}
	return m.CommitHash, nil
}

func (m *VCS) CommitsMatching(ctx context.Context, dir, needle string) ([]string, error) {
	m.Calls = append(m.Calls, "CommitsMatching:"+dir+":"+needle)
	if m.CommitsMatchingFn != nil {
		return m.CommitsMatchingFn(dir, needle)
	}
	return nil, nil
}

func (m *VCS) ChangedPaths(ctx context.Context, dir, hash string) ([]string, error) {
	m.Calls = append(m.Calls, "ChangedPaths:"+dir+":"+hash)
	if m.ChangedPathsFn != nil {
		return m.ChangedPathsFn(dir, hash)
	}
	return nil, nil
}

func (m *VCS) CommitsTouching(ctx context.Context, dir, path string) ([]string, error) {
	m.Calls = append(m.Calls, "CommitsTouching:"+dir+":"+path)
	if m.CommitsTouchingFn != nil {
		return m.CommitsTouchingFn(dir, path)
	}
	return nil, nil
}

func (m *VCS) RevertCommit(ctx context.Context, dir, hash string) (string, error) {
	m.Calls = append(m.Calls, "RevertCommit:"+dir+":"+hash)
	if m.RevertCommitFn != nil {
		return m.RevertCommitFn(dir, hash)
	}
	return "", nil
}

func (m *VCS) Blame(ctx context.Context, dir, path string) ([]port.BlameLine, error) {
	m.Calls = append(m.Calls, "Blame:"+dir+":"+path)
	if m.BlameFn != nil {
		return m.BlameFn(dir, path)
	}
	return nil, nil
}

func (m *VCS) ShowFile(ctx context.Context, dir, hash, path string) ([]byte, error) {
	m.Calls = append(m.Calls, "ShowFile:"+dir+":"+hash+":"+path)
	if m.ShowFileFn != nil {
		return m.ShowFileFn(dir, hash, path)
	}
	return nil, nil
}
