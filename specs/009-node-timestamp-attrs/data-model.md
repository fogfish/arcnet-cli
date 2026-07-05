# Data Model: Node Provenance Timestamps

## `internal/core.Node` (extended)

```go
type Node struct {
    ID    string
    Kind  Kind
    // Published is the source document's declared publication date,
    // propagated to every non-stub, non-schema node the patch creates
    // (spec FR-001), and to a previously-absent node's Published on a later
    // merge (spec FR-010, research.md D3) — never overwritten once
    // non-zero. Zero value (IsZero()) means "not yet set" (a stub, a
    // schema document, or a node from before this feature).
    Published time.Time `json:"published,omitempty"`

    Attrs map[string]any
    Text  string
    Notes string
    HRefs []Link
    Edges []Link
    Links map[string]LinkBlock
}
```

No other `internal/core` type changes. `Patch.Published` is unchanged (already existed).

## Provenance Timestamp Attributes (conceptual, not a Go type)

| Attribute | Where it lives | Format | Set when | Immutability |
|---|---|---|---|---|
| `published` | `Node.Published` (typed field) | `"2006-01-02"` (date-only, matches `Patch.Published`'s existing manifest format) | Node creation (non-stub, non-schema) from `patch.Published` or the node's own already-present value (research.md D11); or first merge that finds it previously zero (research.md D3) | Never overwritten once non-zero (first-writer-wins, never flagged as a conflict) |
| `indexed` | `Node.Attrs["indexed"]` (plain string) | RFC 3339 (`time.RFC3339`), e.g. `"2026-07-05T14:22:31Z"` | Node creation only (non-stub, non-schema) | Set exactly once, at creation; never touched by any later merge |
| `updated` | `Node.Attrs["updated"]` (plain string) | RFC 3339, identical format to `indexed` | A merge into an already-existing node whose rendered content actually differs from before the merge (research.md D6) | Re-set on every application that actually changes the node; absent from a node never yet actually changed by a merge |

`indexed` and `updated` deliberately stay inside the open `Attrs` map (research.md D4) — they carry no cross-kind merge semantics `internal/core.Merge` needs to know about, unlike `Published`.

## Application Timestamp (ephemeral — a local value, not a persisted type)

One `time.Time` (`time.Now().UTC()`), captured once near the top of `internal/app/graph/service.Apply`, formatted once to a `string` (`time.RFC3339`), and reused verbatim as the value written into every node's `indexed` (on create) or `updated` (on an actually-changed merge) for that single invocation — satisfying spec FR-005/FR-009's "identical for all nodes in the patch" requirement by construction (same string, not independently-equal instants).

## New/changed functions

### `internal/core` (`ast.go`, `markdown.go`, `merge.go`)

```go
// ast.go — Node gains the Published field above; no other struct changes.

// markdown.go
func extractPublished(manifest map[string]any) (time.Time, map[string]any)
// used by ParseNode and parsePatchBody's per-node construction, both of
// which currently build an Attrs map from a raw front-matter/yaml-fence
// map inline — extractPublished removes "published" (decoding it via the
// existing decodeManifestDate) before the remaining keys become Attrs.

func renderAttrYAML(kind Kind, id string, published time.Time, attrs map[string]any) ([]byte, error)
// gains the published parameter (previously (kind, id, attrs)); when
// non-zero, "published" (formatted "2006-01-02") is merged into the same
// sorted-attribute-keys loop that already renders every other Attrs key,
// so its position in the output is indistinguishable from an ordinary
// attribute. Both call sites (renderFrontMatter for RenderNode, and
// RenderPatch's per-node fence construction) pass n.Published.

// merge.go
func mergePublished(existing, incoming time.Time) time.Time
// existing if non-zero, else incoming — never flagged. Called once,
// inside mergeCore (shared by MergeUnion/MergeUnionFirstWriter/
// MergeAppend/MergeValidatedOverwrite). MergeNone's existing early return
// (unmodified) already leaves Published untouched — no change needed
// there.
```

### `internal/app/graph/service` (`apply.go`)

```go
func isStub(node core.Node) bool
// true when Attrs is empty and Text/Notes/HRefs/Edges/Links are all
// empty — the exact shape service/subgraph.go's --stubs flag already
// emits (core.Node{ID, Kind} only, spec 007 FR-017).

func nodeContentChanged(existing, merged core.Node) (bool, error)
// renders both sides via core.RenderNode and compares bytes
// (research.md D6); the single mechanism deciding whether a merge earns
// an `updated` stamp, correct for every MergeOp uniformly including
// MergeNone's already-a-no-op case.

func setAttr(attrs map[string]any, key string, value any) map[string]any
// small nil-safe helper: allocates attrs if nil, sets key, returns it —
// used for both "indexed" and "updated".
```

`Apply`'s own exported signature (and `component.Apply`'s delegator) is unchanged — no new parameter, no injected clock (research.md D5). The per-node loop inside `Apply` gains:

- a `stamp := appliedAt.Format(time.RFC3339)` computed once before the loop starts, alongside the existing phase timers
- on create (`!existed`): if `!isStub(node)`, fill `node.Published` if zero (research.md D11) and `node.Attrs = setAttr(node.Attrs, "indexed", stamp)`
- on merge (`existed`): after `core.Merge` returns `merged`, if `nodeContentChanged(existing, merged)`, `merged.Attrs = setAttr(merged.Attrs, "updated", stamp)`

No change to `ApplyResult`, `component.Apply`'s signature, `port.VCS`, `port.SchemaRegistry`, or any Cobra/`cmd/` wiring — this feature is entirely internal to `internal/core` and `internal/app/graph/service`.

## Validation rules (from spec.md Functional Requirements)

- `published`, once non-zero on a node, MUST NOT be overwritten by any later patch's differing value (FR-010, research.md D3).
- `indexed` MUST be identical across every node created by one `Apply` invocation (FR-005) and MUST NOT be modified by any later merge (FR-006).
- `updated` MUST be identical to that same invocation's `indexed` value (FR-009) and MUST be set if and only if `nodeContentChanged` reports true (FR-007/FR-008).
- A stub node (`isStub` true) and a `_schema/` document (never reaching this code path at all, research.md D8) MUST carry none of the three attributes (FR-002/FR-003).
- `published` MUST survive unchanged through `core.RenderPatch` (FR-011) — guaranteed by construction once `renderAttrYAML` renders it (D2), since `RenderPatch`'s per-node fence and `RenderNode`'s front matter share that one function.
