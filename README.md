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
```

`arc init` bootstraps a new, empty knowledge graph in the current directory (or an optional target directory): the canonical folder layout, `_meta/` registry stubs, the `.arc/` local state directory, a `.gitignore`, and a single initial git commit.

See [specs/001-cli-infrastructure/quickstart.md](specs/001-cli-infrastructure/quickstart.md) and [specs/002-arc-init/quickstart.md](specs/002-arc-init/quickstart.md) for the full development quickstarts.
