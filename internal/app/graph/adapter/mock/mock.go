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

import "context"

type VCS struct {
	Tracked      map[string]bool
	IsTrackedErr error
	StageAllErr  error
	CommitHash   string
	CommitErr    error
	Calls        []string
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
