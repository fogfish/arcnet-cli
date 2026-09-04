//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package mock is an in-memory fake implementing port.VCS with
// configurable return values/errors and a call log, for
// internal/app/ctrl/service unit tests.
package mock

import (
	"context"
	"strings"
)

type VCS struct {
	IsAvailableErr error
	InitErr        error
	StagePathsErr  error
	CommitHash     string
	CommitErr      error

	// Calls logs every invocation in order, pathspec included, so a test
	// can assert on the exact set of paths staged and committed rather
	// than merely that staging happened.
	Calls []string
}

func (m *VCS) IsAvailable(ctx context.Context) error {
	m.Calls = append(m.Calls, "IsAvailable")
	return m.IsAvailableErr
}

func (m *VCS) Init(ctx context.Context, dir string) error {
	m.Calls = append(m.Calls, "Init:"+dir)
	return m.InitErr
}

func (m *VCS) StagePaths(ctx context.Context, dir string, paths []string) error {
	m.Calls = append(m.Calls, "StagePaths:"+dir+":"+strings.Join(paths, ","))
	return m.StagePathsErr
}

func (m *VCS) CommitPaths(ctx context.Context, dir, message string, paths []string) (string, error) {
	m.Calls = append(m.Calls, "CommitPaths:"+dir+":"+message+":"+strings.Join(paths, ","))
	if m.CommitErr != nil {
		return "", m.CommitErr
	}
	return m.CommitHash, nil
}
