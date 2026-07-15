//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package mock is an in-memory fake implementing port.VCS and
// port.Fetcher with configurable return values/errors and a call log, for
// internal/app/schema/service unit tests.
package mock

import (
	"bytes"
	"context"
	"io"
)

// VCS mirrors internal/app/ctrl/adapter/mock.VCS's shape, minus
// IsAvailable/Init — port.VCS has no equivalent methods.
type VCS struct {
	StageAllErr error
	CommitHash  string
	CommitErr   error
	Calls       []string
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

// Fetcher is a configurable port.Fetcher fake: Body/Err drive every call
// uniformly, and Calls records each requested URL.
type Fetcher struct {
	Body  []byte
	Err   error
	Calls []string
}

func (m *Fetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	m.Calls = append(m.Calls, url)
	if m.Err != nil {
		return nil, m.Err
	}
	return io.NopCloser(bytes.NewReader(m.Body)), nil
}
