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

// InitOpts carries the repository facts cmd/arc/ctrl resolves through the
// git adapter before calling the service, plus the user's own flag. They
// are passed as plain values rather than probed from inside the service
// because detection MUST happen before the target directory exists
// (research.md D2, D6) — and because every DECISION taken from them still
// belongs to the service.
type InitOpts struct {
	// ParentRepo is the root of the innermost repository enclosing the
	// target, expressed in the same path spelling as the target itself so
	// the two are directly comparable. Empty means no enclosing
	// repository was found.
	ParentRepo string

	// SkipGitInit is the --skip-git-init flag as given by the user.
	SkipGitInit bool

	// TargetIgnored reports whether the parent repository's ignore rules
	// exclude the target. Meaningful only when ParentRepo is non-empty.
	TargetIgnored bool
}

// InitResult is the domain value component.go's Init returns to
// cmd/arc/ctrl, rendered by the bios.Registry[InitResult].
type InitResult struct {
	Root           GraphRoot `json:"path"`
	CommitHash     string    `json:"commit"`
	FoldersCreated []string  `json:"foldersCreated"`

	// Repository is the root of the repository the initial commit landed
	// in, populated in BOTH modes and never empty (FR-027): the graph root
	// itself for a standalone init, an ancestor for --skip-git-init. A
	// consumer distinguishes the two by comparing it to Root — no mode
	// boolean is emitted (FR-028).
	Repository string `json:"repository"`
}
