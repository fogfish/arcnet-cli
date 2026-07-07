//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package kernel holds the schema domain's value types: ARCNET-CORE's
// declared vocabulary of node kinds, merge behaviors, and predicates.
package kernel

import "github.com/fogfish/arcnet-cli/internal/core"

// SchemaKind is the front-matter kind every _schema/ document carries.
const SchemaKind string = "schema"

// NodesDir/PredicatesDir are the two _schema/ subfolders, relative to a
// graph root.
const (
	NodesDir      = "_schema/nodes"
	PredicatesDir = "_schema/predicates"
)

// CoreMergeRules is ARCNET-CORE's four fixed node kinds and their merge
// behavior (CORE §9.1-9.4), seeded by arc init into _schema/nodes/.
var CoreMergeRules = core.MergeRuleSet{
	"source":   core.MergeNone,
	"entity":   core.MergeUnion,
	"resource": core.MergeUnionFirstWriter,
	"timeline": core.MergeAppend,
}

// coreKindDescriptions is a one-line, informational description per fixed
// kind, rendered only into Seed()'s Text field — never parsed back
// structurally.
var coreKindDescriptions = map[string]string{
	"source":   "A citable document a patch itself represents.",
	"entity":   "A concept or subject mentioned across sources, mergeable across contributions.",
	"resource": "A referenced material such as a standard or specification, first-writer-wins on divergence.",
	"timeline": "A derived, chronologically-ordered index of documents by publication period.",
}

// KindDescription returns kind's one-line, informational description, if
// any.
func KindDescription(kind string) string {
	return coreKindDescriptions[kind]
}

// CorePredicates is ARCNET-CORE's thirteen fixed predicate names (CORE
// §7.4), each paired with a one-line description, seeded by arc init into
// _schema/predicates/.
var CorePredicates = map[string]string{
	"mentions":     "A document mentions an entity or resource.",
	"mentionedIn":  "The inverse of mentions.",
	"cites":        "A document cites another as a source or authority.",
	"isCitedBy":    "The inverse of cites.",
	"broader":      "A more general, encompassing concept (skos:broader).",
	"narrower":     "A more specific concept (skos:narrower).",
	"isPartOf":     "A whole this node is a constituent part of.",
	"hasPart":      "The inverse of isPartOf.",
	"requires":     "A dependency this node needs in order to hold or apply.",
	"replaces":     "A prior node this one supersedes.",
	"isReplacedBy": "The inverse of replaces.",
	"conformsTo":   "A standard or specification this node conforms to.",
	"related":      "A general association between two nodes with no more specific predicate (skos:related).",
}
