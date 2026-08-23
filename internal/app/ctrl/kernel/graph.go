//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the ctrl (graph management) domain's value types.
package kernel

import "encoding/json"

// GraphRoot represents the resolved location of a graph, before or after
// initialization.
type GraphRoot struct {
	Root string
}

func (g GraphRoot) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.Root)
}

// ArcNetCoreLayout is a static, pure description of what an empty graph
// must contain. Not user-configurable in this feature.
type ArcNetCoreLayout struct {
	Folders   []string
	SeedFiles map[string]string
}

// DefaultLayout is the canonical eight-folder layout arc init creates
// (ARCNET-CORE §6 v0.11, specs/022-reference-type-folders contract C3): four
// type folders each named for its type character for character, the two
// timeline buckets, and the two _schema/ type folders.
//
// The four content-folder names must stay in agreement with
// graph/service.nodeFolder, which is what arc apply writes through. Nothing
// ties the two together at compile time and a disagreement raises no error
// at runtime — fsys creates missing parent directories on write, so the
// wrong folder would simply appear on first use, leaving a graph with two
// parallel folder sets. ctrl/kernel's own graph_test.go enforces the
// agreement as contract C6.
var DefaultLayout = ArcNetCoreLayout{
	Folders: []string{
		"Source",
		"Entity",
		"Resource",
		"Reference",
		"timeline/yearly",
		"timeline/monthly",
		"_schema/Class",
		"_schema/Property",
	},
	SeedFiles: map[string]string{},
}

// InitResult is the domain value component.go's Init returns to
// cmd/arc/ctrl, rendered by the bios.Registry[InitResult].
type InitResult struct {
	Root           GraphRoot `json:"path"`
	CommitHash     string    `json:"commit"`
	FoldersCreated []string  `json:"foldersCreated"`
}

// UpgradeResult is the domain value component.go's Upgrade returns to
// cmd/arc/ctrl, rendered by the bios.Registry[UpgradeResult]. It mirrors
// InitResult's shape and JSON tag conventions — Root under "path",
// CommitHash under "commit" — so the two control-plane commands present
// the same vocabulary to a --json consumer
// (specs/023-core-vocabulary-conformance data-model.md §6).
type UpgradeResult struct {
	Root       GraphRoot `json:"path"`
	CommitHash string    `json:"commit"`
	// Replaced/Added/Removed are built-in schema document paths relative
	// to the graph root, sorted. Removed covers a built-in document that
	// exists on disk under a name this release no longer seeds.
	Replaced []string `json:"replaced"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	// NeedsReview lists content-node paths whose firstWriteWin-declared
	// text predicate holds more than one paragraph — the shape only the
	// previous append behaviour could have produced (research.md D12).
	// Advisory: it never affects the exit code and is never repaired.
	NeedsReview []string `json:"needsReview"`
	// DryRun records that the run reported without writing.
	DryRun bool `json:"dryRun"`
}
