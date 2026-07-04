# Phase 0 Research: `arc grep`

## D1: Package structure — extends `internal/app/graph`/`cmd/arc/graph`, not a new domain

**Decision**: `arc grep` is a new primary port method on the *existing* `graph` (graph I/O) use-case: `internal/app/graph/kernel/grep.go`, `internal/app/graph/service/grep.go`, `component.go` gains a `Grep(...)` delegator alongside its existing `Apply(...)`, and `cmd/arc/graph/grep.go` sits next to `cmd/arc/graph/apply.go`. No new `internal/app/<domain>` package, no new `cmd/arc/<domain>` package.

**Rationale**: Direct implementation of the user's explicit instruction ("implement arc grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`"). `cmd/arc/graph/apply.go`'s existing `PostRunE` hint already reads `(use "arc grep [<filter>] <pattern>" to fetch content from your graph, ...)` — this feature was already anticipated as a sibling of `apply`, not a new domain the way `arc lint` was.

**Alternatives considered**: A new `internal/app/query`/`internal/app/nav` domain, matching how `arc lint` got its own package — rejected: `arc lint`'s own domain got a dedicated package because it is a distinct *validation* concern with its own port (`port.VCS.CommitsMatching`) and its own multi-rule engine; `arc grep` is a straightforward read of the same graph I/O surface `apply` already reads (mount, walk, parse front-matter), with no port of its own (D13), so folding it into `graph` avoids a fifth/sixth package that would own nothing `graph` doesn't already own. The user's own instruction is explicit on this point regardless.

## D2: `internal/pkg/grep` — a new, dependency-free, `fs.FS`-based content-search library

**Decision**: The reusable, performance-tuned line-matching engine lives at `internal/pkg/grep`, matching ADR 001's own documented "evolution of domain logic" phase 2 ("`/internal/pkg`: further evolution of core types materializes into a stricter definition of applicability boundaries... a self-contained Go module... constrained to dependencies on open-source modules only"). It is the first package in this codebase to occupy that tier. Its public surface takes an `fs.FS` (specifically `fs.FS` + `fs.ReadDirFS`, the read half of `internal/adapter/fsys.Store`) rather than a root directory string, and never imports `os`.

**Rationale**: Constitution Principle VII is an absolute, codebase-wide rule, not a per-feature choice: *"The `os` package's file/directory functions... MUST NOT appear anywhere outside `internal/adapter/fsys` — that package is the sole place they are permitted."* A brief from the user asking for "a parallel walker of directory traversal" and "close files after processed" reads, on its face, like direct `os.ReadDir`/`os.Open` calls — that reading would violate Principle VII. The resolution is architectural, not a scope cut: `internal/pkg/grep` gets everything it needs (concurrent directory listing, concurrent file open/read/close) through the stdlib `io/fs` interfaces alone, and the caller (`internal/app/graph/service.Grep`) passes it the already-mounted `fsys.Store` — which already implements `fs.FS`, `fs.StatFS`, and `fs.ReadDirFS` (`internal/adapter/fsys/types.go`) — with zero adaptation glue. This is a case the constitution's own Principle I anticipates directly: *"if a plan or spec appears to contradict an ADR [or constitution rule], the conflict MUST be raised explicitly and resolved via a new ADR [or, here, a compliant design], not by quietly diverging."* No new ADR is needed here because the constitution's existing rule already has a clean, compliant satisfaction — an `fs.FS`-based design — that fully delivers every one of the user's performance requirements without a single `os.*` call.

**Alternatives considered**: Giving `internal/pkg/grep` its own direct filesystem access (`os.Open`/`os.ReadDir`) as a second sanctioned exception to Principle VII, alongside `internal/adapter/fsys` — rejected outright: the constitution names `internal/adapter/fsys` as *the sole* place, with no feature-level exception carve-out; a second package touching `os` directly is exactly the drift Principle VII exists to prevent, and would also make `internal/pkg/grep` untestable against an in-memory `fstest.MapFS`, undermining its own stated goal of being a generic, reusable library.

## D3: Concurrency design — one bounded pool, no third-party dependency

**Decision**: `grep.Search` walks and scans using **one** bounded worker pool (a buffered `chan struct{}` used as a semaphore, sized `Options.Workers`, default `8`) shared by both directory-traversal and file-scanning work: `fs.ReadDir` at each directory spawns one goroutine per subdirectory (each acquiring/releasing the same semaphore before recursing) and one goroutine per matched file (each acquiring/releasing the semaphore before scanning), coordinated by a single `sync.WaitGroup`. No `golang.org/x/sync/errgroup` or other third-party concurrency helper is added.

**Rationale**: This single mechanism satisfies all three of the user's distinct bullet points ("parallel walker of directory traversal", "parallel file processing", "a bounded worker pool... default is 8") as one coherent design rather than three separately-justified pieces: the walker and the scanner are symmetric participants in the same bounded pool, so the *total* concurrent filesystem work (listings + open file handles) never exceeds `Workers`, which is what "bounded" is actually protecting (file descriptors and CPU, not "walking" and "scanning" as artificially separate budgets). A directory-listing goroutine holds its semaphore slot only for the synchronous duration of `fs.ReadDir` plus dispatching its children as new goroutines — it does not block waiting for those children, so it cannot deadlock against them even at `Workers == 1` (this degrades gracefully to fully sequential, still-correct traversal). Avoiding `golang.org/x/sync` keeps `internal/pkg/grep` a genuinely zero-dependency library (Principle V/YAGNI; ADR 001 phase 2 permits open-source dependencies but does not require them, and a recursive semaphore-bounded fan-out is a well-understood, small amount of code against the stdlib alone).

**Alternatives considered**: Two separate pools (one for directory listing, one for file scanning) — rejected: needlessly splits one resource budget into two arbitrary ones, and the user's spec names a single worker count, not two. `golang.org/x/sync/errgroup` with `SetLimit` — rejected: a real, if small, new dependency for a problem the stdlib already solves cleanly; revisit only if a second consumer of the same pattern makes the boilerplate cost concrete.

## D4: Literal-vs-regex dispatch

**Decision**: `grep.Search` classifies `pattern` once, before scanning any file: if `regexp.QuoteMeta(pattern) == pattern` (the pattern contains no regex metacharacter), it compiles nothing and matches every line with `bytes.Contains`/`bytes.Index` against `[]byte(pattern)`. Otherwise it compiles a `*regexp.Regexp` once (not per file, not per line) and matches with `re.FindIndex`.

**Rationale**: Directly satisfies "literal search with `bytes.Contains` when possible... regex only when the query actually requires it," and is behavior-preserving: a metacharacter-free string matches identically whether treated as a literal substring or compiled as a regexp, so this is a pure performance optimization invisible to `arc grep`'s own contract (spec FR-001 — `<pattern>` is always regex semantics from the caller's point of view). `bytes.Contains` avoids both the compilation cost and the heavier per-match state `regexp.Regexp` carries, which matters because it runs once per line of every scanned file.

**Alternatives considered**: Always compiling `pattern` as a regexp — rejected: measurably slower for the common case (a plain word or phrase search), which is the majority of real `arc grep` invocations per spec User Story 1. Asking the user to opt into literal mode with a flag (e.g. `--fixed-strings`, matching `grep -F`) — rejected as unnecessary surface: the classification is exact and free, so no user-facing flag is needed to get the fast path automatically.

## D5: Buffered reads and buffer reuse

**Decision**: A `sync.Pool` holds `*bufio.Reader` values (default 64KiB internal buffer). Each file-scanning goroutine takes one from the pool, calls `.Reset(f)` to rebind it to the newly-opened `fs.File` with no new allocation, reads the file line-by-line with `ReadBytes('\n')`, then calls `.Reset(nil)` (dropping the reference to the now-closed file) before returning the reader to the pool. The file itself is closed (`defer f.Close()`) as soon as that goroutine's scan finishes.

**Rationale**: Directly satisfies "buffered reads (`bufio.Reader`)", "buffer reuse with `sync.Pool` (minimize memory allocation)", and "close files after processed." Resetting to `nil` before `Put` avoids pinning a closed `fs.File` (and the memory it may reference) in the pool between uses.

**Alternatives considered**: `bufio.Scanner` with a pooled backing buffer — rejected: the user's instruction names `bufio.Reader` specifically, and `ReadBytes('\n')` gives direct access to line boundaries (needed for 1-based line-number bookkeeping) without `Scanner`'s own token-size-limit configuration surface.

## D6: Result shape, ordering, and partial-failure handling

**Decision**: `grep.Search` returns `(Result, error)` where:

```go
type Result struct {
    Matches    []Match  // sorted by (Path, Line) before return
    Unreadable []string // paths that could not be opened/read; scan continued
}
```

A per-file open/read failure is recorded in `Unreadable` and does **not** abort the run. `Search`'s own `error` return is reserved for conditions that mean *no* scanning could meaningfully happen at all: an invalid `pattern` (fails to compile as a regexp, D4) or the root directory itself failing to list. `Matches` is sorted once, after all workers finish, since goroutine completion order is not deterministic and both scriptability (spec SC-005) and deterministic tests need a stable order.

**Rationale**: Directly implements spec FR-008 ("invalid pattern MUST be reported... without scanning any nodes") and FR-012 ("an individual node file cannot be read... continue scanning the remaining nodes"). Sorting is cheap relative to the scanning work it follows and removes an entire class of flaky-order test failures.

**Alternatives considered**: Streaming matches back through a channel as they're found instead of collecting and sorting — rejected: `arc lint`'s own `LintResult` already accumulates its full result in memory before rendering (`internal/app/lint/service/lint.go`), and this feature's own SC-004 performance budget (several thousand nodes, under 10s) does not require a streaming design; consistency with the existing accumulate-then-render pattern (ADR 002 DS-04) was preferred over a bespoke streaming path for one command.

## D7: `Options.Include` — how filtering reaches the content scan without leaking domain concepts into `internal/pkg/grep`

**Decision**: `grep.Options` gains one field, `Include func(path string) bool` (optional — `nil` means "scan every file with the matching extension"). The walker calls `Include(path)` (a cheap, already-computed map lookup) before submitting a file to the worker pool for content scanning, so files excluded by a filter never pay the line-scanning cost at all.

**Rationale**: Keeps `internal/pkg/grep` fully domain-agnostic ("treat files as plain text within the lib" — no `core.Node`, no YAML, no `kind`/`tag`/`attr` vocabulary anywhere in this package), while still letting the caller (`internal/app/graph/service.Grep`, D9) avoid scanning content that a `Filter` (D8) has already excluded, which matters for the same reason D3's bound matters: fewer open file handles and less CPU spent on excluded nodes, without teaching a generic library what a "node" is.

**Alternatives considered**: `grep.Search` accepting an explicit `[]string` of paths to scan instead of walking and filtering itself — rejected: the user's own requirement is that this package *owns* the parallel directory walk; handing it a pre-computed path list would move that walk back out to the caller, defeating the point of D2/D3.

## D8: `internal/core.Filter` — a new, shared node-selection type

**Decision**: A new file, `internal/core/filter.go`, adds:

```go
type Filter struct {
    Kinds        []Kind
    Tags         []string
    Attrs        map[string]string
    AttrPatterns map[string]*regexp.Regexp
}

func (f Filter) Match(node Node) bool
```

`Match` implements VISION.md's Filtering section exactly: `Kinds` is OR'd (a node matches if its `Kind` is in the set; empty set matches every kind), `Tags` is AND'd (every listed tag must be present, read from `node.Attrs["tags"]`, the existing on-disk convention already exercised by `internal/core/merge_test.go`), `Attrs` is AND'd exact-match (case-insensitive for a scalar, membership test for an array attribute), `AttrPatterns` is AND'd regexp-match (same scalar/array distinction). A zero-value `Filter{}` matches every node.

**Rationale**: VISION.md's Filtering section is explicit that this is a *shared* concept: "Several commands accept a filter to narrow the set of nodes they operate on" — `arc list`, `arc popular`, `arc orphans`, `arc relink`, `arc export json`/`dot`, and `arc grep` (this feature) are all specified with the identical `[<filter>]` syntax. `arc grep` is the first of these to actually ship, so this is the first opportunity to place the type correctly rather than accreting it privately inside one command's package. `internal/core` (not `internal/app/graph`) is the right home because `Filter.Match` operates purely on `core.Node`/`core.Kind`, has no dependency on any single use-case's port/service/kernel, and constitution Principle II explicitly favors elevating a genuinely cross-cutting type into a shared package the first time a second consumer is anticipated, rather than waiting for an actual second caller and then having to migrate it out of `internal/app/graph` later (which would be a breaking change for whichever future command's `internal/app/<domain>` copied it in the meantime).

**Alternatives considered**: Defining `Filter` privately inside `internal/app/graph/kernel`, since that is where `arc grep` itself lives — rejected: the moment `arc list` (or any other Filtering-section command) ships, it would either import `internal/app/graph`'s private kernel type (a cross-use-case import ADR 001's port-isolation rules forbid for `port/` and, by the same spirit, for use-case-private `kernel` types) or duplicate the type, which Principle V calls out by name as "no duplicate, divergent implementations of the same capability."

## D9: Front-matter enumeration in `service.Grep` — one pass serves both labeling and filtering

**Decision**: `internal/app/graph/service.Grep` walks every node file the same way `internal/app/lint/service.walkNodeFiles` does (recursive, excluding `.arc/` and `_schema/`, sorted), and for each file reads its raw bytes and runs `core.ParseNode` once. This single pass produces two things simultaneously: (1) a `path → core.Node` index used to label every match's `kind`/`id` columns (required for *every* match, filtered or not — the output format itself needs it), and (2) the `Filter`-membership set (D8) that becomes `grep.Options.Include` (D7). A file that cannot be opened, or that fails to parse as a node at all, is added to the result's `Unreadable` list and excluded from the scan — it can never be labeled meaningfully, so it cannot appear in `Matches` either way (spec Edge Cases: "the same node's `<id>` cannot be determined... that file is excluded from the scan").

**Rationale**: The output contract (`<kind> <id> <line> <text>`, spec FR-004) requires every match to carry a `kind`/`id`, whether or not a filter narrows the result — so this pass is not an optional filtering optimization, it is a hard requirement of the format itself, run once regardless of whether `--kind`/`--tag`/`--attr` were given. Reusing `core.ParseNode` (already proven, already the same parser `arc apply`/`arc lint` use) means node identity/kind extraction never has a second, divergent implementation.

**Consequence, documented not hidden**: a node file's bytes are technically read twice on a scan that includes it — once whole-file via this enumeration pass (`core.ParseNode` needs the complete front-matter+body), once again streamed via `grep.Search`'s own `bufio.Reader` (D5). This is a deliberate simplicity-over-micro-optimization trade (Principle V): it keeps `internal/pkg/grep` fully decoupled from `core.Node`/YAML (D2/D7), and the extra read is cheap against SC-004's target (several thousand nodes, under 10 seconds) — a single additional sequential pass over the same files `arc lint` already performs today.

**Alternatives considered**: Teaching `internal/pkg/grep` to also parse front-matter so a single read serves both purposes — rejected: this is exactly the boundary D2/D7 exist to hold; a generic, potentially-reusable content-search library must not gain a YAML/Markdown-node-shape dependency for one caller's labeling need.

## D10: `.arc/config.yml` — first real use of the previously-dormant `Config` struct

**Decision**: `internal/app/config/kernel.Config` (currently `struct{}`, ARCHITECTURE.md: "dormant, zero callers... kept alive for a future, unrelated configuration need") gains its first real field:

```go
type Config struct {
    Grep GrepConfig `yaml:"grep,omitempty"`
}

type GrepConfig struct {
    Workers      int `yaml:"workers,omitempty"`
    MaxLineWidth int `yaml:"maxLineWidth,omitempty"`
}
```

A zero `Workers`/`MaxLineWidth` (field absent, or `.arc/config.yml` absent entirely) means "use the built-in default" (`8` workers, `80` columns) — resolved once in `cmd/arc/graph/grep.go` at wiring time, not inside `internal/app/config` itself (that package's contract is purely load/save, per its own `component.go`).

**Rationale**: Directly implements the user's "number of workers configurable via `.arc/config` default is 8" and "matched line longer than 80 chars (configurable via `.arc/config`)". `.arc/config.yml` (not a bare `.arc/config`) is this codebase's one existing configuration file — `kernel.ConfigPath` — so this is a naming clarification, not a new file format.

**Alternatives considered**: A grep-private config file or flag-only configuration (`--workers`, `--max-width`) — rejected: the user's instruction names `.arc/config` explicitly, and reusing the existing, already-wired `internal/app/config.Load`/`Save` avoids inventing a second configuration mechanism for a single command (constitution Principle XI's configuration-precedence rule already covers `.arc/config.yml`).

## D11: Highlighting and line-fitting are presentation-only, gated on the same signal as color

**Decision**: `bios.Schema` (ADR 002 DS-05) gains one new field, `Match lipgloss.Style`, present in both `SCHEMA_PLAIN` (`lipgloss.NewStyle()`, i.e. a no-op) and `SCHEMA_COLOR` (e.g. bold + a distinct foreground color). `cmd/arc/graph/grep.go`'s `Human`/`Verbose` printers apply two purely-presentational transforms to each `kernel.Match.Text`, both gated on **`bios.SCHEMA` being the color schema** — the exact same signal ADR 002 DS-05's `SelectSchema` already resolves once, at startup, from TTY/`NO_COLOR`/`TERM=dumb`/`--color`:
1. Wrap the byte range `[Start:End)` in `SCHEMA.Match.Render(...)`.
2. If `len(Text) > MaxLineWidth` (D10), replace the portion of the line outside a window centered on `[Start:End)` with a leading and/or trailing `…`, so the match itself always remains visible within roughly one terminal line.

When `SCHEMA == SCHEMA_PLAIN` (piped output, `NO_COLOR`, `TERM=dumb`, non-TTY with no `--color`), neither transform runs — the full, untruncated, unstyled line is printed.

**Rationale**: The user's own instruction frames both requirements in terminal-display terms ("use colors to highlight... if enabled", "fit the match roughly to one terminal line") — both are about how a match reads on an interactive terminal, not about the data itself. Gating truncation on the same signal as color (rather than inventing a second TTY check) means a script piping `arc grep`'s output never silently loses characters from a long matched line — preserving spec FR-006/FR-007 ("no header/footer... whitespace-delimited so standard tools can parse it") and SC-005 exactly, while still giving an interactive user the requested compact, colorized view. This is a direct application of Principle III ("formatting is presentation, not domain logic") — `kernel.Match.Text` is always the untruncated raw line; only `cmd/arc/graph/grep.go`'s renderer ever shortens or colors it.

**Alternatives considered**: Truncating unconditionally in `Human` mode regardless of TTY — rejected: breaks piping (the stated purpose of the whole command) the moment a matched line exceeds the configured width, which is exactly the failure mode spec Edge Cases and SC-005 guard against. A second, independent `isTTY` check inside `grep.go` — rejected: `bios.SelectSchema` already resolves this exact question once, at root command setup; re-deriving it a second time inside one command's renderer would be a second, potentially-divergent TTY-detection site (Principle V).

**Known limitation, documented not hidden**: when `pattern` matches more than once on the same line, only the *first* occurrence's span is highlighted/used as the truncation anchor (`kernel.Match` carries a single `Start`/`End` pair, not a list) — sufficient for a human to locate the match, consistent with spec FR-005's "report that line once, not once per match within the line," and avoiding a variable-length highlight-span list in the domain value for a cosmetic-only refinement.

## D12: Exit-code convention — corrected to match this codebase's existing `bios.ErrSilent` pattern

**Decision**: `cmd/arc/graph/grep.go`'s `RunE` always prints its full result (human/verbose/JSON) first, then:
- Returns `nil` when at least one match was found.
- Returns `bios.ErrSilent` (exit `1`, no second error line — DS-07, identical to `arc lint`'s existing use for "violations found") when the run completed cleanly but zero matches were found — whether because `pattern` matched nothing or because a `Filter` matched zero nodes.
- Returns a real `error` (exit `1`, *with* a human-readable banner via `cmd/arc/main.go`'s single `Execute()`-adjacent formatting site) only for a genuine refusal to run at all: an invalid `pattern`, or the target not being an initialized graph.

**Rationale**: spec.md's Assumptions section, as first drafted during `/speckit-specify`, described "the conventional three-way split used by standard search tools" (match/no-match/error as three distinct exit codes) — that does not match how any existing command in this codebase signals outcomes; `arc lint`/`arc apply` both use exactly this two-way `bios.ErrSilent`-vs-real-error split, and ADR 002 DS-07 reserves a third, distinct exit code for "a meaningfully distinct failure class," set explicitly in `PostRunE` — introducing one here for "zero matches" (which is not a failure, just an empty result) would be a novel pattern with no precedent anywhere else in the CLI. The spec's Assumptions bullet was corrected in place during this planning phase (constitution Principle I: a conflict between a spec artifact and established, binding convention must be resolved, not silently diverged from) to describe the two-way split precisely: "ran, found nothing" is distinguished from "refused to run" by the presence of an error message, not a third exit-code value — matching every other command in this CLI.

**Alternatives considered**: Implementing a genuine three-way exit code (`0`/`1`/`2`) via an explicit `PostRunE`-set code — rejected: no existing command in this codebase does this, and DS-07 explicitly scopes that mechanism to "a meaningfully distinct failure class," which "zero matches" is not (it is `arc lint`'s "violations found" case, not its "target not a graph" case). Introducing it here first, for one command, without a documented cross-cutting need, would be exactly the kind of scope creep Principle V/YAGNI guards against.

## D13: No new port

**Decision**: `internal/app/graph/service.Grep` takes only `fsys.Mounter` (plus the already-in-hand `core.Filter`, pattern string, and resolved `kernel.GrepConfig`) — no `port.VCS`, no `port.SchemaRegistry`.

**Rationale**: `arc grep` never touches git history and never registers a kind or predicate — it is a pure read of already-parsed node content, the narrowest possible dependency set, following the same "a port declares only what the use-case actually needs" rule (ADR 001 port isolation rule 2) that already scoped `lint.port.VCS` down to one method versus `graph.port.VCS`'s larger surface.

## D14: Filter flags stay local to `cmd/arc/graph/grep.go` for now

**Decision**: The `--kind`/`--tag`/`--attr` options struct (ADR 002 DS-02) is defined, unexported, inside `cmd/arc/graph/grep.go` itself — not promoted to a shared `cmd/`-level location yet.

**Rationale**: `arc grep` is the first command in this codebase to actually implement VISION.md's Filtering section; DS-02 explicitly permits (but does not require) reusing one options struct across multiple commands, and Principle V/YAGNI favors waiting for an actual second consumer (the next Filtering-section command to ship) before extracting a shared location, rather than guessing its shape now. `internal/core.Filter` itself (D8) is already the reusable, tested part — the CLI-flag-parsing wrapper around it is cheap to promote later and not worth generalizing speculatively today.
