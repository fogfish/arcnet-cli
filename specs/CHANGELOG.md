# Changelog

# 2026-07-02

/speckit-specify `arc init [<dir>]` — initialize a new knowledge graph: create the canonical folder layout (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`); write stub files `_meta/predicates.md` and `_meta/aliases.md`; create `.arc/` for arc-managed state (see Graph Root from VISION.md); run `git init` and create `.gitkeep` for empty folders; write `.gitignore` excluding `.arc/`; stage everything and produce the initial commit `graph(init): empty knowledge graph` (CORE §11 https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md)

/speckit-plan graph management (control plane) is own domain `internal/app/ctrl`. Also maintain same hierarchy in `cmd/arc/crtl`. Integrate `git` as first class concept via invocation of command-line tool, informing user of `arc` about the git tool progress. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux0design-system.md).       

# 2026-07-01

/speckit-specify setup the infrastructure for development of cli called `arc`. The infrastructure includes (1) an empty cobra application; (2) github actions to test, check and release application; (3) goreleaser configuration and github actions integrations.

/speckit-plan setup the infrastructure following the mandatory libraries defined by the constitution.md. Use https://github.com/fogfish/iq/tree/main as an example on how to setup GitHub Action and GoReleasing for testing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-test.yml, linting (staticcheck) https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-code.yml and releasing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/build.yml https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.goreleaser.yaml

