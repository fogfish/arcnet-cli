//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the schema domain's value types: ARCNET-CORE's
// declared vocabulary of predicates and types (CORE §10/§11), seeded by
// arc init and read back by internal/app/schema/service.Resolve.
package kernel

import "github.com/fogfish/arcnet-cli/internal/core"

// TypesDir/PredicatesDir are the two _schema/ subfolders, relative to a
// graph root (renamed from the existence-only NodesDir — CORE §9.2).
const (
	TypesDir      = "_schema/types"
	PredicatesDir = "_schema/predicates"
)

// CorePredicateDefs is every predicate ARCNET-CORE §10 documents (research.md
// D6, spec FR-007) — content, metadata/control, structural, semantic,
// citation, type-specific, and the schema mechanism's own vocabulary
// (§10.8). "@id"/"@type" (§10.1) are deliberately excluded: they are
// structural identity fields stripped out of Attrs/Edges before a Node is
// ever constructed (internal/core.identityFields), never looked up through
// core.Index.Predicates by any consumer — the reference ARCNET-CORE example
// graph's own _schema/predicates/ folder likewise carries no such files.
var CorePredicateDefs = map[string]core.PredicateDef{
	"tags": {Role: "meta", Merge: core.MergeUnion, Description: "Topical tags for discoverability."},
	"text": {Role: "text", Merge: core.MergeAppend, Aligned: "schema:text", Description: "Generic prose predicate; each contribution appends to the existing prose rather than overwriting it."},

	"published": {Role: "meta", Merge: core.MergeImmutable, Description: "ISO-8601 production date of the document a node derives from; drives the timeline."},
	"created":   {Role: "meta", Merge: core.MergeImmutable, Description: "ISO-8601 timestamp the node was created in the graph."},
	"updated":   {Role: "meta", Merge: core.MergeLastWriteWin, Description: "ISO-8601 timestamp of the node's last modification."},

	"mentions":    {Role: "link", Merge: core.MergeUnion, Aligned: "schema:mentions", Description: "Asserts that the source document mentions the entity; recorded under the source's own Mentions block."},
	"mentionedIn": {Role: "link", Merge: core.MergeUnion, Aligned: "schema:subjectOf", Description: "The inverse of mentions — recorded as a backlink under the entity's own mentionedIn block."},

	"broader":      {Role: "edge", Merge: core.MergeUnion, Aligned: "skos:broader", Description: "Generalization: the target is the more general concept, the subject a kind or specialization of it."},
	"narrower":     {Role: "edge", Merge: core.MergeUnion, Aligned: "skos:narrower", Description: "The inverse of broader — an optional backlink from the more general concept to the specialization."},
	"isPartOf":     {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:isPartOf", Description: "Composition (part-whole): the subject is a component or member of the whole named by the target."},
	"hasPart":      {Role: "edge", Merge: core.MergeUnion, Aligned: "schema:hasPart", Description: "The inverse of isPartOf — an optional backlink from the whole to a component."},
	"requires":     {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:requires", Description: "Functional dependency: the subject needs the target to function, hold, or be delivered."},
	"replaces":     {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:replaces", Description: "Supersession over time: the subject supplants an older target."},
	"isReplacedBy": {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:isReplacedBy", Description: "The inverse of replaces — an optional backlink from the superseded subject to its successor."},
	"conformsTo":   {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:conformsTo", Description: "Standard adherence: the subject complies with a named specification or schema."},
	"related":      {Role: "edge", Merge: core.MergeUnion, Aligned: "skos:related", Description: "A non-hierarchical, non-compositional association between two subjects, used only when no more specific predicate applies."},

	"cites":            {Role: "link", Merge: core.MergeUnion, Aligned: "cito:cites", Description: "The general-purpose citation predicate; the source's own structural link to a cited resource."},
	"citesAsEvidence":  {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:citesAsEvidence", Description: "Cites the target as evidence for the citing statement."},
	"citesAsAuthority": {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:citesAsAuthority", Description: "Cites the target as an authoritative source for the citing statement."},
	"supports":         {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:supports", Description: "The citing statement is supported by the target."},
	"confirms":         {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:confirms", Description: "The citing statement confirms findings in the target."},
	"extends":          {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:extends", Description: "The citing statement extends work in the target."},
	"critiques":        {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:critiques", Description: "The citing statement critiques the target."},
	"disputes":         {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:disputes", Description: "The citing statement disputes claims in the target."},
	"refutes":          {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:refutes", Description: "The citing statement refutes claims in the target."},
	"isCitedBy":        {Role: "link", Merge: core.MergeUnion, Aligned: "cito:isCitedBy", Description: "The inverse of any citation predicate — recorded as a backlink under the cited node's own isCitedBy block."},

	"title":       {Role: "meta", Merge: core.MergeImmutable, Description: "The document title as published — distinct from @id when @id is a derived citekey."},
	"abstract":    {Role: "text", Merge: core.MergeAppend, Description: "A short prose summary of the document."},
	"authors":     {Role: "meta", Merge: core.MergeUnion, Description: "Ordered list of author names."},
	"url":         {Role: "meta", Merge: core.MergeFillIfEmpty, Description: "Canonical location of the document or work."},
	"doi":         {Role: "meta", Merge: core.MergeFillIfEmpty, Description: "Digital object identifier."},
	"category":    {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "John F. Sowa's top-level category, decoded into a bag of words (e.g. independent/physical/continuant/object)."},
	"aliases":     {Role: "meta", Merge: core.MergeUnion, Description: "Alternative names for the entity."},
	"definition":  {Role: "text", Merge: core.MergeAppend, Description: "A one-to-three sentence definition of the subject."},
	"notes":       {Role: "text", Merge: core.MergeAppend, Description: "Additional prose."},
	"ref":         {Role: "meta", Merge: core.MergeImmutable, Description: "Resource type: a citable work or a topic/area tracked for reading or research."},
	"year":        {Role: "meta", Merge: core.MergeFillIfEmpty, Description: "Year of publication."},
	"status":      {Role: "meta", Merge: core.MergeLastWriteWin, Description: "read or backlog — a backlog resource is a research target."},
	"relevance":   {Role: "text", Merge: core.MergeAppend, Description: "A one-to-two sentence note on why the resource matters."},
	"granularity": {Role: "meta", Merge: core.MergeImmutable, Description: "yearly or monthly."},
	"entries":     {Role: "link", Merge: core.MergeAppend, Description: "The source nodes whose published date falls in this period, ordered by date."},
	"heading":     {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "A human-readable title for the period, shown in place of the bare @id (period code)."},

	"role":        {Role: "meta", Merge: core.MergeImmutable, Description: "One of meta/text/href/edge/link (CORE §5): the predicate's serialization position."},
	"merge":       {Role: "meta", Merge: core.MergeImmutable, Description: "One of the merge behaviors (CORE §9.3): how contributions to this predicate combine."},
	"label":       {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "Human-readable title shown as a link-role predicate's heading; defaults to the predicate name, capitalized."},
	"aligned":     {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "The standard-vocabulary term this predicate maps to, or arc:<name> if graph-native."},
	"description": {Role: "text", Merge: core.MergeAppend, Description: "Prose describing the predicate's or type's meaning — the body text of a Property/Class node."},
	"required":    {Role: "link", Merge: core.MergeUnion, Label: "Requires", Description: "Asserts that the class requires the target predicate on every conforming instance."},
	"optional":    {Role: "link", Merge: core.MergeUnion, Description: "Asserts that the class permits the target predicate."},
}

// CoreTypeDefs is ARCNET-CORE's four fixed node types (CORE §11, seeded for
// arc init since spec 005) plus Property/Class themselves, since a schema
// node's own "@type" value is itself a type in use (CORE §10.8, spec
// FR-007, research.md D6). Kept for continuity/documentation even though no
// longer consulted by core.Merge's per-predicate dispatch (data-model.md
// "CoreTypeDefs seed values").
var CoreTypeDefs = map[string]core.TypeDef{
	"source": {
		Merge:       core.MergeImmutable,
		Required:    []string{"title", "published", "abstract", "mentions"},
		Optional:    []string{"authors", "url", "cites", "tags", "doi"},
		Description: "A node for one ingested document — the provenance origin other nodes derive from.",
	},
	"entity": {
		Merge:       core.MergeUnion,
		Required:    []string{"category", "definition", "mentionedIn"},
		Optional:    []string{"aliases", "tags"},
		Description: "A node for a subject occurring in sources, typed by Sowa category.",
	},
	"resource": {
		Merge:       core.MergeFirstWriteWin,
		Required:    []string{"ref", "relevance"},
		Optional:    []string{"url", "isCitedBy", "authors", "year", "doi", "status", "notes"},
		Description: "A node for an external work the graph points to but has not ingested, or a topic/area tracked for reading or research.",
	},
	"timeline": {
		Merge:       core.MergeAppend,
		Required:    []string{"granularity", "entries"},
		Optional:    []string{"heading"},
		Description: "A production-date index of ingested documents.",
	},
	"Property": {
		Merge:       core.MergeUnion,
		Required:    []string{"role", "merge", "description"},
		Optional:    []string{"label", "aligned"},
		Description: "A predicate schema node: the mechanism CORE uses to register a predicate's own vocabulary as an ordinary graph node.",
	},
	"Class": {
		Merge:       core.MergeUnion,
		Required:    []string{"merge", "description"},
		Optional:    []string{"required", "optional"},
		Description: "A type schema node: the mechanism CORE uses to register a @type value's own vocabulary as an ordinary graph node.",
	},
}
