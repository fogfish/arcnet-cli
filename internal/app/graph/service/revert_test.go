//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	graphmock "github.com/fogfish/arcnet-cli/internal/app/graph/adapter/mock"
	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestRevertGuardNotAGraph(t *testing.T) {
	store := newMemStore()

	_, err := service.Revert(context.Background(), memMounter{store: store}, &graphmock.VCS{}, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

// research.md D1: zero matches refuses.
func TestRevertNoIngestCommitRefuses(t *testing.T) {
	store := newGraphStore()
	vcs := &graphmock.VCS{}

	_, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNoIngestCommit)))
}

// research.md D1 (corrected — BUG-001): more than one match is the
// expected result of a prior retract-then-reapply cycle for sourceID, not
// an integrity anomaly — the newest match (hashes[0], CommitsMatching's
// own newest-first ordering) is always the currently active ingest
// commit, and Revert acts on it without refusing.
func TestRevertUsesNewestIngestCommitWhenMultipleMatchesExist(t *testing.T) {
	store := newGraphStore()
	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"newer456", "older123"}, nil },
		ChangedPathsFn:    func(dir, hash string) ([]string, error) { return []string{"sources/foo-2026-x.md"}, nil },
		CommitsTouchingFn: func(dir, path string) ([]string, error) { return []string{"newer456"}, nil },
		RevertCommitFn:    func(dir, hash string) (string, error) { return "rev789", nil },
	}

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("whole-commit", result.Approach)).
		Should(it.Equal("rev789", result.CommitHash))
	it.Then(t).
		Should(it.Seq(vcs.Calls).Contain("ChangedPaths:/graph:newer456")).
		Should(it.Seq(vcs.Calls).Contain("RevertCommit:/graph:newer456"))
}

// research.md D2 / Clarifications Session 2026-07-12: an already-retracted
// document's source node is already absent — a safe no-op, not an error.
func TestRevertSkipsWhenSourceNodeAlreadyRemoved(t *testing.T) {
	store := newGraphStore()
	vcs := &graphmock.VCS{
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
	}

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(result.Skipped)).
		Should(it.Equal("foo-2026-x", result.Document))
	it.Then(t).ShouldNot(it.Seq(vcs.Calls).Contain("ChangedPaths:/graph:ingest123"))
}

// spec.md FR-016: a failure at the final commit step propagates the
// error and never fabricates a commit hash — the per-node path's own
// node removals/rewrites are left uncommitted (recoverable via git,
// mirroring apply.go's own bounded-rollback precedent), not silently
// discarded or partially reported as success.
func TestRevertCommitFailurePropagatesWithoutFabricatingResult(t *testing.T) {
	store := newGraphStore()
	store.files["sources/foo-2026-x.md"] = []byte("---\n\"@id\": foo-2026-x\n\"@type\": source\n---\n# foo-2026-x\n")
	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
		ChangedPathsFn: func(dir, hash string) ([]string, error) {
			return []string{"sources/foo-2026-x.md", "entities/Widget.md"}, nil
		},
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			if path == "sources/foo-2026-x.md" {
				return []string{"ingest123"}, nil
			}
			return []string{"later999", "ingest123"}, nil
		},
		CommitErr: errors.New("commit failed"),
	}

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Equal("", result.CommitHash))
	it.Then(t).Should(it.Seq(vcs.Calls).Contain("StageAll:/graph"))
}

// research.md D3/D4: nothing has touched the ingest commit's own files
// since — takes the whole-commit git revert path.
func TestRevertWholeCommitPathWhenNothingTouchedSince(t *testing.T) {
	store := newGraphStore()
	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
		ChangedPathsFn:    func(dir, hash string) ([]string, error) { return []string{"sources/foo-2026-x.md"}, nil },
		CommitsTouchingFn: func(dir, path string) ([]string, error) { return []string{"ingest123"}, nil },
		RevertCommitFn:    func(dir, hash string) (string, error) { return "rev456", nil },
	}

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("whole-commit", result.Approach)).
		Should(it.Equal("rev456", result.CommitHash))
	it.Then(t).Should(it.Seq(vcs.Calls).Contain("RevertCommit:/graph:ingest123"))
}

// research.md D3: a later commit touched one of the ingest commit's own
// files — the whole-operation eligibility test fails and the per-node
// path is taken instead, even though the source node itself remains
// exclusively owned (invariant: it is always removed).
func TestRevertDispatchesToPerNodeWhenLaterCommitTouchedAPath(t *testing.T) {
	store := newGraphStore()
	store.files["sources/foo-2026-x.md"] = []byte("---\n\"@id\": foo-2026-x\n\"@type\": source\n---\n# foo-2026-x\n")
	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
		ChangedPathsFn: func(dir, hash string) ([]string, error) {
			return []string{"sources/foo-2026-x.md", "entities/Widget.md"}, nil
		},
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			if path == "sources/foo-2026-x.md" {
				return []string{"ingest123"}, nil
			}
			return []string{"later999", "ingest123"}, nil
		},
		CommitHash: "commitABC",
	}

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("per-node", result.Approach)).
		Should(it.Equal(1, result.Removed["source"])).
		Should(it.Equal("commitABC", result.CommitHash))
	it.Then(t).Should(it.Seq(store.removed).Contain("sources/foo-2026-x.md"))
	it.Then(t).Should(it.Seq(vcs.Calls).Contain("Commit:/graph:graph(revert): foo-2026-x — per-node reconciliation\n\nRemoved: 1 nodes (source: 1)\nReconciled: 0 nodes ()\nLinks removed: 0\nReverted-Document: foo-2026-x\n"))
}

// research.md D7: a shared node's paragraph is stripped only when every
// one of its lines is blamed to the reverted patch's own ingest commit —
// a paragraph contributed by a later patch is left untouched.
func TestRevertReconcilesSharedNodeStrippingOnlyBlamedParagraph(t *testing.T) {
	store := newGraphStore()
	node := core.Node{
		ID:   "Widget",
		Type: "entity",
		Texts: map[string]string{
			"notes": "First paragraph from foo-2026-x.\n\nSecond paragraph from later patch.",
		},
	}
	rendered, err := core.RenderNode(node, coreIndexFixture)
	it.Then(t).Should(it.Nil(err))
	store.files["entities/Widget.md"] = rendered

	lines := strings.Split(strings.TrimSuffix(string(rendered), "\n"), "\n")
	firstLine := -1
	for i, l := range lines {
		if l == "First paragraph from foo-2026-x." {
			firstLine = i + 1
		}
	}
	it.Then(t).ShouldNot(it.Equal(-1, firstLine))

	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
		ChangedPathsFn: func(dir, hash string) ([]string, error) {
			return []string{"sources/foo-2026-x.md", "entities/Widget.md"}, nil
		},
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			if path == "sources/foo-2026-x.md" {
				return []string{"ingest123"}, nil
			}
			return []string{"later999", "ingest123"}, nil
		},
		BlameFn: func(dir, path string) ([]port.BlameLine, error) {
			return []port.BlameLine{{Number: firstLine, Commit: "ingest123"}}, nil
		},
		CommitHash: "commitDEF",
	}
	store.files["sources/foo-2026-x.md"] = []byte("---\n\"@id\": foo-2026-x\n\"@type\": source\n---\n# foo-2026-x\n")

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, result.Reconciled["entity"]))

	content := string(store.files["entities/Widget.md"])
	it.Then(t).
		ShouldNot(it.String(content).Contain("First paragraph from foo-2026-x.")).
		Should(it.String(content).Contain("Second paragraph from later patch."))
}

// research.md D9: no line of a shared node's content is attributed to the
// reverted patch — a safe, successful no-op, not an error, and no write.
func TestRevertLeavesSharedNodeUnchangedWhenNoAttribution(t *testing.T) {
	store := newGraphStore()
	node := core.Node{
		ID:    "Widget",
		Type:  "entity",
		Texts: map[string]string{"notes": "Only a later patch's paragraph."},
	}
	rendered, err := core.RenderNode(node, coreIndexFixture)
	it.Then(t).Should(it.Nil(err))
	store.files["entities/Widget.md"] = rendered
	store.files["sources/foo-2026-x.md"] = []byte("---\n\"@id\": foo-2026-x\n\"@type\": source\n---\n# foo-2026-x\n")

	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
		ChangedPathsFn: func(dir, hash string) ([]string, error) {
			return []string{"sources/foo-2026-x.md", "entities/Widget.md"}, nil
		},
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			if path == "sources/foo-2026-x.md" {
				return []string{"ingest123"}, nil
			}
			return []string{"later999", "ingest123"}, nil
		},
		BlameFn:    func(dir, path string) ([]port.BlameLine, error) { return nil, nil },
		CommitHash: "commitGHI",
	}

	result, err := service.Revert(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, "/graph", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, result.Reconciled["entity"]))
	it.Then(t).Should(it.Equal(string(rendered), string(store.files["entities/Widget.md"])))

	var outcome string
	for _, n := range result.Nodes {
		if n.Path == "entities/Widget.md" {
			outcome = n.Kind
		}
	}
	it.Then(t).Should(it.Equal("unchanged", outcome))
}

// research.md D8's two provenance sub-cases (marker-shaped Texts value
// parsing/resolution) are unit-tested directly in revert_internal_test.go
// (package service), not here: a conflict-marker-shaped Texts value does
// not currently survive a core.RenderNode -> core.ParseNode round trip —
// its "=======" delimiter line is parsed as a Markdown setext-heading
// underline for the immediately preceding line, silently dropping the
// "existing" side and both delimiter lines (a pre-existing
// internal/core gap, independent of this feature, out of scope to fix
// here per plan.md's "internal/core stays untouched" constraint) — so no
// black-box fixture reaching this code path through a real
// readExistingNode call is currently constructible.

// localGraphMounter mounts a real, os-backed fsys.Store rooted at dir —
// used only where enumerateNodes' real directory walk matters (backlink
// sweep across multiple files), since the in-package memStore fixture's
// own ReadDir is an always-empty stub.
type localGraphMounter struct{}

func (localGraphMounter) Mount(root string) (fsys.Store, error) { return fsys.Local{}.Mount(root) }

func writeGraphFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
}

// research.md D6: removing an exclusively-owned node sweeps every
// referrer's backlinks in one pass — an ordinary referrer's Edges entry
// and a timeline period's own "cites" bullet alike (the reverse index
// already covers both with no format-specific code, per D6's load-bearing
// discovery).
func TestRevertRemovesExclusiveNodeAndSweepsBacklinksIncludingTimeline(t *testing.T) {
	dir := t.TempDir()
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, ".arc"), 0o755)))
	writeGraphFile(t, dir, "sources/foo-2026-x.md", "---\n\"@id\": foo-2026-x\n\"@type\": source\n---\n# foo-2026-x\n")
	writeGraphFile(t, dir, "entities/OldWidget.md", "---\n\"@id\": OldWidget\n\"@type\": entity\n---\n# OldWidget\n")
	writeGraphFile(t, dir, "entities/Gadget.md", "---\n\"@id\": Gadget\n\"@type\": entity\n---\n# Gadget\n\n- relatesTo:: [[OldWidget]]\n")
	writeGraphFile(t, dir, "timeline/yearly/2026.md", "---\n\"@id\": \"2026\"\n\"@type\": timeline\nperiod: \"2026\"\ngranularity: yearly\n---\n# 2026\n\n- cites:: [[OldWidget]] — OldWidget — 2026-01-01\n")

	vcs := &graphmock.VCS{
		Tracked:           map[string]bool{"sources/foo-2026-x.md": true},
		CommitsMatchingFn: func(dir, needle string) ([]string, error) { return []string{"ingest123"}, nil },
		ChangedPathsFn: func(dir, hash string) ([]string, error) {
			// entities/DummyOther.md never existed on disk in this fixture
			// — it exists purely to make the whole-operation eligibility
			// test (D3) fail, forcing the per-node path (D5/D6) even
			// though both real paths below remain individually exclusive.
			return []string{"sources/foo-2026-x.md", "entities/OldWidget.md", "entities/DummyOther.md"}, nil
		},
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			if path == "entities/DummyOther.md" {
				return []string{"later999", "ingest123"}, nil
			}
			return []string{"ingest123"}, nil
		},
		CommitHash: "commitPQR",
	}

	result, err := service.Revert(context.Background(), localGraphMounter{}, vcs, bios.NewReporter(true, true), coreIndexFixture, dir, "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Removed["source"])).
		Should(it.Equal(1, result.Removed["entity"])).
		Should(it.Equal(2, result.LinksRemoved))

	_, statErr := os.Stat(filepath.Join(dir, "entities", "OldWidget.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	gadget, err := os.ReadFile(filepath.Join(dir, "entities", "Gadget.md"))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(string(gadget)).Contain("OldWidget"))

	yearly, err := os.ReadFile(filepath.Join(dir, "timeline", "yearly", "2026.md"))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(string(yearly)).Contain("OldWidget"))
}
