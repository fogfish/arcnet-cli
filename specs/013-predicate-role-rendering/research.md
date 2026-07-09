# Research: Schema-Driven Link Rendering

## D1: Where the rendering decision lives — `Index` threaded into `core.RenderNode`/`RenderPatch`

**Decision**: `RenderNode(n Node) ([]byte, error)` becomes `RenderNode(n Node, index Index) ([]byte, error)`;
`RenderPatch(p Patch) ([]byte, error)` becomes `RenderPatch(p Patch, index Index) ([]byte, error)`; the
unexported `renderNodeBody(n Node) []byte` becomes `renderNodeBody(n Node, index Index) []byte`.

**Rationale**: `Index`/`PredicateDef` (`Role`, `Label`) already live in `internal/core` (`rules.go`) — this is
the exact precedent spec 012 already established for `core.Merge(existing, incoming Node, index Index, ...)`.
Threading `Index` into `RenderNode`/`RenderPatch` introduces no new import and no ADR 001 dependency-direction
violation: `internal/core` still depends on nothing outside itself. The alternative — having `internal/core`
call back into `internal/app/schema` to resolve a graph's schema itself — would be the actual violation
(`internal/core`'s own package doc: "No dependency on any `internal/app/<use-case>`").

**Alternatives considered**:
- A package-level `SetIndex(Index)` / global mutable state on `internal/core` — rejected: violates Principle
  IV (immutability, no hidden side channel) and would make `RenderNode` non-pure and order-dependent across
  concurrent callers.
- A new `internal/core/render` sub-package taking `Index` at construction — rejected as unnecessary
  indirection (YAGNI): `RenderNode`/`RenderPatch` are already free functions; adding a parameter is the
  smaller diff and matches `core.Merge`'s own shape exactly.

## D2: Partitioning `Node.Edges` by role, and the flat/grouped render algorithm

**⚠️ Superseded in part — BUG-001 (2026-07-09)**: the algorithm below, as originally decided, applies
identically to both `RenderNode` and `RenderPatch` via one shared `renderNodeBody`. That is correct for
`RenderNode` (ARCNET-CORE §5: a `link`-role predicate's occurrences form their own `## Predicate` block) but
wrong for `RenderPatch`: ARCNET-CORE §14.2 fixes a patch document's headings (`H1`/`H2`) to `@type`/`@id`
structure exclusively — "node bodies use bold labels, never headings" — so `RenderPatch`'s link-role groups
MUST render under a `**Label**` bold-label paragraph, not a `"## " + label` heading. The partition/grouping/
ordering/omission logic below (which predicate goes in which bucket, sort-by-label, the single-group
omission) is unaffected and still shared by both callers; only the final markup emitted for a link-role
group's heading differs by caller. See spec.md FR-014 (added) and the corresponding fix in
contracts/render-shape-contract.md.

**Decision**: `renderNodeBody` partitions `n.Edges` into two ordered collections, in one pass, preserving each
predicate's first-seen relative order:
- **edge-role occurrences**: rendered as a single **bare** bulleted list (no heading), each line via the
  existing `renderLinkBullet`, in original document order across every edge-role predicate — this is exactly
  today's rendering for the flat case, unchanged in format.
- **link-role occurrences**: grouped by predicate name; one `"## " + label + "\n"` heading per distinct
  predicate, followed by that predicate's occurrences (again via `renderLinkBullet`) in original relative
  order; groups ordered by their resolved label, ascending (`sort.Strings`), for a deterministic, canonical
  output independent of input order (FR-010 explicitly permits this reordering).
- The bare edge-role list is written **before** any heading-grouped link-role block, matching
  `walkNodeBody`'s already-fixed parser grammar exactly: leading prose → **one optional bare list** → zero or
  more heading+list blocks → trailing prose (`internal/core/markdown.go` lines ~511-521). Emitting output in
  this order is what makes it round-trip correctly through the existing, unchanged parser (FR-011): the bare
  list lands in the parser's single "ungrouped edges" slot, and each heading+list pair lands in the parser's
  labeled-block loop.

**Rationale**: This reuses 100% of the existing parser (no parse-time change, FR-011) and 100% of the existing
bullet-line format (`renderLinkBullet`/`markupFor`) for both flat and grouped occurrences — the only new
code is where each predicate's occurrences land in the output, and whether a heading precedes them.

**Alternatives considered**: Interleaving edge-role bullets and link-role groups in original Edges order
(rather than "all flat bullets first, then all groups") — rejected: the parser's bare-list slot is a single
contiguous list at one fixed position (right after leading prose); scattering flat bullets between heading
blocks isn't a shape `walkNodeBody` can even recognize without a parser change, which is out of scope.

## D3: Role resolution for a predicate, including the unregistered fallback

**Decision**: New helper, mirroring `merge.go`'s own `resolveMergeOp` precedent exactly:

```go
// resolveRenderRole looks up predicate's declared Role in index, falling
// back to "edge" (flat, the conservative shape — never invents a heading
// the author didn't declare) when the predicate has no schema document yet,
// mirroring resolveMergeOp's own unregistered-predicate precedent.
func resolveRenderRole(index Index, predicate string) string {
	if def, ok := index.Predicates[predicate]; ok {
		return def.Role
	}
	return "edge"
}
```

**Rationale**: Directly satisfies spec.md's Assumptions section and FR-013; reuses the exact naming/shape
convention `merge.go:resolveMergeOp` already established for the identical "predicate absent from Index"
situation, so a reader of both files recognizes the same idiom.

## D4: Heading label resolution

**Decision**: `def.Label` if non-empty, else `titleCaseType(predicate)` (the existing helper, already used for
`RenderPatch`'s `"# " + titleCaseType(typ)` type headings) — matching `schema.go`'s own already-documented
default: `"label": ... "defaults to the predicate name, capitalized."`.

**Rationale**: No new capitalization logic; reuses the one helper `internal/core/markdown.go` already has for
an identical "predicate/type name → display heading" need.

## D5: The single-link-role-predicate-body heading omission (FR-006/FR-007) — presence-based, not permission-based

**Decision**: The omission condition is evaluated purely from what's **present** in `n.Edges` at render time:
zero edge-role occurrences present, **and** exactly one distinct link-role predicate name present (with one or
more occurrences of it). When both hold, that single group renders as a bare list (landing in the same
parser bare-list slot as D2's flat case) instead of a `"## Label"` block. In every other case (any edge-role
occurrence present, or two-or-more distinct link-role predicates present), every link-role predicate's heading
renders.

This deliberately does **not** consult the node's `@type`'s Class-node `Required`/`Optional` list (which the
`/speckit-plan` command's own input text proposed, framed as "is `entries` the type's *only possible* edge
predicate"). spec.md's own Edge Cases section already resolved this ambiguity toward presence, not permission:
*"a type permitting more predicates that simply weren't used in this instance is treated the same as a type
that only ever allows the one predicate."* Implementing permission-based instead would produce the **opposite**
answer for that exact edge case (a type allowing two link predicates, only one of which happens to be used on
a given node, would keep its heading under permission-based, but spec.md requires it to still omit). Presence-
based is simultaneously simpler (no `TypeDef.Required`/`Optional` lookup, no dependency on `Node.Type` at all
inside `renderNodeBody`) and the literal, already-ratified requirement — not a shortcut taken at the expense of
correctness.

**Rationale**: For every currently-seeded type (`timeline`'s `entries` is its only link/edge-bearing predicate
at all — `granularity`/`heading` are both `meta`, front-matter-only), presence-based and permission-based
produce an identical result today; the divergence only matters for a hypothetical future type with two-or-more
optional link predicates, which spec.md has already settled.

## D6: Signature/call-site blast radius

**Bugfix note (BUG-001, 2026-07-09)**: this table's `RenderPatch` rows (`humanSubgraphPrinter.Show`,
`subgraphGetHandler`) supply the same `Index` value D2 always assumed would drive an identical rendering
algorithm to `RenderNode`. The `Index` sourcing itself is unaffected by this bugfix — only D2's downstream
choice of markup (heading vs. bold label) needs to branch on which function (`RenderNode` vs `RenderPatch`)
is doing the rendering, not on anything about how `Index` was obtained.

**Decision** — every existing production call site of `core.RenderNode`/`core.RenderPatch`, and how each
supplies its `Index`:

| Call site | New `Index` source |
|---|---|
| `internal/app/schema/service/schema.go` `Seed()` (2 calls: `predicateNode`/`typeNode`) | `core.Index{Predicates: kernel.CorePredicateDefs, Types: kernel.CoreTypeDefs}`, built once inline — pure, no I/O, exactly the built-in vocabulary being seeded. Needed because `typeNode`'s `Edges` (`required`/`optional`, both `role: link` per `CorePredicateDefs`) must resolve to grouped rendering for `_schema/types/*.md` seed documents to come out correct. |
| `internal/app/schema/service/schema.go` `registerIfAbsent` (called by `RegisterType`/`RegisterPredicate`) | `core.Index{}` (zero value). Safe by inspection: both callers construct a `core.Node` with `Edges: nil` (`RegisterType`/`RegisterPredicate`'s literal node values carry only `Attrs`/`Texts`) — `renderNodeBody`'s role-partitioning path is only reached when `len(node.Edges) > 0`, so the empty Index can never be consulted. Documented with a one-line comment at the call site rather than threading a real Index through two exported signatures for no behavioral difference (YAGNI). |
| `internal/app/graph/service/apply.go` `nodeContentChanged`, `writeNode` | The `index core.Index` parameter `Apply` already receives (spec 012) — both functions gain an `index Index` parameter, threaded from their two call sites inside `Apply`. No new resolution; already guaranteed valid by `Apply`'s own existing `.arc`/schema preflight. |
| `cmd/arc/graph/subgraph.go` `humanSubgraphPrinter.Show` | New `index core.Index` field on `humanSubgraphPrinter`, populated in `RunE` via a new local `resolveIndexOrDefault(store)` helper (D7) — this command has never required `.arc`/`_schema` to exist (unlike `apply`/`lint`) and must not start requiring it now just to pick a bullet style. |
| `cmd/arc/graph/serve.go` `nodeGetHandler`, `subgraphGetHandler` | Same `resolveIndexOrDefault(store)` helper, resolved once in `buildServer` and passed into both handler constructors alongside `dir`. |

**Rationale**: Every production call site already has, or can trivially obtain, an `Index` value without
introducing a new hard dependency on `.arc`/`_schema` where one didn't already exist.

## D7: Graceful degradation for read-only, schema-optional commands (`arc subgraph`, `arc serve`)

**Decision**: New small helper in `cmd/arc/graph` (not `internal/app/graph/service` — this is a
presentation-time convenience, not a service-layer concern, and keeps `Subgraph`/`NodeGet`'s existing
signatures and their entire existing test suites untouched):

```go
// resolveIndexOrDefault resolves store's schema index for rendering
// purposes only, falling back to the zero Index (every predicate renders
// edge-role/flat, D3) when the directory isn't a fully schema-populated
// graph. arc subgraph/arc serve have never required .arc/_schema to exist
// (unlike arc apply/arc lint's own hard preflight) — this feature must not
// make choosing a bullet-vs-heading shape a new reason those commands stop
// working against a bare directory of node files.
func resolveIndexOrDefault(store fsys.Store) core.Index {
	index, err := appschema.Resolve(store)
	if err != nil {
		return core.Index{}
	}
	return index
}
```

**Rationale**: `internal/app/graph/service/grep.go`'s existing comment and `subgraph`/`serve`'s existing test
fixtures (`writeGrepNode` — bare `.md` files, no `.arc/`, no `_schema/`) confirm these two commands were
deliberately built to need nothing beyond node files themselves. Making `schema.Resolve`'s error hard-fail
here would be a new, disproportionate reliability regression for a purely cosmetic rendering choice, and
would break every existing subgraph/serve E2E test's fixture. Falling back to `core.Index{}` reuses D3's
already-required "unregistered predicate ⇒ edge/flat" fallback uniformly — a directory with no schema at all
behaves identically to a directory where every individual predicate happens to be unregistered, which is
already a required, spec'd behavior (FR-013), not a new concept.

**Alternatives considered**: Hard-failing subgraph/serve when `.arc`/`_schema` is absent, matching `apply`'s
strictness — rejected: out of proportion to this feature's scope (FR-012 explicitly keeps merge/apply
behavior untouched; equally, this feature must not incidentally tighten subgraph/serve's own preflight
contract, which no FR calls for).

## D8: Existing tests that assert today's "always flat" behavior and must be rewritten

**Decision**: Two existing tests in `internal/core/markdown_test.go` assert behavior this feature deliberately
inverts, and are rewritten (not merely extended) as part of implementation:

- `TestRenderNodeEdgesFlatBulletedListNoGroupedHeadings` — asserts a node mixing `replaces` (edge),
  `mentions`/`mentionedIn` (link) all render as flat bullets with no `"## "` anywhere. Rewritten to assert the
  schema-driven mixed shape instead: `replaces` stays a flat bullet, `mentions`/`mentionedIn` each render
  under their own heading — the same fixture, corrected expectation, doubling as this feature's canonical
  "CORE §11-style worked example" test the `/speckit-plan` input text asked for.
- `TestCosmeticExceptionGroupedHeadingFlattensOnRoundTrip` — asserts a `"## Label"`-grouped input always
  flattens on re-render. Rewritten to assert normalization *toward the predicate's declared role* instead of
  *always toward flat*: the fixture's grouped predicate, if link-role, stays grouped (byte-normalized, not
  flattened); a variant fixture with an edge-role predicate written grouped is asserted to flatten. This
  directly exercises FR-009 (normalize inconsistent input to canonical shape) in both directions.

**Rationale**: Leaving these two tests as-is would make the test suite internally contradictory (one test
asserting flat-always while new tests assert schema-driven) and would fail the moment the new behavior ships
— they are exactly the tests the request's own "today, arc's parser detects the shape it finds and preserves
it" describes as being retired.

## D9: `ARCHITECTURE.md` Glossary touch-up

**Decision**: The **Node** glossary entry's `Edges` clause currently reads "every outgoing structural link, in
document order, regardless of how the source document grouped it" — still accurate for the in-memory shape
and for parsing (both untouched, FR-011), but is read easily as implying rendering also ignores grouping,
which is no longer true. Append one clause noting that *rendering* (not the in-memory representation) derives
flat-vs-grouped from each predicate's own schema `Role`, cross-referencing this spec. No other glossary entry
needs a new term — this feature adds no new domain concept beyond the already-glossary'd `Role`/`Label`
fields of **Predicate Schema Node**.

## D10: `RenderNode` vs `RenderPatch` link-role markup diverges by format (BUG-001)

**Decision**: D2's "one `## Predicate` block per link-role predicate" markup choice governs `RenderNode`
only. `RenderPatch` renders the identical link-role group (same partition, same label resolution, same
single-group omission) under a `"**" + label + "**\n"` bold-label paragraph instead, followed by that
group's bullets — never a `## Label` heading. This is not a new decision so much as D2's original decision
scoped correctly: ARCNET-CORE §5 (graph node files) and §14.2 (patch documents) always specified different
body markup for link-role groups; D2 simply failed to consult §14.2 when generalizing "one shared
`renderNodeBody`" to both `RenderNode` and `RenderPatch`.

**Rationale**: ARCNET-CORE §14.2's patch structure is fixed and load-bearing: `H1 = @type`, `H2 = @id`, one
`H2` per node, with no other heading permitted anywhere in a patch document ("Markdown headings are reserved
for type and identity; node bodies use bold labels, never headings"). A `## Label` heading inside a patch's
per-node body is indistinguishable, to a reader or to `parsePatchBody`'s own `isSectionBoundary` heuristic
recognizing only heading-immediately-followed-by-yaml-fence as a node boundary, from a poorly-formed or
ambiguous document — `arc`'s own parser happens to tolerate it via that heuristic, but an external,
spec-conforming patch consumer is not obligated to. Rendering link-role groups as bold labels in
`RenderPatch` matches this codebase's own pre-existing bold-label parsing precedent
(`internal/core/markdown.go`'s `blockTitle`/`boldLabel`, introduced for BUG-003), so no parser change is
required — `walkNodeBody` already recognizes a `**Label**` paragraph immediately followed by a list as a
predicate-grouped block.

**Alternatives considered**: Leaving `RenderPatch` unchanged (still emitting `## Label` headings) on the
grounds that `arc`'s own round-trip tolerates it — rejected: this is a spec-conformance defect against the
external, authoritative ARCNET-CORE format, not merely an internal round-trip property; "our own parser
doesn't choke on it" is not the same as "the format is correct."
