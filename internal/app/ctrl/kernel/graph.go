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

var DefaultLayout = ArcNetCoreLayout{
	Folders: []string{
		"sources",
		"entities",
		"resources",
		"timeline/yearly",
		"timeline/monthly",
		"_schema/types",
		"_schema/predicates",
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
