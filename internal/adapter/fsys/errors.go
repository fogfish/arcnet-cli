//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package fsys

import "github.com/fogfish/faults"

const (
	ErrRootNotDirectory = faults.Safe1[string]("%s is not a directory")
	ErrRootCreate       = faults.Safe1[string]("failed to create graph root at %s")
	ErrCreate           = faults.Safe1[string]("failed to create %s")
	ErrRemove           = faults.Safe1[string]("failed to remove %s")
)
