# Architecture

`arc` is a single-binary Cobra CLI (module `github.com/fogfish/arcnet-cli`) built per [ADR 001](adrs/001-system-architecture.md) (hexagonal/onion layering, screaming architecture) and [ADR 002](adrs/002-ux-design-system.md) (CLI UX design system). Both ADRs are Accepted and BINDING (constitution Principle I).

## Directory Structure

```text
cmd/arc/                    # sole primary (driving) adapter: Cobra command tree
├── main.go                 # entrypoint, calls newRootCmd().Execute()
├── root.go                 # root command, DS-03 persistent flags, PersistentPreRun schema selection
├── ctrl/                   # Cobra wiring for the ctrl (graph management) domain
│   ├── init.go              # `arc init` command: flag/arg parsing, calls internal/app/ctrl.Init,
│   │                         #   seeded by internal/app/schema.Seed() — pure, no network access
│   └── apply_schema.go      # `arc apply schema <patch.md>|<url>|arcnet:<name>` command: attached as
│                             #   a child of graph.NewApplyCmd() in cmd/arc/root.go (borrows the
│                             #   `apply` verb for naming consistency, per user direction); calls
│                             #   internal/app/schema.ApplyPatch (specs/018-apply-schema-patch)
├── graph/                  # Cobra wiring for the graph (graph I/O) domain
│   ├── apply.go             # `arc apply` command: flag/arg parsing, calls
│   │                         #   internal/app/schema.Resolve then internal/app/graph.Apply
│   ├── revert.go            # `arc revert <source-id> [--force|-f]` command:
│   │                         #   destructive-operation confirmation gate
│   │                         #   (internal/bios.Confirm, unless --force) then
│   │                         #   calls internal/app/graph.Revert (specs/016-arc-revert)
│   ├── grep.go              # `arc grep` command: <pattern> arg, --type/--tag/--attr local
│   │                         #   flags, calls internal/app/graph.Grep, renders via
│   │                         #   bios.Registry (highlight/truncate presentation only)
│   ├── subgraph.go          # `arc subgraph` command: <basename> arg, --depth local flag,
│   │                         #   reuses grep.go's optsFilter, calls internal/app/graph.Subgraph,
│   │                         #   writes core.RenderPatch's bytes verbatim to stdout — no
│   │                         #   bios.SCHEMA styling (specs/007-arc-subgraph, research.md D10)
│   └── serve.go             # `arc serve [--http <addr>]` command: the codebase's second
│                             #   primary-adapter family (ADR 003) — registers node_get/
│                             #   node_grep/subgraph_get as MCP Tools on an mcp.Server, calling
│                             #   internal/app/graph.NodeGet/Grep/Subgraph exactly like every
│                             #   Cobra command does, over stdio by default or Streamable
│                             #   HTTP/SSE when --http names a Bind Address
│                             #   (specs/008-arc-serve-mcp)
└── lint/                   # Cobra wiring for the lint (graph conformance validation) domain
    └── lint.go               # `arc lint` command: flag/arg parsing, calls
                              #   internal/app/schema.Resolve then internal/app/lint.Lint

internal/
├── bios/                    # shared kernel (ADR 002 DS-04/05/06) — output modes, color schema,
│                             #   progress reporter. Reused by every future command; not tied to
│                             #   any single use-case.
│                             #   confirm.go adds Confirm(prompt string) (bool, error), a
│                             #   TTY-gated destructive-operation confirmation gate
│                             #   (research.md D10, specs/016-arc-revert) — the first command
│                             #   in this codebase whose default behavior deletes a tracked file.
├── core/                    # shared, use-case-independent core domain (ADR 001's "core domain"
│                             #   evolution phase): the graph AST (ARCNET-AST §4-6) as plain Go
│                             #   types, a goldmark-backed Markdown↔AST codec, the CORE §10 merge
│                             #   algebra, CORE §9.4 timeline-period derivation, and the
│                             #   PredicateDef/TypeDef/Index value types (specs/011-machine-readable-
│                             #   schema, replacing the earlier MergeRuleSet). No dependency on any
│                             #   internal/app/<use-case> — ARCNET-CORE's actual declared type/merge/
│                             #   predicate defaults live in internal/app/schema instead. Also holds
│                             #   Filter{Types,Tags,Attrs,AttrPatterns}/Filter.Match(Node) — the
│                             #   shared node-selection type every VISION.md Filtering-section
│                             #   command consumes (specs/006-arc-grep-content-search, research.md D8).
│                             #   RenderPatch(Patch) ([]byte, error) is the structural inverse of
│                             #   ParsePatch (specs/007-arc-subgraph, research.md D2): CORE §12.2
│                             #   patch-exchange serialization, grouped by Type/ID (research.md D9).
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
│   ├── git/                 # shared, cross-use-case git adapter (ADR 001 "phase 2" adapter tier,
│   │                         #   promoted from internal/app/ctrl/adapter/git once a second use-case
│   │                         #   needed git access, research.md D4 in specs/003-apply-patch/). The
│   │                         #   one concrete Git type satisfies ctrl.port.VCS, graph.port.VCS,
│   │                         #   lint.port.VCS, AND schema.port.VCS structurally (ADR 001 port
│   │                         #   isolation rule 1) — its CommitsMatching method
│   │                         #   (specs/004-arc-lint/research.md D12) is the one addition lint
│   │                         #   needed, read-only (git log, never a write).
│   └── http/                # shared, cross-use-case HTTP-fetch adapter (ADR 001 "phase 2" adapter
│                             #   tier): Client.Fetch(ctx, url) (io.ReadCloser, error), backed by
│                             #   net/http.Client with a default, overridable timeout — this
│                             #   codebase's first genuinely network-calling capability
│                             #   (specs/018-apply-schema-patch, research.md D2). Satisfies
│                             #   internal/app/schema/port.Fetcher structurally.
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
    ├── schema/                # fifth domain use-case, no cmd/ package of its own (its Cobra command,
    │   │                       #   `arc apply schema`, lives in cmd/arc/ctrl instead — see below):
    │   │                       #   isolates ARCNET-CORE's declared vocabulary of predicates and
    │   │                       #   types as machine-readable _schema/predicates/*.md (Property
    │   │                       #   nodes) and _schema/types/*.md (Class nodes) documents, each
    │   │                       #   declaring its own role/merge/label/aligned (predicates) or
    │   │                       #   merge/required/optional (types), per CORE §9.1/§9.2
    │   │                       #   (specs/011-machine-readable-schema). Consumed by arc init/arc
    │   │                       #   apply/arc apply schema (and read-only by arc lint), never invoked
    │   │                       #   directly of its own. Gained its first port/adapter subdirectory
    │   │                       #   (specs/018-apply-schema-patch) for ApplyPatch's own URL-fetch/git-
    │   │                       #   commit needs; Seed/Resolve/RegisterType/RegisterPredicate's I/O
    │   │                       #   remains the already-shared internal/adapter/fsys, consumed
    │   │                       #   directly.
    │   ├── kernel/             # CorePredicateDefs, CoreTypeDefs (ARCNET-CORE §10/§11's full
    │   │                        #   built-in vocabulary), TypesDir/PredicatesDir path constants;
    │   │                        #   ApplySchemaResult and ArcnetCatalogBaseURL (a var, not a const,
    │   │                        #   purely for one E2E test's httptest.Server seam,
    │   │                        #   specs/018-apply-schema-patch)
    │   ├── port/                # schema-private secondary ports (specs/018-apply-schema-patch):
    │   │                         #   VCS{StageAll,Commit} — narrower than graph.port.VCS, satisfied
    │   │                         #   structurally by internal/adapter/git.VCS — and
    │   │                         #   Fetcher{Fetch(ctx,url) (io.ReadCloser,error)}, satisfied by
    │   │                         #   internal/adapter/http.Client
    │   ├── adapter/
    │   │   └── mock/             # in-memory fake VCS/Fetcher for service unit tests
    │   ├── service/             # use-case logic (Seed, Resolve, RegisterType, RegisterPredicate,
    │   │                        #   ApplyPatch) — Resolve fails fast (core.Index, error) on a missing
    │   │                        #   schema folder or any malformed document, never skips one;
    │   │                        #   ApplyPatch validates every patch-carried node is Property/Class
    │   │                        #   and every node decodes/renders cleanly before any _schema/ write
    │   │                        #   begins (no rollback bookkeeping needed, unlike graph.Apply)
    │   └── component.go         # primary port: Seed(), Resolve(store) (core.Index, error),
    │                             #   RegisterType(store, typ), RegisterPredicate(store, predicate),
    │                             #   ApplyPatch(ctx, mounter, vcs, fetcher, reporter, dir, source)
    │                             #   (kernel.ApplySchemaResult, error); Component{} additionally
    │                             #   satisfies graph.port.SchemaRegistry structurally
    │
    ├── graph/                 # third domain use-case: graph mutation / graph I/O
    │   ├── kernel/              # domain value types (ApplyResult, RevertResult, Match, GrepResult,
    │   │                          #   SubgraphResult — specs/007-arc-subgraph)
    │   ├── port/                 # graph-private secondary ports: VCS (widened by
    │   │                          #   specs/016-arc-revert with six ingest-commit/blame/revert
    │   │                          #   primitives, contracts/vcs-port-contract.md), and
    │   │                          #   SchemaRegistry
    │   │                          #   (RegisterType/RegisterPredicate — satisfied structurally by
    │   │                          #   internal/app/schema's Component, ADR 001 port isolation rule 1)
    │   ├── adapter/
    │   │   └── mock/             # in-memory fake VCS for service unit tests
    │   ├── service/              # use-case logic (Apply, Revert, Grep, Subgraph) — Apply's per-node
    │   │                          #   loop's auto-discovery hook registers a previously-unseen
    │   │                          #   type/predicate into _schema/ in the same commit as the
    │   │                          #   triggering patch (spec.md FR-012); Grep enumerates+parses
    │   │                          #   every node file (excluding .arc/ and _schema/), builds a
    │   │                          #   Filter-membership set, and delegates the actual line scan to
    │   │                          #   internal/pkg/grep.Search (specs/006-arc-grep-content-search);
    │   │                          #   Subgraph shares Grep's walkNodeFiles enumeration, then runs
    │   │                          #   two independent, capped BFS passes (direct/backlink) from a
    │   │                          #   seed node and serializes the result via core.RenderPatch
    │   │                          #   (specs/007-arc-subgraph, research.md D3/D4/D5) — no port of
    │   │                          #   its own, strictly read-only like Grep; Revert locates a
    │   │                          #   source-id's ingest commit and retracts its contribution via
    │   │                          #   a whole-commit git revert (nothing has touched its files
    │   │                          #   since) or a per-node reconciliation otherwise — removing an
    │   │                          #   exclusively-owned node (sweeping every backlink, including
    │   │                          #   timeline entries) or stripping only a shared node's own
    │   │                          #   blame-attributed text, resolving a conflict marker's
    │   │                          #   provenance where blame alone cannot (specs/016-arc-revert,
    │   │                          #   contracts/revert-algorithm-contract.md)
    │   └── component.go          # primary port: Apply(ctx, mounter, vcs, reporter, index,
    │                              #   schema, dir, patchPath) (kernel.ApplyResult, error) — index is
    │                              #   the core.Index internal/app/schema.Resolve returns
    │                              #   (specs/011-machine-readable-schema, replacing the earlier
    │                              #   (core.MergeRuleSet, map[string]bool) pair);
    │                              #   Revert(ctx, mounter, vcs, reporter, index, dir, sourceID)
    │                              #   (kernel.RevertResult, error) (specs/016-arc-revert);
    │                              #   Grep(ctx, mounter, filter, pattern, cfg, dir) (kernel.GrepResult, error);
    │                              #   Subgraph(ctx, mounter, filter, basename, depth, cfg, dir)
    │                              #   (kernel.SubgraphResult, error); NodeGet(ctx, mounter, dir, id)
    │                              #   (core.Node, error) and EnsureGraph(ctx, mounter, dir) error
    │                              #   (specs/008-arc-serve-mcp — arc serve's node_get tool and
    │                              #   startup preflight, backed by service/node.go reusing
    │                              #   enumerateNodes/guardIsGraph)
    │
    └── lint/                  # fifth domain use-case: graph conformance validation (CORE §14/§16)
        ├── kernel/              # domain value types (Rule, Violation, NodeStatus, LintResult, Sowa tables)
        ├── port/                 # lint-private secondary port (VCS) — narrowest of the three port.VCS
        ├── adapter/
        │   └── mock/             # in-memory fake VCS for service unit tests
        ├── service/              # use-case logic (Lint): enumeration (excludes .arc/ and _schema/
        │                          #   entirely), raw-text line locator, one checker per CORE §14/§16
        │                          #   rule — strictly read-only, never writes to fsys.Store and
        │                          #   never commits
        └── component.go          # primary port: Lint(ctx, mounter, vcs, reporter, index,
                                    #   dir) (kernel.LintResult, error) — index is the same
                                    #   core.Index arc apply consumes (specs/011-machine-readable-schema)
```

`internal/app/ctrl` is the first `internal/` package in this codebase, so ADR 001's `componentX` layout (`kernel/`, `port/`, `adapter/`, `service/`, `component.go`) now takes full effect. `internal/bios` and `internal/adapter/fsys` are deliberately shared, not use-case-private, since every future command needs an output/color/reporter kernel and every future graph-root-mounting command needs the same filesystem mount contract (research.md D3/D5 in `specs/002-arc-init/`). `internal/core` is the project's first core-domain package (ADR 001's own evolution model): the graph AST and its canonical Markdown serialization are a model invariant shared by every future graph-reading command, not an `apply`-specific concern, so they live below the use-case layer. `internal/adapter/git` is the first adapter promoted to the shared tier once a second use-case (`graph`) needed the same capability `ctrl` already had (research.md D4 in `specs/003-apply-patch/`), mirroring `internal/adapter/fsys`'s precedent. `internal/app/schema` (`specs/005-graph-schema-first-class/`) is the fifth `internal/app/<domain>` use-case and the first to have neither a `cmd/` package of its own nor a `port`/`adapter` subdirectory: it isolates ARCNET-CORE's declared vocabulary of node kinds, merge behaviors, and predicates, replacing the retired `_meta/` registry stubs and `.arc/config.yml`'s merge-rule content with versioned, human-readable `_schema/` documents (research.md D1/D2/D5 in `specs/005-graph-schema-first-class/`).

## Command Grammar (Principle IX)

This project uses **bare top-level verbs** (`arc init`, `arc apply`, `arc list`, ...), not noun-verb nesting — permitted by ADR 002 DS-01 because the entire tool operates on exactly one kind of subject, a knowledge graph. Every subcommand follows this convention without exception. The sole exception to "bare" is `arc apply schema`, a child of `arc apply` attached for naming consistency with `arc apply <patch.md>` even though its business logic and conceptual home (schema/config management) lives in `cmd/arc/ctrl`, not `cmd/arc/graph` (specs/018-apply-schema-patch, per explicit user direction).

## Glossary

| Term | Definition |
|---|---|
| **Graph Root** | The directory tree representing one knowledge graph instance; identified by the presence of a `.arc/` directory at its top level. Resolved and mounted via `internal/adapter/fsys` (`ResolveLocalRoot` then `Mounter.Mount`). |
| **Canonical Folder** | One of the fixed top-level directories every graph must contain: `sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_schema/types/`, `_schema/predicates/`. Defined statically by `internal/app/ctrl/kernel.ArcNetCoreLayout`. |
| **Schema Index** | The in-memory `internal/core.Index{Predicates map[string]PredicateDef, Types map[string]TypeDef}` `internal/app/schema/service.Resolve` builds once per command invocation from a graph's own `_schema/` documents — the single runtime source of truth `arc apply`/`arc lint` both consume, replacing the earlier `(core.MergeRuleSet, map[string]bool)` pair (specs/011-machine-readable-schema). |
| **Predicate Schema Node** | A `Property`-typed node at `_schema/predicates/<name>.md` (CORE §9.1): mandatory `role` (one of `meta`/`text`/`href`/`edge`/`link`) and `merge` attributes, optional `label`/`aligned`, mandatory descriptive body — decoded into `core.PredicateDef`. Its `merge` attribute is the sole authority `arc apply` consults to reconcile that predicate wherever it occurs, on any node of any type (spec 012 FR-013) — a node's own `@type` no longer determines merge behavior. Seeded for ARCNET-CORE's full §10 vocabulary by `arc init`; auto-registered (`role: edge`, `merge: union`, a placeholder description) the first time `arc apply` encounters an unrecognized predicate (spec FR-010); never overwritten automatically once present (spec FR-011). Replaces the existence-only Predicate Schema Document spec 005 introduced. |
| **Type Schema Node** | A `Class`-typed node at `_schema/types/<name>.md` (CORE §9.2, renamed from `_schema/nodes/<kind>.md`): an optional `merge` attribute (the arc apply-specific bridge field beyond CORE's own documented shape, spec 011 FR-015 — retained only for schema-document-shape continuity on built-in/auto-registered documents, never validated as mandatory, spec 012 FR-020, Bugfix 018/BUG-001), mandatory descriptive body, and zero or more `required`/`optional` predicate-name bullets a conforming instance must/may carry — decoded into `core.TypeDef`. Its own `@id`/name and every node's `@type` value referencing it MUST be CamelCase — begin with an uppercase letter (`specs/019-camelcase-node-types`); `arc apply` rejects a non-conforming patch H1/`@type` outright, and `arc lint`'s `RuleTypeCase` flags a non-conforming pre-existing type definition or reference. Seeded for ARCNET-CORE's four fixed types (`Source`/`Entity`/`Resource`/`Timeline`) plus `Property`/`Class` themselves by `arc init`; auto-registered (`merge: union`, empty `required`/`optional`, a placeholder description) the first time `arc apply` encounters an unrecognized type; never overwritten automatically once present. Replaces the existence-only Node-Kind Schema Document spec 005 introduced. |
| **`rdfs:subClassOf`** | An `edge`-role predicate a Type Schema Node declares zero or more of (`- rdfs:subClassOf:: [[<base-type-name>]]`), naming another registered type this type inherits its predicate contract from. Multiple declarations mean multiple inheritance; declarations chain transitively to any depth. Resolved entirely within `internal/app/schema/service.Resolve`/`Seed` at schema-indexing time — no other package, including `internal/core` and `internal/app/lint`, has any notion of type hierarchy. `specs/017-subclass-of-predicate`. |
| **`Node`** | The built-in `Class`-typed type every other type (except `Property`/`Class` themselves) implicitly inherits from, whether or not it declares an explicit `rdfs:subClassOf` edge toward it: `Required: [published, created]`, `Optional: [tags, text, updated, scoreZ, scoreC]`. Never a node's own `@type` in practice — it exists only to be inherited from, factoring the cross-cutting contract every content type previously redeclared directly out into one place. `specs/017-subclass-of-predicate`. |
| **Effective (Inherited) Contract** | The fully flattened `Required`/`Optional` predicate set `internal/app/schema/service.Resolve` computes for a type by recursively unioning in every `rdfs:subClassOf` ancestor's own effective contract (deduplicated, required-always-wins-over-optional), including the implicit `Node` base. This is the only contract shape any consumer of `core.Index.Types` (`arc lint`'s conformance checks foremost) ever sees — `core.TypeDef` carries no raw hierarchy, only the resolved result. A cycle or a reference to an unregistered base type fails schema loading (`ErrSchemaCycle`/`ErrSchemaUnresolvedBase`) before any other schema-dependent work proceeds. `specs/017-subclass-of-predicate`. |
| **Arc State Directory** | The `.arc/` directory holding tool-managed local state, never versioned alongside graph content (excluded via `.gitignore`). Its presence is what distinguishes an initialized Graph Root from an empty directory. |
| **Initial Commit** | The single git commit produced by `arc init` that records a graph's creation, with the mandatory subject line `graph(init): empty knowledge graph` (CORE §11.3). |
| **Node** | The graph's addressable unit (ARCNET-AST §4): one Markdown file on disk, or one `## <ID>` section inside a patch. Identity (`ID`, from front-matter `"@id"`) and category (`Type`, from `"@type"`) are both mandatory, open-vocabulary, and never derived by fallback — `"@id"` must equal the file's basename. Everything else is one of `Attrs` (a `map[string][]Predicate`, every front-matter key besides `"@id"`/`"@type"`/`"published"`), `Texts` (a `map[string]string` of named prose fields), `HRefs` (inline mentions extracted from `Texts`), or `Edges` (every outgoing structural link, in document order, regardless of how the source document grouped it). Parsing still ignores original grouping, unchanged (`specs/010-predicate-node-model`); rendering now derives flat-vs-grouped shape from each predicate's own schema `Role` instead (`specs/013-predicate-role-rendering`). `internal/core.Node` (`specs/010-predicate-node-model`, supersedes specs/003-apply-patch's `Kind`/`Text`/`Notes`/`Links` shape). |
| **Predicate** | One value contributed to a Node's `Attrs` entry (AST §7): exactly one of `Value` (a scalar, as authored) or `Target` (a reference-valued attribute's target basename, optionally paired with `Alias`) is set. Every `Attrs` key holds a non-empty, ordered list of `Predicate` — one element for a single-valued attribute, several for a multi-valued one; a single-element list renders back to a bare YAML scalar, a multi-element list to a sequence. `internal/core.Predicate` (`specs/010-predicate-node-model`). |
| **Text Predicate / Prose Field** | A named entry in a Node's `Texts` map — e.g. a `Source`'s `abstract`, an `Entity`'s `definition`, every kind's `notes`. Keyed via `textPredicateFor(Type, leading bool)`, a small hardcoded `@type`→predicate-name lookup table that is an explicit, temporary stopgap pending spec 011's Schema Index; this increment's structural parser still recognizes only two prose positions per node (leading, trailing), so `Texts` supports open keys as a representation without yet supporting more than two populated keys per node. `internal/core.Node.Texts` (`specs/010-predicate-node-model`, research.md D4). |
| **Patch** | A CORE §12 Markdown document — one manifest (`document`, `published`, `title`, `stats`) plus H1-kind/H2-node sections — that `arc apply` ingests into the graph. Parsed by `internal/core.ParsePatch` into `internal/core.Patch`. |
| **Node Contribution** | One H2 node section within a patch: the create-or-merge unit `arc apply` applies to the graph, one per patch-carried `internal/core.Node`. |
| **Source Node** | A node of kind `Source` — the citable document a patch itself represents; every one of its predicates reconciles by its own declared Merge Behavior (typically `immutable`), never by a single whole-node rule. |
| **Entity/Resource Node** | A node of kind `Entity` or `Resource` — CORE's fixed kinds for concepts and referenced material, mergeable across multiple contributing patches; each predicate present on either kind reconciles by its own declared Merge Behavior, not by the node's own kind. |
| **Timeline Entry** | One chronologically-ordered bullet appended to a `timeline/yearly/<YYYY>.md` or `timeline/monthly/<YYYY-MM>.md` period file, derived from a patch's `published` manifest field (CORE §9.4, `internal/core.TimelinePeriods`/`.TimelineEntry`). |
| **Merge Behavior** | One of the `internal/core.MergeOp` seven-value canonical vocabulary (CORE §9.3: `immutable`, `union`, `firstWriteWin`, `fillIfEmpty`, `lastWriteWin`, `append`, `validatedOverwrite`) a *predicate* — not a node's `@type` — declares itself against on its own Predicate Schema Node. `internal/core.Merge` reconciles every predicate present on a merged node individually, looking up each one's own behavior in `core.Index.Predicates[name].Merge` (spec 012, per-predicate dispatch); a `TypeDef.Merge` value still exists on a Type Schema Node for schema-document-shape continuity but is no longer consulted by reconciliation. |
| **Ingest Commit** | The single git commit `arc apply` produces per invocation, subject naming the applied document, with per-kind created/merged stats and a `Source-Id:` trailer (CORE §11.3). A newly auto-registered Schema Document lands in this same commit (spec FR-012). |
| **Violation** | One failed CORE §14 checklist rule, produced by `arc lint`: the rule that fired, the file and line (or "not applicable"), a human-readable message, and — for violations spanning more than one file (e.g. a basename collision) — every related path. `internal/app/lint/kernel.Violation`. |
| **Lint Run** | One `arc lint` invocation: walks every node file in the graph, runs every applicable CORE §14 rule against it, and aggregates every violation found without stopping at the first one (spec FR-013). Strictly read-only — the first graph-inspecting command in this codebase that never writes to `fsys.Store` or git history. Schema Documents under `_schema/` are excluded from this walk entirely (spec FR-015). `internal/app/lint/kernel.LintResult`. |
| **Checklist Rule** | One named CORE §14/§16 conformance check (`internal/app/lint/kernel.Rule`), e.g. unique basenames, resolvable links, Source citekey identity, Entity Sowa category, registered predicates, one ingest commit per document, absence of merge-conflict markers, a node's own type-declared Requires/Optional predicate contract, `"@id"`/`"@type"` front-matter quoting, schema-driven citation-predicate recognition, predicate-role structural conformance (`specs/014-lint-spec-conformance`), and CamelCase class-name casing for both a schema type definition and a node's own `@type` reference (`typeCase`, `specs/019-camelcase-node-types`). |
| **Extension Profile Checklist** | `arc lint`'s CORE §10/§14/§16 check for a non-built-in node type: recognized (present in the resolved `core.Index.Types`) vs. unrecognized (`unrecognizedKind`); for a recognized type, its instances are further checked against that type's own declared `required`/`optional` predicate contract (`typeRequires`/`typeOptional`) and each occurrence's structural position against its predicate's declared `role` (`predicateRole`) — closing the field-schema conformance gap `specs/004-arc-lint/research.md` D11 originally left open (`specs/014-lint-spec-conformance`). |
| **Filter** | The optional, composable node-selection criteria (`Types` OR'd, `Tags`/`Attrs`/`AttrPatterns` AND'd) shared by every VISION.md Filtering-section command; a zero-value `Filter{}` matches every node. `internal/core.Filter`, `Filter.Match(Node) bool` (`specs/006-arc-grep-content-search`, research.md D8) — `arc grep` is the first command to consume it. |
| **Match** | One reported occurrence of `arc grep`'s `<pattern>` on a single line within a single node's file: the node's `type`/`id`, the 1-based line number, and the full matched line text. `internal/pkg/grep.Match` (path/line/text/byte-offsets, domain-agnostic) is mapped into `internal/app/graph/kernel.Match` (type/id-labeled) for rendering. |
| **Grep Run** | One `arc grep` invocation: enumerates and parses every node file (excluding `.arc/` and `_schema/`), narrows the scan to nodes passing a `Filter`, and reports every matching line across every scanned node in a single pass, never stopping at the first match. Strictly read-only, like `arc lint`. `internal/app/graph/kernel.GrepResult`, `internal/app/graph/service.Grep` (`specs/006-arc-grep-content-search`). |
| **Seed Node** | The node named by `arc subgraph`'s `<basename>` argument — always present in its extraction's output, never excluded by a `Filter`. `specs/007-arc-subgraph`. |
| **Reachable Node** | Any node other than the seed found within `arc subgraph`'s requested hop count by following structural `Edges`/`Links` in either direction; subject to the optional `Filter` and to its traversal direction's cap. `specs/007-arc-subgraph`. |
| **Subgraph** | The seed node plus the set of reachable nodes selected for one `arc subgraph` extraction, serialized as one patch-exchange document grouped by type via `internal/core.RenderPatch`. `internal/app/graph/kernel.SubgraphResult`, `internal/app/graph/service.Subgraph` (`specs/007-arc-subgraph`). |
| **Traversal Cap** | A configurable ceiling — `subgraph.directCap` (outgoing, default `4096`) and `subgraph.backlinkCap` (incoming, default `1024`), `internal/app/config/kernel.SubgraphConfig` — on how many nodes `arc subgraph` retains per traversal direction before filtering; when exceeded, the highest-degree candidates are kept and the run still succeeds (soft cap). `specs/007-arc-subgraph`, research.md D4/D5. |
| **MCP Tool** | One callable capability `arc serve` registers on its `mcp.Server` via `mcp.AddTool` — `node_get`, `node_grep`, or `subgraph_get`. Each is a thin wrapper: decode MCP JSON arguments, call the identical `internal/app/graph` primary-port function every Cobra command already calls, render the result as markdown text (`core.RenderNode`/`RenderPatch`, or a new table for `node_grep`), never new business logic (ADR 003). `specs/008-arc-serve-mcp`. |
| **Transport** | The wire framing `arc serve` runs its `mcp.Server` over: `mcp.StdioTransport` by default (newline-delimited JSON over stdin/stdout) or `mcp.NewStreamableHTTPHandler` (Streamable HTTP/SSE) when `--http <addr>` is given. Both front the identical registered tool set — only the framing differs (spec SC-007). ADR 003, `specs/008-arc-serve-mcp`. |
| **Bind Address** | The `[host]:port` value `arc serve --http <addr>` parses via `resolveHTTPAddr`: a bare port or `:port` (no host) resolves to `127.0.0.1` (loopback-only); an explicit host binds exactly that host. A syntactically invalid address, or one already in use, refuses to start (spec FR-003/FR-005). `specs/008-arc-serve-mcp`, research.md D5. |
| **Provenance Timestamp Attributes** | `published`/`indexed`/`updated` — a node's provenance readable directly from its own file. `published` (`internal/core.Node.Published`, a typed field, date-only) is the source document's declared publication date, filled once on creation or first merge and never overwritten thereafter; `indexed`/`updated` (plain `Attrs` strings, RFC 3339) are stamped exclusively by `internal/app/graph/service.Apply` — `indexed` once at node creation, `updated` on any later merge that actually changes the node's rendered content. A stub node or a `_schema/` document carries none of the three. `specs/009-node-timestamp-attrs`. |
| **Application Timestamp** | One `time.Now().UTC()` captured once near the top of a single `internal/app/graph/service.Apply` invocation, formatted once (RFC 3339) and reused verbatim as the value stamped into every node's `indexed` (on create) or `updated` (on an actually-changed merge) for that invocation — guaranteeing every node touched by one application shares an identical value. `specs/009-node-timestamp-attrs`, research.md D5. |
| **Exclusively-Owned Node** | A node file path `p` for which `len(CommitsTouching(p)) == 1` — the reverted patch's own ingest commit is the only commit that ever changed it. `arc revert` deletes such a node outright and sweeps every referrer's backlink to it (research.md D5/D6, `specs/016-arc-revert`). |
| **Shared Node** | A node file path `p` for which `len(CommitsTouching(p)) > 1` and the reverted patch's ingest commit is one of them — at least one other patch has also touched it since. `arc revert` never deletes a shared node; it strips only the reverted patch's own attributable text contribution (`git blame`-mapped paragraphs, or a resolved conflict marker), leaving `Attrs`/`Edges`/`HRefs` untouched (research.md D7-D9, `specs/016-arc-revert`). |
| **Reconciliation Approach** | `arc revert`'s own `RevertResult.Approach`: `"whole-commit"` when every path the ingest commit touched passes its per-path eligibility test (nothing has touched it since — a plain `git revert` applies), or `"per-node"` otherwise (a node-by-node walk classifying each touched path as an Exclusively-Owned Node or a Shared Node). Computed once per revert (research.md D3/D4, `specs/016-arc-revert`). |
| **Ingest Commit** (`arc revert`) | The same commit `arc apply` produces (see the earlier **Ingest Commit** entry), located for a given `source-id` via `CommitsMatching(dir, "Source-Id: "+sourceID)` — `arc revert`'s own starting point, reusing the identical `Source-Id:` trailer identity `arc lint`'s `RuleIngestCommit` already relies on rather than a second lookup convention. `internal/app/graph/service.Revert`, research.md D1, `specs/016-arc-revert`. |
| **`arcnet:` Catalog Reference** | A single positional input to `arc apply schema` beginning with the literal prefix `arcnet:` — the remainder is a path within the official arcnet extensions catalog, resolved against the fixed base `kernel.ArcnetCatalogBaseURL` (`https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/`) and fetched exactly like a directly supplied `http(s)://` URL. A bare `arcnet:` with nothing after the prefix is rejected before any fetch attempt. `internal/app/schema/service.classifySource`, research.md D1/D1a, `specs/018-apply-schema-patch`. |
| **`kernel.ApplySchemaResult`** | The value `internal/app/schema/service.ApplyPatch` returns: `Source` (the resolved local path or URL the patch was read from), `Created`/`Merged` (counts keyed `"predicate"`/`"type"`), and `CommitHash` (empty on a no-op re-apply — no `Skipped` boolean, unlike `graph.kernel.ApplyResult`, since a schema patch carries no source-tracking idempotency concept). `internal/app/schema/kernel.ApplySchemaResult`, `specs/018-apply-schema-patch`. |
