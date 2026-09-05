<h1 align="center">arc</h1>
<p align="center"><strong>Your knowledge graph as plain Markdown, versioned by git</strong></p>

<p align="center">
  <a href="https://github.com/fogfish/arcnet-cli/releases">
    <img src="https://img.shields.io/github/v/tag/fogfish/arcnet-cli?label=version" />
  </a>
  <a href="https://github.com/fogfish/arcnet-cli/actions/">
    <img src="https://github.com/fogfish/arcnet-cli/workflows/build/badge.svg" />
  </a>
  <a href="http://github.com/fogfish/arcnet-cli">
    <img src="https://img.shields.io/github/last-commit/fogfish/arcnet-cli.svg" />
  </a>
  <a href="https://coveralls.io/github/fogfish/arcnet-cli?branch=main">
    <img src="https://coveralls.io/repos/github/fogfish/arcnet-cli/badge.svg?branch=main" />
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" />
  </a>
</p>

---

`arc` turns a folder of Markdown files into a knowledge graph you own. It ingests **patch documents** — self-contained Markdown serializations of what a paper, an article, or an AI chat session taught you — merges them into typed, interlinked nodes, records every ingest as a git commit, and serves the result back to your LLM tooling over the Model Context Protocol.

There is no database and no proprietary format. The graph is a git repository of Markdown files: readable in Obsidian, searchable with `grep`, diffable in a pull request, and outliving any tool that made it — including this one.

> [!IMPORTANT]
> **`arc` does not produce content — it manages it.** Turning a paper, an article, or a chat transcript into a patch document is an *extraction* step performed by an external system (an LLM). The patch format and the extraction contract are specified at **[fogfish/arcnet-spec](https://github.com/fogfish/arcnet-spec)**. The simplest way to get started is **[CHAT-TO-PATCH.md](https://github.com/fogfish/arcnet-spec/blob/main/CHAT-TO-PATCH.md)** — a ready-to-use prompt that converts a ChatGPT (or Claude, or any assistant) session into a knowledge graph patch. See [Producing content](#producing-content).


## Features

* **Plain Markdown, plain git** — every node is one Markdown file with YAML front-matter; every ingest is one commit with full provenance. No lock-in, no index to rebuild, no server to run.
* **Typed, self-describing graph** — the vocabulary lives *in* the graph as a first-class, versioned `_schema/` tree: one document per class, one per predicate, each declaring its own merge behaviour.
* **Idempotent, mergeable ingestion** — re-applying a patch is a safe no-op; re-ingesting a lightly edited document merges per-predicate instead of duplicating or clobbering.
* **Clean inverse** — `arc revert` retracts a document's entire contribution, reconciling nodes other patches have since enriched.
* **Conformance linting** — `arc lint` validates the whole graph against the ARCNET-CORE checklist and reports every violation with file and line.
* **Whole-graph statistics** — `arc stats` reports a graph's size, composition by type, broken links and ingestion coverage in one read-only pass; `--verbose` adds connectivity, hubs, schema coverage and content volume.
* **Precise retrieval** — search node content with `arc grep`, extract a self-contained neighbourhood with `arc subgraph`, both narrowed by a triple-based `--type`/`--tag`/`--attr`/`--predicate` filter.
* **Native MCP server** — `arc serve` exposes eight read-only tools over stdio or HTTP, so an agent can read your graph directly instead of being pasted context.
* **Extension-agnostic** — domain profiles add new types and predicates through `arc apply schema`; no special subcommand, no rebuild.

> [!NOTE]
> `arc` is pre-1.0 and experimental. Breaking vocabulary changes are shipped without a migration path, by design — re-initialize instead.


## Installation

<details>
<summary>macOS - Homebrew (Recommended)</summary>

```bash
brew tap fogfish/arcnet-cli https://github.com/fogfish/arcnet-cli
brew install arcnet-cli
```

Upgrade to the latest version

```bash
brew upgrade arcnet-cli
```

</details>

<details>
<summary>Direct binaries download (Linux, Windows)</summary>

Download the executable for your platform from the [latest release](https://github.com/fogfish/arcnet-cli/releases/latest), rename it to `arc`, make it executable, and add it to your PATH.

</details>

<details>
<summary>Build from sources</summary>

Requires Go [installed](https://go.dev/doc/install).

```bash
go install github.com/fogfish/arcnet-cli/cmd/arc@latest
```

Or from a checkout of this repository:

```bash
go build -o arc ./cmd/arc
```

</details>

Verify the installation:

```bash
arc --version
arc --help
```

`arc` requires **git** on your PATH. It needs no configuration, no API key, and no network access — except `arc apply schema` when you point it at a URL or the `arcnet:` catalog.


## Producing content

`arc` is the *librarian*, not the *author*. It manages an already-distilled knowledge graph; it does not read your papers or your chat logs for you.

The unit of exchange is a **patch document**: one Markdown file, one front-matter manifest, nodes grouped by type under `# Source` / `# Entity` / `# Reference` headings, and `predicate:: [[Target]]` edges between them. Producing one is the job of an external system — in practice, an LLM following an extraction prompt.

* **The format and the extraction contract are specified at [github.com/fogfish/arcnet-spec](https://github.com/fogfish/arcnet-spec).** Start with [`ARCNET-CORE.md`](https://github.com/fogfish/arcnet-spec/blob/main/ARCNET-CORE.md) for the node model, identity rules, edges, merges, and the patch format itself.
* **The simplest way to produce your first patch is [CHAT-TO-PATCH.md](https://github.com/fogfish/arcnet-spec/blob/main/CHAT-TO-PATCH.md)** — a complete, copy-paste prompt that converts a working session with an assistant into a patch document: a `Source` for the session, an `Entity` per durable subject, a `Reference` per external work it pointed at, and a `Thought` per core insight, decision, or open question the session produced. Paste the prompt at the end of a ChatGPT session, save the reply as `<citekey>.patch.md`, and hand it to `arc apply`.

Any producer works — a script, a hand-written file, a different model, your own prompt — as long as it emits conformant patch documents.


## Quick Start

**1. Create a graph.**

```bash
arc init my-graph
cd my-graph
```

This is a complete, offline bootstrap: the canonical folder layout, a versioned `_schema/` seeded with the entire ARCNET-CORE built-in vocabulary (one document per type and per predicate), the `.arc/` state directory with its own rule keeping it out of version control, and the initial git commit.

**Adding a graph to a project you already version.** `arc init` refuses to run inside an existing git repository rather than nesting a new one inside yours; it names the repository it found and the flag that overrides the refusal:

```bash
cd my-project
arc init --skip-git-init notes     # graph in notes/, committed to my-project's own history
```

With `--skip-git-init` no repository is created, the commit contains only the files initialization itself wrote — anything you had modified, staged or untracked is left exactly as it was — and your project's own `.gitignore` is never read, created or modified. If a canonical folder name such as `Source/` is already taken, `arc` refuses and points you at the recovery: initialize into a subfolder.

```
my-graph/
├── Source/              # documents ingested into the graph
├── Entity/              # durable subjects — concepts, systems, protocols
├── Reference/           # external works cited but not ingested
├── Resource/            # attached resources
├── timeline/            # yearly/ and monthly/ chronology, derived on ingest
├── _schema/
│   ├── Class/           # one document per node type
│   └── Property/        # one document per predicate, with its merge policy
├── .arc/                # local state, excluded from git by its own .gitignore
└── .git/
```

**2. Produce a patch.** Take a chat session you want to keep, append the [CHAT-TO-PATCH.md](https://github.com/fogfish/arcnet-spec/blob/main/CHAT-TO-PATCH.md) prompt, and save the assistant's reply outside the graph directory — say `~/patches/2026-june-tls13.patch.md`.

That prompt emits `Thought` nodes, which are a domain extension rather than core vocabulary. Import the extension once, into this graph:

```bash
arc apply schema arcnet:domain-core-thought.md
```

**3. Ingest it.**

```bash
arc apply ~/patches/2026-june-tls13.patch.md
```

```
Applied 2026-june-tls13: +1 Entity, +1 Source (commit 082fe27)
```

One patch, one commit. `arc` creates or merges every node the patch carries, derives the timeline entries, and auto-registers any previously-unseen type or predicate into `_schema/` in the same commit. Applying the same patch twice is a no-op.

**4. Read the graph back.**

```bash
arc grep TLS
```

```
Source    rescorla-2024-tls13  6   title: 'TLS 1.3: Design and Rationale'
Source    rescorla-2024-tls13  12  A design retrospective on the TLS 1.3 handshake.
Timeline  2024-03              10  - cites:: [[rescorla-2024-tls13]] — *TLS 1.3: …* — 2024-03-11
```

```bash
arc subgraph "Transport Layer Security" --depth 2
```

`arc subgraph` prints the seed node plus everything within `--depth` hops as a *patch-exchange document* — the same format `arc apply` reads. That makes it equally suitable for piping into another graph or pasting into an LLM prompt as hand-tuned context.

**5. Check it.**

```bash
arc lint
```

Every node is validated against the ARCNET-CORE conformance checklist — front-matter, unique basenames, resolvable `[[links]]`, citekey identity, Sowa category, predicate registration, provenance, one ingest commit per document. Violations are reported with file and line; the run never stops at the first one.

**6. Measure it.**

```bash
arc stats
arc stats --verbose
```

`arc stats` reports how many nodes the graph holds, how they break down by declared type, how many links point at nothing, and how much was ingested in each year. `--verbose` adds edges by predicate, monthly ingestion, disconnected and stub nodes, degree and hub ranking, schema coverage and content volume. Nodes are recognized by what each file declares, never by the folder holding it, so reorganizing the graph changes no reported figure. Broken links are a reported figure, not a failure: `arc stats` exits 0 whenever it produced a report — gating on graph health stays `arc lint`'s job.

**7. Serve it to your agent.**

```bash
arc serve --http :8080
```

An MCP server over stdio, exposing your graph read-only to any MCP client.

> [!TIP]
> Keep patch documents **outside** the graph directory. A `.patch.md` file left inside the graph is itself walked by `arc lint` and reported as a malformed node.


## Usage

`arc` operates on the graph in the **current working directory** — the folder containing `.arc/`. Run commands from the graph root.

```bash
# bootstrap
arc init [<dir>]                          # initialize a new, empty knowledge graph
arc init [<dir>] --skip-git-init          # ...into an existing git repository, using its history

# ingest
arc apply <patch.md>                      # ingest one patch document
arc apply batch <dir>                     # ingest every patch in a directory tree
arc apply schema <patch.md>|<url>|arcnet:<name>   # import Class/Property definitions

# retract
arc revert <source-id> [--force]          # retract a document's contribution

# read
arc grep <pattern> [filter]               # search node content
arc subgraph <basename> [--depth n] [filter] [--stubs]
arc lint [--skip <rule[,rule...]>]        # validate against the conformance checklist
arc lint rules                            # list every rule --skip can name
arc stats [--verbose]                     # report the graph's shape and health

# serve
arc serve [--http <addr>]                 # MCP server over stdio or HTTP
```

Use `arc help <command>` for details.

### Global options

| Flag              | Description                                      |
| ----------------- | ------------------------------------------------ |
| `-q`, `--quiet`   | Suppress progress output; errors are still shown |
| `-v`, `--verbose` | Show additional diagnostic detail                |
| `--json`          | Machine-readable structured output               |
| `-C`, `--color`   | Force-enable color (auto-detected otherwise)     |

### Filtering

`arc grep` and `arc subgraph` accept a filter that narrows which nodes they operate on. Under the hood a filter is a conjunction of `(source, predicate, target)` triple statements, so a node can be selected by what it *is* (its attributes) or by what it *points at* (its relations), uniformly.

| Flag                       | Semantics                                                                                          |
| -------------------------- | -------------------------------------------------------------------------------------------------- |
| `--type <type>`            | Node type equals `<type>`. Repeatable, **OR**                                                      |
| `--tag <tag>`              | Node's `tags` contains `<tag>`. Repeatable, **AND**                                                |
| `--attr <name>=<value>`    | Attribute `<name>` equals `<value>` (case-insensitive; membership for arrays). Repeatable, **AND** |
| `--attr <name>~=<pattern>` | Attribute `<name>` matches the regexp `<pattern>`. Repeatable, **AND**                             |
| `--predicate <name>`       | *(`arc subgraph` only)* Follow only relations named `<name>` during expansion. Repeatable, **OR**  |

`--type`/`--tag`/`--attr` narrow **which reached nodes survive** into the result; `--predicate` narrows **what gets reached** in the first place. They compose freely:

```bash
arc grep --tag cryptography --attr status=mature "TLS 1\.3"
arc subgraph TLS --predicate cites --type Source --depth 2
```

See [specs/VISION.md](specs/VISION.md) for the full filter model, including the MCP wire shape.


## Typical workflows

### Chat sessions as cross-session memory

The core loop `arc` was built for: a working session is an ingestable document. Distill it, ingest it, and the graph becomes memory your next session can query.

```bash
# once per graph — the extraction prompt emits Thought nodes
arc apply schema arcnet:domain-core-thought.md

# after each session: append the CHAT-TO-PATCH prompt, save the reply, ingest it
arc apply ~/patches/2026-june-tls13.patch.md

# later: what have I concluded about this?
arc grep --type Thought "handshake"
arc subgraph "Transport Layer Security" --depth 2
```

Entities and thoughts accumulate across sessions and merge by identity, so the tenth session about a subject enriches the same nodes the first one created rather than starting over.

### Backfilling a library

Already have a pile of patches — a year of sessions, a reading list, output from a batch extraction run?

```bash
arc apply batch ./patches
```

`arc` walks the directory recursively, skips Markdown that carries no patch manifest, never descends into hidden directories, and applies each patch oldest-publication-date-first, one commit per patch. Documents already in the graph are skipped, so re-running over a growing directory — or resuming an interrupted run — is safe. Failures are recorded and the run continues; `--fail-fast` halts at the first one instead. The closing summary reports every count, every failure with its reason, and every merge conflict flagged across the run; `--json` emits the same result machine-readably; the exit code is non-zero if any patch failed. The patch directory itself is never modified.

### Extending the vocabulary

Domain profiles add types and predicates without touching `arc` itself:

```bash
arc apply schema arcnet:domain-article.md        # official catalog shorthand
arc apply schema https://example.org/media.schema.md
arc apply schema ./my-domain.schema.md           # your own
```

Each Class/Property definition is created or merged into `_schema/` and committed. A patch containing any non-Class/Property node fails the whole operation *before* any write happens. The `arcnet:` shorthand resolves into the [official arcnet-spec catalog](https://github.com/fogfish/arcnet-spec/tree/main/schema) (`core.md`, `domain-article.md`, `domain-core-thought.md`, `domain-incident.md`).

Every predicate declares its merge behaviour from the closed set of six — `immutable`, `union`, `firstWriteWin`, `fillIfEmpty`, `lastWriteWin`, `append` — and a schema document declaring anything else is refused outright. Merge is a property of a *predicate*, never of a type, so type definitions carry no merge declaration at all.

### Serving the graph to an LLM agent

```bash
arc serve                          # stdio, for a local MCP client
arc serve --http :8080             # Streamable HTTP/SSE, loopback only
arc serve --http 0.0.0.0:8080      # bind explicitly
```

A bare port or `:port` binds `127.0.0.1` only; an explicit host binds exactly that host. The server is strictly read-only and never modifies the graph or its git history.

Point an MCP client at it — for example, Claude Code:

```bash
claude mcp add arcnet -- arc serve
```

Eight read-only tools are exposed:

| Tool               | Purpose                                                                                                           |
| ------------------ | ----------------------------------------------------------------------------------------------------------------- |
| `schema`           | The graph's full ontology — every class and predicate, with descriptions. **Recommended first call of a session** |
| `node_get`         | One node's full stored content by `@id`                                                                           |
| `node_grep`        | Regexp search over node content, optionally filtered                                                              |
| `node_match`       | Every distinct `{id, property, value}` fact satisfying a filter                                                   |
| `node_links`       | A node's outgoing relations as `{predicate, target}` rows                                                         |
| `node_backlinks`   | Every relation targeting a node, as `{source, predicate}` rows                                                    |
| `subgraph_get`     | The resolved subgraph around a seed, to a hop depth, filterable                                                   |
| `context_retrieve` | Free-text query + attribute match + one-hop expansion, ranked and truncated                                       |

Every connecting client is told at session start to call `schema` first.

### Retracting a document

Ingested something you shouldn't have, or want to re-ingest a document from scratch with a better extraction?

```bash
arc revert rescorla-2024-tls13
arc apply ~/patches/rescorla-2024-tls13.v2.patch.md
```

`arc revert` locates the ingest commit by its `Source-Id:` trailer and retracts that patch's contribution — a whole-commit `git revert` when nothing has since touched any file it changed, or a per-node reconciliation otherwise: removing a node outright when the reverted patch was its sole author, and stripping only the reverted patch's own text contribution from a node another patch has since enriched. Because this removes graph content, it asks for confirmation unless `--force`/`-f` is given. Re-reverting an already-retracted document is a safe no-op.

### Collaborating on a graph

The graph *is* a git repository, so sharing needs no special mechanism:

```bash
git remote add origin git@github.com:me/my-graph.git
git push -u origin main
```

Contributors clone it, `arc apply` their own patches, and open pull requests — where a diff of the graph is a readable diff of Markdown. Run `arc lint` in CI to gate merges.


## Documentation

- **[ARCNET specification](https://github.com/fogfish/arcnet-spec)** — the file format and folder convention `arc` implements, tool-agnostic
- **[CHAT-TO-PATCH.md](https://github.com/fogfish/arcnet-spec/blob/main/CHAT-TO-PATCH.md)** — the extraction prompt: a chat session in, a patch document out
- **[specs/VISION.md](specs/VISION.md)** — the roadmap and the full command model, including filtering
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — how `arc` is built
- **Per-feature quickstarts** — [cli](specs/001-cli-infrastructure/quickstart.md), [init](specs/002-arc-init/quickstart.md), [apply](specs/003-apply-patch/quickstart.md), [lint](specs/004-arc-lint/quickstart.md), [schema](specs/005-graph-schema-first-class/quickstart.md), [grep](specs/006-arc-grep-content-search/quickstart.md), [subgraph](specs/007-arc-subgraph/quickstart.md), [serve](specs/008-arc-serve-mcp/quickstart.md), [revert](specs/016-arc-revert/quickstart.md)


## Contributing

`arc` is [MIT](LICENSE) licensed and accepts contributions via GitHub pull requests:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request


## License

[![See LICENSE](https://img.shields.io/github/license/fogfish/arcnet-cli.svg?style=for-the-badge)](LICENSE)
