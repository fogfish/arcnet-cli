# Changelog

# 2026-07-02

/speckit-specify `arc apply <patch.md>` — apply a patch file to the graph (CORE §12.3): parse the patch manifest (`kind: patch`, `document`, `published`, `stats`); check idempotency and skip with a clear message if `sources/<id>.md` is already tracked (CORE §11.2); for each H1/H2 node section reconstruct the node object (ARCNET-AST §4 ); **create** new node files when the basename does not exist; **merge** into existing files per the kind's declared merge operation — `none` for `source`, `union` for `entity`, `union first-writer` for `resource`, and per-profile operation for domain/extension kinds (CORE §10); derive and append timeline entries from the source's `published` date (CORE §9.4); produce exactly one git commit with the mandatory subject, stats, and `Source-Id:` trailer (CORE §11.3); update the local index cache (Phase 4) atomically within the same filesystem transaction

See specifications:
* CORE https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md
* ARCNET-AST https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md

/speckit-plan graph I/O is own domain `internal/app/graph`. Also maintain same hierarchy in `cmd/arc/graph`. Define the graph AST https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md as core domain (`internal/core`). Parse the patch using `github.com/yuin/goldmark` markdown parser into AST and then use AST to patch the graph itself. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux0design-system.md). Establish YAML based config at `.arc/config.yml` (as part of `arc init`), this version of the config defines only merge rules per kind, the config management is implemented at `internal/app/config`. The default merge rules for all supported profiles is defined at `github.com/fogfish/arcnet-spec`. 

---

/speckit-specify `arc init [<dir>]` — initialize a new knowledge graph: create the canonical folder layout (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`); write stub files `_meta/predicates.md` and `_meta/aliases.md`; create `.arc/` for arc-managed state (see Graph Root from VISION.md); run `git init` and create `.gitkeep` for empty folders; write `.gitignore` excluding `.arc/`; stage everything and produce the initial commit `graph(init): empty knowledge graph` (CORE §11 https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md)

/speckit-plan graph management (control plane) is own domain `internal/app/ctrl`. Also maintain same hierarchy in `cmd/arc/ctrl`. Integrate `git` as first class concept via invocation of command-line tool, informing user of `arc` about the git tool progress. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux0design-system.md).       

# 2026-07-01

/speckit-specify setup the infrastructure for development of cli called `arc`. The infrastructure includes (1) an empty cobra application; (2) github actions to test, check and release application; (3) goreleaser configuration and github actions integrations.

/speckit-plan setup the infrastructure following the mandatory libraries defined by the constitution.md. Use https://github.com/fogfish/iq/tree/main as an example on how to setup GitHub Action and GoReleasing for testing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-test.yml, linting (staticcheck) https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-code.yml and releasing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/build.yml https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.goreleaser.yaml

