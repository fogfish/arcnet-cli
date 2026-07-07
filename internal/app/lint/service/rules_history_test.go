//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

type historyMockVCS struct {
	commits map[string][]string
	err     error
}

func (m historyMockVCS) CommitsMatching(ctx context.Context, dir, needle string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.commits[needle], nil
}

func TestCheckIngestCommitExactlyOneMatch(t *testing.T) {
	vcs := historyMockVCS{commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123"}}}
	node := core.Node{Type: "source", ID: "foo-2026-x"}

	out, err := checkIngestCommit(context.Background(), vcs, "/graph", node, "sources/foo-2026-x.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckIngestCommitZeroMatches(t *testing.T) {
	vcs := historyMockVCS{commits: map[string][]string{}}
	node := core.Node{Type: "source", ID: "foo-2026-x"}

	out, err := checkIngestCommit(context.Background(), vcs, "/graph", node, "sources/foo-2026-x.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIngestCommit, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("0 matching commits"))
}

func TestCheckIngestCommitMoreThanOneMatch(t *testing.T) {
	vcs := historyMockVCS{commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123", "def456"}}}
	node := core.Node{Type: "source", ID: "foo-2026-x"}

	out, err := checkIngestCommit(context.Background(), vcs, "/graph", node, "sources/foo-2026-x.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.String(out[0].Message).Contain("2 matching commits"))
}

func TestCheckIngestCommitNonSourceExempt(t *testing.T) {
	vcs := historyMockVCS{}
	node := core.Node{Type: "entity", ID: "Widget"}

	out, err := checkIngestCommit(context.Background(), vcs, "/graph", node, "entities/Widget.md")

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(0, len(out)))
}

func TestCheckIngestCommitErrorPropagates(t *testing.T) {
	vcs := historyMockVCS{err: errors.New("git log failed")}
	node := core.Node{Type: "source", ID: "foo-2026-x"}

	_, err := checkIngestCommit(context.Background(), vcs, "/graph", node, "sources/foo-2026-x.md")

	it.Then(t).ShouldNot(it.Nil(err))
}
