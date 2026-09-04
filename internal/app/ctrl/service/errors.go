//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import "github.com/fogfish/faults"

const (
	ErrGitUnavailable     = faults.Type("git is required but was not found on PATH")
	ErrAlreadyInitialized = faults.Safe1[string]("%s is already an initialized graph")
	ErrTargetNotEmpty     = faults.Safe1[string]("%s is not empty; arc init requires an empty or non-existent directory")
	ErrLayoutWrite        = faults.Safe1[string]("failed to write graph layout at %s")

	// specs/031-init-existing-git-repo. Wording is part of the contract:
	// FR-006 requires the refusal to name both the enclosing repository
	// and the flag that overrides it, and FR-031 requires a collision to
	// name the conflicting path and the subfolder recovery.
	ErrInsideRepository   = faults.Safe2[string, string]("%s is inside the existing git repository %s; use --skip-git-init to add the graph to that repository instead of nesting a new one inside it")
	ErrNoParentRepository = faults.Safe1[string]("%s is not inside a git repository; --skip-git-init requires an existing repository to add the graph to")
	ErrLayoutCollision    = faults.Safe1[string]("%s already exists; initialize the graph into a subfolder instead")
	ErrTargetIgnored      = faults.Safe1[string]("%s is excluded by the repository's ignore rules; the graph could never be committed")
)
