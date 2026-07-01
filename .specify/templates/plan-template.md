# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go [version, e.g., 1.22 — match `go.mod`]

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss` [plus any feature-specific external client/SDK — check for an existing adapter first, Principle VII]

**Storage**: [if applicable, e.g., local files under XDG config dir, a remote API, or N/A]

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit and colocated E2E tests (constitution Principles VI, VIII — no alternative assertion library)

**Target Platform**: [OS/arch targets declared in `.goreleaser.yaml`, e.g., linux/darwin/windows amd64+arm64]

**Project Type**: Single Cobra CLI binary (constitution Principle III — no web/mobile split)

**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]

**Constraints**: [domain-specific, e.g., <200ms p95, <100MB memory, offline-capable or NEEDS CLARIFICATION]

**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

[Gates determined based on constitution file]

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Expand the tree below with the concrete packages this
  feature touches (real command/domain names). This project is a single
  Cobra-based Go CLI (constitution Principle III) — there is no web/mobile
  layout to choose between; only the internal package boundaries vary
  per feature.
-->

```text
cmd/
└── <command>/
    ├── <command>.go        # Cobra command: flag parsing, RunE, output formatting only
    └── <command>_test.go   # E2E test(s), one per spec.md acceptance scenario, via sut() (Principle VIII)

internal/
└── <domain>/
    ├── <type>.go           # domain types, port interfaces — no cobra, no cmd/ imports (Principle III)
    ├── <type>_test.go      # unit tests, github.com/fogfish/it/v2 (Principle VI)
    └── adapter/
        └── <adapter>.go    # driven adapter implementing the port (Principle VII)

testdata/                   # fixtures colocated with the E2E test(s) above (Principle VIII)
```

**Structure Decision**: [Name the concrete command(s) and domain package(s)
this feature adds or touches, replacing the `<command>`/`<domain>` placeholders
above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
