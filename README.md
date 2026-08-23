# arcnet-cli
CLI for Knowledge Graph

## Install

Download a release binary from the [releases page](https://github.com/fogfish/arcnet-cli/releases), or build from source:

```bash
go build -o arc ./cmd/arc
```

## Quick start

```bash
./arc --help
./arc --version
./arc init
./arc apply rescorla-2026-tls13.patch.md
./arc apply batch ./patches
./arc apply schema arcnet:media.schema.md
./arc revert rescorla-2026-tls13
./arc lint
./arc grep TLS
./arc subgraph TLS
./arc serve
```

`arc init` bootstraps a new, empty knowledge graph in the current directory (or an optional target directory): the canonical folder layout, a first-class, versioned `_schema/` seeded with ARCNET-CORE v0.11's full built-in vocabulary — every core type and predicate, one document each — the `.arc/` local state directory, a `.gitignore`, and a single initial git commit. Initialization is fully offline — no network access required.

Every seeded predicate declares its merge behaviour from CORE §9.3's closed set of six — `immutable`, `union`, `firstWriteWin`, `fillIfEmpty`, `lastWriteWin`, `append` — and a schema document declaring anything else is refused outright. Type definitions carry no merge declaration at all: merge is a property of a predicate, not of a type. Type-specific prose (a document's `abstract`, an entity's `definition`, a reference's `relevance`, a definition's `description`) is single-valued and first-fixed, so re-ingesting a lightly edited document preserves the established text and flags the divergence for review rather than accumulating paragraphs. The universal base type requires no predicates; a document requires its own `title`, `published`, `abstract`, and `mentions`, a timeline period requires only its `cites`, and an external-work reference requires only its `title`.

> `arc` is pre-1.0 and experimental. This vocabulary correction is a breaking change for a graph seeded by an earlier release — no migration path is provided, by design; re-initialize instead.

`arc apply` ingests a document patch into an already-initialized graph: it creates or merges every node the patch carries, derives and appends timeline entries, auto-registers any previously-unseen node kind or predicate into `_schema/` in the same commit, and produces exactly one commit. Re-applying an already-tracked document is a safe no-op.

`arc apply batch <dir>` ingests a whole directory of patches in one invocation: it walks `<dir>` recursively, passes over Markdown that declares no patch manifest, never descends into hidden directories, and applies every applicable patch through that same single-patch algorithm — oldest publication date first (ties broken by relative path), one commit per patch. Documents already tracked in the graph are skipped, so re-running over a growing directory, or resuming an interrupted run, is safe. A patch that cannot be applied is recorded with its reason and the run continues; `--fail-fast` halts at the first failure instead, leaving the remainder unprocessed. The closing summary reports every count plus each failure and every merge conflict flagged across the whole run, `--json` emits the same result machine-readably, and the command exits non-zero if any patch failed. The patch directory itself is never modified.

`arc apply schema <patch.md> | <url> | arcnet:<name>` imports Property/Class schema definitions from a patch document — a local file, a URL, or a short `arcnet:<name>` reference into the official arcnet extensions catalog — creating or merging each one into `_schema/`. Any non-Property/Class node anywhere in the patch fails the whole operation before any write happens.

`arc revert <source-id>` retracts a patch document's contribution from the graph: a whole-commit `git revert` when nothing has since touched any file it changed, or a per-node reconciliation otherwise — removing a node outright when the reverted patch was its sole author, or stripping only the reverted patch's own text contribution from a node another patch has since enriched. Removing graph content is destructive, so `arc revert` asks for confirmation unless `--force`/`-f` is given. Re-reverting an already-retracted document is a safe no-op.

`arc lint` validates the graph against the full CORE §14 conformance checklist — front-matter/kind, unique basenames, resolvable links, source citekey identity, entity Sowa category, predicate registration, citation predicates, one ingest commit per document, and absence of merge-conflict markers — reporting every violation with its file and line. `_schema/` documents are exempt from these checks. It is strictly read-only.

`arc grep <pattern>` scans every node file's content for lines matching a regexp, optionally narrowed by a `--type`/`--tag`/`--attr` filter (see Filtering in [specs/VISION.md](specs/VISION.md)), printing one `<type>  <id>  <line>  <text>` row per match — suitable for piping to standard tools. It is strictly read-only.

`arc subgraph <basename>` extracts a seed node plus everything reachable from it within `--depth` hops (both outgoing and incoming structural connections, default `1`), optionally narrowed by the same `--type`/`--tag`/`--attr` filter, and serializes the result as one patch-exchange document — ready to re-ingest via `arc apply` or paste into an LLM prompt. It is strictly read-only.

`arc serve [--http <addr>]` starts a Model Context Protocol (MCP) server exposing three read-only tools — `node_get`, `node_grep`, `subgraph_get` — backed by the same use-case functions `arc grep`/`arc subgraph` already call, so an LLM client can read the graph directly. It serves over stdio by default, or over Streamable HTTP/SSE when `--http <addr>` is given (a bare port or `:port` binds loopback-only; an explicit host binds exactly that host). It is strictly read-only.

See [specs/001-cli-infrastructure/quickstart.md](specs/001-cli-infrastructure/quickstart.md), [specs/002-arc-init/quickstart.md](specs/002-arc-init/quickstart.md), [specs/003-apply-patch/quickstart.md](specs/003-apply-patch/quickstart.md), [specs/004-arc-lint/quickstart.md](specs/004-arc-lint/quickstart.md), [specs/005-graph-schema-first-class/quickstart.md](specs/005-graph-schema-first-class/quickstart.md), [specs/006-arc-grep-content-search/quickstart.md](specs/006-arc-grep-content-search/quickstart.md), [specs/007-arc-subgraph/quickstart.md](specs/007-arc-subgraph/quickstart.md), [specs/008-arc-serve-mcp/quickstart.md](specs/008-arc-serve-mcp/quickstart.md), and [specs/016-arc-revert/quickstart.md](specs/016-arc-revert/quickstart.md) for the full development quickstarts.
