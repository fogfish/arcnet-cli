//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the graph (graph I/O) domain's value types.
package kernel

import "github.com/fogfish/arcnet-cli/internal/core"

// ApplyResult is the domain value component.go's Apply returns to
// cmd/arc/graph, rendered by bios.Registry[ApplyResult].
type ApplyResult struct {
	// Document is the applied patch's source id.
	Document string `json:"document"`
	// Skipped is true when the idempotency check (FR-003) found the
	// document already tracked; every other field is zero-valued then.
	Skipped bool `json:"skipped"`
	// Created holds node counts by kind, newly created.
	Created map[core.Kind]int `json:"created"`
	// Merged holds node counts by kind, merged into existing nodes.
	Merged map[core.Kind]int `json:"merged"`
	// Conflicts holds relative paths of node files that received a
	// conflict marker (FR-013), for the PostRunE hint.
	Conflicts []string `json:"conflicts"`
	// Warnings holds one sentence per node whose kind was not found in
	// the resolved MergeRuleSet and was therefore applied using the
	// default union behavior (spec FR-018); empty when every kind was
	// recognized.
	Warnings []string `json:"warnings"`
	// CommitHash is the short hash of the single resulting commit; empty
	// when Skipped.
	CommitHash string `json:"commit"`
	// Timeline holds period codes touched ("2026", "2026-04").
	Timeline []string `json:"timeline"`
}
