//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package git is the shared, cross-use-case os/exec-backed git adapter
// (ADR 001's application-level adapter tier, promoted from
// internal/app/ctrl/adapter/git per research.md D4, mirroring
// internal/adapter/fsys's existing precedent). Its one concrete Git type
// satisfies both internal/app/ctrl/port.VCS and internal/app/graph/port.VCS
// structurally (ADR 001 port isolation rule 1) — no vendor/subprocess types
// (exec.Cmd, exec.ExitError) leak through either.
package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fogfish/faults"

	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

const (
	ErrGitNotFound = faults.Type("git binary not found on PATH")
	ErrGitInit     = faults.Type("git init failed")
	ErrGitStage    = faults.Type("git add failed")
	ErrGitCommit   = faults.Type("git commit failed")
	ErrGitLsFiles  = faults.Type("git ls-files failed")
	ErrGitLog      = faults.Type("git log failed")
	ErrGitDiffTree = faults.Type("git diff-tree failed")
	ErrGitRevert   = faults.Type("git revert failed")
	ErrGitBlame    = faults.Type("git blame failed")
	ErrGitShow     = faults.Type("git show failed")
)

// execError captures combined stdout+stderr from a failed git subprocess
// invocation, for inclusion in the wrapped error message.
type execError struct {
	output string
	err    error
}

func (e execError) Error() string {
	output := strings.TrimSpace(e.output)
	if output == "" {
		return e.err.Error()
	}
	return output + ": " + e.err.Error()
}

func (e execError) Unwrap() error { return e.err }

// VCS backs every port.VCS operation with a real git subprocess, reporting
// completion of each step through bios.Reporter. Reporter output is a
// --verbose-only affair (the caller chooses a silentReporter by default,
// see cmd/arc/ctrl/init.go) consolidated to three steps — availability,
// preparing the repository, and committing — not one per subprocess call
// (BUG-001, research.md D2 Bugfix): StageAll intentionally reports nothing
// of its own, since it is part of "preparing" the repository that Init
// already reports.
type VCS struct {
	Reporter bios.Reporter
}

func New(reporter bios.Reporter) VCS {
	return VCS{Reporter: reporter}
}

func (v VCS) IsAvailable(ctx context.Context) error {
	const label = "Checking git availability"
	start := time.Now()

	if _, err := run(ctx, "", "--version"); err != nil {
		v.Reporter.Error(label, err)
		return ErrGitNotFound.With(err)
	}

	v.Reporter.Done(label, time.Since(start))
	return nil
}

func (v VCS) Init(ctx context.Context, dir string) error {
	const label = "Preparing git repository"
	start := time.Now()

	if _, err := run(ctx, dir, "init"); err != nil {
		v.Reporter.Error(label, err)
		return ErrGitInit.With(err)
	}

	v.Reporter.Done(label, time.Since(start))
	return nil
}

func (v VCS) StageAll(ctx context.Context, dir string) error {
	if _, err := run(ctx, dir, "add", "-A"); err != nil {
		return ErrGitStage.With(err)
	}
	return nil
}

func (v VCS) Commit(ctx context.Context, dir, message string) (string, error) {
	const label = "Committing empty graph"
	start := time.Now()

	if _, err := run(ctx, dir, "commit", "-m", message); err != nil {
		v.Reporter.Error(label, err)
		return "", ErrGitCommit.With(err)
	}

	out, err := run(ctx, dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		v.Reporter.Error(label, err)
		return "", ErrGitCommit.With(err)
	}

	v.Reporter.Done(label, time.Since(start))
	return strings.TrimSpace(string(out)), nil
}

// IsTracked reports whether path is already tracked by git in dir (CORE
// §11.2's documented idempotency check), via `git ls-files
// --error-unmatch <path>`. Exit 0 means tracked; git's own "not tracked"
// exit status for --error-unmatch is an expected outcome, not an error, and
// is reported as (false, nil) — only a genuine unexpected failure (git
// missing, dir not a repository) is returned as (false, err).
func (v VCS) IsTracked(ctx context.Context, dir, path string) (bool, error) {
	_, err := run(ctx, dir, "ls-files", "--error-unmatch", path)
	if err == nil {
		return true, nil
	}

	var eerr execError
	if errors.As(err, &eerr) {
		var exitErr *exec.ExitError
		if errors.As(eerr.err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	return false, ErrGitLsFiles.With(err)
}

// CommitsMatching returns the hashes of every commit reachable from any ref
// (`--all`) whose message contains needle, matched literally (`--fixed-
// strings`, so a citekey containing regex metacharacters is never
// misinterpreted as a pattern) — internal/app/lint's CORE §11.1 "one
// ingest commit per document" check (research.md D12).
func (v VCS) CommitsMatching(ctx context.Context, dir, needle string) ([]string, error) {
	out, err := run(ctx, dir, "log", "--all", "--fixed-strings", "--grep="+needle, "--format=%H")
	if err != nil {
		return nil, ErrGitLog.With(err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// ChangedPaths lists every path hash's commit touched (`git diff-tree
// --no-commit-id --name-only -r --root <hash>`) — root-commit-safe: `--root`
// makes diff-tree diff a root commit against the empty tree (its default
// behavior for a root commit is to show nothing at all) while remaining a
// no-op for any non-root commit, so one invocation is correct either way
// unlike a plain two-dot `git diff` (research.md D3).
func (v VCS) ChangedPaths(ctx context.Context, dir, hash string) ([]string, error) {
	out, err := run(ctx, dir, "diff-tree", "--no-commit-id", "--name-only", "-r", "--root", hash)
	if err != nil {
		return nil, ErrGitDiffTree.With(err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// CommitsTouching returns every commit that ever changed path, newest
// first (`git log --follow --format=%H -- <path>`) — the single primitive
// both whole-operation eligibility (D3) and per-node exclusivity (D5) are
// expressed in terms of. `--follow` matters for a node file that was ever
// renamed.
func (v VCS) CommitsTouching(ctx context.Context, dir, path string) ([]string, error) {
	out, err := run(ctx, dir, "log", "--follow", "--format=%H", "--", path)
	if err != nil {
		return nil, ErrGitLog.With(err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// RevertCommit reverts hash (`git revert --no-edit <hash>`) then resolves
// the resulting commit's short hash, mirroring Commit's own commit +
// rev-parse pattern (research.md D4).
func (v VCS) RevertCommit(ctx context.Context, dir, hash string) (string, error) {
	if _, err := run(ctx, dir, "revert", "--no-edit", hash); err != nil {
		return "", ErrGitRevert.With(err)
	}

	out, err := run(ctx, dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", ErrGitRevert.With(err)
	}

	return strings.TrimSpace(string(out)), nil
}

// blameHeaderPattern matches a `git blame --line-porcelain` header line
// (`<sha1> <orig-line> <final-line> [<num-lines>]`) — the only line shape
// in that output starting with a bare 40-character hex commit hash.
var blameHeaderPattern = regexp.MustCompile(`^([0-9a-f]{40}) \d+ (\d+)(?: \d+)?$`)

// Blame returns one port.BlameLine per current line of path (`git blame
// --line-porcelain HEAD -- <path>`) — content itself is never needed, only
// which commit last touched each final line number (research.md D7).
func (v VCS) Blame(ctx context.Context, dir, path string) ([]port.BlameLine, error) {
	out, err := run(ctx, dir, "blame", "--line-porcelain", "HEAD", "--", path)
	if err != nil {
		return nil, ErrGitBlame.With(err)
	}

	var lines []port.BlameLine
	for _, line := range strings.Split(string(out), "\n") {
		m := blameHeaderPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		lines = append(lines, port.BlameLine{Number: n, Commit: m[1]})
	}
	return lines, nil
}

// showFileMissingMarkers are the two "path did not exist at this commit"
// message shapes `git show <hash>:<path>` produces — a normal, expected
// outcome (contracts/vcs-port-contract.md), never a fatal error.
var showFileMissingMarkers = []string{"does not exist in", "exists on disk, but not in"}

// ShowFile returns path's raw bytes as they existed at hash (`git show
// <hash>:<path>`). A path absent from the tree at hash is not an error —
// it returns (nil, nil), the same shape IsTracked already uses to
// distinguish an expected "not tracked" exit from a genuine failure.
func (v VCS) ShowFile(ctx context.Context, dir, hash, path string) ([]byte, error) {
	out, err := run(ctx, dir, "show", hash+":"+path)
	if err == nil {
		return out, nil
	}

	var eerr execError
	if errors.As(err, &eerr) {
		for _, marker := range showFileMissingMarkers {
			if strings.Contains(eerr.output, marker) {
				return nil, nil
			}
		}
	}

	return nil, ErrGitShow.With(err)
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return nil, execError{output: buf.String(), err: err}
	}

	return buf.Bytes(), nil
}
