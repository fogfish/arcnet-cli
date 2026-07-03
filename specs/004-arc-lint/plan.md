# Implementation Plan: Validate Graph Conformance (`arc lint`)

**Branch**: `004-arc-lint` | **Date**: 2026-07-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-arc-lint/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Implement `arc lint`: walk every node file in an initialized graph and check it against the full CORE §14 conformance checklist — valid front-matter/`kind`, unique basenames, resolvable `[[link]]`s, `source` citekey identity (§6.2), `entity` four-word Sowa `category` (§9.2.1), derived-node provenance (§3.4), camelCase/registered predicates (§7.3), registered `cito:`-aligned citation predicates (§8), one `graph(ingest):` commit per document (§11.1), extension-kind recognition, and absence of unresolved git merge-conflict markers — reporting every violation with file path and line number, never stopping at the first one found (spec FR-013). Per the user's explicit instruction, lint is its own domain: **`internal/app/lint`** (mirroring `internal/app/ctrl`'s `kernel/port/adapter/service/component.go` layout exactly) hosted by a matching **`cmd/arc/lint`** command package. Lint is strictly read-only (spec FR-014) — it never writes to `fsys.Store` and never commits; its one external dependency beyond the filesystem is a new, narrow, lint-private `port.VCS` (`CommitsMatching`, wrapping `git log --all --grep=<needle> --fixed-strings --format=%H`) satisfied structurally by the already-shared `internal/adapter/git.Git` (a second method added to that promoted adapter, following `specs/003-apply-patch/research.md` D4's precedent exactly — no second git client). Reusing `internal/core.ParseNode` for structural fields is not enough on its own: that parser deliberately discards Markdown source positions once it produces a `core.Node` (`specs/003-apply-patch`'s AST never needed them), so `internal/app/lint/service` pairs `core.ParseNode`'s structural result with a lint-private, line-oriented raw-text scan of the same file's bytes (regexp-based, not a second Markdown parser) to attribute every violation to a concrete line number — a new capability, not a change to `internal/core`'s existing, already-shipped parsing contract. UX follows ADR 002 exactly as `arc init`/`arc apply` already established it: DS-04's `Registry[T]{Human, Verbose}` split maps directly onto the user's requested behavior — the `Human` renderer lists only nodes with issues, the existing `--verbose`/`-v` global flag resolves to the `Verbose` renderer that additionally lists every node's individual pass/fail status — both ending with one overall graph-status summary line; no new flags are introduced.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`, `github.com/fogfish/it/v2`, `github.com/yuin/goldmark`/`goldmark-meta` (via existing `internal/core.ParseNode`, not a new direct dependency), `gopkg.in/yaml.v3` (parsing `_meta/predicates.md`'s registry format and `.arc/config.yml` via existing `internal/app/config.Resolve`) — all existing, no new third-party dependency added by this feature; the system `git` binary via the existing shared `internal/adapter/git`, gaining one new method

**Storage**: The mounted graph root, read exclusively through the existing `internal/adapter/fsys` `Store`/`Mounter` (no changes to that package) — lint performs **zero writes**, the first graph-reading command in the codebase with that property (spec FR-014)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated at `cmd/arc/lint/lint_test.go` via the existing `sut()` helper, one per spec.md acceptance scenario (constitution Principles VI, VIII); unit tests for `internal/app/lint/service` covering each CORE §14 rule individually (one deliberately-broken fixture graph per rule under `testdata/`, per spec.md SC-003) against fakes of `fsys.Mounter`/`fsys.Store` and a mock `lint.port.VCS`; unit tests for the new raw-text line-locator against known byte offsets; an integration test for the promoted `internal/adapter/git.Git.CommitsMatching` against a real local `git` repository and `t.TempDir()`, mirroring `internal/adapter/git/git_test.go`'s existing precedent for `IsTracked`

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — fourth `internal/app/<domain>` use-case (`lint`, after `ctrl`, `graph`, `config`)

**Performance Goals**: Spec SC-004 — a graph of several thousand nodes completes a full lint run in under 30 seconds; achievable with a single sequential filesystem walk plus one `git log` invocation per distinct `source` node, no per-node subprocess spawn beyond that

**Constraints**: Target directory MUST already be an initialized graph (spec FR-017, same guard `arc apply` already uses — `guardIsGraph`, checking `.arc`); lint MUST NOT stop at the first violation (spec FR-013); lint MUST make no filesystem or git-history changes under any circumstance (spec FR-014); lint operates fully local/offline (spec Assumptions)

**Scale/Scope**: One new bare-verb command (`arc lint`), one new use-case package (`internal/app/lint` with `Lint`), one new method on the existing shared `internal/adapter/git.Git` (`CommitsMatching`) behind a new, lint-private `internal/app/lint/port.VCS`, no changes to `internal/core`'s public `Node`/`ParseNode` contract (the line-locator is new, additive code in `internal/app/lint`, not a modification of `core.ParseNode`'s existing return shape), no new external dependencies

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Directory Structure and Glossary sections MUST be updated in the same PR to add `internal/app/lint`, `cmd/arc/lint`, and new glossary terms (Violation, Lint Run, Checklist Rule, Predicate Registry, Extension Profile Checklist — from spec.md Key Entities). `tasks.md` MUST include this task. |
| II — DDD & Glossary | Yes | PASS — new glossary terms defined in data-model.md/spec.md Key Entities, copied into ARCHITECTURE.md per the Principle I obligation above |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/lint` is Cobra wiring only; `internal/app/lint/{kernel,port,service}` holds domain logic and the one narrow port per ADR 001's `componentX` layout, identical in shape to `internal/app/ctrl`/`internal/app/graph`; the raw-text line-locator is pure, I/O-free logic and lives in `internal/app/lint/service`, not `internal/core` (research.md D3 — it is a lint-specific concern, not a graph-format-wide one, so it does not belong in the shared AST package per Principle V's YAGNI guidance) |
| IV — Functional Programming Style | Yes | PASS — the line-locator and every per-rule checker are pure functions (`[]byte`/`core.Node` in, `[]kernel.Violation` out); no inline comments; enforced during implementation |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — `lint.port.VCS` declares exactly one method (`CommitsMatching`), narrower than both `ctrl.port.VCS` and `graph.port.VCS`, since lint never initializes, stages, or commits; FR-011's extension-profile checklist is deliberately scoped to what the existing config/registration mechanism can actually check today (kind recognized vs. not) rather than speculatively designing a per-kind field-schema mechanism that does not exist yet anywhere in this codebase (research.md D11, flagged explicitly, not silently narrowed) |
| VI — TDD | Yes | PASS — E2E and service unit tests written first, one fixture graph per CORE §14 rule (spec SC-003); `internal/adapter/git.Git.CommitsMatching`'s integration test uses the real local `git` against `t.TempDir()`, matching existing `IsTracked` precedent — no real git subprocess needed in the service-level unit tests, which use a mock `lint.port.VCS` |
| VII — External Integration & Adapter Consistency | Yes | PASS — the one new external touchpoint (`git log`) goes through the *same* promoted `internal/adapter/git` adapter every other use-case shares (no duplicate git client, mirroring `specs/003-apply-patch/research.md` D4); all filesystem I/O goes through the existing, unmodified `internal/adapter/fsys`; `CommitsMatching` is a fast, local subprocess call with no meaningful timeout/cancellation surface beyond the existing `context.Context` plumbing already used by every other `port.VCS` method |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's 3 user stories / 8 acceptance scenarios map 1:1 to E2E tests in `cmd/arc/lint/lint_test.go` |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (continuing `arc init`/`arc apply`'s precedent), zero new flags (the existing global `--verbose`/`-v` is reused exactly as DS-03 intends — no command-local option struct needed since lint has no arguments and no bespoke flags), DS-04's `Registry[T]{Human, Verbose}` split implements the user's requested normal-vs-verbose behavior by construction rather than a bespoke branch |
| X — Terminal Output, Color & Interactivity | Yes | PASS — reuses the existing `internal/bios` DS-04/05/06 kernel unchanged; the overall graph-status summary line is lint's "successful operation states what changed" equivalent (a read-only command has no state change to report, so it reports the *finding* instead — the same spirit: never a silent, unexplained exit) |
| XI — Configuration, Env & Secrets | Yes | PASS — reads the existing `.arc/config.yml` via `internal/app/config.Resolve` unchanged; no new configuration surface; no secrets involved |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected failure (not-a-graph, unreadable file, malformed predicates registry) declared as a `faults.Type`/`faults.SafeN` constant in `internal/app/lint/service/errors.go`, wrapped via `.With()`, matching `specs/002-arc-init`/`specs/003-apply-patch`'s established convention exactly |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — adds a new, additive `--json` schema (`kernel.LintResult`); no breaking change to any existing `--json` contract; no telemetry introduced |

**ADR 001 port isolation rule 1** (explicit check, since this plan again extends the shared git adapter mid-project, as `specs/003-apply-patch` did): satisfied — `lint.port.VCS` is a third, separate, narrow, use-case-private interface (`ctrl.port.VCS`, `graph.port.VCS`, now `lint.port.VCS`), each satisfied structurally by the one shared `internal/adapter/git.Git` concrete type; no interface is widened to serve a use-case that does not need its full surface.

One entry in Complexity Tracking below (FR-011's deliberately narrowed extension-profile-checklist scope) — a documented, non-speculative scope decision, not a structural violation. No other unresolved violations.

## Project Structure

### Documentation (this feature)

```text
specs/004-arc-lint/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── cli-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── root.go               # + registers lint.NewLintCmd(); no new persistent flags
    └── lint/                  # NEW — Cobra wiring for the lint domain
        ├── lint.go             # package lint: NewLintCmd() *cobra.Command — no args, no local flags;
        │                       #   mounts the graph, calls internal/app/config.Resolve then
        │                       #   internal/app/lint.Lint, renders via bios.Registry (Human/Verbose split
        │                       #   implements normal-vs--verbose per the user's instruction), sets a
        │                       #   distinct non-zero exit code when violations are found (DS-07)
        └── lint_test.go        # E2E tests, one per spec.md acceptance scenario, via sut()

internal/
├── adapter/
│   └── git/                  # existing, gains one method (no new package)
│       ├── git.go              # + CommitsMatching(ctx, dir, needle string) ([]string, error) —
│       │                       #   `git log --all --fixed-strings --grep=<needle> --format=%H`
│       └── git_test.go         # + integration test for CommitsMatching, real git, t.TempDir()
│
└── app/
    └── lint/                  # NEW — graph conformance validation (research.md D1)
        ├── kernel/
        │   ├── lint.go          # Violation, NodeStatus, LintResult, Rule constants (CORE §14 items)
        │   └── lint_test.go
        ├── port/
        │   └── vcs.go           # VCS: CommitsMatching (lint-private, narrowest of the three port.VCS)
        ├── adapter/
        │   └── mock/
        │       └── mock.go      # fake VCS for service unit tests
        ├── service/
        │   ├── lint.go          # Lint use-case: walk graph, run each CORE §14 checker, aggregate
        │   ├── locate.go        # raw-text line locator (regexp over file bytes, no goldmark) —
        │   │                     #   front-matter/kind line, [[Target]] occurrences, predicate tokens,
        │   │                     #   conflict markers (research.md D3)
        │   ├── rules_frontmatter.go   # front-matter/kind validity, unique basenames, conflict markers
        │   ├── rules_links.go        # [[link]] resolution, derived-node provenance (§3.4)
        │   ├── rules_identity.go     # source citekey==basename (§6.2), entity Sowa category (§9.2.1)
        │   ├── rules_predicates.go   # camelCase + _meta/predicates.md registration (§7.3), citations (§8)
        │   ├── rules_history.go      # one graph(ingest): commit per document (§11.1), via port.VCS
        │   ├── errors.go             # ErrNotAGraph, ErrPredicatesUnreadable
        │   └── lint_test.go          # unit tests, one fixture per CORE §14 rule, against fakes/mock
        ├── README.md
        └── component.go          # primary port: Lint(ctx, mounter, vcs, rules, dir) (kernel.LintResult, error)

testdata/
└── lint/                     # fixture graphs, one deliberately-broken-per-rule (spec SC-003) +
                               #   one fully-conformant graph (spec SC-002), shared by service and E2E tests

ARCHITECTURE.md               # + Directory Structure/Glossary updated (Principle I obligation above)
```

**Structure Decision**: This feature adds the project's fourth `internal/app/<domain>` use-case (`lint`), following `internal/app/ctrl`/`internal/app/graph`/`internal/app/config`'s already-established `kernel/port/adapter/service/component.go` layout exactly, per the user's explicit instruction that "linter is own domain `internal/app/lint`" with "the same hierarchy in `cmd/arc/lint`". It extends the existing shared `internal/adapter/git` with one new method rather than introducing a fourth git client, mirroring `specs/003-apply-patch/research.md` D4's promotion precedent. No existing package's public contract changes: `internal/core.ParseNode`/`RenderNode`, `internal/app/config.Resolve`, and `internal/adapter/fsys` are all consumed unchanged. The command surface adds `cmd/arc/lint/lint.go`, a sibling package to `cmd/arc/ctrl` and `cmd/arc/graph`, registered into the existing root command with no new persistent or command-local flags.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| FR-011 (extension-kind profile checklist) is implemented only as "kind recognized (core or config-registered) vs. not," not full per-kind field-schema validation | CORE §10/§14 describes a domain profile as declaring its own front-matter/body schema, but no mechanism for a profile to *declare* that schema exists anywhere in this codebase yet (`.arc/config.yml` only carries a kind→`MergeOp` map, per `specs/003-apply-patch`) — there is nothing concrete to validate against beyond recognition | Designing a new profile-schema declaration format speculatively, with no consumer requesting one yet and no real profile-schema example to validate the design against, was rejected as scope creep beyond what this feature's own spec (§14's checklist, not a new schema DSL) asked for. A follow-up feature can add profile-schema declarations once a real domain profile needs field-level validation; `arc lint` will pick it up the same way it already picks up kind registration, by reading `.arc/config.yml` |

## Bugfix Log

_(none yet)_
