//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package mock is an in-memory fake implementing port.VCS with
// configurable return values/errors and a call log, for
// internal/app/lint/service unit tests.
package mock

import "context"

type VCS struct {
	// Commits maps a needle (as passed to CommitsMatching) to the commit
	// hashes it should return.
	Commits map[string][]string
	Err     error
	Calls   []string
}

func (m *VCS) CommitsMatching(ctx context.Context, dir, needle string) ([]string, error) {
	m.Calls = append(m.Calls, "CommitsMatching:"+dir+":"+needle)
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Commits[needle], nil
}
