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
			"timeline/yearly", "timeline/monthly", "_meta",
		)).
		Should(it.Equal(6, len(kernel.DefaultLayout.Folders)))
}

func TestDefaultLayoutMetaStubs(t *testing.T) {
	it.Then(t).
		Should(it.Equal(2, len(kernel.DefaultLayout.MetaStubs)))

	_, hasPredicates := kernel.DefaultLayout.MetaStubs["_meta/predicates.md"]
	_, hasAliases := kernel.DefaultLayout.MetaStubs["_meta/aliases.md"]
	it.Then(t).
		Should(it.True(hasPredicates)).
		Should(it.True(hasAliases))
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
