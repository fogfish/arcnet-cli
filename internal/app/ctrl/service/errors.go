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
