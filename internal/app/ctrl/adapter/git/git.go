//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package git is the os/exec-backed real implementation of port.VCS.
package git

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/fogfish/faults"

	"github.com/fogfish/arcnet-cli/internal/bios"
)

const (
	ErrGitNotFound = faults.Type("git binary not found on PATH")
	ErrGitInit     = faults.Type("git init failed")
	ErrGitStage    = faults.Type("git add failed")
	ErrGitCommit   = faults.Type("git commit failed")
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
