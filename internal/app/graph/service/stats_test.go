//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// statsStore is a read-only fsys.Store backed by fstest.MapFS. arc stats
// never writes, so a store whose Create/Remove always fail is a faithful
// stand-in — and a failing test would be the first sign the read-only
// guarantee (spec FR-023) had been broken.
type statsStore struct{ fstest.MapFS }

func (statsStore) Create(name string) (fsys.File, error) { return nil, errors.New("read-only store") }
func (statsStore) Remove(name string) error              { return errors.New("read-only store") }

type statsMounter struct{ store statsStore }

func (m statsMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

func statsBare(files map[string]string) statsMounter {
	fs := fstest.MapFS{}
	for path, content := range files {
		fs[path] = &fstest.MapFile{Data: []byte(content)}
	}
	return statsMounter{store: statsStore{fs}}
}

// statsFake builds an initialized in-memory graph holding files.
func statsFake(files map[string]string) statsMounter {
	out := statsBare(files)
	out.store.MapFS[".arc/.gitkeep"] = &fstest.MapFile{}
	return out
}

func statsOver(t *testing.T, files map[string]string, detail bool) (int, int, int) {
	t.Helper()
	r, err := service.Stats(context.Background(), statsFake(files), core.Index{}, "/graph", detail)
	it.Then(t).Must(it.Nil(err))
	return r.Nodes, r.Edges, r.BrokenLinks
}

const statsSchemaClass = `---
"@id": "Entity"
"@type": Class
---
# Entity

A declared type.
`

const statsSchemaProperty = `---
"@id": "cites"
"@type": Property
role: edge
merge: union
---
# cites

A declared predicate.
`

// The census counts schema documents; the content population, which every
// link figure is computed over, does not (research.md D1).
func TestStatsSplitsCensusFromContentPopulation(t *testing.T) {
	nodes, edges, broken := statsOver(t, map[string]string{
		"_schema/Class/Entity.md":   statsSchemaClass,
		"_schema/Property/cites.md": statsSchemaProperty,
		"Entity/A.md":               "---\n\"@id\": \"A\"\n\"@type\": Entity\n---\n# A\n\n- cites:: [[cites]]\n",
	}, false)

	it.Then(t).
		Should(it.Equal(3, nodes)).
		Should(it.Equal(1, edges)).
		Should(it.Equal(1, broken))
}

// A schema document is recognized by its declared type, so filing one
// outside _schema/ changes nothing (spec FR-004a).
func TestStatsRecognizesSchemaByDeclaredType(t *testing.T) {
	nodes, _, _ := statsOver(t, map[string]string{
		"vocabulary/Entity.md": statsSchemaClass,
		"Entity/A.md":          "---\n\"@id\": \"A\"\n\"@type\": Entity\n---\n# A\n\nProse.\n",
	}, false)

	it.Then(t).Should(it.Equal(2, nodes))
}

// A distinct unresolved target counts once per referencing node, however
// many times that node repeats it (spec FR-006).
func TestStatsCountsUnresolvedTargetOncePerNode(t *testing.T) {
	_, _, broken := statsOver(t, map[string]string{
		"Entity/A.md": "---\n\"@id\": \"A\"\n\"@type\": Entity\n---\n# A\n\nSees [[X]] and [[X]].\n\n- cites:: [[X]]\n- cites:: [[X]]\n",
		"Entity/B.md": "---\n\"@id\": \"B\"\n\"@type\": Entity\n---\n# B\n\n- cites:: [[X]]\n",
	}, false)

	it.Then(t).Should(it.Equal(2, broken))
}

// A Timeline node's period is its own id, and the granularity that id
// declares decides which breakdown it lands in (research.md D5).
func TestStatsReadsPeriodFromNodeIdentity(t *testing.T) {
	store := statsFake(map[string]string{
		"anywhere/2026.md":       "---\n\"@id\": \"2026\"\n\"@type\": Timeline\n---\n# 2026\n\n- cites:: [[a]]\n- cites:: [[b]]\n",
		"elsewhere/2026-04.md":   "---\n\"@id\": \"2026-04\"\n\"@type\": Timeline\n---\n# 2026-04\n\n- cites:: [[a]]\n",
		"elsewhere/quarterly.md": "---\n\"@id\": \"quarterly\"\n\"@type\": Timeline\n---\n# quarterly\n\nNot a period code.\n",
	})

	r, err := service.Stats(context.Background(), store, core.Index{}, "/graph", true)
	it.Then(t).Must(it.Nil(err))

	it.Then(t).
		Should(it.Equal(1, len(r.Ingestion))).
		Should(it.Equal("2026", r.Ingestion[0].Period)).
		Should(it.Equal(2, r.Ingestion[0].Count)).
		Should(it.Equal(1, len(r.Detail.IngestionMonthly))).
		Should(it.Equal("2026-04", r.Detail.IngestionMonthly[0].Period))
}

// An even population takes the mean of the two central degrees; zero-degree
// nodes stay in the denominator (research.md D9).
func TestStatsMedianOfEvenPopulation(t *testing.T) {
	store := statsFake(map[string]string{
		"Entity/A.md": "---\n\"@id\": \"A\"\n\"@type\": Entity\n---\n# A\n\nProse.\n",
		"Entity/B.md": "---\n\"@id\": \"B\"\n\"@type\": Entity\n---\n# B\n\n- cites:: [[A]]\n",
		"Entity/C.md": "---\n\"@id\": \"C\"\n\"@type\": Entity\n---\n# C\n\n- cites:: [[A]]\n- cites:: [[B]]\n",
		"Entity/D.md": "---\n\"@id\": \"D\"\n\"@type\": Entity\n---\n# D\n\n- cites:: [[A]]\n- cites:: [[B]]\n- cites:: [[C]]\n",
	})

	r, err := service.Stats(context.Background(), store, core.Index{}, "/graph", true)
	it.Then(t).Must(it.Nil(err))

	it.Then(t).
		Should(it.Equal(1.5, r.Detail.MedianOutDegree)).
		Should(it.Equal(1.5, r.Detail.AvgOutDegree))
}

// The hub ranking is capped at ten and breaks ties by ascending id, so a
// graph with many equally-referenced nodes still ranks deterministically
// (spec FR-015, FR-019).
func TestStatsTopReferencedCapsAndBreaksTies(t *testing.T) {
	files := map[string]string{}
	body := "---\n\"@id\": \"hub\"\n\"@type\": Entity\n---\n# hub\n"
	for i := 0; i < 12; i++ {
		id := string(rune('a' + i))
		files["Entity/"+id+".md"] = "---\n\"@id\": \"" + id + "\"\n\"@type\": Entity\n---\n# " + id + "\n"
		body += "- cites:: [[" + id + "]]\n"
	}
	files["Entity/hub.md"] = body

	r, err := service.Stats(context.Background(), statsFake(files), core.Index{}, "/graph", true)
	it.Then(t).Must(it.Nil(err))

	ids := make([]string, 0, len(r.Detail.TopReferenced))
	for _, e := range r.Detail.TopReferenced {
		ids = append(ids, e.ID)
	}
	it.Then(t).Should(it.Seq(ids).Equal("a", "b", "c", "d", "e", "f", "g", "h", "i", "j"))
}

// Verbosity gates computation, not rendering: a default run produces no
// detail section at all (spec FR-018).
func TestStatsOmitsDetailUnlessAsked(t *testing.T) {
	store := statsFake(map[string]string{
		"Entity/A.md": "---\n\"@id\": \"A\"\n\"@type\": Entity\n---\n# A\n\nProse.\n",
	})

	plain, err := service.Stats(context.Background(), store, core.Index{}, "/graph", false)
	it.Then(t).Must(it.Nil(err))
	verbose, err := service.Stats(context.Background(), store, core.Index{}, "/graph", true)
	it.Then(t).Must(it.Nil(err))

	it.Then(t).
		Should(it.True(plain.Detail == nil)).
		Should(it.True(verbose.Detail != nil))
}

// A directory carrying no .arc/ is refused through the shared sentinel
// every graph command uses (spec FR-011).
func TestStatsRefusesDirectoryThatIsNotAGraph(t *testing.T) {
	mounter := statsBare(map[string]string{"notes.md": "# Notes\n"})

	_, err := service.Stats(context.Background(), mounter, core.Index{}, "/graph", false)

	it.Then(t).Should(it.Fail(func() error { return err }).Contain("is not an initialized graph"))
}

// A malformed node is recorded and the scan continues; host markdown is
// recorded apart from it and counted in nothing (spec FR-009, SC-006).
func TestStatsSeparatesUnreadableFromForeignFiles(t *testing.T) {
	store := statsFake(map[string]string{
		"Entity/Good.md":   "---\n\"@id\": \"Good\"\n\"@type\": Entity\n---\n# Good\n\nProse.\n",
		"Entity/Broken.md": "---\n\"@id\": \"Broken\n\"@type\": [Entity\n  bad: : :\n---\n# Broken\n",
		"README.md":        "# Host project\n\nOrdinary host markdown.\n",
	})

	r, err := service.Stats(context.Background(), store, core.Index{}, "/graph", false)
	it.Then(t).Must(it.Nil(err))

	it.Then(t).
		Should(it.Equal(1, r.Nodes)).
		Should(it.Seq(r.Unreadable).Equal("Entity/Broken.md")).
		Should(it.Seq(r.Foreign).Equal("README.md"))
}

// Every published date the fixture omits is counted, and none of the
// figures depend on a node having one (spec FR-017).
func TestStatsCountsNodesWithoutPublicationDate(t *testing.T) {
	store := statsFake(map[string]string{
		"Source/dated.md":   "---\n\"@id\": \"dated\"\n\"@type\": Source\npublished: " + time.Now().Format("2006-01-02") + "\n---\n# dated\n\nProse.\n",
		"Source/undated.md": "---\n\"@id\": \"undated\"\n\"@type\": Source\n---\n# undated\n\nProse.\n",
	})

	r, err := service.Stats(context.Background(), store, core.Index{}, "/graph", true)
	it.Then(t).Must(it.Nil(err))

	it.Then(t).Should(it.Equal(1, r.Detail.Content.NodesWithoutPublished))
}
