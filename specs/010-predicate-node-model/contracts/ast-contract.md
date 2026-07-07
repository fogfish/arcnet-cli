# AST Contract: `internal/core` (supersedes specs/003-apply-patch's `Node`/`Link`/`LinkBlock` shape)

Public surface any graph-reading command (`apply`, `lint`, `grep`,
`subgraph`, `serve`) depends on. goldmark types never appear here (carried
over from specs/003-apply-patch research.md D2/D3). Only the parts of the
003 contract that change are restated here; `Merge`'s function signature,
`TimelinePeriods`/`TimelineEntry`, and the merge-rule vocabulary functions
are unchanged in signature (their behavior over the new shapes is described
below where it differs).

## Types

```go
type Predicate struct {
    Value  any
    Target string
    Alias  string
}

type Node struct {
    ID        string
    Type      string
    Published time.Time
    Attrs     map[string][]Predicate
    Texts     map[string]string
    HRefs     []Link
    Edges     []Link
}
```

`Kind` (the named type) and `LinkBlock` no longer exist. `Link` is
unchanged (`Predicate string`, `Target string`, `Alias string`).

## Parsing

```go
func ParsePatch(r io.Reader) (Patch, error)
func ParseNode(r io.Reader) (Node, error)
```

- `ParsePatch`/`ParseNode` return `ErrManifestInvalid` when any of the
  following hold, with a message naming the specific problem (never a
  generic "invalid manifest"):
  - front matter contains a `kind` key at all (old-format signal);
  - `"@id"` is absent or empty;
  - `"@type"` is absent or empty;
  - for `ParseNode` specifically, `"@id"` does not equal the file's
    basename with `.md` stripped.
  - No fallback to `id`/`title`/`period` is attempted under any
    circumstance — this replaces specs/003's `deriveNodeID` fallback chain
    entirely (research.md D7).
  - **BUG-001**: for `ParsePatch`'s per-node sections specifically,
    `"@id"`/`"@type"` being absent from the node's own yaml fence is not
    itself a rejection — the `"## <ID>"` heading satisfies `"@id"`, and the
    enclosing `"# <Type>"` heading satisfies `"@type"`, mirroring the
    pre-existing CORE §12.2 convention. A yaml-fence `"@id"`/`"@type"` key
    is optional; if present, it MUST agree with the corresponding heading,
    or the node is rejected as inconsistent. This carve-out does not apply
    to `ParseNode`, which has no section headings and must find both keys
    in the file's own front matter.
- Every front-matter key other than `"@id"`/`"@type"`/`"published"` is
  wrapped into `Attrs[key] = []Predicate{...}`: a YAML scalar produces one
  `Predicate{Value: scalar}`; a YAML sequence produces one `Predicate` per
  element, in order. Unrecognized keys are preserved exactly as any other
  key (AST invariant 5, carried over unchanged).
- Body walking keeps specs/003's structural recognition (leading prose /
  optional bare list / heading-or-bold-label-plus-list blocks / trailing
  prose) but changes what it produces: leading and trailing prose become
  `Texts[textPredicateFor(node.Type, true)]` and
  `Texts[textPredicateFor(node.Type, false)]` respectively (empty string
  omits the key rather than storing `""`); the bare list and every
  heading/bold-label-plus-list block's items are all appended, in the
  order encountered across the whole body, to one `Edges` slice — no
  per-block grouping key is retained (research.md D5).
- `[[Target]]`/`[[Target|alias]]`/`[predicate:: [[Target]]]` bracket markup
  embedded inline inside a `Texts` value is still stripped and recorded
  into `HRefs`, in the order encountered — unchanged from specs/003's
  `Text`/`Notes` behavior, just applied per `Texts` key instead of to two
  fixed fields.

**Bugfix**: 2026-07-07 — BUG-001 Updated from bugfix patch: added the
per-node patch-section heading carve-out for `"@id"`/`"@type"` above.

## Serialization

```go
func RenderNode(n Node) ([]byte, error)
```

- Front matter renders `"@id"` and `"@type"` first (both quoted YAML
  keys), then every other `Attrs` key sorted alphabetically: a
  single-element `[]Predicate` renders as a bare scalar; a multi-element
  list renders as a YAML sequence (research.md D3). `published`, when
  non-zero, renders exactly as specs/009 defined.
- Body renders every `Texts` key present (order: the node's declared
  leading-slot key first if present, then any other keys sorted
  alphabetically, then the trailing-slot key last if present — matching
  the original leading-prose/edges/trailing-prose visual layout), then
  `Edges` as one flat bulleted list (`- predicate:: [[Target]]` or bare
  `[[Target]]` when `Predicate` is empty), in `Edges`' stored order. No
  `"## <Label>"` grouped rendering is produced (research.md D6 — deferred
  to spec 013).
- Inline wikilink reconstruction into `Texts` values from `HRefs` is
  unchanged in algorithm from specs/003's contract (best-effort, not
  guaranteed byte-exact for a repeated display substring), just applied
  per `Texts` key instead of to two fixed fields.
- Round-trips losslessly with `ParseNode`/`ParsePatch` (AST §3.6, spec
  FR-014), with the same documented best-effort `HRefs` exception as
  specs/003, plus the explicitly permitted cosmetic exception that a node
  originally written with a grouped-heading link block round-trips to a
  flat bulleted list (content and connectivity unchanged, layout is not).
  A second round-trip of already-rendered output is byte-for-byte stable
  (spec FR-015).

## Old-format rejection (spec FR-012/FR-013, research.md D7)

`ParseNode`/`ParsePatch` never partially parse an old-format file. On
detecting any of the conditions listed under Parsing above, they return
immediately with `ErrManifestInvalid`, before any body walking begins, and
the caller (`service.Apply`, `service.subgraph`, lint's loader, etc.)
propagates that error unchanged — no partial `Node` is ever constructed or
written.

## Merge (research.md data-model.md "Relationships / Lifecycle")

```go
func Merge(existing, incoming Node, op MergeOp, sourceID string) (merged Node, conflicts []string, err error)
```

Signature unchanged from specs/003. Behavior over the new shape:

- `Texts` merges key-by-key over the union of both nodes' keys, applying
  the same scalar-merge policy specs/003's `Text`/`Notes` handling used,
  now generalized to any key name instead of two fixed ones.
- `Attrs` merges key-by-key over the union of keys; each key's
  `[]Predicate` list is merged as one list per the node kind's `MergeOp`
  (union/first-writer/append/etc.) — this feature does not change
  per-predicate merge *policy*, only the shape (list-of-`Predicate`
  instead of a bare scalar) that policy is applied to.
- `Edges` merges as one unioned `Link` list — the same union behavior
  specs/003 applied to `Edges`, now also covering what used to be
  `Links`' contents, since there is only one collection to union.
- `HRefs` merge is unchanged.
- `conflicts` reporting is unchanged in kind (a list of `Attrs`/prose key
  names that diverged), generalized to whatever `Texts`/`Attrs` keys are
  actually present rather than the fixed `"text"`/`"notes"` literals.

## Merge-rule vocabulary

Unchanged in function/constant signatures from specs/003's contract,
except `MergeRuleSet` is `map[string]MergeOp` (was `map[Kind]MergeOp`) and
`Lookup(kind string) (op MergeOp, ok bool)` (was `Lookup(kind Kind)`) —
mechanical consequence of research.md D1.
