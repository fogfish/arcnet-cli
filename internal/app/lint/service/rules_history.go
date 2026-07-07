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
	"fmt"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/lint/port"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// checkIngestCommit reports a RuleIngestCommit violation when a source
// node's document does not have exactly one graph(ingest): commit carrying
// its Source-Id trailer (research.md D12/spec FR-010).
func checkIngestCommit(ctx context.Context, vcs port.VCS, dir string, node core.Node, path string) ([]kernel.Violation, error) {
	if node.Type != "source" {
		return nil, nil
	}

	hashes, err := vcs.CommitsMatching(ctx, dir, "Source-Id: "+node.ID)
	if err != nil {
		return nil, err
	}
	if len(hashes) == 1 {
		return nil, nil
	}

	return []kernel.Violation{{
		Rule:    kernel.RuleIngestCommit,
		Path:    path,
		Line:    0,
		Message: fmt.Sprintf("%d matching commits found for this document", len(hashes)),
	}}, nil
}
