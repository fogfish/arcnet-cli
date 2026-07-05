# AST Contract Delta: `internal/core` (supersedes the `Node`/`Merge` sections of `specs/003-apply-patch/contracts/ast-contract.md`)

This delta documents the surface changes this feature makes to `internal/core`. Everything in `specs/003-apply-patch/contracts/ast-contract.md` not called out below is unchanged.

## `Node` gains a typed `Published` field

```go
type Node struct {
    ID        string
    Kind      Kind
    Published time.Time `json:"published,omitempty"`
    Attrs     map[string]any
    Text      string
    Notes     string
    HRefs     []Link
    Edges     []Link
    Links     map[string]LinkBlock
}
```

- `Published` is zero (`IsZero()`) when not yet set — a stub node, a `_schema/` document (which never reaches this type via the ordinary node path, research.md D8), or any node predating this feature.
- `ParseNode`/`parsePatchBody` no longer leave a `"published"` front-matter/yaml-fence key in `Attrs` — it is decoded into `Published` instead (research.md D2). Every other unrecognized key is still preserved verbatim in `Attrs` exactly as before (AST invariant 5, unchanged).
- `RenderNode`/`RenderPatch` render `Published` (when non-zero, `"2006-01-02"` format) back into the same sorted-attribute position it would occupy as an ordinary `Attrs` key — on-disk shape is unaffected by the fact that it is now a typed field internally.

## `Merge` — `Published` fills once, first-writer-wins, never flagged

```go
func Merge(existing, incoming Node, op MergeOp, sourceID string) (merged Node, conflicts []string, err error)
```

- Unchanged signature and every existing documented behavior (`specs/003-apply-patch/contracts/ast-contract.md`'s Merge section still applies for `Text`/`Notes`/`Attrs`/`Edges`/`Links`/`HRefs`).
- **New**: `merged.Published` is `existing.Published` if non-zero, else `incoming.Published` — for every op except `MergeNone` (whose existing early return already leaves `Published` untouched, since it returns `existing` verbatim). Never appears in the returned `conflicts` list; `Published` divergence is never a conflict, by design (research.md D3).

## `ParseNode`/`ParsePatch`/`RenderNode`/`RenderPatch` — unchanged signatures, extended behavior

```go
func ParsePatch(r io.Reader) (Patch, error)
func ParseNode(r io.Reader) (Node, error)
func RenderNode(n Node) ([]byte, error)
func RenderPatch(p Patch) ([]byte, error)
```

No signature changes. `ParseNode`'s Node now carries `Published`; `RenderNode`/`RenderPatch` include it in their output whenever non-zero. Round-trip (`ParseNode(RenderNode(n))` reproduces `n`, per the existing lossless-conversion invariant this contract already documents) holds for `Published` the same way it already holds for every other field.
