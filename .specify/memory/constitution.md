
<!--
================================================================================
SYNC IMPACT REPORT
================================================================================
Version change: 1.1.0 → 1.2.0 (MINOR)
Reason: `github.com/charmbracelet/lipgloss` added as an explicit mandatory
dependency for all colored/styled terminal output (Principle X and
[Mandatory Libraries & Tooling](#mandatory-libraries--tooling)), replacing
the earlier implicit "color MUST be disabled automatically..." rule with a
named implementation requirement: raw ANSI escape sequences MUST NOT be
embedded directly in formatting code.

Integration (no version bump, metadata + template wiring only):
  - RATIFICATION_DATE resolved to 2026-06-30 (the date this constitution and
    its two accepted ADRs were adopted into this project), closing the
    previously deferred TODO.
  - `.specify/templates/tasks-template.md` restructured so the mandatory
    Phase 2 (Design Preconditions, 2a-2e) and Phase N (Constitution
    Compliance Verification) sections required by Governance > Task List
    Requirements are baked into the template rather than left to be
    reconstructed per feature.
  - `.specify/templates/plan-template.md` Project Structure section updated
    to present the fixed `cmd/` + `internal/<domain>/` + `adrs/` layout
    (Principle III) as the default, in place of the generic web/mobile
    project-type options.
  - `.specify/templates/spec-template.md` reviewed: no change needed: its
    user-story/acceptance-scenario shape already satisfies Principle VIII's
    1:1 spec-to-E2E-test mapping requirement.
  - Deferred: `ARCHITECTURE.md` still does not exist (Principle I). Creating
    it requires real domain/glossary content, not template propagation, so
    it is left as a follow-up rather than stubbed here.

--------------------------------------------------------------------------------
Previous: 1.0.0 → 1.1.0 (MINOR)
Reason: Amendment grounding three guidance areas in concrete, verifiable
reference patterns, plus splitting a combined principle for clarity:
  - VI    — unit/E2E assertion library is now an explicit mandate
            (`github.com/fogfish/it/v2`), not "choose one and document"
  - VIII  — E2E/acceptance test pattern is now an explicit mandate: tests
            colocated with the Cobra command package (`cmd/<cmd>_test.go`),
            exercised via a `sut()` helper that calls the command's `RunE`
            directly and captures redirected stdout, fixtures under
            `testdata/` — replacing the earlier "exec the compiled binary
            or invoke root Execute()" guidance
  - XIII  — split into XIII (Distribution & Release Engineering, explicit
            GoReleaser mandate, https://goreleaser.com) and XIV
            (Versioning, Security & Compatibility)
  - NEW   — "Mandatory Libraries & Tooling" section added: an explicit,
            non-optional inventory of CLI framework, assertion library,
            static analysis, release automation, and required CI workflows

These specific rules (the `sut()`/`RunE` test helper shape, the `fogfish/it/v2`
mandate, the GoReleaser config shape, the CI workflow split) were verified
against a real, public Cobra CLI codebase's test suite, GoReleaser config,
and GitHub Actions workflows rather than assumed generically. The source
repository is not named in this document; only the resulting, generalized
rules are retained — consistent with how the rest of this constitution
avoids naming any single adopting project.

All cross-references in the Compliance Checklist, When Constraints Cannot Be
Met, and Governance sections were updated for the XIII/XIV split.

--------------------------------------------------------------------------------
Previous: (new) → 1.0.0
Reason: Initial draft. Derived by generalizing an existing spec-kit backend
SaaS constitution into a project-agnostic constitution for Cobra-based
(github.com/spf13/cobra) command-line applications distributed as open-source
tools. All product-specific material (SaaS tenancy, AWS Bedrock/LLM
integration, serverless IaC, REST/web frontend, DynamoDB persistence) has
been removed or replaced with CLI-specific equivalents grounded in
https://clig.dev (Command Line Interface Guidelines).

This document is NOT affiliated with, and does not name, any specific
product, company, or codebase. It is intended to be copied into
`.specify/memory/constitution.md` of any Cobra-based CLI project and adopted
as-is, or lightly edited for that project's domain.

Structural conventions referenced (package layout `cmd/` + `internal/`,
persistent root flags, build-time version injection, GoReleaser-based
release pipeline) reflect widely-used, publicly documented patterns common
to mature open-source Go CLI tools — they are generic engineering practice,
not tied to any single example project.

Principles carried over from the source constitution (largely unchanged):
  I, II, IV, V, VI — architecture documentation, DDD/glossary, functional
  style, SOLID/YAGNI, TDD. These are stack-agnostic Go engineering practices.

Principles replaced (backend/SaaS-specific → CLI-specific):
  III  — Hexagonal Architecture: reframed around `cmd/` (Cobra, driving
         adapter) vs `internal/` (domain) vs adapters for external systems.
  VII  — Persistence Pattern Consistency → External Integration & Adapter
         Consistency (cloud APIs, REST clients, file systems, caches).
  VIII — E2E Acceptance Testing: reframed around exercising the compiled
         binary / root command, golden-file output, no frontend/Playwright.
  IX   — AI Agent & LLM Integration → Command & Flag Design (CLIG
         compliance): the core CLI UX grammar (subcommands, flags, help,
         confirmation, exit codes).
  X    — Serverless Infrastructure & IaC → Terminal Output, Color &
         Interactivity (TTY detection, NO_COLOR, signals, responsiveness).
  XI   — REST API Design & Documentation → Configuration, Environment
         Variables & Secrets (XDG, precedence order, secret handling).
  XII  — Frontend Architecture & UI Standards → Documentation & Help System
         (Cobra help text, man pages, README, error messages).
  XIII — SaaS Security, Tenancy & Data Isolation → Distribution, Versioning,
         Security & Release Engineering (GoReleaser, SemVer, supply chain,
         telemetry consent).

Templates:
  ⚠ This draft has not yet been wired into any `.specify/templates/*`
    files. If adopted, review `plan-template.md`, `spec-template.md`, and
    `tasks-template.md` for CLI-specific Constitution Check alignment.

Deferred TODOs:
  - TODO(PROJECT_NAME): replace the bracketed token in the title once this
    constitution is adopted into a specific CLI project's repository.
  - TODO(ARCHITECTURE_MD): ARCHITECTURE.md does not exist until the adopting
    project creates it (Principle I).
  - TODO(RATIFICATION_DATE): set to the date this draft is formally accepted
    by the adopting project's maintainers.
================================================================================
-->

# ARCNET CLI Constitution

This constitution governs the design and engineering of a command-line
application built in Go with `github.com/spf13/cobra`. It is written to be
adopted, verbatim or with light edits, by any open-source CLI project — it
contains no references to a specific product, company, or cloud account.
Wherever this document says "the tool" or "the CLI", substitute the name of
the project that adopts it.

This constitution treats the [Command Line Interface Guidelines](https://clig.dev)
(CLIG) as the authoritative source for user-facing CLI behavior. Where this
document is silent, CLIG governs; where this document is explicit, it MUST
be followed even if a contributor's prior CLI experience suggests otherwise.

Beyond user-facing behavior, the [Mandatory Libraries & Tooling](#mandatory-libraries--tooling)
section below fixes the concrete toolchain — assertion library, static
analysis, release pipeline, required CI workflows. These are explicit,
non-substitutable choices, not illustrative examples a project is free to
swap out.

## Core Principles

### I. Architecture Documentation & Decision Records

Architecture and major decisions MUST be documented, and Architecture
Decision Records are BINDING.

**Rules**:
- [ARCHITECTURE.md](ARCHITECTURE.md) is the single source of truth for system architecture
- When a feature changes or touches architecture, [ARCHITECTURE.md](ARCHITECTURE.md) MUST be updated in the same PR
- Non-functional requirements (performance, startup latency, supported platforms, compatibility guarantees) MUST be documented in [ARCHITECTURE.md](ARCHITECTURE.md)
- Major architectural decisions MUST be recorded in [adrs/](adrs/) as Architecture Decision Records
- ADR files MUST follow the format `NNN-decision-title.md` (e.g., `001-command-package-layout.md`)
- ADRs MUST include: Context, Decision, Consequences, Status (Proposed/Accepted/Deprecated/Superseded)
- **ADRs with status "Accepted" are BINDING**: all code MUST follow the patterns documented in accepted ADRs
- Deviation from an accepted ADR is NOT permitted without a new superseding ADR
- When an ADR establishes a pattern (e.g., port/adapter shape, output schema, flag-naming convention), that pattern MUST be followed consistently across every command
- **Before implementing, read every ADR referenced in the plan's Constitution Check section**: verify no plan decision contradicts an accepted ADR. When a conflict is found, the ADR takes precedence; the plan MUST be corrected before implementation begins
- **Before adding a new external system integration (cloud API, REST endpoint, package registry, queue), check whether an adapter for that capability already exists**: a second, divergent client for the same external system is an architecture violation
- **Resolving cross-cutting design tension by ignoring an ADR is FORBIDDEN**: if a plan or spec appears to contradict an ADR, the conflict MUST be raised explicitly and resolved via a new ADR, not by quietly diverging

**Rationale**: Contributors and AI agents require rapid, accurate understanding of architectural constraints to contribute effectively and maintain consistency. Making ADRs binding (not merely advisory) prevents architectural drift and keeps the command surface predictable as the project grows.

### II. Domain-Driven Design & Glossary Management

Domain concepts MUST be explicitly modeled, documented, and maintained in the glossary, independent of the CLI's command surface.

**Rules**:
- This project follows Domain-Driven Design (DDD): the CLI is a thin client over a domain model that exists independently of Cobra
- When the domain model is updated or new domain concepts are introduced, they MUST be added to [ARCHITECTURE.md](ARCHITECTURE.md)
- All domain terms MUST be added to the Glossary section in [ARCHITECTURE.md](ARCHITECTURE.md), including: term name, definition, relationships to other domain concepts
- Ubiquitous language from the domain model MUST be used consistently in code, command/flag names, help text, and documentation — a flag named `--name` and a domain field named `Identifier` for the same concept is a glossary violation
- **Before defining a new domain type inside a command package, check `internal/<domain>/` for an existing type**: introducing `type Target string` inside `cmd/` when `internal/resource.ID` already exists duplicates the domain model. Reuse the existing type or elevate a genuinely shared type into `internal/<domain>/`

**Rationale**: Consistent domain language prevents ambiguity between what a flag is named, what the help text calls it, and what the code calls it — all three MUST refer to the same domain concept under the same name.

### III. Hexagonal Architecture & Clean Boundaries

The codebase MUST separate the command-line surface from domain logic using hexagonal/onion architecture with the Cobra command tree as the sole driving adapter. [ADR 001](../../adrs/001-system-architecture.md) defines mandatory and binding principles of system architecture

**Rules**:
- `cmd/` contains ONLY Cobra command definitions: flag/argument parsing, input validation, invoking application services, and formatting output. `cmd/` MUST NOT contain business logic, retry loops, or direct external-system calls
- `internal/<domain>/` (or `pkg/<domain>/` for packages intended to be imported by other Go programs) contains domain logic and port interfaces. Domain code MUST NOT import `github.com/spf13/cobra` or any `cmd/`-package type
- Driven adapters (cloud SDK clients, HTTP/REST clients, file-system access, local databases, package-manager APIs) live in `internal/<domain>/adapter/` (or equivalent) and implement port interfaces defined by the domain
- **Port interfaces MUST be domain-level operations only**: vendor SDK types (cloud provider structs, REST client response types, ORM types) MUST NOT appear in port package signatures. A port exposes verbs like `List`, `Fetch`, `Apply` over domain types, never the underlying client's request/response structs
- This is NOT dogmatic: pragmatic deviations are allowed for small, single-command tools, but domain logic MUST remain unit-testable without spinning up Cobra or a live network connection
- The directory structure MUST be explained in [ARCHITECTURE.md](ARCHITECTURE.md); if it is unclear or misleading, it MUST be clarified with the user and the document updated

**Rationale**: Hexagonal architecture keeps the domain logic testable in isolation (no `cobra.Command`, no live cloud credentials required to run unit tests) and lets the command surface evolve — renaming a flag, adding a subcommand alias — without touching business rules.

### IV. Functional Programming Style

Functional Programming Style is NON-NEGOTIABLE. All Go code MUST follow a functional-first design philosophy emphasizing clarity through structure rather than commentary.

**Rules**:
- **No Inline Comments**: Code MUST NOT include inline or block comments. The only exception is GoDoc comments for exported (public) functions, types, and packages. Readability must come from the code itself
- **Self-Explanatory Naming**: All identifiers MUST be descriptive and unambiguous. Variable names clearly express purpose and contents; function names describe what they do, not how; avoid abbreviations unless universally understood
- **Small, Focused Functions**: Functions MUST be short (not more than 25 lines), focused on a single responsibility, and easy to understand in isolation. If a function requires explanation, refactor it into smaller composable pieces
- **Composition Over Complexity**: Favor composing small functions over large, monolithic ones. Functions MUST be easily chainable. Prefer passing data through transformations rather than mutating shared state
- **Immutability by Default**: Functions MUST avoid mutating inputs whenever possible. Prefer returning new values. Limit side effects and isolate them at the boundaries of the system (adapters), never in domain logic
- **Readability as a Requirement**: Code MUST be understandable without external explanation. If naming and structure are insufficient to convey intent, the code must be rewritten rather than annotated
- **Segregation**: Go code MUST accept interfaces, return structs

**Rationale**: Pure functions are testable without mocks, composable into pipelines, and predictable under concurrency — properties that matter even for a single-binary CLI tool, especially one that fans out concurrent requests to external systems.

### V. Code Quality & Simplicity

All contributors MUST follow SOLID principles and implement only what the current feature requires (YAGNI).

**Rules**:
- **Single Responsibility**: Functions, types, methods, and packages MUST exhibit natural cohesion. A package's name MUST describe its single purpose; catch-all packages (`utils`, `common`, `helpers`) are forbidden. Each subcommand's business logic exists in its own package, separate from its Cobra wiring
- **Open/Closed**: Compose simple types into more complex ones using embedding. Types are open for extension via embedding but closed for modification
- **Liskov Substitution**: Express dependencies between packages in terms of interfaces, not concrete types. Prefer small, single-method interfaces so implementations faithfully satisfy their contract (*require no more, promise no less*)
- **Interface Segregation**: Functions and methods MUST depend only on the behavior they actually need. Accept the narrowest interface that satisfies requirements; return concrete structs
- **Dependency Inversion**: High-level packages MUST NOT import low-level packages directly. Push concrete dependencies up to `main`/`cmd` wiring; lower-level code depends only on interfaces. The import graph MUST be acyclic and wide rather than deep

**Rationale**: SOLID principles structure functions, types, and methods into packages that exhibit natural cohesion, and define dependencies between packages in terms of interfaces rather than concrete types — essential for a CLI tool that will accumulate subcommands over time without becoming a single sprawling package.

### VI. Test-Driven Development & Automated Testing

Code quality and correctness MUST be ensured through Test-Driven Development (TDD) and automated tests, not manual validation.

**Rules**:
- **Tests MUST be written FIRST and MUST FAIL before implementation begins** (red-green-refactor cycle)
- **"Starting red" means tests MUST compile AND fail semantically (for the right reasons)**:
  - Tests MUST compile successfully with no compilation errors
  - Tests MUST fail because the functionality under test is missing or incomplete, not because the method doesn't exist
  - It is ACCEPTABLE to implement minimal structure to make tests compile (stub types, empty methods, interface definitions)
  - **Skipping or marking tests as pending is FORBIDDEN in the red phase**
  - **Tests MUST NOT contain comments marking them as "in the red phase"** (e.g., `// TODO: implement`) — tests are written once and turn green naturally
- Tests MUST drive design: implementation decisions emerge from test requirements, not vice versa
- Tests MUST change as little as possible during implementation; major test churn indicates poorly derived tests
- Automated tests MUST cover happy paths, error cases, and edge cases (including malformed flags, missing required arguments, and non-TTY/piped input)
- **`github.com/fogfish/it/v2` MUST be the sole assertion library** for unit and E2E tests, used via its fluent `it.Then(t).Should(...)` style (see [Mandatory Libraries & Tooling](#mandatory-libraries--tooling)); packages MUST NOT mix in another assertion library (e.g., `testify`, stdlib-only comparisons) alongside it
- Table-driven tests MUST be used for flag validation and other parameterized scenarios
- **Bash/shell scripts MUST NOT be used to validate unit-level code correctness**; they are reserved for optional smoke-test scripts driving a real binary against example workflows (Principle VIII) and for infrastructure tasks (CI/CD, release, system-level checks)
- Test coverage MUST be verifiable via `go test ./... -cover`
- Integration tests MUST test real behavior end-to-end where feasible (real file I/O against a temp directory, a real local process); external network/cloud calls MUST be isolated behind the ports from Principle III and exercised via fakes/mocks in unit tests

**Rationale**: TDD ensures tests are independent verification of requirements, not retrofitted validation. Failure must be semantic — tests compile and run but functionality is missing — not syntactic. Minimal scaffolding to make tests compile is acceptable; it is the implementation logic that must not precede the test.

### VII. External Integration & Adapter Consistency

Every integration with a system outside the CLI's own process MUST follow a consistent port/adapter pattern.

**Rules**:
- All external integrations (cloud provider APIs, REST/gRPC services, message queues, package registries, local databases, or the filesystem when used as a state store) MUST be accessed through a port interface defined in the domain package, with the concrete client living in an adapter package
- **Before adding a new adapter, verify whether an adapter for that capability already exists** in a shared package; duplicate clients for the same external system are forbidden
- Adapter interfaces MUST be small (Interface Segregation): one interface per capability (e.g., a `Lister` and a `Fetcher`, not one interface mirroring the entire vendor SDK)
- **Vendor SDK types MUST NOT leak through port interfaces**: a port exports domain types and standard-library types only
- Network and long-running calls MUST accept and respect `context.Context` for cancellation, and MUST apply a sensible default timeout, overridable by a flag or config value (CLIG Robustness)
- If the tool caches remote data, the caching strategy (location, TTL, invalidation) MUST be documented in [ARCHITECTURE.md](ARCHITECTURE.md) and MUST be bypassable via a flag (e.g., `--no-cache`) or environment variable for debugging
- Adapters that handle credentials or tokens MUST NOT log secret material at any verbosity level, including `--verbose`/`--debug` output

**Rationale**: A CLI tool's reliability is dominated by how it talks to the outside world. Consistent, narrow, swappable adapters keep domain logic testable without live credentials and prevent the same external dependency from being wired up three different ways across subcommands.

### VIII. End-to-End Acceptance Testing & Spec Traceability

All features MUST have comprehensive end-to-end (E2E) acceptance tests where each scenario in `spec.md` maps 1:1 to a test, exercised through the same `*cobra.Command` a user actually invokes.

**Rules**:
- **Every acceptance scenario in `specs/[NNN-feature-name]/spec.md` MUST have a corresponding E2E test**
- E2E tests MUST be written BEFORE implementation begins (acceptance-test-driven development)
- **E2E tests MUST be colocated with the Cobra command they exercise, in the same package**, named `<command>_test.go` (e.g., `cmd/list_test.go`) — not in a separate top-level test tree disconnected from the command source. This follows standard Go convention: a test lives next to the code it tests
- **E2E tests MUST invoke the command's `RunE` function directly through the `*cobra.Command` value** (`cmd.RunE(cmd, args)`), after setting any flag-bound package-level variables the way Cobra itself would after parsing `--flags`. This exercises the real production handler — the same function Cobra dispatches to in normal operation — never a duplicate test-only code path
- **A shared `sut` (system-under-test) helper MUST wrap command execution and capture its output** by redirecting `os.Stdout` through an `os.Pipe`, running the command, restoring `os.Stdout`, and returning the captured output string alongside the returned `error`:
  ```go
  func sut(cmd *cobra.Command, args []string) (string, error) {
      stdout := os.Stdout
      r, w, _ := os.Pipe()
      os.Stdout = w

      ch := make(chan string)
      go func() {
          var buf bytes.Buffer
          io.Copy(&buf, r)
          ch <- buf.String()
      }()

      err := cmd.RunE(cmd, args)

      w.Close()
      os.Stdout = stdout
      return <-ch, err
  }
  ```
- **External systems MUST be replaced with a mock/fake configuration profile or injected fake adapter** before calling `sut` (e.g., a `profile = "mock"` switch read by the adapter wiring from Principle VII); E2E tests MUST NOT require live network access or live credentials to pass in CI
- Test fixtures (input files, sample manifests, golden output) MUST live in a `testdata/` directory colocated with the test file — Go's toolchain already ignores `testdata/` by default
- **E2E tests MUST compile AND fail semantically initially (red phase)**:
  - Assertions MUST use `github.com/fogfish/it/v2` (Principle VI) and target the captured stdout content and the returned `error` with concrete expected values, not placeholder always-fail assertions
  - For commands with a `--json` output mode, assertions MUST validate against the documented JSON schema, not merely "non-empty"
  - **Skipping or marking tests as pending is FORBIDDEN**; `t.Skip()` calls MUST NOT appear in E2E test files
  - **Tests MUST NOT contain "red phase" comments** (e.g., `// TODO: implement`) — they are written once and turn green naturally
  - It is ACCEPTABLE to implement minimal structure to make E2E tests compile (a command registered with a `RunE` that returns a "not implemented" error)
- Each test function SHOULD include a comment showing the equivalent command-line invocation under test (e.g., `// tool agent -f testdata/prompt.md`), in addition to the spec scenario reference: `// Scenario X.Y from specs/[NNN-feature]/spec.md`
- E2E tests MUST change minimally during implementation; major changes indicate tests were derived from implementation, not from the spec
- E2E tests turn GREEN when implementation satisfies the acceptance criteria
- E2E tests MUST be independent and isolated: no shared mutable package state beyond the flag-bound variables explicitly set at the top of each test, no dependency on execution order
- A separate, optional shell-driven smoke-test script MAY exist to exercise full real-world workflows against example inputs as a manual or scheduled CI check; it MUST NOT replace the `go test`-based E2E suite described above, which is what gates every PR

**Rationale**: Testing through the command's own `RunE` — the exact function Cobra dispatches to in production — proves the command behaves correctly for the user invoking it, without the cost and flakiness of spawning a subprocess per test case. Colocating tests with the command package keeps the relationship between a command and its tests obvious and avoids a parallel `tests/e2e/` directory structure drifting out of sync with `cmd/`. Capturing stdout via `os.Pipe` keeps production code free of test-only seams (no writer needs to be threaded through just to make output testable).

### IX. Command & Flag Design (CLIG Compliance)

The command and flag surface MUST follow the [Command Line Interface Guidelines](https://clig.dev), implemented exclusively with `github.com/spf13/cobra`. [ADR 002](../../adrs/002-ux-design-system.md) defines binding principles of system architecture.

**Rules**:
- **`github.com/spf13/cobra` is the sole command-line parsing framework**; no hand-rolled flag parsing, and no second parsing library introduced alongside it
- Subcommand naming MUST be consistent project-wide: pick either noun-verb (e.g., `tool resource create`) or verb-noun ordering once, document the choice in [ARCHITECTURE.md](ARCHITECTURE.md), and apply it to every subcommand without exception
- **No catch-all or implicit subcommands**; arbitrary abbreviation of subcommand names is FORBIDDEN — only explicit `Aliases` declared on the `cobra.Command` are permitted
- Flags are preferred over positional arguments, except for a single primary "subject" argument per command (e.g., a resource name or file path) where a flag would be redundant
- Every flag MUST have a long form (`--flag`); single-letter shorthands are reserved for the most frequently used flags and MUST follow established conventions where one exists (`-h` help, `-v` verbose, `-q` quiet, `-o` output, `-n` name/dry-run-count, `-f` file/force, `-a` all, `-d` debug/delete, `-u` user)
- `-h`, `--help`, and a `help` subcommand MUST all work and MUST short-circuit all other flag processing
- Running a command with missing required arguments MUST print concise help (one-line description, 1-2 examples, flag summary) — never a raw panic, stack trace, or Go error value
- **Destructive or irreversible operations MUST require explicit confirmation**, or an explicit `--yes`/`--force` flag for non-interactive use; confirmation rigor MUST scale with the blast radius (a single-resource delete needs a lighter prompt than a bulk/recursive delete)
- Exit codes MUST be meaningful: `0` success, non-zero failure. Distinct non-zero codes SHOULD distinguish usage errors from runtime errors, and the meaning of each code MUST be documented
- A `--no-input`/non-interactive mode MUST exist for every command that could otherwise prompt, so the tool is safe to run unattended in scripts and CI
- **Secrets MUST NOT be accepted directly as flag values** (visible via `ps`, shell history, and CI logs); accept secrets via a file path, stdin, or a named environment variable referencing a secret store instead
- Arguments and flags SHOULD be order-independent; multiple positional arguments of the *same kind* (e.g., several file paths) are acceptable to enable shell globbing, but a command MUST NOT use two-or-more positional arguments for *different* purposes

**Rationale**: CLIG codifies what decades of UNIX and modern CLI tools have converged on as predictable, scriptable, and humane command-line behavior. Cobra is the single chosen implementation of that grammar so every subcommand inherits the same parsing, help generation, and error-handling behavior for free.

### X. Terminal Output, Color & Interactivity

Output MUST adapt to its audience — human at a terminal, or another program at the end of a pipe — and MUST never leave the user wondering whether the tool is alive.

**Rules**:
- Primary output goes to `stdout`; diagnostics, progress, and errors go to `stderr`
- The tool MUST detect whether `stdout` is a TTY and choose human-readable formatting by default; `--json` MUST emit machine-readable structured output, and `--plain` MUST emit script-friendly tabular output, regardless of TTY detection
- **`github.com/charmbracelet/lipgloss` MUST be used for all colored/styled terminal output** (see [Mandatory Libraries & Tooling](#mandatory-libraries--tooling)); raw ANSI escape sequences (e.g. `"\033[32m%s\033[0m"`) MUST NOT be embedded directly in formatting code
- **Color MUST be disabled automatically** when: output is not a TTY, the `NO_COLOR` environment variable is set (to any value), `TERM=dumb`, or `--no-color` is passed. Color MUST never be the sole carrier of information (pair it with text/symbols)
- A `--quiet`/`-q` flag MUST suppress non-essential output; a `--verbose`/`-v` flag MUST reveal additional diagnostic detail. The tool MUST function correctly with neither flag set
- Animated progress indicators (spinners, progress bars) MUST render only to a TTY and MUST be automatically suppressed when output is piped or redirected
- **Ctrl-C (SIGINT) MUST exit promptly.** If an operation is mid-flight, the tool MUST announce what it is doing (e.g., "interrupted, cleaning up...") rather than hanging silently; a second Ctrl-C MUST force-exit immediately. Prefer crash-only design — fast, safe exit — over best-effort cleanup that can itself hang
- The tool MUST produce some output within roughly 100ms of invocation for any operation that takes longer, confirming to the user that it has started, not frozen
- Successful operations that change state MUST briefly explain what changed; silent success on a state-changing command is a defect

**Rationale**: A CLI is read by both humans and scripts. Treating `--json`/`--plain` as the stable, scriptable contract and everything else (color, spinners, table formatting) as a free-to-change human convenience lets the tool be friendly in a terminal without becoming a breaking change every time the table layout improves.

### XI. Configuration, Environment Variables & Secrets

Configuration MUST be predictable, layered, and never silently destructive.

**Rules**:
- Configuration precedence, highest to lowest, MUST be: **command-line flags → environment variables → project-level config file → user-level config file → system-wide config file**
- Config file locations MUST follow the **XDG Base Directory Specification** on Linux (with documented OS-equivalents for macOS/Windows) — no ad hoc dotfiles in arbitrary locations
- The tool MUST ask for confirmation before modifying a configuration file it does not own, and MUST prefer creating a new config file over silently appending to an existing one
- Environment variables specific to this tool MUST use uppercase letters, digits, and underscores, prefixed with the project's own namespace (e.g., `TOOLNAME_*`). Recognized cross-tool variables (`NO_COLOR`, `DEBUG`, `EDITOR`, `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`/`NO_PROXY`, `TERM`, `PAGER`, `SHELL`) MUST be honored using their standard names — never reinvented under the project's own prefix
- **Secrets (API keys, tokens, passwords) MUST NOT be required to live in long-lived shell environment variables when an alternative exists**; prefer a secret file with restrictive permissions, an OS keychain, or a short-lived credential provider. Secrets MUST NEVER be logged, echoed in `--verbose`/`--debug` output, or embedded in error messages
- Reading a project-local `.env` file is acceptable for local development convenience but MUST NOT be the only supported way to configure the tool for CI/non-interactive use

**Rationale**: Predictable precedence and standard locations are what let a user reason about "why did the tool pick this value" without reading source code. Treating secrets as a distinct, higher-care category prevents the most common CLI security defect: a credential captured in shell history, CI logs, or a crash report.

### XII. Documentation & Help System

The tool MUST be self-documenting from the terminal, with web documentation as a secondary, deep-linkable reference.

**Rules**:
- Every command and subcommand MUST populate Cobra's `Short`, `Long`, and `Example` fields — none left empty for a user-facing command — plus the automatically generated flag listing
- Top-level help (`--help` on the root command) MUST include a pointer to web documentation and an issue-reporting URL
- A `README.md` at the repository root MUST document installation, a quick-start example, and a link to the full command reference
- A generated command reference (e.g., via `cobra/doc`) SHOULD be produced as part of the release process for any tool intended for general distribution; man pages SHOULD be generated where the target platforms support them
- Documentation MUST be updated in the same PR as the command/flag change it describes; help text and README MUST NOT drift from actual behavior
- **Expected/anticipated errors MUST be rewritten into human-readable guidance** before reaching the user — never a raw library error, stack trace, or Go `panic`. Where a correction is knowable (e.g., a typo'd subcommand), the tool SHOULD suggest it
- Unexpected/internal errors MUST include enough detail to file a useful bug report (and, where feasible, a ready-to-use issue URL); the user MUST NOT see a bare `panic:` trace in normal operation

**Rationale**: A CLI's primary documentation surface is the CLI itself — `--help` is read far more often than the README. Keeping `Short`/`Long`/`Example` mandatory and synchronized with behavior, and rewriting expected errors into guidance, is what makes the tool feel solid rather than merely functional.

### XIII. Distribution & Release Engineering

The tool MUST be released as a versioned, verifiable, single-binary artifact via an explicit, automated release pipeline.

**Rules**:
- **[GoReleaser](https://goreleaser.com) MUST be the release pipeline**, configured via a `.goreleaser.yaml` (or `.goreleaser.yml`) at the repository root; no manually assembled release artifacts and no alternative release tool
- The tool MUST build as a single, statically linked binary (`CGO_ENABLED=0` where feasible) for each supported OS/architecture target, declared explicitly in the `builds:` section (e.g., `linux`, `darwin`, `windows`); unsupported OS/arch combinations MUST be explicitly excluded via `ignore:`, not silently left broken
- Archives MUST use GoReleaser's `binary` format (a bare binary, not a tarball wrapping a single file) unless a project has a documented reason to ship additional files (shell completion scripts, a license file) alongside the binary
- A checksum file (e.g., `checksums.txt`) MUST be generated by the `checksum:` section and published alongside every release's binaries
- The release pipeline MUST run in CI via `goreleaser/goreleaser-action` (or equivalent), triggered after the build and test workflow passes and a version tag is created (Principle XIV) — never run ad hoc from a contributor's machine for an official release
- Where the target ecosystem has a package-manager convention, a package manifest SHOULD be published as part of the same pipeline (e.g., a Homebrew formula via GoReleaser's `brews:` section, publishing to a tap repository); the formula's smoke test MUST invoke `<binary> --version` (or equivalent) to confirm the installed binary actually runs
- The changelog generated by the release pipeline's `changelog:` section MUST exclude non-user-facing commit categories (e.g., `^docs:`, `^test:`) so release notes stay focused on user-visible change
- Uninstallation MUST be straightforward and documented (e.g., remove the single binary and the documented config directory)

**Rationale**: A CLI tool's trust is earned through verifiable, reproducible releases. GoReleaser is named explicitly — not "a release tool of your choice" — because the cross-platform build matrix, checksum generation, and package-manager publishing it automates are exactly the steps that are easy to get subtly wrong by hand; naming one tool keeps every adopting project's release process auditable in the same way.

### XIV. Versioning, Security & Compatibility

The tool's version number and scriptable contracts MUST be trustworthy, and the tool MUST respect user data by default.

**Rules**:
- Versioning MUST follow **Semantic Versioning**; the `--version` flag MUST report the exact released version, injected at build time via linker flags — never hardcoded in source
- Version tags SHOULD be created by CI from an explicit, documented increment rule (e.g., a marker in the commit message selecting major/minor/patch, defaulting to patch when absent), not pushed manually by a contributor, so the tag and the release it triggers (Principle XIII) are always in lockstep
- **Breaking changes to command names, flag names/semantics, or any output consumed by scripts (`--json`/`--plain` schemas) MUST bump the major version**, and SHOULD be preceded by a deprecation warning on `stderr` in at least one prior minor release
- Human-formatted default output (tables, colored text, column layout) is explicitly **not** a stable contract and MAY change in minor/patch releases; only `--json` and `--plain` output are stable, scriptable contracts
- **`govulncheck` (or equivalent) MUST scan third-party dependencies in CI before every release**; known-critical vulnerabilities MUST block release
- **The tool MUST NOT collect usage analytics, telemetry, or crash reports without explicit, documented opt-in consent.** If telemetry exists, what is collected, why, and how to disable it MUST be documented in the README

**Rationale**: Separating "how we ship the bits" (Principle XIII) from "what the version number and the data we collect promise the user" (this principle) keeps each concern independently reviewable — a change to the GoReleaser config and a change to the deprecation policy are different kinds of risk and should not be hidden inside one combined principle.

## Mandatory Libraries & Tooling

The following libraries and tools are not illustrative options — they are mandated by this constitution for every project that adopts it, so that test idioms, lint gates, and release mechanics are identical across every CLI project under this governance, and a contributor's tooling investment carries over directly between projects.

**CLI Framework**
- `github.com/spf13/cobra` MUST be used for all command/flag parsing (Principle IX). No alternative CLI framework (`urfave/cli`, `kingpin`, a hand-rolled wrapper around the stdlib `flag` package) is permitted for the project's own command surface.

**Terminal Output & Styling**
- `github.com/charmbracelet/lipgloss` MUST be used for all colored/styled terminal output (Principle X). Color/style definitions are `lipgloss.Style` values, selected per the TTY/`NO_COLOR`/`--color` rules in Principle X — never raw ANSI escape sequences written by hand, and never a second styling library introduced alongside it.

**Testing**
- `github.com/fogfish/it/v2` MUST be the sole assertion library for unit and E2E tests (Principles VI, VIII), used via its fluent `it.Then(t).Should(...)` style.
- E2E/acceptance tests MUST follow the colocated `cmd/<command>_test.go` plus `sut()`-helper pattern defined in Principle VIII.
- `go test -coverprofile=profile.cov $(go list ./... | grep -v /examples/)` (excluding any `examples/` directory used for runnable documentation, if present) MUST be runnable from CI; coverage MUST be published to a coverage-reporting service (e.g., Coveralls via `shogo82148/actions-goveralls` or equivalent) so the coverage trend is visible on every PR.

**Static Analysis**
- `staticcheck` (`honnef.co/go/tools/cmd/staticcheck`) MUST run in CI on every pull request via a dedicated workflow and MUST block merge on failure. Contributors SHOULD run it locally before pushing.
- `govulncheck` MUST run before every release (Principle XIV) to scan for known-vulnerable dependencies.

**Release & Distribution**
- [GoReleaser](https://goreleaser.com) MUST be the sole release pipeline (Principle XIII), driven by a `.goreleaser.yaml`/`.goreleaser.yml` at the repository root and invoked in CI via `goreleaser/goreleaser-action`.

**Continuous Integration (GitHub Actions)**

A project adopting this constitution MUST implement at minimum these workflows under `.github/workflows/`:
- **`build`** — triggered on push to the default branch: `go build ./...`, `go test` with coverage, an automatic SemVer tag increment (Principle XIV), then a GoReleaser release of that tag
- **`check-code`** (or equivalent name) — triggered on pull request open/synchronize: runs the static analysis gate (`staticcheck`); MUST be a required check before merge
- **`check-test`** (or equivalent name) — triggered on pull request open/synchronize: `go build ./...` and `go test` with coverage reporting; MUST be a required check before merge
- The Go toolchain version used in CI MUST be pinned explicitly (e.g., `actions/setup-go` with a fixed `go-version`), never left to float to "latest"

**Rationale**: Naming specific tools — rather than leaving "pick an assertion library" or "pick a release tool" open per project — eliminates a recurring source of churn when a contributor moves between CLI projects under this constitution: the test idioms, the lint gate, and the release mechanics are identical everywhere.

## Development Requirements

### Compliance Checklist

**PRECONDITIONS (must complete BEFORE implementation begins)**:

- [ ] Domain model designed: all entities, aggregates, and value objects identified and documented
- [ ] Domain concepts added to [ARCHITECTURE.md](ARCHITECTURE.md) Glossary section
- [ ] Command/flag surface designed: subcommand names, flags, output schema (`--json`) agreed before coding (Principle IX)
- [ ] External integration ports designed for any new adapter (Principle VII)
- [ ] **E2E acceptance tests written, colocated with their Cobra commands (`cmd/<command>_test.go`), for all spec scenarios (Principle VIII)**
- [ ] **E2E tests compile successfully and fail semantically before implementation (Principle VIII)**

**Implementation Phase**:

- [ ] [ARCHITECTURE.md](ARCHITECTURE.md) reflects architectural changes (if any)
- [ ] Major decisions recorded in [adrs/](adrs/) with correct numbering
- [ ] Code follows patterns established in accepted ADRs
- [ ] Commands implemented exactly as designed: flag names, help text, exit codes (Principle IX)
- [ ] Domain logic uses ports (interfaces); Cobra wiring and adapters are separated (Principle III)
- [ ] **Unit tests written FIRST, compile successfully, and fail semantically before implementation (Principle VI)**
- [ ] **Unit tests drive design: implementation emerges from test requirements (Principle VI)**
- [ ] **Unit tests change minimally during implementation (Principle VI)**
- [ ] Automated tests included (unit, integration, or both) with meaningful coverage
- [ ] No Bash scripts used for unit-level code correctness validation (only optional smoke-test scripts and infrastructure tasks)
- [ ] Unit and E2E tests use `github.com/fogfish/it/v2` exclusively (Principle VI, [Mandatory Libraries & Tooling](#mandatory-libraries--tooling))
- [ ] New external integrations follow the port/adapter pattern (Principle VII)
- [ ] Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose` (Principle X)
- [ ] Configuration precedence and XDG locations respected; no secrets logged (Principle XI)
- [ ] Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [ ] Release artifacts (if applicable) build via the GoReleaser pipeline with version injected at build time (Principle XIII)
- [ ] Versioning, deprecation, and telemetry-consent rules followed for any user-facing or scriptable-output change (Principle XIV)
- [ ] **E2E tests turn GREEN as implementation satisfies acceptance criteria (Principle VIII)**
- [ ] **E2E tests changed minimally during implementation (Principle VIII)**
- [ ] **All spec scenarios have passing E2E tests colocated with their Cobra commands (Principle VIII)**

### When Constraints Cannot Be Met

If **Principle I (Architecture Documentation & ADRs)** requires deviation:

1. STOP implementation immediately
2. Document why the accepted ADR pattern cannot be followed
3. Draft a new ADR proposing an alternative approach with rationale
4. Mark the new ADR as superseding the previous ADR
5. Do NOT implement the deviation without an accepted superseding ADR

If **Principle VI (TDD & Automated Testing)** cannot be satisfied:

1. Document the specific reason why TDD or automated testing is not feasible
2. Propose an alternative validation approach (e.g., manual verification checklist for a one-off migration script)
3. Escalate to project maintainers for exception approval
4. Do NOT skip test-first development without explicit justification

If **Principle VII (External Integration & Adapter Consistency)** cannot be satisfied:

1. STOP implementation immediately
2. Document why a port/adapter cannot be introduced for the new external system
3. Propose an alternative approach with technical justification
4. Create an ADR documenting the exception and rationale
5. Do NOT call vendor SDKs directly from `cmd/` or domain packages without an accepted ADR

If **Principle VIII (E2E Acceptance Testing & Spec Traceability)** cannot be satisfied:

1. STOP implementation immediately
2. Document why E2E tests cannot be written for spec scenarios or why 1:1 mapping is not feasible
3. Propose an alternative acceptance-testing approach with technical justification
4. Escalate to project maintainers for exception approval
5. Do NOT ship commands without E2E acceptance tests mapped to spec scenarios

If **Principle IX (Command & Flag Design / CLIG Compliance)** cannot be satisfied:

1. STOP implementation immediately
2. Document precisely which CLIG rule cannot be followed and why (e.g., a third-party constraint forces a non-standard flag name)
3. Escalate to project maintainers for sign-off before merging
4. Do NOT introduce a second flag-parsing mechanism alongside Cobra to work around the constraint

If **Principle XI (Configuration, Environment Variables & Secrets)** cannot be satisfied:

1. STOP implementation immediately
2. Document the specific constraint preventing standard precedence, XDG layout, or secret handling
3. Escalate to project maintainers — secret-handling exceptions require explicit written approval
4. Do NOT ship code that logs secrets or accepts them only via plaintext flags without an accepted ADR

If **Principle XIII (Distribution & Release Engineering)** cannot be satisfied:

1. STOP implementation immediately
2. Document why GoReleaser, the single-binary build, or the checksum/changelog requirements cannot be applied
3. Escalate to project maintainers for exception approval
4. Do NOT hand-assemble or manually publish release artifacts outside the mandated pipeline without an accepted ADR

If **Principle XIV (Versioning, Security & Compatibility)** cannot be satisfied:

1. STOP implementation immediately
2. Document why the SemVer contract, the automated tagging rule, the `govulncheck` gate, or the telemetry-consent rule cannot be applied
3. Escalate to project maintainers for exception approval
4. Do NOT release a binary that collects telemetry without consent or breaks a documented `--json`/`--plain` contract without a major version bump

## Governance

### Amendment Procedure

1. Propose amendment via PR to [.specify/memory/constitution.md](.specify/memory/constitution.md)
2. Include rationale, impacted templates, and migration plan
3. Increment CONSTITUTION_VERSION per semantic versioning rules
4. Update the Sync Impact Report with changes
5. Propagate changes to dependent templates and documentation

### Versioning Policy

- **MAJOR**: Backward-incompatible governance/principle removals or redefinitions
- **MINOR**: New principle/section added or materially expanded guidance
- **PATCH**: Clarifications, wording, typo fixes, non-semantic refinements

### Compliance Review

- All PRs MUST verify compliance with this constitution
- Reviewers MUST challenge complexity and request justification when principles are violated
- Reviewers MUST verify adherence to accepted ADRs (Principle I)
- Reviewers MUST verify new external integrations follow the port/adapter pattern (Principle VII)
- Reviewers MUST verify command/flag changes follow CLIG conventions (Principle IX)
- Reviewers MUST verify terminal output respects TTY detection and `NO_COLOR` (Principle X)
- Reviewers MUST verify colored/styled output uses `github.com/charmbracelet/lipgloss`, with no raw ANSI escape sequences in formatting code (Principle X)
- Reviewers MUST verify no secrets are logged or required only via plaintext flags (Principle XI)
- Reviewers MUST verify help text is populated and accurate for changed commands (Principle XII)
- Reviewers MUST verify TDD was followed: tests written first, compiled, and failed semantically (Principle VI)
- Reviewers MUST verify unit and E2E tests use `github.com/fogfish/it/v2` exclusively (Principle VI)
- Reviewers MUST verify E2E tests exist for all spec scenarios and changed minimally during implementation (Principle VIII)
- Reviewers MUST verify E2E tests are colocated with their command and exercise the command's `RunE` via the `sut()` pattern, not an internal function bypassing it (Principle VIII)
- Reviewers MUST verify release artifacts are produced via the mandated GoReleaser pipeline (Principle XIII)
- Reviewers MUST verify release/versioning changes preserve the `--json`/`--plain` stability contract (Principle XIV)
- Reviewers MUST verify mandated libraries and tooling (Cobra, `fogfish/it/v2`, `staticcheck`, GoReleaser, required CI workflows) are used unchanged, per [Mandatory Libraries & Tooling](#mandatory-libraries--tooling)
- Template files in [.specify/templates/](.specify/templates/) provide execution workflows that enforce these principles

### Task List Requirements

Every feature's `tasks.md` file MUST include these mandatory sections from [tasks-template.md](.specify/templates/tasks-template.md):

**MANDATORY SECTIONS** (cannot be omitted):
1. **Phase 2: Design Preconditions** - Constitution PRECONDITIONS implementation
   - Phase 2a: Domain Model & Glossary (Principles II, V)
   - Phase 2b: Command & Flag Contract Design (Principle IX)
   - Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)
   - Phase 2d: E2E Acceptance Test Design (Principle VIII)
   - Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

2. **Phase N: Constitution Compliance Verification** - Constitution Implementation Phase checklist
   - Design Phase Verification tasks
   - Implementation Phase Verification tasks grouped by principle
   - All verification tasks with explicit principle references

**OPTIONAL PHASES** (include when applicable — omit when not needed):
- **Phase 0: Pre-implementation Refactoring** — include when the feature requires significant changes
  to existing code (rename, restructure, extract interfaces, split files). MUST be submitted as a
  separate PR from feature work. All existing tests MUST pass after refactoring.
- **Phase 2.5: Command Boilerplate** — include when the feature introduces new subcommands requiring
  new ports, adapters, and Cobra wiring. Creates empty-but-compiling scaffolding (commands returning
  a "not implemented" error, empty adapter methods) as a separate PR before business logic. Enables
  focused, incremental code review: structural scaffold PR → business logic PRs.

**CUSTOMIZABLE SECTIONS** (adapt to feature):
- Phase 1: Setup (project-specific initialization)
- Phase 2.5: Foundational Infrastructure (feature-specific foundation)
- Phase 3+: User Stories (based on spec.md user stories)
- Additional Polish (optional improvements)

**Enforcement**:
- AI agents generating tasks.md MUST retain Phase 2 and Phase N sections verbatim (adapting only task
  descriptions to the specific feature)
- Omitting mandatory sections violates this constitution and blocks feature completion

**Version**: 1.2.0 | **Ratified**: 2026-06-30 | **Last Amended**: 2026-06-30
