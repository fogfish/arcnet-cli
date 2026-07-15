# Implementation Plan: Import Schema Definitions via `arc apply schema`

**Branch**: `018-apply-schema-patch` | **Date**: 2026-07-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/018-apply-schema-patch/spec.md`

## Summary

Add `arc apply schema <patch.md> | <url> | arcnet:<name>`: a command that
parses a patch document (the same manifest/node-section format `arc apply`
already reads) restricted to `Property`/`Class` node sections, and
creates/merges each one into the graph's `_schema/predicates/`/
`_schema/types/` documents. An `arcnet:<name>` input resolves to a fixed
location in the official arcnet extensions catalog
(`https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/<name>`)
and is fetched exactly like a directly supplied URL. Any non-`Property`/
`Class` node anywhere in the patch fails the whole operation before any
write happens. Business logic lives in `internal/app/schema`
(the domain package that already owns `_schema/` reads via `Resolve` and
writes via `RegisterType`/`RegisterPredicate`); the Cobra wiring lives in
`cmd/arc/ctrl` (per user direction — this is a controller/schema-management
operation that borrows the `apply` verb for naming consistency with
`arc apply`, not a graph-content operation) and is attached as a child of
the existing `graph.NewApplyCmd()` command in `cmd/arc/root.go`.

## Technical Context

**Language/Version**: Go 1.26.5 (go.mod)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`, `github.com/fogfish/it/v2`; stdlib `net/http` and `net/url` newly introduced for URL-input support (no existing HTTP adapter in the codebase — verified via repo-wide search, research.md D2)

**Storage**: Local filesystem via `internal/adapter/fsys` (`_schema/predicates/*.md`, `_schema/types/*.md`); git working tree via `internal/adapter/git` for the resulting commit

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated with `cmd/arc/ctrl/apply_schema_test.go` via the `sut()` helper (Principle VIII), one per spec.md acceptance scenario

**Target Platform**: linux/darwin/windows amd64+arm64 (`.goreleaser.yaml`, unchanged by this feature)

**Project Type**: Single Cobra CLI binary (Principle III)

**Performance Goals**: N/A — single invocation, patch documents are small (tens to low hundreds of node sections); no throughput target

**Constraints**: URL fetch MUST apply a sensible default timeout, overridable, and MUST fail closed (no partial schema writes) per Principle VII

**Scale/Scope**: One new CLI command, one new domain operation in an existing package, one new shared adapter (`internal/adapter/http`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle III (Hexagonal Architecture)**: New business logic goes in `internal/app/schema/service`, exposed through `internal/app/schema/component.go`; `cmd/arc/ctrl/apply_schema.go` contains only flag parsing, source-kind detection delegation, and output formatting. PASS.
- **Principle VII (External Integration & Adapter Consistency)**: URL fetch is a new external integration; verified no existing adapter covers it (research.md D2). A new narrow port (`internal/app/schema/port.Fetcher`) plus a new shared adapter (`internal/adapter/http`) are added, context-respecting with a default, overridable timeout. Git commit reuses the existing shared `internal/adapter/git` package via a new narrow `internal/app/schema/port.VCS`, satisfied structurally without new adapter code (ADR 001 port isolation rule 1, the same technique `ctrl`/`graph`'s own narrower `VCS` ports already use). PASS.
- **Principle VIII (E2E Testing & Spec Traceability)**: `cmd/arc/ctrl/apply_schema_test.go` gets one test per spec.md acceptance scenario, invoking `RunE` directly via `sut()`, with `internal/app/schema/adapter/mock` fakes for `VCS`/`Fetcher` swapped in before each call — no live network or git process in the E2E suite. PASS.
- **Principle IX (Command & Flag Design)**: `apply schema` keeps the existing verb-first ordering (`arc apply <thing>`) already established by `arc apply`/`arc apply <patch.md>`; a single positional "subject" argument (patch path or URL) with no ambiguity about which positional it is. PASS.
- **Principle I (ADRs are binding)**: No conflict identified with ADR 001 (system architecture: cmd/domain/adapter split, port isolation) or ADR 002 (UX design system: CLIG-compliant flags/output). PASS.

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/018-apply-schema-patch/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/arc/ctrl/
├── apply_schema.go        # Cobra command: `arc apply schema <patch.md>|<url>`, attached as
│                           # a child of graph.NewApplyCmd() in cmd/arc/root.go
└── apply_schema_test.go   # E2E tests, one per spec.md acceptance scenario, via sut()

internal/app/schema/
├── component.go            # gains ApplyPatch(...) delegator into service.ApplyPatch
├── port/
│   ├── vcs.go               # new: narrow VCS{StageAll, Commit} port, private to schema
│   └── fetcher.go            # new: narrow Fetcher{Fetch(ctx, url) (io.ReadCloser, error)} port
├── kernel/
│   └── apply.go             # new: ApplySchemaResult domain value type
├── service/
│   ├── apply.go             # new: ApplyPatch — validate-all-then-write, reuse decodePredicateDef/decodeTypeDef
│   └── errors.go             # gains ErrDisallowedNodeType, ErrPatchRead
└── adapter/mock/
    └── mock.go               # new: in-memory VCS + Fetcher fakes for unit/E2E tests

internal/adapter/http/
├── http.go                 # new shared adapter: context-respecting GET with default timeout
└── http_test.go
```

**Structure Decision**: Business logic (validation, create/merge, commit) lives
in `internal/app/schema` alongside the domain's existing `Resolve`/
`RegisterType`/`RegisterPredicate` operations, since this feature's output is
exactly the `_schema/` documents that package already owns end-to-end. The
Cobra command lives in `cmd/arc/ctrl` per explicit user direction (a
schema/config-management concern, grouped with `arc init`, even though it is
attached under the `apply` verb for naming consistency with the existing
graph-content `arc apply` command). A new shared `internal/adapter/http`
package is added at the same level as `internal/adapter/fsys`/`git`, since
URL fetching is a capability any future command could also need, not
something private to this one use-case.

## Complexity Tracking

*No violations — table omitted.*
