# Phase 0 Research: `arc apply`

## D1: Package structure — `internal/core`, `internal/app/graph`, `internal/app/config`

**Decision**: Three new packages, plus one promotion of existing code:

- **`internal/core`** — the graph AST (ARCNET-AST §4-6) as plain Go types (`Node`, `Link`, `LinkBlock`, `Patch`), the CORE §9/§10 kind/merge-operation vocabulary (`Kind`, `MergeOp`, `CoreMergeRules`, `KnownProfileMergeRules`, `MergeRuleSet`), the goldmark-backed Markdown↔AST codec, the CORE §10 merge algebra, and CORE §9.4 timeline-period derivation. Per the user's instruction ("Define the graph AST ... as core domain") and ADR 001's own evolution model ("`/internal/core`: initially, the domain is a solid part of the application — a collection of core types... allowed dependencies on themselves or on open-source modules only").
- **`internal/app/graph`** — the graph-mutation use-case (mirrors `internal/app/ctrl`'s `kernel/port/service/component.go` layout). Houses `Apply` now; `Retract`/`Reapply`/batch apply are separate, later VISION.md commands that will live in the same package per the user's "graph I/O is own domain" framing.
- **`internal/app/config`** — the `.arc/config.yml` load/save/resolve use-case, structured the same way, minus a `port/` package (see D5).
- **Promotion**: `internal/app/ctrl/adapter/git` → `internal/adapter/git` (shared). See D4.

**Rationale**: `internal/core` is the natural home for the AST *and* its canonical serialization, because ARCNET-AST §3.6 declares lossless Markdown⇄model⇄patch conversion a *model invariant*, not an apply-specific concern — every future graph-reading command (`lint`, `index build`, `retract`) needs the same parser, so it belongs below the use-case layer, not inside `internal/app/graph`. `internal/app/graph` and `internal/app/config` follow ADR 001's `componentX` layout exactly, matching `internal/app/ctrl`'s existing precedent.

**Alternatives considered**: A `port.PatchParser` interface inside `internal/app/graph`, implemented by a goldmark adapter — rejected. Ports exist to isolate *variable or live* dependencies (filesystem, subprocess, network) so they can be faked in unit tests; goldmark is a pure, deterministic, in-process parser with no I/O of its own, so hiding it behind an injectable port buys no testability that a plain exported function (`core.ParsePatch(io.Reader) (Patch, error)`) does not already have — same reasoning that keeps `encoding/json` un-ported elsewhere in Go code. Keeping the AST and its parser inside `internal/app/graph` instead of `internal/core` — rejected per the lossless-conversion/multi-consumer argument above.

## D2: `internal/core` AST shape (ARCNET-AST §4-6)

**Decision**:

```go
type Kind string
type MergeOp string

const (
    MergeNone              MergeOp = "none"
    MergeUnion             MergeOp = "union"
    MergeUnionFirstWriter  MergeOp = "union-first-writer"
    MergeAppend            MergeOp = "append"
    MergeValidatedOverwrite MergeOp = "validated-overwrite"
)

type Link struct {
    Predicate string
    Target    string
    Alias     string
}

type LinkBlock struct {
    Title string // display heading, e.g. "Mentions" — derived, not independently stored (AST §4)
    Seq   []Link
}

type Node struct {
    ID    string
    Kind  Kind
    Attrs map[string]any // front-matter scalars, excluding kind (AST §4)
    Text  string
    Notes string
    HRefs []Link
    Edges []Link
    Links map[string]LinkBlock
}

type Patch struct {
    Document  string
    Published time.Time
    Title     string
    Stats     map[string]any
    Nodes     []Node
}
```

**Rationale**: This is a direct, minimal transliteration of ARCNET-AST §4 ("Node Object") and §5 ("Graph") into Go, plus a `Patch` wrapper for CORE §12.2's manifest fields (`kind: patch`, `document`, `published`, recommended `title`/`stats`). `hrefs`/`edges`/`links` are kept as three distinct fields exactly per AST §3 ("A Link is stored on the node for one of three distinct purposes, and never two at once for the same purpose") rather than collapsed into one generic link list, so a future consumer (e.g. `arc lint` checking dangling links) can honor AST invariant 3 ("`hrefs` ... never treated as a source of navigable edges") without re-deriving which list is which.

**Alternatives considered**: Reusing goldmark's own `ast.Node` tree as the application-wide AST — rejected: it is a Markdown-syntax tree (headings, lists, emphasis), not the CORE/AST-spec's domain-level Node/Link model: front-matter, `::` edge syntax, and `[[wikilink]]` targets have no first-class goldmark node types, so a translation layer is unavoidable regardless, and vendor AST types must not leak through the domain boundary (constitution Principle VII's "vendor SDK types MUST NOT leak through port interfaces" applies in spirit even though this is a pure parser, not a port).

## D3: Markdown parsing — goldmark, confined to `internal/core`

**Decision**: `internal/core` depends on `github.com/yuin/goldmark` (with the `github.com/yuin/goldmark-meta` extension for YAML front-matter) internally. `goldmark`'s AST types (`ast.Node`, `text.Reader`, etc.) never appear outside the file(s) implementing `ParsePatch`/`ParseNode`; every exported function in `internal/core` returns only the plain types from D2.

`ParsePatch(r io.Reader) (Patch, error)` walks the parsed Markdown tree once: the front-matter block becomes the manifest fields; each `# <Kind>` H1 heading opens a kind section; each `## <basename>` H2 heading under it opens a node, whose fenced ` ```yaml ` block is unmarshaled into `Attrs`, and whose remaining body (prose, `predicate:: [[Target]]` list/body-form bullets, `## <Predicate>`-headed blocks) is walked to populate `Text`/`Edges`/`Links`/`HRefs` per AST §6. Bullet-level `predicate::` syntax and `[[Target]]`/`[[Target|alias]]` link syntax are not native goldmark constructs, so they are recognized by inspecting each list item's/paragraph's raw text within the already-parsed tree (goldmark still owns list/paragraph/heading structure; only the CORE-specific inline grammar is hand-parsed against goldmark's raw text spans) — this is a bounded, single-purpose regex/scan over each already-isolated inline text span, not a re-implementation of Markdown parsing itself.

**Rationale**: This is exactly what the user's instruction specifies ("Parse the patch using `github.com/yuin/goldmark` markdown parser into AST and then use AST to patch the graph itself"), and matches CORE §12.2's structure (H1 = kind, H2 = identity, fenced yaml block = scalar attrs, remaining body = prose/edges) closely enough that the walk is a direct structural mapping, not a heuristic one.

**Alternatives considered**: A hand-rolled line-oriented parser (no goldmark) — rejected, contradicts the explicit instruction and reimplements Markdown block structure (fenced code blocks, list nesting) goldmark already handles correctly. A full CommonMark-to-AST round-trip preserving every Markdown node — rejected as unneeded: AST §2 states deep Markdown parsing by a consumer is "out of scope and is to be avoided" beyond what resolves to `text`/`Link`/attribute members.

## D4: Git adapter promotion — `internal/app/ctrl/adapter/git` → `internal/adapter/git`

**Decision**: Move the existing git adapter to `internal/adapter/git` (shared, cross-use-case), matching `internal/adapter/fsys`'s precedent. `internal/app/ctrl/port.VCS` (unchanged: `IsAvailable`, `Init`, `StageAll`, `Commit`) and a new, `graph`-private `internal/app/graph/port.VCS` (`IsTracked`, `StageAll`, `Commit` — narrower, `apply` never calls `git init` or re-checks availability since a graph it operates on is already initialized) are both satisfied structurally by the one promoted concrete `git.Git` type — per ADR 001 port isolation rule 1 ("If use-case B needs a behaviour that use-case A's infrastructure provides, use-case B defines its own narrow port interface... and the wiring layer (`cmd/`) connects the two").

`git.Git` gains one new method, `IsTracked(ctx, dir, path string) (bool, error)`, wrapping `git ls-files --error-unmatch <path>` (CORE §11.2's documented idempotency check: exit 0 = tracked, non-zero = not tracked — the method returns `(false, nil)` for the expected "not tracked" exit status, and only a genuine unexpected failure as `(false, err)`).

**Rationale**: `port.VCS`-shaped git access is now needed by two use-cases (`ctrl`, `graph`); ADR 001 itself describes this exact trigger as the adapter's second evolution phase ("further evolution of an adapter causes generalization toward application-level reusability... grouped by technology dependency"). Keeping two separate, narrow port interfaces (rather than widening `ctrl.port.VCS` to cover `IsTracked` too) preserves Interface Segregation — `ctrl.Init` has no reason to depend on an `IsTracked` method it never calls.

**Alternatives considered**: A second, independent git-subprocess adapter under `internal/app/graph/adapter/git` — rejected: a second, divergent client for the same external system is an explicit constitution Principle VII violation ("Before adding a new adapter, verify whether an adapter for that capability already exists... duplicate clients for the same external system are forbidden"). This promotion is flagged as a **Phase 0 (Pre-implementation Refactoring)** task, per the tasks-template's optional phase for "significant changes to existing code... submitted as a separate PR from feature work," since it touches already-shipped `002-arc-init` code with no behavior change of its own.

## D5: `.arc/config.yml` — shape, location, and use-case decoupling

**Decision**: `internal/app/config` depends directly on `fsys.Store` (no private `port/` package needed), the same documented exception `internal/app/ctrl` already established for filesystem access (specs/002-arc-init/research.md D3: "filesystem access is explicitly cross-use-case... the interface belongs at the shared adapter tier, not duplicated per use-case").

```go
// internal/app/config/kernel/config.go
type Config struct {
    MergeRules core.MergeRuleSet // yaml: mergeRules
}

// internal/app/config/component.go
func Resolve(store fsys.Store) (core.MergeRuleSet, error)
func Save(store fsys.Store, cfg kernel.Config) error
```

`Resolve` loads `.arc/config.yml` if present (malformed YAML is a hard error, matching this codebase's existing "refuse rather than guess" posture) and returns `core.CoreMergeRules` unioned with the file's declared rules; if the file is absent, it returns `core.CoreMergeRules` alone — this is what "a graph with no domain kinds registered" (spec User Story 3, Acceptance Scenario 2) means concretely: no config file, or a config file that declares no kind beyond the three built-ins.

Because `internal/core.CoreMergeRules` is a shared, dependency-free domain constant (not a `config`-use-case-owned value), `internal/app/ctrl/kernel.DefaultLayout.MetaStubs` gains one more entry — `core.ConfigPath` (`".arc/config.yml"`) → `core.CoreMergeRules.YAML()` (a marshal helper living in `internal/core` alongside the type, reused unmodified by `config.Save`) — written by `arc init`'s existing `writeLayout` loop with zero new code path. **`internal/app/ctrl` never imports `internal/app/config`** — both independently depend on the shared `internal/core` constant, preserving ADR 001's "use-cases are strictly decoupled... no reference... not even structs or interfaces" rule. `cmd/arc/graph/apply.go` is the one place that calls `appconfig.Resolve(store)` before calling `appgraph.Apply(...)`, passing the resolved `core.MergeRuleSet` in as a plain value — composition happens at the wiring layer, per ADR 001's own guidance for cross-cutting concerns not covered by a shared kernel package.

**Known tension, flagged not silently resolved**: `.arc/` is entirely `.gitignore`d (`specs/002-arc-init`), so `.arc/config.yml` is **local to one clone**, not shared via git the way the graph content itself is. Two collaborators cloning the same graph repository can end up with different locally-registered domain kinds, so the same patch could be accepted by one collaborator's `arc apply` and refused by another's. This is what the user's literal instruction (`.arc/config.yml`) specifies, and is consistent with `.arc/`'s documented purpose (VISION.md: "arc-managed state that is not part of the versioned graph") — but it is a real, user-visible consequence worth surfacing rather than silently accepting, and a candidate for a future ADR if shared, versioned kind-registration is later required.

A pre-existing graph created by `arc init` **before** this feature shipped has no `.arc/config.yml` at all; `Resolve`'s "absent file → `core.CoreMergeRules` alone" fallback means such a graph keeps working for the three core kinds with no migration step, simply without any domain kind pre-registered (consistent with "no domain kinds registered" being the correct, unsurprising default).

**Rationale**: Resolves spec FR-018/FR-019/FR-020's registration requirement with the smallest addition that satisfies "this version of the config defines only merge rules per kind" (user instruction) — a flat `kind → mergeOp` map, edited directly as YAML by a user who wants to opt into a domain profile's kind (e.g. copying `hypothesis: validated-overwrite` from `internal/core.KnownProfileMergeRules`, which ships as the ready-made defaults for the two example profiles CORE's own spec repository documents — ARCNET-DOMAIN-ARTICLE.md's `hypothesis`/`aporia`, ARCNET-DOMAIN-CORE-THOUGHT.md's `thought`). No dedicated `arc config` mutation command is introduced in this iteration — the instruction describes "config management" (load/save/resolve), not a CLI surface for editing it; hand-editing the YAML file is sufficient for "this version."

**Alternatives considered**: A `port.ConfigStore` interface duplicated inside both `ctrl` and `graph` — rejected as unnecessary ceremony for a capability (`internal/core` constant + `fsys.Store`) both already have direct, legitimate access to. `internal/app/ctrl` importing `internal/app/config`'s `component.go` directly to seed the file — rejected: this is exactly the "use-case depends on another use-case" ADR 001 forbids; routing the shared value through `internal/core` avoids it entirely with less code, not more.

## D6: Merge algebra (CORE §10) lives in `internal/core`

**Decision**:

```go
// internal/core/merge.go
func Merge(existing, incoming Node, op MergeOp, sourceID string) (merged Node, conflicts []string, err error)
```

- `MergeNone`: if `existing` is the zero Node (does not yet exist), the caller creates it verbatim; if it already exists, `Merge` returns `existing` unchanged (`source`, spec FR-007).
- `MergeUnion`: `Edges`/`HRefs`/each `Links[predicate].Seq` are unioned (`(predicate, target)` pairs deduplicated, order-preserving, existing entries first); multi-valued `Attrs` (slice-typed) are unioned the same way; scalar `Attrs`/`Text`/`Notes` are first-writer-wins — a divergent incoming scalar is left unchanged in `merged` but its field name is appended to `conflicts`, and the raw divergent value is embedded as a VISION.md-style conflict marker (D7) so the file itself stays human-readable and diff-friendly.
- `MergeUnionFirstWriter`: identical to `MergeUnion`, except a scalar `Attrs` field that is empty/absent on `existing` is filled from `incoming` with **no** conflict — only a genuine value-vs-value divergence on an already-populated field is flagged.
- `MergeValidatedOverwrite`: multi-valued fields union exactly as above; **every** scalar field is treated first-writer-wins with conflicts never flagged, since CORE §10 reserves overwriting a validation-owned scalar for "an optional validation pass" this feature does not implement (no domain-profile schema exists yet to say *which* fields are validation-owned). This is a deliberate, flagged simplification, not a silent gap — see the note in spec.md's Out of Scope and this decision's Alternatives below.
- `MergeAppend` is not handled by `Merge` at all — CORE §12.2 excludes index kinds (`timeline`) from patches entirely ("Index kinds (`timeline`) are not carried"), so no patch-carried `Node` is ever merged with this operation; timeline entries are produced by D8's separate, dedicated logic.

**Rationale**: CORE §10 is graph-format algebra, independent of any one use-case, and reusable by a future `arc retract`/`arc resolve` — keeping it in `internal/core` next to the `Node`/`MergeOp` types it operates on avoids splitting one cohesive piece of domain logic across `internal/core` and `internal/app/graph`.

**Alternatives considered**: Implementing per-kind merge switches inline in `internal/app/graph/service` — rejected: `internal/app/graph/service.Apply` should orchestrate I/O and sequencing (read patch, look up existing node, call `Merge`, write result, stage, commit), not re-derive CORE §10's algebra, which the constitution's Screaming Architecture guidance (ADR 001) puts in the domain layer, not the use-case's orchestration layer. Fully implementing `validated-overwrite`'s "validation-pass-owned fields" now — rejected as speculative: no domain profile schema mechanism exists to declare which fields are validation-owned, and none of the three format-fixed kinds (`source`/`entity`/`resource`) use this operation, so there is no concrete case to validate the design against yet.

## D7: Conflict marker format (VISION.md, "Merge conflicts")

**Decision**: When `Merge` flags a scalar field as conflicted, the field's value in the returned `Node.Attrs` (or `Text`, if the conflicted field is the body prose) becomes the literal multi-line string:

```
<<<<<<< <existing-source-id>
<existing value>
=======
<incoming value>
>>>>>>> <incoming-source-id>
```

reproducing VISION.md's own documented example verbatim (git-style conflict markers, so the file stays readable and diff-friendly, and so a future `arc conflicts`/`arc resolve` command — out of this feature's scope per spec.md — can find every unresolved conflict by grepping for `<<<<<<<`). `existing-source-id`/`incoming-source-id` are not stored on `Node` itself (nodes do not carry a provenance-per-field history in the AST model); `internal/app/graph/service.Apply` threads the two source ids (`document` field from the graph's already-recorded provenance for `existing`, vs. the current patch's `document` for `incoming`) into `Merge`'s `sourceID` parameter and a similarly-threaded "which source wrote the value currently in `existing`" lookup. Where `existing`'s original writer cannot be determined (a node created before this provenance convention existed, or a hand-edited file), the marker's left side falls back to the literal token `existing`.

**Rationale**: Reusing VISION.md's own example format exactly, rather than inventing a different one, keeps this feature consistent with the rest of the roadmap's documented (if not-yet-built) `arc conflicts`/`arc resolve` commands, so those commands — when built — parse a format this feature already produces rather than requiring a follow-up format migration.

**Alternatives considered**: A structured `needsReview: [field, ...]` front-matter list instead of inline markers — rejected: VISION.md explicitly specifies the inline marker format ("`arc` writes the conflict directly into the node file using git-style conflict markers"), and an additional structured list would be redundant with what the marker itself already encodes (which field, by construction — its own value contains the marker).

## D8: Timeline derivation (CORE §9.4)

**Decision**: `internal/core/timeline.go` exposes:

```go
func TimelinePeriods(published time.Time) (yearly, monthly string) // "2026", "2026-04"
func TimelineEntry(id, title string, authors []string, published time.Time) string // one rendered bullet
```

`internal/app/graph/service.Apply` reads (or, if absent, creates) `timeline/yearly/<YYYY>.md` and `timeline/monthly/<YYYY-MM>.md` via `fsys.Store`, inserts `TimelineEntry(...)` in chronological order among existing entries (a small, in-package insertion-sort over already-parsed entries — reusing `core.ParseNode` to read the existing file's `entries` list), and writes the result back. A freshly created period file gets the CORE §9.4 front-matter (`kind: timeline`, `period`, `granularity`) and a human-readable `# <Month> <Year>` / `# <Year>` heading.

**Rationale**: Direct implementation of CORE §9.4's documented shape and `arc init`'s pre-existing `timeline/yearly/`, `timeline/monthly/` folders (already created empty by `002-arc-init`). Reusing `core.ParseNode` for the read-back-and-insert step (rather than hand-parsing the existing bullet list) keeps exactly one Markdown-parsing code path in the whole codebase.

**Alternatives considered**: Always appending at the end of the file regardless of order — rejected: CORE §9.4 requires entries "ordered by date," and out-of-order timeline files would misrepresent the graph's own documented invariant to any later reader.

## D9: Command grammar, UX (ADR 002 compliance)

**Decision**: `arc apply <patch.md>` — a bare top-level verb (DS-01, continuing the D6 precedent from `specs/002-arc-init/research.md`: "the entire `arc` tool operates on exactly one kind of subject"), hosted in `cmd/arc/graph/apply.go` (package `graph`, mirroring `internal/app/graph`, exactly as `cmd/arc/ctrl` mirrors `internal/app/ctrl` — the package name is not the CLI verb, matching existing precedent). `<patch.md>` is the command's one positional "subject" argument (DS-02/DS-09), no command-local flags beyond the DS-03 persistent root flags. `--dry-run` and `--batch` are separate VISION.md bullets and out of this feature's scope (spec.md Assumptions) — not wired here.

Reporter steps (DS-06, flat mode — a single-phase-per-call command like `arc init`, not a multi-phase task tree per DS-08): `"Reading patch file"`, `"Checking idempotency"`, `"Applying node contributions"`, `"Updating timeline"`, `"Committing"`. Renderer: `bios.Registry[kernel.ApplyResult]{Human: humanApplyPrinter{}}` — human line states counts created/merged by kind and the commit hash, or the "already tracked, nothing to do" message for the idempotent-skip path (constitution Principle X: "successful operations that change state MUST briefly explain what changed" — the skip path explains *why nothing* changed, satisfying the same spirit). `PostRunE` hint: when `ApplyResult.Conflicts` is non-empty, a hint naming the conflicted file(s) — directly actionable, satisfying DS-12's "conditional on the flags/state actually used in this invocation" rule; no hint otherwise (DS-12 does not require a hint on every command).

**Rationale**: Follows `specs/002-arc-init`'s already-established, binding conventions (D6 bare-verb grammar, DS-04/05/06/12 patterns) with zero new UX decisions needed — ADR 002 is already fully expressive for this command's shape.

**Alternatives considered**: `arc graph apply` (noun-verb) — rejected, breaks the already-fixed bare-verb convention and VISION.md's own naming for every command in its roadmap.

## D10: Error annotation (`github.com/fogfish/faults`, constitution Mandatory Libraries & Tooling)

**Decision**: Every expected failure (malformed patch manifest, unparsable patch body, target not an initialized graph, unrecognized node kind, malformed `.arc/config.yml`, mid-run I/O failure) is a package-level `faults.Type`/`faults.SafeN` constant in the owning package's `errors.go`, wrapped via `.With()`, matched via `errors.Is()` — same convention `specs/002-arc-init/research.md` D7 established. No new pattern introduced.
