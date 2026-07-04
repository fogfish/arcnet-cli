# Architecture

`arc` is a single-binary Cobra CLI (module `github.com/fogfish/arcnet-cli`) built per [ADR 001](adrs/001-system-architecture.md) (hexagonal/onion layering, screaming architecture) and [ADR 002](adrs/002-ux-design-system.md) (CLI UX design system). Both ADRs are Accepted and BINDING (constitution Principle I).

## Directory Structure

```text
cmd/arc/                    # sole primary (driving) adapter: Cobra command tree
├── main.go                 # entrypoint, calls newRootCmd().Execute()
├── root.go                 # root command, DS-03 persistent flags, PersistentPreRun schema selection
├── ctrl/                   # Cobra wiring for the ctrl (graph management) domain
│   └── init.go              # `arc init` command: flag/arg parsing, calls internal/app/ctrl.Init,
│                             #   seeded by internal/app/schema.Seed() — pure, no network access
├── graph/                  # Cobra wiring for the graph (graph I/O) domain
│   ├── apply.go             # `arc apply` command: flag/arg parsing, calls
│   │                         #   internal/app/schema.Resolve then internal/app/graph.Apply
│   └── grep.go              # `arc grep` command: <pattern> arg, --kind/--tag/--attr local
│                             #   flags, calls internal/app/graph.Grep, renders via
│                             #   bios.Registry (highlight/truncate presentation only)
└── lint/                   # Cobra wiring for the lint (graph conformance validation) domain
    └── lint.go               # `arc lint` command: flag/arg parsing, calls
                              #   internal/app/schema.Resolve then internal/app/lint.Lint

internal/
├── bios/                    # shared kernel (ADR 002 DS-04/05/06) — output modes, color schema,
│                             #   progress reporter. Reused by every future command; not tied to
│                             #   any single use-case.
├── core/                    # shared, use-case-independent core domain (ADR 001's "core domain"
│                             #   evolution phase): the graph AST (ARCNET-AST §4-6) as plain Go
│                             #   types, a goldmark-backed Markdown↔AST codec, the CORE §10 merge
│                             #   algebra, CORE §9.4 timeline-period derivation, and the
│                             #   MergeRuleSet value type (Union/Lookup). No dependency on any
│                             #   internal/app/<use-case> — ARCNET-CORE's actual declared kind/merge/
│                             #   predicate defaults live in internal/app/schema instead. Also holds
│                             #   Filter{Kinds,Tags,Attrs,AttrPatterns}/Filter.Match(Node) — the
│                             #   shared node-selection type every VISION.md Filtering-section
│                             #   command consumes (specs/006-arc-grep-content-search, research.md D8).
├── pkg/                     # NEW tier (ADR 001 "evolution of domain logic" phase 2): generic,
│                             #   reusable domain services promoted out of internal/core once they
│                             #   need stricter isolation. First occupant:
│   └── grep/                  # domain-agnostic, dependency-free, fs.FS-based content-search
│                               #   library — Search(ctx, fsys, pattern, Options) (Result, error);
│                               #   no dependency on internal/core or internal/app/*, never imports
│                               #   os (constitution Principle VII; specs/006-arc-grep-content-search,
│                               #   research.md D2)
├── adapter/
│   ├── fsys/                # shared, cross-use-case filesystem adapter (ADR 001 "phase 2" adapter
│   │                         #   tier). The ONLY package permitted to call os's file/directory
│   │                         #   functions (constitution Principle VII, Mandatory Libraries &
│   │                         #   Tooling: "Filesystem Abstraction"). Built on stdlib io/fs/io.Writer
│   │                         #   only — no third-party filesystem library.
│   └── git/                 # shared, cross-use-case git adapter (ADR 001 "phase 2" adapter tier,
│                             #   promoted from internal/app/ctrl/adapter/git once a second use-case
│                             #   needed git access, research.md D4 in specs/003-apply-patch/). The
│                             #   one concrete Git type satisfies ctrl.port.VCS, graph.port.VCS, AND
│                             #   lint.port.VCS structurally (ADR 001 port isolation rule 1) — its
│                             #   CommitsMatching method (specs/004-arc-lint/research.md D12) is the
│                             #   one addition lint needed, read-only (git log, never a write).
└── app/
    ├── ctrl/                 # first domain use-case: graph management / control plane
    │   ├── kernel/            # domain value types (GraphRoot, ArcNetCoreLayout, InitResult)
    │   ├── port/               # ctrl-private secondary port (VCS) — not imported by other use-cases
    │   ├── adapter/
    │   │   └── mock/           # in-memory fake VCS for service unit tests
    │   ├── service/            # use-case logic (Init)
    │   └── component.go        # primary port: Init(ctx, mounter, vcs, dir, schemaSeed) (kernel.InitResult, error)
    │
    ├── config/                # second domain use-case: `.arc/config.yml` load/save — gained its
    │   │                       #   first real field (Grep) in specs/006-arc-grep-content-search,
    │   │                       #   after sitting dormant (zero callers) since
    │   │                       #   specs/005-graph-schema-first-class shipped
    │   ├── kernel/             # domain value types (Config, ConfigPath); Config.Grep GrepConfig
    │   │                        #   {Workers,MaxLineWidth} is its first real field — a zero/absent
    │   │                        #   value resolves to the built-in default at the cmd/ wiring layer
    │   ├── service/             # use-case logic (Load, Save)
    │   └── component.go         # primary port: Load(store), Save(store, cfg)
    │
    ├── schema/                # fifth domain use-case, no cmd/ package or port/adapter subdirectory:
    │   │                       #   isolates ARCNET-CORE's declared vocabulary of node kinds, merge
    │   │                       #   behaviors, and predicates as versioned _schema/nodes/*.md and
    │   │                       #   _schema/predicates/*.md documents. Its only I/O is the
    │   │                       #   already-shared internal/adapter/fsys, consumed directly (no
    │   │                       #   use-case-private external dependency to abstract). Consumed only
    │   │                       #   by arc init/arc apply (and read-only by arc lint), never invoked
    │   │                       #   directly of its own.
    │   ├── kernel/             # CoreMergeRules, CorePredicates (ARCNET-CORE §9/§7.4 built-ins),
    │   │                        #   SchemaKind, NodesDir/PredicatesDir path constants
    │   ├── service/             # use-case logic (Seed, Resolve, RegisterKind, RegisterPredicate)
    │   └── component.go         # primary port: Seed(), Resolve(store), RegisterKind(store, kind),
    │                             #   RegisterPredicate(store, predicate); Component{} additionally
    │                             #   satisfies graph.port.SchemaRegistry structurally
    │
    ├── graph/                 # third domain use-case: graph mutation / graph I/O
    │   ├── kernel/              # domain value types (ApplyResult, Match, GrepResult)
    │   ├── port/                 # graph-private secondary ports: VCS, and SchemaRegistry
    │   │                          #   (RegisterKind/RegisterPredicate — satisfied structurally by
    │   │                          #   internal/app/schema's Component, ADR 001 port isolation rule 1)
    │   ├── adapter/
    │   │   └── mock/             # in-memory fake VCS for service unit tests
    │   ├── service/              # use-case logic (Apply, Grep) — Apply's per-node loop's
    │   │                          #   auto-discovery hook registers a previously-unseen
    │   │                          #   kind/predicate into _schema/ in the same commit as the
    │   │                          #   triggering patch (spec.md FR-012); Grep enumerates+parses
    │   │                          #   every node file (excluding .arc/ and _schema/), builds a
    │   │                          #   Filter-membership set, and delegates the actual line scan to
    │   │                          #   internal/pkg/grep.Search (specs/006-arc-grep-content-search)
    │   └── component.go          # primary port: Apply(ctx, mounter, vcs, reporter, rules,
    │                              #   predicates, schema, dir, patchPath) (kernel.ApplyResult, error);
    │                              #   Grep(ctx, mounter, filter, pattern, cfg, dir) (kernel.GrepResult, error)
    │
    └── lint/                  # fifth domain use-case: graph conformance validation (CORE §14)
        ├── kernel/              # domain value types (Rule, Violation, NodeStatus, LintResult, Sowa tables)
        ├── port/                 # lint-private secondary port (VCS) — narrowest of the three port.VCS
        ├── adapter/
        │   └── mock/             # in-memory fake VCS for service unit tests
        ├── service/              # use-case logic (Lint): enumeration (excludes .arc/ and _schema/
        │                          #   entirely), raw-text line locator, one checker per CORE §14
        │                          #   rule — strictly read-only, never writes to fsys.Store and
        │                          #   never commits
        └── component.go          # primary port: Lint(ctx, mounter, vcs, reporter, rules,
                                    #   predicates, dir) (kernel.LintResult, error)
```

`internal/app/ctrl` is the first `internal/` package in this codebase, so ADR 001's `componentX` layout (`kernel/`, `port/`, `adapter/`, `service/`, `component.go`) now takes full effect. `internal/bios` and `internal/adapter/fsys` are deliberately shared, not use-case-private, since every future command needs an output/color/reporter kernel and every future graph-root-mounting command needs the same filesystem mount contract (research.md D3/D5 in `specs/002-arc-init/`). `internal/core` is the project's first core-domain package (ADR 001's own evolution model): the graph AST and its canonical Markdown serialization are a model invariant shared by every future graph-reading command, not an `apply`-specific concern, so they live below the use-case layer. `internal/adapter/git` is the first adapter promoted to the shared tier once a second use-case (`graph`) needed the same capability `ctrl` already had (research.md D4 in `specs/003-apply-patch/`), mirroring `internal/adapter/fsys`'s precedent. `internal/app/schema` (`specs/005-graph-schema-first-class/`) is the fifth `internal/app/<domain>` use-case and the first to have neither a `cmd/` package of its own nor a `port`/`adapter` subdirectory: it isolates ARCNET-CORE's declared vocabulary of node kinds, merge behaviors, and predicates, replacing the retired `_meta/` registry stubs and `.arc/config.yml`'s merge-rule content with versioned, human-readable `_schema/` documents (research.md D1/D2/D5 in `specs/005-graph-schema-first-class/`).

## Command Grammar (Principle IX)

This project uses **bare top-level verbs** (`arc init`, `arc apply`, `arc list`, ...), not noun-verb nesting — permitted by ADR 002 DS-01 because the entire tool operates on exactly one kind of subject, a knowledge graph. Every subcommand follows this convention without exception.

## Glossary

| Term | Definition |
|---|---|
| **Graph Root** | The directory tree representing one knowledge graph instance; identified by the presence of a `.arc/` directory at its top level. Resolved and mounted via `internal/adapter/fsys` (`ResolveLocalRoot` then `Mounter.Mount`). |
| **Canonical Folder** | One of the fixed top-level directories every graph must contain: `sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_schema/nodes/`, `_schema/predicates/`. Defined statically by `internal/app/ctrl/kernel.ArcNetCoreLayout`. |
| **Schema Document** | A versioned, human-readable Markdown document under `_schema/` describing one recognized node kind or predicate — `kind: schema` front-matter, parsed/rendered by the same, unmodified `internal/core.ParseNode`/`RenderNode` every ordinary content node uses. Replaces the retired Metadata Stub/Kind Registration concepts. See Node-Kind Schema Document/Predicate Schema Document below. `internal/app/schema`. |
| **Node-Kind Schema Document** | A Schema Document at `_schema/nodes/<kind>.md`: its `id` is the node kind's name, its `merge` attribute is the `Merge Behavior` `arc apply` uses for that kind. Seeded for ARCNET-CORE's four fixed kinds by `arc init`; auto-registered (always with `merge: union`) the first time `arc apply` encounters an unrecognized kind (spec FR-010); never overwritten automatically once present (spec FR-011) — a human hand-editing its `merge` value is what a later `arc apply` invocation actually uses. |
| **Predicate Schema Document** | A Schema Document at `_schema/predicates/<name>.md`, carrying no `merge` attribute — its mere presence is what registers a predicate name. Seeded for ARCNET-CORE's thirteen fixed predicates (CORE §7.4) by `arc init`; auto-registered the first time `arc apply` encounters an unrecognized predicate. |
| **Arc State Directory** | The `.arc/` directory holding tool-managed local state, never versioned alongside graph content (excluded via `.gitignore`). Its presence is what distinguishes an initialized Graph Root from an empty directory. |
| **Initial Commit** | The single git commit produced by `arc init` that records a graph's creation, with the mandatory subject line `graph(init): empty knowledge graph` (CORE §11.3). |
| **Patch** | A CORE §12 Markdown document — one manifest (`document`, `published`, `title`, `stats`) plus H1-kind/H2-node sections — that `arc apply` ingests into the graph. Parsed by `internal/core.ParsePatch` into `internal/core.Patch`. |
| **Node Contribution** | One H2 node section within a patch: the create-or-merge unit `arc apply` applies to the graph, one per patch-carried `internal/core.Node`. |
| **Source Node** | A node of kind `source` (CORE's fixed, always-recognized `MergeNone` kind) — the citable document a patch itself represents. |
| **Entity/Resource Node** | A node of kind `entity` (`MergeUnion`) or `resource` (`MergeUnionFirstWriter`) — CORE's fixed kinds for concepts and referenced material, mergeable across multiple contributing patches. |
| **Timeline Entry** | One chronologically-ordered bullet appended to a `timeline/yearly/<YYYY>.md` or `timeline/monthly/<YYYY-MM>.md` period file, derived from a patch's `published` manifest field (CORE §9.4, `internal/core.TimelinePeriods`/`.TimelineEntry`). |
| **Merge Behavior** | The `internal/core.MergeOp` (`none`, `union`, `union-first-writer`, `append`, `validated-overwrite`) a node's kind is registered against, determining how `internal/core.Merge` reconciles an incoming contribution with an existing node. Now sourced from a Node-Kind Schema Document's `merge` attribute, resolved via `internal/app/schema.Resolve`. |
| **Ingest Commit** | The single git commit `arc apply` produces per invocation, subject naming the applied document, with per-kind created/merged stats and a `Source-Id:` trailer (CORE §11.3). A newly auto-registered Schema Document lands in this same commit (spec FR-012). |
| **Violation** | One failed CORE §14 checklist rule, produced by `arc lint`: the rule that fired, the file and line (or "not applicable"), a human-readable message, and — for violations spanning more than one file (e.g. a basename collision) — every related path. `internal/app/lint/kernel.Violation`. |
| **Lint Run** | One `arc lint` invocation: walks every node file in the graph, runs every applicable CORE §14 rule against it, and aggregates every violation found without stopping at the first one (spec FR-013). Strictly read-only — the first graph-inspecting command in this codebase that never writes to `fsys.Store` or git history. Schema Documents under `_schema/` are excluded from this walk entirely (spec FR-015). `internal/app/lint/kernel.LintResult`. |
| **Checklist Rule** | One named CORE §14 conformance check (`internal/app/lint/kernel.Rule`), e.g. unique basenames, resolvable links, source citekey identity, entity Sowa category, registered predicates, one ingest commit per document, absence of merge-conflict markers. |
| **Extension Profile Checklist** | `arc lint`'s CORE §10/§14 check for a non-built-in node kind: recognized (present in the resolved `core.MergeRuleSet`) vs. unrecognized, deliberately scoped to kind-recognition only — no per-kind field-schema declaration mechanism exists yet in this codebase (plan.md Complexity Tracking, `specs/004-arc-lint/research.md` D11; unaffected by `specs/005-graph-schema-first-class`, which adds kind/merge/predicate *recognition* storage, not field-level schema declaration). |
| **Filter** | The optional, composable node-selection criteria (`Kinds` OR'd, `Tags`/`Attrs`/`AttrPatterns` AND'd) shared by every VISION.md Filtering-section command; a zero-value `Filter{}` matches every node. `internal/core.Filter`, `Filter.Match(Node) bool` (`specs/006-arc-grep-content-search`, research.md D8) — `arc grep` is the first command to consume it. |
| **Match** | One reported occurrence of `arc grep`'s `<pattern>` on a single line within a single node's file: the node's `kind`/`id`, the 1-based line number, and the full matched line text. `internal/pkg/grep.Match` (path/line/text/byte-offsets, domain-agnostic) is mapped into `internal/app/graph/kernel.Match` (kind/id-labeled) for rendering. |
| **Grep Run** | One `arc grep` invocation: enumerates and parses every node file (excluding `.arc/` and `_schema/`), narrows the scan to nodes passing a `Filter`, and reports every matching line across every scanned node in a single pass, never stopping at the first match. Strictly read-only, like `arc lint`. `internal/app/graph/kernel.GrepResult`, `internal/app/graph/service.Grep` (`specs/006-arc-grep-content-search`). |
