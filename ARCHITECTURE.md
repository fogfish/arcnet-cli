# Architecture

`arc` is a single-binary Cobra CLI (module `github.com/fogfish/arcnet-cli`) built per [ADR 001](adrs/001-system-architecture.md) (hexagonal/onion layering, screaming architecture) and [ADR 002](adrs/002-ux-design-system.md) (CLI UX design system). Both ADRs are Accepted and BINDING (constitution Principle I).

## Directory Structure

```text
cmd/arc/                    # sole primary (driving) adapter: Cobra command tree
├── main.go                 # entrypoint, calls newRootCmd().Execute()
├── root.go                 # root command, DS-03 persistent flags, PersistentPreRun schema selection
├── ctrl/                   # Cobra wiring for the ctrl (graph management) domain
│   └── init.go              # `arc init` command: flag/arg parsing, calls internal/app/ctrl.Init,
│                             #   composes internal/app/config.Default's config-seed fetch
└── graph/                  # Cobra wiring for the graph (graph I/O) domain
    └── apply.go             # `arc apply` command: flag/arg parsing, calls
                              #   internal/app/config.Resolve then internal/app/graph.Apply

internal/
├── bios/                    # shared kernel (ADR 002 DS-04/05/06) — output modes, color schema,
│                             #   progress reporter. Reused by every future command; not tied to
│                             #   any single use-case.
├── core/                    # shared, use-case-independent core domain (ADR 001's "core domain"
│                             #   evolution phase): the graph AST (ARCNET-AST §4-6) as plain Go
│                             #   types, a goldmark-backed Markdown↔AST codec, the CORE §10 merge
│                             #   algebra, CORE §9.4 timeline-period derivation, and the CORE §9/§10
│                             #   kind/merge-rule vocabulary. No dependency on any internal/app/<use-case>.
├── adapter/
│   ├── fsys/                # shared, cross-use-case filesystem adapter (ADR 001 "phase 2" adapter
│   │                         #   tier). The ONLY package permitted to call os's file/directory
│   │                         #   functions (constitution Principle VII, Mandatory Libraries &
│   │                         #   Tooling: "Filesystem Abstraction"). Built on stdlib io/fs/io.Writer
│   │                         #   only — no third-party filesystem library.
│   └── git/                 # shared, cross-use-case git adapter (ADR 001 "phase 2" adapter tier,
│                             #   promoted from internal/app/ctrl/adapter/git once a second use-case
│                             #   needed git access, research.md D4 in specs/003-apply-patch/). The
│                             #   one concrete Git type satisfies both ctrl.port.VCS and
│                             #   graph.port.VCS structurally (ADR 001 port isolation rule 1).
└── app/
    ├── ctrl/                 # first domain use-case: graph management / control plane
    │   ├── kernel/            # domain value types (GraphRoot, ArcNetCoreLayout, InitResult)
    │   ├── port/               # ctrl-private secondary port (VCS) — not imported by other use-cases
    │   ├── adapter/
    │   │   └── mock/           # in-memory fake VCS for service unit tests
    │   ├── service/            # use-case logic (Init)
    │   └── component.go        # primary port: Init(ctx, mounter, vcs, dir, configSeed) (kernel.InitResult, error)
    │
    ├── config/                # second domain use-case: `.arc/config.yml` load/save/resolve/default
    │   ├── kernel/             # domain value types (Config)
    │   ├── port/                # config-private secondary port (Fetcher) for the config-seed fetch
    │   ├── adapter/
    │   │   ├── http/            # stdlib net/http-backed real Fetcher
    │   │   └── mock/            # fake Fetcher for service unit tests
    │   ├── service/             # use-case logic (Load, Save, Resolve, Default)
    │   └── component.go         # primary port: Resolve(store), Save(store, cfg), Default(ctx, fetcher)
    │
    └── graph/                 # third domain use-case: graph mutation / graph I/O
        ├── kernel/              # domain value types (ApplyResult)
        ├── port/                 # graph-private secondary port (VCS) — narrower than ctrl's
        ├── adapter/
        │   └── mock/             # in-memory fake VCS for service unit tests
        ├── service/              # use-case logic (Apply)
        └── component.go          # primary port: Apply(ctx, mounter, vcs, rules, dir, patchPath) (kernel.ApplyResult, error)
```

`internal/app/ctrl` is the first `internal/` package in this codebase, so ADR 001's `componentX` layout (`kernel/`, `port/`, `adapter/`, `service/`, `component.go`) now takes full effect. `internal/bios` and `internal/adapter/fsys` are deliberately shared, not use-case-private, since every future command needs an output/color/reporter kernel and every future graph-root-mounting command needs the same filesystem mount contract (research.md D3/D5 in `specs/002-arc-init/`). `internal/core` is the project's first core-domain package (ADR 001's own evolution model): the graph AST and its canonical Markdown serialization are a model invariant shared by every future graph-reading command, not an `apply`-specific concern, so they live below the use-case layer. `internal/adapter/git` is the first adapter promoted to the shared tier once a second use-case (`graph`) needed the same capability `ctrl` already had (research.md D4 in `specs/003-apply-patch/`), mirroring `internal/adapter/fsys`'s precedent.

## Command Grammar (Principle IX)

This project uses **bare top-level verbs** (`arc init`, `arc apply`, `arc list`, ...), not noun-verb nesting — permitted by ADR 002 DS-01 because the entire tool operates on exactly one kind of subject, a knowledge graph. Every subcommand follows this convention without exception.

## Glossary

| Term | Definition |
|---|---|
| **Graph Root** | The directory tree representing one knowledge graph instance; identified by the presence of a `.arc/` directory at its top level. Resolved and mounted via `internal/adapter/fsys` (`ResolveLocalRoot` then `Mounter.Mount`). |
| **Canonical Folder** | One of the fixed top-level directories every graph must contain: `sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`. Defined statically by `internal/app/ctrl/kernel.ArcNetCoreLayout`. |
| **Metadata Stub** | A registry placeholder file (`_meta/predicates.md`, `_meta/aliases.md`) that later commands append controlled-vocabulary entries to. |
| **Arc State Directory** | The `.arc/` directory holding tool-managed local state, never versioned alongside graph content (excluded via `.gitignore`). Its presence is what distinguishes an initialized Graph Root from an empty directory. |
| **Initial Commit** | The single git commit produced by `arc init` that records a graph's creation, with the mandatory subject line `graph(init): empty knowledge graph` (CORE §11.3). |
| **Patch** | A CORE §12 Markdown document — one manifest (`document`, `published`, `title`, `stats`) plus H1-kind/H2-node sections — that `arc apply` ingests into the graph. Parsed by `internal/core.ParsePatch` into `internal/core.Patch`. |
| **Node Contribution** | One H2 node section within a patch: the create-or-merge unit `arc apply` applies to the graph, one per patch-carried `internal/core.Node`. |
| **Source Node** | A node of kind `source` (CORE's fixed, always-recognized `MergeNone` kind) — the citable document a patch itself represents. |
| **Entity/Resource Node** | A node of kind `entity` (`MergeUnion`) or `resource` (`MergeUnionFirstWriter`) — CORE's fixed kinds for concepts and referenced material, mergeable across multiple contributing patches. |
| **Timeline Entry** | One chronologically-ordered bullet appended to a `timeline/yearly/<YYYY>.md` or `timeline/monthly/<YYYY-MM>.md` period file, derived from a patch's `published` manifest field (CORE §9.4, `internal/core.TimelinePeriods`/`.TimelineEntry`). |
| **Merge Behavior** | The `internal/core.MergeOp` (`none`, `union`, `union-first-writer`, `append`, `validated-overwrite`) a node's kind is registered against, determining how `internal/core.Merge` reconciles an incoming contribution with an existing node. |
| **Ingest Commit** | The single git commit `arc apply` produces per invocation, subject naming the applied document, with per-kind created/merged stats and a `Source-Id:` trailer (CORE §11.3). |
| **Kind Registration** | An entry in `.arc/config.yml`'s `mergeRules` map associating a domain-specific node kind with a `Merge Behavior`, beyond CORE's fixed kinds. An unregistered kind still applies, using the safe `union` default, with a warning (spec FR-018). |
