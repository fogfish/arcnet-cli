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
// graph root: "_schema/Class" holds the Class-typed type schema nodes (CORE
// §9.2) and "_schema/Property" the Property-typed predicate schema nodes
// (§9.1).
//
// _schema/ is a namespace prefix rather than a type folder, but its two
// children are type folders and so obey CORE §6 (v0.11) like any other:
// each is named for the type it holds, character for character
// (specs/022-reference-type-folders, contract C2). The Go identifiers keep
// naming the concept — the folder of type nodes, the folder of predicate
// nodes — while their values name the path; both Seed and Resolve read
// these constants, so the pair is the single source of truth for where
// schema documents live.
const (
	TypesDir      = "_schema/Class"
	PredicatesDir = "_schema/Property"
)

// CorePredicateDefs is every predicate ARCNET-CORE §10 documents (research.md
// D6, spec FR-007) — content, metadata/control, structural, semantic,
// citation, type-specific, and the schema mechanism's own vocabulary
// (§10.8). "@id"/"@type" (§10.1) are deliberately excluded: they are
// structural identity fields stripped out of Attrs/Edges before a Node is
// ever constructed (internal/core.identityFields), never looked up through
// core.Index.Predicates by any consumer — the reference ARCNET-CORE example
// graph's own predicate-schema folder likewise carries no such files.
var CorePredicateDefs = map[string]core.PredicateDef{
	"tags": {Role: "meta", Merge: core.MergeUnion, Description: "Topical tags for discoverability."},
	"text": {Role: "text", Merge: core.MergeAppend, Aligned: "schema:text", Description: "Generic prose predicate; each contribution appends to the existing prose rather than overwriting it."},

	"published": {Role: "meta", Merge: core.MergeImmutable, Description: "ISO-8601 production date of the document a node derives from; drives the timeline."},
	"created":   {Role: "meta", Merge: core.MergeImmutable, Description: "ISO-8601 timestamp the node was created in the graph."},
	"updated":   {Role: "meta", Merge: core.MergeLastWriteWin, Description: "ISO-8601 timestamp of the node's last modification."},
	"indexed":   {Role: "meta", Merge: core.MergeImmutable, Aligned: "arc:indexed", Description: "ISO-8601 timestamp (second resolution) marking when the node first entered the graph; set once at creation by arc apply and never modified by any later merge (spec 009)."},

	"scoreZ": {Role: "meta", Merge: core.MergeLastWriteWin, Aligned: "arc:scoreZ", Description: "A graph-analytics z-score (e.g. centrality); the most recent value written wins, so a recomputation pass always supersedes the previous score."},
	"scoreC": {Role: "meta", Merge: core.MergeLastWriteWin, Aligned: "arc:scoreC", Description: "A graph-analytics centrality-style score; the most recent value written wins, so a recomputation pass always supersedes the previous score."},

	"mentions":    {Role: "link", Merge: core.MergeUnion, Aligned: "schema:mentions", Description: "Asserts that the source document mentions the entity; recorded under the source's own Mentions block."},
	"mentionedIn": {Role: "link", Merge: core.MergeUnion, Aligned: "schema:subjectOf", Description: "The inverse of mentions — recorded as a backlink under the mentioned node's own mentionedIn block."},

	"broader":      {Role: "edge", Merge: core.MergeUnion, Aligned: "skos:broader", Description: "Generalization: the target is the more general concept, the subject a kind or specialization of it."},
	"narrower":     {Role: "edge", Merge: core.MergeUnion, Aligned: "skos:narrower", Description: "The inverse of broader — an optional backlink from the more general concept to the specialization."},
	"isPartOf":     {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:isPartOf", Description: "Composition (part-whole): the subject is a component or member of the whole named by the target."},
	"hasPart":      {Role: "edge", Merge: core.MergeUnion, Aligned: "schema:hasPart", Description: "The inverse of isPartOf — an optional backlink from the whole to a component."},
	"requires":     {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:requires", Description: "Functional dependency: the subject needs the target to function, hold, or be delivered."},
	"replaces":     {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:replaces", Description: "Supersession over time: the subject supplants an older target."},
	"isReplacedBy": {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:isReplacedBy", Description: "The inverse of replaces — an optional backlink from the superseded subject to its successor."},
	"conformsTo":   {Role: "edge", Merge: core.MergeUnion, Aligned: "dcterms:conformsTo", Description: "Standard adherence: the subject complies with a named specification or schema."},
	"related":      {Role: "edge", Merge: core.MergeUnion, Aligned: "skos:related", Description: "A non-hierarchical, non-compositional association between two subjects, used only when no more specific predicate applies."},
	"referencedBy": {Role: "edge", Merge: core.MergeUnion, Description: "A non-hierarchical, non-compositional asymmetric association when the object's own node doesn't explicitly link the subject back."},

	"cites":            {Role: "link", Merge: core.MergeUnion, Aligned: "cito:cites", Description: "The general-purpose citation predicate; a source's own structural link to a cited reference, or a timeline's chronological reference to a source node it contains."},
	"citesAsEvidence":  {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:citesAsEvidence", Description: "Cites the target as evidence for the citing statement."},
	"citesAsAuthority": {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:citesAsAuthority", Description: "Cites the target as an authoritative source for the citing statement."},
	"supports":         {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:supports", Description: "The citing statement is supported by the target."},
	"confirms":         {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:confirms", Description: "The citing statement confirms findings in the target."},
	"extends":          {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:extends", Description: "The citing statement extends work in the target."},
	"critiques":        {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:critiques", Description: "The citing statement critiques the target."},
	"disputes":         {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:disputes", Description: "The citing statement disputes claims in the target."},
	"refutes":          {Role: "edge", Merge: core.MergeUnion, Aligned: "cito:refutes", Description: "The citing statement refutes claims in the target."},
	"isCitedBy":        {Role: "link", Merge: core.MergeUnion, Aligned: "cito:isCitedBy", Description: "The inverse of any citation predicate — recorded as a backlink under the cited node's own isCitedBy block."},

	"author": {Role: "meta", Merge: core.MergeUnion, Aligned: "schema:author", Description: "The author of the content. Registered alongside the plural \"authors\" (§10.7), which is the array form the type definitions reference; both carry the same standard alignment."},
	"about":  {Role: "meta", Merge: core.MergeUnion, Description: "What the content is about — its subject matter: one of technique, theory, platform, system, technology, language, framework, or field."},
	"genre":  {Role: "meta", Merge: core.MergeUnion, Description: "The genre of the content: one of paper, standard, tool, dataset, or post."},

	"title":       {Role: "meta", Merge: core.MergeImmutable, Aligned: "schema:title", Description: "The document title as published — distinct from @id when @id is a derived citekey."},
	"abstract":    {Role: "text", Merge: core.MergeFirstWriteWin, Aligned: "schema:abstract", Description: "A short prose summary of the document."},
	"authors":     {Role: "meta", Merge: core.MergeUnion, Aligned: "schema:author", Description: "Ordered list of author names."},
	"url":         {Role: "meta", Merge: core.MergeFillIfEmpty, Aligned: "schema:url", Description: "Canonical location of the document or work."},
	"doi":         {Role: "meta", Merge: core.MergeFillIfEmpty, Aligned: "schema:doi", Description: "Digital object identifier."},
	"category":    {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "John F. Sowa's top-level category, decoded into a bag of words (e.g. independent/physical/continuant/object)."},
	"aliases":     {Role: "meta", Merge: core.MergeUnion, Aligned: "skos:altLabel", Description: "Alternative names for the entity."},
	"definition":  {Role: "text", Merge: core.MergeFirstWriteWin, Description: "A one-to-three sentence definition of the subject."},
	"notes":       {Role: "text", Merge: core.MergeAppend, Description: "Additional prose."},
	"ref":         {Role: "meta", Merge: core.MergeImmutable, Description: "Reference type: a citable work or a topic/area tracked for reading or research."},
	"year":        {Role: "meta", Merge: core.MergeImmutable, Description: "Year of publication."},
	"status":      {Role: "meta", Merge: core.MergeLastWriteWin, Description: "read or backlog — a backlog reference is a research target."},
	"relevance":   {Role: "text", Merge: core.MergeFirstWriteWin, Description: "A one-to-two sentence note on why the reference matters."},
	"granularity": {Role: "meta", Merge: core.MergeImmutable, Description: "yearly or monthly."},
	"heading":     {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "A human-readable title for the period, shown in place of the bare @id (period code)."},
	"period":      {Role: "meta", Merge: core.MergeImmutable, Aligned: "arc:period", Description: "A timeline node's own period code (YYYY or YYYY-MM), duplicated from its @id so a bare 4-digit yearly value always decodes as a YAML string rather than an integer."},

	"role":        {Role: "meta", Merge: core.MergeImmutable, Description: "One of meta/text/href/edge/link (CORE §5): the predicate's serialization position."},
	"merge":       {Role: "meta", Merge: core.MergeImmutable, Description: "One of the merge behaviors (CORE §9.3): how contributions to this predicate combine."},
	"label":       {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "Human-readable title shown as a link-role predicate's heading; defaults to the predicate name, capitalized."},
	"aligned":     {Role: "meta", Merge: core.MergeFirstWriteWin, Description: "The standard-vocabulary term this predicate maps to, or arc:<name> if graph-native."},
	"description": {Role: "text", Merge: core.MergeFirstWriteWin, Description: "Prose describing the predicate's or type's meaning — the body text of a Property/Class node."},
	"required":    {Role: "link", Merge: core.MergeUnion, Label: "Requires", Description: "Asserts that the class requires the target predicate on every conforming instance."},
	"optional":    {Role: "link", Merge: core.MergeUnion, Description: "Asserts that the class permits the target predicate."},

	"subClassOf": {Role: "edge", Merge: core.MergeUnion, Aligned: "rdfs:subClassOf", Description: "Declares that the subject type inherits every predicate the target type requires or permits (spec 017)."},
}

// CoreTypeDefs is ARCNET-CORE's five fixed content types (CORE §11, seeded
// for arc init since spec 005) plus the universal Node base and
// Property/Class themselves, since a schema node's own "@type" value is
// itself a type in use (CORE §10.8, research.md D6).
//
// No entry declares a merge behaviour: CORE §9.3 retired type-level merge
// in favour of per-predicate merge, and §10.8 registers "merge" as a
// predicate Property alone uses. core.TypeDef carries no such field for a
// literal here to set (specs/023-core-vocabulary-conformance FR-005).
var CoreTypeDefs = map[string]core.TypeDef{
	// §11.2 requires "published" of a Source directly. Before spec 023 it
	// arrived by inheritance from Node, so promoting it here is neutral
	// for Source and is what lets Node's own Requires empty out (FR-008).
	"Source": {
		Required: []string{"title", "published", "abstract", "mentions"},
		// "author"/"about"/"genre" are §10.2 document-metadata predicates.
		// FR-011 registers them; FR-012 additionally requires that a patch
		// using them lint clean, which registration alone cannot deliver —
		// checkTypeOptional reports any predicate a node's own type does
		// not declare. Permitting them on Source, the type §10.2's own
		// examples attach them to, is what closes that half (data-model.md
		// §4 lists only the registration).
		Optional:    []string{"author", "authors", "about", "genre", "url", "cites", "tags", "doi", "indexed"},
		Description: "A node for one ingested document — the provenance origin other nodes derive from.",
	},
	"Entity": {
		Required: []string{"category", "definition", "mentionedIn"},
		Optional: []string{
			"aliases", "tags", "notes", "indexed", "mentions",
			"broader", "narrower", "isPartOf", "hasPart", "requires", "replaces", "isReplacedBy", "conformsTo", "related", "referencedBy",
		},
		Description: "A node for a subject occurring in sources, typed by Sowa category.",
	},
	// CORE §11.4 lists "notes" as Resource's sole optional; "indexed" is
	// carried in addition because it is not a CORE predicate at all but an
	// arc extension (arc:indexed, spec 009) that arc apply stamps on every
	// node it creates. Omitting it would make every Resource the tool
	// itself writes fail its own type-conformance check — the same reason
	// Source, Entity, and Timeline each carry it, and the same kind of
	// documented divergence as Timeline's arc-internal "period".
	"Resource": {
		Required:    []string{"text", "tags", "mentionedIn"},
		Optional:    []string{"notes", "indexed"},
		Description: "A fragment of an ingested document's content that is relevant to the graph but does not warrant its own dedicated type; tag-classified so a recurring pattern can later be promoted into a proper domain type.",
	},
	// §11.5 requires "cites" alone. "granularity" and "period" stay
	// declared as Optional — §11.5's own worked example carries a
	// granularity — but a period node that holds only its chronological
	// citations is conformant and must lint clean (FR-009).
	"Timeline": {
		Required:    []string{"cites"},
		Optional:    []string{"granularity", "period", "heading", "indexed", "mentions", "mentionedIn"},
		Description: "A production-date index of ingested documents.",
	},
	// §11.6 v0.11's normative Class block requires "title" alone. "ref",
	// "relevance", "status", and "notes" are retained as Optional so
	// §11.6's own worked example — which carries a ref and a status —
	// conforms to the type it illustrates, and so spec 022's body-prose
	// keying (a Reference's leading prose is its "relevance") survives.
	//
	// This SUPERSEDES a recorded Clarification in
	// specs/022-reference-type-folders, which required title/ref/relevance
	// on the strength of v0.10's 0.9→0.10 revision note. That note is gone
	// from v0.11 (plan.md F3, research.md D3). Raised explicitly rather
	// than diverged from quietly; the change is a pure relaxation, so no
	// graph that lints clean today can start failing.
	//
	// Plus the same arc-extension "indexed" Resource carries above and for
	// the same reason.
	"Reference": {
		Required:    []string{"title"},
		Optional:    []string{"url", "authors", "year", "doi", "isCitedBy", "ref", "relevance", "status", "notes", "indexed"},
		Description: "A node for an external work the graph points to but has not ingested, or a topic/area tracked for reading or research.",
	},
	// §11.1: every node carries "@id"/"@type" and nothing else
	// universally, so the universal base requires NOTHING (FR-007).
	// "published"/"created" move to Optional, where they stay declarable
	// on every type by inheritance — a node that already carries either
	// keeps it, and only the lint expectation changes. A type that really
	// does require one says so itself, as Source now does.
	"Node": {
		Required:    []string{},
		Optional:    []string{"published", "created", "tags", "text", "updated", "scoreZ", "scoreC"},
		Description: "The graph's implicit universal base type: every content type (Source, Entity, Resource, Timeline, Reference) inherits its Required/Optional contract via rdfs:subClassOf, whether declared explicitly or not. Never itself a node's own @type — it exists only to be inherited from (spec 017).",
	},
	"Property": {
		Required:    []string{"role", "merge", "description"},
		Optional:    []string{"label", "aligned"},
		Description: "A predicate schema node: the mechanism CORE uses to register a predicate's own vocabulary as an ordinary graph node.",
	},
	// §9.2 makes a Class document's description mandatory — decodeTypeDef
	// already rejects an empty one, so promoting it to Requires can break
	// no loadable graph. "subClassOf" is declared because CoreTypeBases
	// writes a subClassOf:: bullet onto five seeded Class documents while
	// Class itself never declared the predicate — a typeOptional violation
	// the seeded tree inflicts on itself (FR-029).
	"Class": {
		Required:    []string{"description"},
		Optional:    []string{"required", "optional", "subClassOf"},
		Description: "A type schema node: the mechanism CORE uses to register a @type value's own vocabulary as an ordinary graph node.",
	},
}

// CoreTypeBases declares each seeded content type's explicit rdfs:subClassOf
// base(s) — written to Seed's rendered output for self-description (spec
// 017, data-model.md), redundant with the implicit Node base every type
// except Node/Property/Class already receives at resolve time regardless of
// what this map says.
var CoreTypeBases = map[string][]string{
	"Source":    {"Node"},
	"Entity":    {"Node"},
	"Resource":  {"Node"},
	"Timeline":  {"Node"},
	"Reference": {"Node"},
}

// RetiredBuiltIns names every _schema/ document a PREVIOUS release seeded
// that this one does not — the set arc upgrade deletes from an existing
// graph (contract C3.3, kernel.UpgradeResult.Removed).
//
// It is a historical record, not a derived value: once a name leaves
// CorePredicateDefs/CoreTypeDefs there is nothing left in the code to
// discover that it was ever built in, so an entry must be added here at the
// same time the definition is removed, or graphs in the field keep an
// orphaned document forever.
//
// Empty today: specs/023-core-vocabulary-conformance only corrects and adds
// definitions — it retires none. (Spec 022's folder renames are a different
// migration and are explicitly out of scope for arc upgrade, contract C3.9.)
var RetiredBuiltIns = []string{}
