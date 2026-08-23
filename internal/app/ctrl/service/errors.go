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
)

const (
	// ErrNotInitialized is returned by Upgrade when the target directory
	// carries no ".arc/" marker — it is not a graph, so there is no
	// built-in vocabulary to bring up to date.
	ErrNotInitialized = faults.Safe1[string](`%s is not an initialized graph, run "arc init" first`)

	// ErrUpgradeRead is returned when an existing built-in schema document
	// or content node cannot be read; names the offending path.
	ErrUpgradeRead = faults.Safe1[string]("failed to read %s")

	// ErrUpgradeWrite is returned when a built-in schema document cannot be
	// replaced or removed; names the offending path.
	ErrUpgradeWrite = faults.Safe1[string]("failed to update schema document %s")

	// ErrUpgradeUnresolvable is returned when the freshly written built-in
	// vocabulary does not itself resolve. Seed() output is always
	// resolvable, so this is a programming error rather than a condition a
	// user's graph can cause (contract C3.8).
	ErrUpgradeUnresolvable = faults.Type("upgraded schema failed to resolve — this is a bug in arc, please report it")
)
