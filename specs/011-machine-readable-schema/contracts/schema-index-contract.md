# Contract: `core.Index` and `internal/app/schema` Go API

Supersedes `core.MergeRuleSet`/`map[string]bool` as the shape threaded through `arc apply`/`arc lint` (research.md D1, D8, D9).

## `internal/core` additions (`rules.go`)

```go
type PredicateDef struct {
    Role        string
    Merge       MergeOp
    Label       string
    Aligned     string
    Description string
}

type TypeDef struct {
    Merge       MergeOp
    Required    []string
    Optional    []string
    Description string
}

type Index struct {
    Predicates map[string]PredicateDef
    Types      map[string]TypeDef
}
```

`MergeRuleSet`, `.Lookup`, `.Union` are removed (research.md D9) — `Index.Types[name].Merge` plus a plain map presence check (`_, ok := index.Types[name]`) replaces `MergeRuleSet.Lookup(name)`.

## `internal/app/schema` (kernel + service)

```go
// kernel/schema.go
const (
    PredicatesDir = "_schema/predicates"
    TypesDir      = "_schema/types" // renamed from NodesDir = "_schema/nodes"
)

var CorePredicateDefs map[string]core.PredicateDef // full CORE §10 vocabulary
var CoreTypeDefs      map[string]core.TypeDef      // source/entity/resource/timeline + Property/Class

// service/schema.go
func Seed() map[string][]byte
func Resolve(store fsys.Store) (core.Index, error)
func RegisterType(store fsys.Store, typ string) (created bool, err error)      // renamed from RegisterKind
func RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
```

- `Seed()`: pure, no I/O, renders every `CorePredicateDefs`/`CoreTypeDefs` entry as a conformant Property/Class node (schema-document-contract.md), keyed by on-disk path. Panics on a render failure exactly as today (a built-in, always-valid value failing to render is a programming error, not a runtime condition).
- `Resolve(store)`: first checks `.arc/` presence (research.md D2) — absent yields the existing "not a graph" error family; present but `_schema/predicates/`/`_schema/types/` absent, or containing any document failing the read contract in schema-document-contract.md, yields a new `ErrSchemaMissing`/`ErrSchemaInvalid` (`faults.Type`/`faults.SafeN`, naming the file and field) before any command using the result makes a change. Never returns a partially-populated `Index`.
- `RegisterType`/`RegisterPredicate`: idempotent (`registerIfAbsent`, unchanged precedent) — writes only when absent, using the auto-registration shapes in schema-document-contract.md.

## `internal/app/schema.Component` (primary port, unchanged shape beyond the rename)

```go
func Seed() map[string][]byte
func Resolve(store fsys.Store) (core.Index, error)

type Component struct{}
func (Component) RegisterType(store fsys.Store, typ string) (created bool, err error)
func (Component) RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
```

## Consumers

```go
// internal/app/graph/port/schema.go
type SchemaRegistry interface {
    RegisterType(store fsys.Store, typ string) (created bool, err error) // renamed
    RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
}

// internal/app/graph/service/apply.go
func Apply(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter,
    index core.Index, schema port.SchemaRegistry, dir, patchPath string) (kernel.ApplyResult, error)

// internal/app/lint/service/lint.go
func Lint(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter,
    index core.Index, dir string) (kernel.LintResult, error)
```

`cmd/arc/graph/apply.go` and `cmd/arc/lint/lint.go` each call `appschema.Resolve(store)` once and pass the resulting `core.Index` straight through — no change to either command's own flag/output surface.
