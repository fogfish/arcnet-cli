# Architecture

`arc` is a single-binary Cobra CLI (module `github.com/fogfish/arcnet-cli`) built per [ADR 001](adrs/001-system-architecture.md) (hexagonal/onion layering, screaming architecture) and [ADR 002](adrs/002-ux-design-system.md) (CLI UX design system). Both ADRs are Accepted and BINDING (constitution Principle I).

## Directory Structure

```text
cmd/arc/                    # sole primary (driving) adapter: Cobra command tree
├── main.go                 # entrypoint, calls newRootCmd().Execute()
├── root.go                 # root command, DS-03 persistent flags, PersistentPreRun schema selection
└── ctrl/                   # Cobra wiring for the ctrl (graph management) domain
    └── init.go              # `arc init` command: flag/arg parsing, calls internal/app/ctrl.Init

internal/
├── bios/                    # shared kernel (ADR 002 DS-04/05/06) — output modes, color schema,
│                             #   progress reporter. Reused by every future command; not tied to
│                             #   any single use-case.
├── adapter/
│   └── fsys/                # shared, cross-use-case filesystem adapter (ADR 001 "phase 2" adapter
│                             #   tier). The ONLY package permitted to call os's file/directory
│                             #   functions (constitution Principle VII, Mandatory Libraries &
│                             #   Tooling: "Filesystem Abstraction"). Built on stdlib io/fs/io.Writer
│                             #   only — no third-party filesystem library.
└── app/
    └── ctrl/                 # first domain use-case: graph management / control plane
        ├── kernel/            # domain value types (GraphRoot, ArcNetCoreLayout, InitResult)
        ├── port/               # ctrl-private secondary port (VCS) — not imported by other use-cases
        ├── adapter/
        │   ├── git/            # os/exec-backed real VCS implementation
        │   └── mock/           # in-memory fake VCS for service unit tests
        ├── service/            # use-case logic (Init)
        └── component.go        # primary port: Init(ctx, mounter, vcs, dir) (kernel.InitResult, error)
```

`internal/app/ctrl` is the first `internal/` package in this codebase, so ADR 001's `componentX` layout (`kernel/`, `port/`, `adapter/`, `service/`, `component.go`) now takes full effect. `internal/bios` and `internal/adapter/fsys` are deliberately shared, not use-case-private, since every future command needs an output/color/reporter kernel and every future graph-root-mounting command needs the same filesystem mount contract (research.md D3/D5 in `specs/002-arc-init/`).

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
