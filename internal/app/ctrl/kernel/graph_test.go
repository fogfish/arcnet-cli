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
	"sort"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	schemakernel "github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
)

func TestDefaultLayoutFolders(t *testing.T) {
	it.Then(t).
		Should(it.Seq(kernel.DefaultLayout.Folders).Contain(
			"Source", "Entity", "Resource", "Reference",
			"timeline/yearly", "timeline/monthly",
			"_schema/Class", "_schema/Property",
		)).
		Should(it.Equal(8, len(kernel.DefaultLayout.Folders)))
}

func TestDefaultLayoutSeedFilesEmptyByDefault(t *testing.T) {
	it.Then(t).Should(it.Equal(0, len(kernel.DefaultLayout.SeedFiles)))
}

func TestInitResultJSONShape(t *testing.T) {
	result := kernel.InitResult{
		Root:           kernel.GraphRoot{Root: "/tmp/my-graph"},
		CommitHash:     "a1b2c3d",
		FoldersCreated: []string{"Source", "_meta"},
	}

	it.Then(t).Should(it.Json(result).Equiv(`{
		"path": "/tmp/my-graph",
		"commit": "a1b2c3d",
		"foldersCreated": ["Source", "_meta"]
	}`))

	b, err := json.Marshal(result)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(string(b)).Contain(`"path":"/tmp/my-graph"`))
}

// ---------------------------------------------------------------------------
// specs/022-reference-type-folders — ARCNET-CORE v0.11
// ---------------------------------------------------------------------------

// contentTypeNames returns every seeded content type that owns a flat type
// folder. Timeline is excluded (contract C2: its nodes are bucketed under
// timeline/, never filed at Timeline/<id>.md); CoreTypeBases already
// excludes the schema-mechanism types Property/Class and the abstract base
// Node, none of which is ever a node's own @type.
func contentTypeNames() []string {
	out := make([]string, 0, len(schemakernel.CoreTypeBases))
	for name := range schemakernel.CoreTypeBases {
		if name == "Timeline" {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// TestDefaultLayoutAgreesWithNodeFolderDerivation enforces contract C6, the
// drift hazard research.md D3 identified: arc init creates folders from this
// static list while arc apply writes to a path graph/service.nodeFolder
// computes, and nothing else ties the two together. Because fsys creates
// missing parent directories on write, a disagreement raises no error at
// runtime — it surfaces only as a graph carrying two parallel folder sets.
//
// The import direction matters. ctrl/kernel is a leaf value package and must
// not depend on graph/service, so this test cannot call nodeFolder. Under
// contract C1 nodeFolder is the identity function, so asserting against the
// type name itself asserts against nodeFolder's result. The reciprocal
// half — that nodeFolder really is the identity — is asserted beside the
// function, in graph/service's own apply_test.go.
func TestDefaultLayoutAgreesWithNodeFolderDerivation(t *testing.T) {
	folders := make(map[string]bool, len(kernel.DefaultLayout.Folders))
	for _, folder := range kernel.DefaultLayout.Folders {
		folders[folder] = true
	}

	types := make(map[string]bool, len(schemakernel.CoreTypeBases))
	for _, name := range contentTypeNames() {
		types[name] = true
		// Every content type has a folder named for it, verbatim.
		it.Then(t).Should(it.True(folders[name]))
	}

	// And every non-functional folder belongs to some content type — no
	// orphan folder, and none surviving under a retired name.
	for _, folder := range kernel.DefaultLayout.Folders {
		if strings.HasPrefix(folder, "timeline/") || strings.HasPrefix(folder, "_schema/") {
			continue
		}
		it.Then(t).Should(it.True(types[folder]))
	}
}

// TestDefaultLayoutSchemaFoldersMatchSchemaPathConstants pins the other half
// of contract C2: _schema/ is a namespace prefix whose two children are
// themselves type folders. The layout must name the very paths schema/kernel
// seeds into and resolves from, or arc init would create one pair of folders
// and the schema loader would look in another.
func TestDefaultLayoutSchemaFoldersMatchSchemaPathConstants(t *testing.T) {
	it.Then(t).Should(it.Seq(kernel.DefaultLayout.Folders).Contain(
		schemakernel.PredicatesDir,
		schemakernel.TypesDir,
	))
}
