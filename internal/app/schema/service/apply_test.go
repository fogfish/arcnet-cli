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
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	schemamock "github.com/fogfish/arcnet-cli/internal/app/schema/adapter/mock"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/schema/service"
)

type fakeMounter struct{ store *fakeStore }

func (m fakeMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

// fakeApplyReporter records every Step call, mirroring graph/service's own
// fakeReporter precedent.
type fakeApplyReporter struct{ steps []string }

func (r *fakeApplyReporter) Start(string)               {}
func (r *fakeApplyReporter) Step(label string)          { r.steps = append(r.steps, label) }
func (r *fakeApplyReporter) Done(string, time.Duration) {}
func (r *fakeApplyReporter) Error(string, error)        {}

const propertyPatch = `---
kind: patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

The item's weight in kilograms.
`

const propertyPatchUpdated = `---
kind: patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

Measured in kilograms (SI).
`

const mixedSchemaPatch = `---
kind: patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

The item's weight in kilograms.

# Class

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Class
merge: union
` + "```" + `

A physical item tracked by the Acme extension.

- required:: [[acmeWeight]]
`

const mixedValidInvalidSchemaPatch = `---
kind: patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

The item's weight in kilograms.

# Class

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Class
merge: union
` + "```" + `

A physical item tracked by the Acme extension.

# entity

## Acme Corp
` + "```yaml" + `
"@id": "Acme Corp"
"@type": entity
category: [independent, social, continuant, collection]
` + "```" + `

The company behind the extension.
`

const entityOnlyPatch = `---
kind: patch
document: acme-corp-note
published: 2026-07-15
title: Acme Corp
---
# entity

## Acme Corp
` + "```yaml" + `
"@id": "Acme Corp"
"@type": entity
category: [independent, social, continuant, collection]
` + "```" + `

The company behind the extension.
`

// spec 018 US1 Acceptance Scenarios 1-3: a Property-only, Class-only, and
// mixed patch each create the corresponding schema definitions.
func TestApplyPatchCreatesPredicateAndTypeFromMixedPatch(t *testing.T) {
	store := newSeededStore()
	store.files["patch.md"] = mixedSchemaPatch
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{CommitHash: "abc123"}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	result, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["predicate"])).
		Should(it.Equal(1, result.Created["type"])).
		Should(it.Equal("abc123", result.CommitHash))

	it.Then(t).
		Should(it.String(store.written[kernel.PredicatesDir+"/acmeWeight.md"]).Contain("meta")).
		Should(it.String(store.written[kernel.TypesDir+"/Widget.md"]).Contain("union"))
	it.Then(t).
		Should(it.Seq(vcs.Calls).Contain("StageAll:/graph")).
		Should(it.Equal(2, len(vcs.Calls))).
		Should(it.True(strings.HasPrefix(vcs.Calls[1], "Commit:/graph:schema(apply):")))
}

// spec 018 US3 Acceptance Scenario 1: re-applying with a changed field
// merges it into the existing predicate per its declared merge behavior
// (description: append) rather than duplicating a new document.
func TestApplyPatchReappliedWithChangedFieldMergesPredicate(t *testing.T) {
	store := newSeededStore()
	store.files["patch.md"] = propertyPatch
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{CommitHash: "abc123"}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	_, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")
	it.Then(t).Should(it.Nil(err))

	store.files["patch.md"] = propertyPatchUpdated
	result, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, result.Created["predicate"])).
		Should(it.Equal(1, result.Merged["predicate"])).
		ShouldNot(it.Equal("", result.CommitHash))
	it.Then(t).
		Should(it.String(store.written[kernel.PredicatesDir+"/acmeWeight.md"]).Contain("kilograms.")).
		Should(it.String(store.written[kernel.PredicatesDir+"/acmeWeight.md"]).Contain("Measured in kilograms (SI)."))
}

// spec 018 US3 Acceptance Scenario 2: an unchanged re-apply reports zero
// created/merged and produces no commit.
func TestApplyPatchNoOpReappliedReportsZeroAndNoCommit(t *testing.T) {
	store := newSeededStore()
	store.files["patch.md"] = propertyPatch
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{CommitHash: "abc123"}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	_, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")
	it.Then(t).Should(it.Nil(err))
	callsAfterFirst := len(vcs.Calls)

	result, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, result.Created["predicate"])).
		Should(it.Equal(0, result.Merged["predicate"])).
		Should(it.Equal("", result.CommitHash))
	it.Then(t).Should(it.Equal(callsAfterFirst, len(vcs.Calls)))
}

// spec 018 US2 Acceptance Scenario 1: a disallowed node type fails the
// operation, naming the offending node's id/type, with zero writes.
func TestApplyPatchRejectsDisallowedNodeTypeWithZeroWrites(t *testing.T) {
	store := newSeededStore()
	store.files["patch.md"] = entityOnlyPatch
	writtenBefore := len(store.written)
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	_, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).
		Should(it.String(err.Error()).Contain("Acme Corp")).
		Should(it.String(err.Error()).Contain("entity"))
	it.Then(t).Should(it.Equal(writtenBefore, len(store.written)))
}

// spec 018 US2 Acceptance Scenario 2: a patch mixing valid Property/Class
// sections with one disallowed section writes none of it — not even the
// otherwise-valid sections.
func TestApplyPatchRejectsMixedValidAndInvalidWithZeroWrites(t *testing.T) {
	store := newSeededStore()
	store.files["patch.md"] = mixedValidInvalidSchemaPatch
	writtenBefore := len(store.written)
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	_, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Equal(writtenBefore, len(store.written)))
	it.Then(t).Should(it.Equal(0, len(vcs.Calls)))
}

// research.md D1: an http(s) source is dispatched through port.Fetcher,
// never mounted as a local file.
func TestApplyPatchFetchesFromURLSource(t *testing.T) {
	store := newSeededStore()
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{CommitHash: "abc123"}
	fetcher := &schemamock.Fetcher{Body: []byte(propertyPatch)}
	reporter := &fakeApplyReporter{}

	result, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "https://example.org/x.schema.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["predicate"])).
		Should(it.Equal("https://example.org/x.schema.md", result.Source))
	it.Then(t).Should(it.Seq(fetcher.Calls).Contain("https://example.org/x.schema.md"))
}

// research.md D1a: an arcnet: reference resolves against
// kernel.ArcnetCatalogBaseURL before being fetched.
func TestApplyPatchArcnetPrefixResolvesAgainstCatalogBaseURL(t *testing.T) {
	original := kernel.ArcnetCatalogBaseURL
	kernel.ArcnetCatalogBaseURL = "https://example.org/catalog/"
	t.Cleanup(func() { kernel.ArcnetCatalogBaseURL = original })

	store := newSeededStore()
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{CommitHash: "abc123"}
	fetcher := &schemamock.Fetcher{Body: []byte(propertyPatch)}
	reporter := &fakeApplyReporter{}

	result, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "arcnet:media.schema.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("arcnet:media.schema.md", result.Source))
	it.Then(t).Should(it.Seq(fetcher.Calls).Contain("https://example.org/catalog/media.schema.md"))
}

// spec FR-002a edge case: a bare "arcnet:" reference is rejected before any
// fetch attempt.
func TestApplyPatchEmptyArcnetReferenceRejectedWithNoFetch(t *testing.T) {
	store := newSeededStore()
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	_, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "arcnet:")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(fetcher.Calls)))
}

// spec FR-010: the command requires an already-initialized graph.
func TestApplyPatchFailsWhenNotAGraph(t *testing.T) {
	store := &fakeStore{files: map[string]string{}, dirs: map[string]bool{}, written: map[string]string{}}
	mounter := fakeMounter{store: store}
	vcs := &schemamock.VCS{}
	fetcher := &schemamock.Fetcher{}
	reporter := &fakeApplyReporter{}

	_, err := service.ApplyPatch(context.Background(), mounter, vcs, fetcher, reporter, "/graph", "patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
}
