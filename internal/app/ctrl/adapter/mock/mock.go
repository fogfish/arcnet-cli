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

import "context"

type VCS struct {
	IsAvailableErr error
	InitErr        error
	StageAllErr    error
	CommitHash     string
	CommitErr      error
	Calls          []string
}

func (m *VCS) IsAvailable(ctx context.Context) error {
	m.Calls = append(m.Calls, "IsAvailable")
	return m.IsAvailableErr
}

func (m *VCS) Init(ctx context.Context, dir string) error {
	m.Calls = append(m.Calls, "Init:"+dir)
	return m.InitErr
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
