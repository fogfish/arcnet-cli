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
	ErrNotAGraph = faults.Safe1[string]("%s is not an initialized graph")
	ErrPatchRead = faults.Safe1[string]("failed to read patch file %s")
	ErrNodeWrite = faults.Safe1[string]("failed to write %s")
)
