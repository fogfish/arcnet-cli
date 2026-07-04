# Phase 1 Data Model: `arc grep`

Value types are immutable (constitution Principle IV) and carry no Cobra, `os/exec`, raw `os.*` filesystem, or goldmark AST types.

## Shared core (`internal/core`)

### Filter (new, research.md D8)

Node-selection criteria shared by every `[<filter>]`-accepting command (VISION.md Filtering); `arc grep` is the first consumer.

| Field | Type | Semantics |
|---|---|---|
| `Kinds` | `[]Kind` | Empty matches every kind; otherwise OR — a node matches if `Kind` is any listed value |
| `Tags` | `[]string` | Empty matches every node; otherwise AND — every listed tag must be present in `node.Attrs["tags"]` |
| `Attrs` | `map[string]string` | Empty matches every node; otherwise AND — `name=value`, case-insensitive equality for a scalar attribute, membership test for an array attribute |
| `AttrPatterns` | `map[string]*regexp.Regexp` | Empty matches every node; otherwise AND — `name~=pattern`, regexp match against a scalar, or against any element of an array attribute |

`Filter{}` (zero value) matches every node — mirrors VISION.md's "an absent or empty filter object matches all nodes" exactly. `Filter.Match(node Node) bool` is the sole exported behavior; no method mutates `Filter` or `Node`.

## `internal/pkg/grep` (new, research.md D2-D7)

Domain-agnostic content-search library — no dependency on `internal/core`, `internal/app/*`, or any use-case's vocabulary.

### Match

| Field | Type | Notes |
|---|---|---|
| `Path` | `string` | `fs.FS`-relative path (`fs.ValidPath` form: relative, `/`-separated, no leading/trailing slash) |
| `Line` | `int` | 1-based line number within `Path` |
| `Text` | `string` | Full line text, no trailing newline |
| `Start` | `int` | Byte offset within `Text` where the (first) match begins |
| `End` | `int` | Byte offset within `Text` where the (first) match ends |

### Options

| Field | Type | Notes |
|---|---|---|
| `Extension` | `string` | Required file suffix, e.g. `".md"`; empty defaults to `".md"` |
| `Workers` | `int` | Bounded pool size (research.md D3); `<= 0` defaults to `8` |
| `Include` | `func(path string) bool` | Optional (research.md D7); `nil` means "scan every file matching `Extension`" |

### Result

| Field | Type | Notes |
|---|---|---|
| `Matches` | `[]Match` | Sorted by `(Path, Line)` before return (research.md D6) |
| `Unreadable` | `[]string` | Paths that could not be opened/read; scan continued for the rest |

### Search

```go
func Search(ctx context.Context, fsys fs.FS, pattern string, opts Options) (Result, error)
```

`fsys` must implement `fs.FS` and `fs.ReadDirFS` (`internal/adapter/fsys.Store` already does, unmodified). `error` is non-nil only for: `pattern` failing to compile as a regexp (research.md D4/D6), the root directory itself failing to list, or `ctx` cancellation — never for a single file's read failure (that goes into `Result.Unreadable`).

## Application values (`internal/app/graph/kernel`)

### Match (new)

One reported line, in one node's file, that matched `arc grep`'s pattern — the row `cmd/arc/graph` renders as `<kind>  <id>  <line>  <text>`.

| Field | Type | Notes |
|---|---|---|
| `Kind` | `core.Kind` | The owning node's kind |
| `ID` | `string` | The owning node's parsed identity (basename-derived, same as `core.Node.ID`) |
| `Path` | `string` | Node file path, relative to the graph root |
| `Line` | `int` | 1-based line number within `Path` |
| `Text` | `string` | Full, untruncated, unstyled matched line — presentation (highlighting/truncation) is applied only in `cmd/arc/graph/grep.go` (research.md D11), never here |
| `Start` | `int` | Byte offset within `Text` where the match begins (carried through from `grep.Match`) |
| `End` | `int` | Byte offset within `Text` where the match ends |

### GrepResult (new)

The domain value `component.go`'s `Grep` returns to `cmd/arc/graph`, rendered by `bios.Registry[GrepResult]`.

| Field | Type | Notes |
|---|---|---|
| `Root` | `string` | The graph root that was searched |
| `Pattern` | `string` | The regexp pattern searched for |
| `Matches` | `[]Match` | Every match found, across every node passing `Filter` (empty when nothing matched — spec FR-009) |
| `Unreadable` | `[]string` | Node files that could not be read or parsed and were excluded from the scan (spec FR-012, Edge Cases) |

`cmd/arc/graph/grep.go` derives its exit signal (research.md D12) from `len(GrepResult.Matches) == 0`, exactly as `cmd/arc/lint` derives its own exit signal from `len(LintResult.Violations)`.

## Configuration (`internal/app/config/kernel`)

### Config (extended — first real field, research.md D10)

| Field | Type | Notes |
|---|---|---|
| `Grep` | `GrepConfig` | `yaml:"grep,omitempty"` |

### GrepConfig (new)

| Field | Type | Notes |
|---|---|---|
| `Workers` | `int` | `yaml:"workers,omitempty"`; `<= 0` (including absent) resolves to the built-in default `8` |
| `MaxLineWidth` | `int` | `yaml:"maxLineWidth,omitempty"`; `<= 0` (including absent) resolves to the built-in default `80` |

Resolution (zero → default) happens once, in `cmd/arc/graph/grep.go`, immediately after `internal/app/config.Load` — `internal/app/config`'s own `Load`/`Save` contract is unchanged (still a pure YAML round-trip, no defaulting logic inside that package).

## Presentation (`internal/bios`)

### Schema (extended, research.md D11)

| Field (new) | Type | `SCHEMA_PLAIN` | `SCHEMA_COLOR` |
|---|---|---|---|
| `Match` | `lipgloss.Style` | `lipgloss.NewStyle()` (no-op) | e.g. `lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))` |

No other `Schema` field changes. `SelectSchema`'s TTY/`NO_COLOR`/`TERM=dumb`/`--color` resolution logic is unchanged and is the exact signal research.md D11 reuses for gating truncation.

## Ports

None. `internal/app/graph/service.Grep` depends only on `fsys.Mounter` (research.md D13) — no `port.VCS`, no `port.SchemaRegistry`.

## Filesystem I/O

All reads go through `fsys.Store` (`internal/adapter/fsys`, unchanged). `arc grep` mounts the graph root the same way `arc apply`/`arc lint` do and uses the identical `Store.Stat(".arc")` guard (`guardIsGraph`). `arc grep` never calls `Store.Create`, `Store.Remove`, or any `File` write method — strictly read-only, like `arc lint`.

## Validation rules (from spec Functional Requirements)

| Rule | Source | Enforced in |
|---|---|---|
| `<pattern>` required, regexp semantics | FR-001 | `cobra.ExactArgs(1)` in `cmd/arc/graph/grep.go`; `grep.Search`'s classification (research.md D4) |
| No filter ⇒ every node scanned | FR-002 | `service.Grep`'s enumeration pass (research.md D9) builds an all-nodes `Include` when `Filter{}` |
| Filter ⇒ only matching nodes scanned, VISION.md semantics | FR-003 | `core.Filter.Match` (research.md D8), consumed as `grep.Options.Include` (research.md D7) |
| One output line per matching line, `kind`/`id`/`line`/`text` | FR-004 | `kernel.Match` shape; `cmd/arc/graph`'s Human/Verbose printers |
| Every matching line reported, never stop at first, never double-report | FR-005 | `grep.Search`'s per-line scan (one `Match` per matching line, research.md D5/D6) |
| One match per output line, stable field order | FR-006 | `kernel.Match`/renderer shape — no header/footer text interleaved |
| No header/footer/summary mixed into matches | FR-007 | `cmd/arc/graph`'s Human/Verbose printers emit only match rows (+ `bios.Reporter`/hint on `stderr`, never `stdout`) |
| Invalid pattern ⇒ clear error, no scan | FR-008 | `grep.Search`'s upfront classification/compile step (research.md D4/D6) returns a hard error before any file is opened |
| Zero matches ⇒ clean exit, distinguishable from error | FR-009 | `cmd/arc/graph/grep.go`'s `bios.ErrSilent` convention (research.md D12) |
| Read-only, no graph/git mutation | FR-010 | Structural — `service.Grep` never receives a write-capable dependency |
| Refuse when target is not an initialized graph | FR-011 | `service.Grep`'s `guardIsGraph`, before enumeration begins |
| Unreadable file ⇒ reported, scan continues | FR-012 | `service.Grep`'s enumeration pass + `grep.Search`'s `Result.Unreadable` (research.md D6/D9), merged into `GrepResult.Unreadable` |
