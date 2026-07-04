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
./arc lint
```

`arc init` bootstraps a new, empty knowledge graph in the current directory (or an optional target directory): the canonical folder layout, a first-class, versioned `_schema/` seeded with every ARCNET-CORE node kind and predicate, the `.arc/` local state directory, a `.gitignore`, and a single initial git commit. Initialization is fully offline — no network access required.

`arc apply` ingests a document patch into an already-initialized graph: it creates or merges every node the patch carries, derives and appends timeline entries, auto-registers any previously-unseen node kind or predicate into `_schema/` in the same commit, and produces exactly one commit. Re-applying an already-tracked document is a safe no-op.

`arc lint` validates the graph against the full CORE §14 conformance checklist — front-matter/kind, unique basenames, resolvable links, source citekey identity, entity Sowa category, predicate registration, citation predicates, one ingest commit per document, and absence of merge-conflict markers — reporting every violation with its file and line. `_schema/` documents are exempt from these checks. It is strictly read-only.

See [specs/001-cli-infrastructure/quickstart.md](specs/001-cli-infrastructure/quickstart.md), [specs/002-arc-init/quickstart.md](specs/002-arc-init/quickstart.md), [specs/003-apply-patch/quickstart.md](specs/003-apply-patch/quickstart.md), [specs/004-arc-lint/quickstart.md](specs/004-arc-lint/quickstart.md), and [specs/005-graph-schema-first-class/quickstart.md](specs/005-graph-schema-first-class/quickstart.md) for the full development quickstarts.
