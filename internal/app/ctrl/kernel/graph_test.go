//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel_test

import (
	"encoding/json"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
)

func TestDefaultLayoutFolders(t *testing.T) {
	it.Then(t).
		Should(it.Seq(kernel.DefaultLayout.Folders).Contain(
			"sources", "entities", "resources",
			"timeline/yearly", "timeline/monthly",
			"_schema/nodes", "_schema/predicates",
		)).
		Should(it.Equal(7, len(kernel.DefaultLayout.Folders)))
}

func TestDefaultLayoutSeedFilesEmptyByDefault(t *testing.T) {
	it.Then(t).Should(it.Equal(0, len(kernel.DefaultLayout.SeedFiles)))
}

func TestInitResultJSONShape(t *testing.T) {
	result := kernel.InitResult{
		Root:           kernel.GraphRoot{Root: "/tmp/my-graph"},
		CommitHash:     "a1b2c3d",
		FoldersCreated: []string{"sources", "_meta"},
	}

	it.Then(t).Should(it.Json(result).Equiv(`{
		"path": "/tmp/my-graph",
		"commit": "a1b2c3d",
		"foldersCreated": ["sources", "_meta"]
	}`))

	b, err := json.Marshal(result)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(string(b)).Contain(`"path":"/tmp/my-graph"`))
}
