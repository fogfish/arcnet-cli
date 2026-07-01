# Feature Specification: CLI Development Infrastructure Bootstrap

**Feature Branch**: `001-cli-infrastructure`

**Created**: 2026-07-01

**Status**: Draft

**Input**: User description: "setup the infrastructure for development of cli called `arc`. The infrastructure includes (1) an empty cobra application; (2) github actions to test, check and release application; (3) goreleaser configuration and github actions integrations."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bootstrap a runnable CLI skeleton (Priority: P1)

A contributor clones the repository for the first time and needs a working, buildable `arc` command-line application skeleton — with no business commands yet — so that all future feature work has a real binary, root command, `--help`, and `--version` to build on top of, following the project's hexagonal `cmd/`/`internal/` layout (ADR 001).

**Why this priority**: Nothing else in this feature can be verified or built upon without a compiling, runnable binary. It is the foundation every other story depends on.

**Independent Test**: Can be fully tested by running `go build ./...` followed by executing the produced binary with `--help` and `--version`, and delivers a functioning, distributable command-line entry point even before any real subcommand exists.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** a contributor runs the Go build for the module, **Then** the build succeeds and produces an `arc` executable.
2. **Given** the built `arc` executable, **When** it is invoked with no arguments or with `--help`, **Then** it prints usage information and exits successfully (matching CLIG expectations).
3. **Given** the built `arc` executable, **When** it is invoked with `--version`, **Then** it prints a version identifier and exits successfully.
4. **Given** the project source tree, **When** a contributor inspects the layout, **Then** command wiring lives under `cmd/` and no business/domain logic exists outside of it (there is none yet), per ADR 001.

---

### User Story 2 - Automated verification on every pull request (Priority: P2)

A contributor opens a pull request and needs automated feedback — build, tests with coverage, and static analysis — without waiting on a human reviewer to run checks locally, so that problems are caught before merge.

**Why this priority**: Continuous verification is what makes the skeleton from Story 1 safe to build on; without it, every subsequent feature risks silently breaking the build or introducing regressions.

**Independent Test**: Can be fully tested by opening a pull request against the default branch and observing that check runs are automatically triggered and reported on the PR, and delivers merge-blocking quality gates independent of any release capability.

**Acceptance Scenarios**:

1. **Given** an open pull request, **When** it is opened or updated with new commits, **Then** a build-and-test check automatically runs `go build` and `go test` with coverage and reports pass/fail status on the PR.
2. **Given** an open pull request, **When** it is opened or updated with new commits, **Then** a static-analysis check automatically runs and reports pass/fail status on the PR.
3. **Given** a pull request where the build-and-test check or the static-analysis check has failed, **When** a maintainer attempts to merge, **Then** the merge is blocked until the check passes.
4. **Given** a pull request where both checks pass, **When** a maintainer views the PR, **Then** the reported test coverage trend is visible.

---

### User Story 3 - Automated versioned release on merge (Priority: P3)

A maintainer merges an accepted change into the default branch and needs a versioned, downloadable release artifact to be produced automatically, so that users can install the latest `arc` build without any manual packaging step.

**Why this priority**: Release automation delivers the end-user-visible value of the whole feature (an installable binary) but depends on Stories 1 and 2 already being in place (something to build, and confidence that it works).

**Independent Test**: Can be fully tested by merging a change into the default branch and observing that a new version tag and a corresponding set of release artifacts appear automatically, and delivers a fully hands-off release pipeline independent of any specific application feature.

**Acceptance Scenarios**:

1. **Given** a change merged into the default branch, **When** the merge completes, **Then** the build pipeline automatically determines and creates the next semantic-version tag with no manual version editing.
2. **Given** a newly created version tag, **When** the release pipeline runs, **Then** it produces cross-platform release artifacts (binaries and checksums) and publishes them as a release, without any manually assembled artifacts.
3. **Given** a completed release, **When** a user views the release notes, **Then** the changelog reflects user-facing changes only (non-user-facing commit categories are excluded).
4. **Given** the release pipeline, **When** it runs, **Then** it uses a pinned Go toolchain version rather than an unpinned "latest" version.

### Edge Cases

- What happens when a pull request's static-analysis check and build-and-test check both fail? Both statuses MUST be individually visible and merging MUST remain blocked until both pass.
- What happens when a commit is pushed directly to the default branch without a pull request? The build pipeline still runs, still gates the release on tests passing, and still triggers a new version tag and release only after tests pass.
- What happens when there are no user-facing commits since the last release (only excluded categories, e.g., docs/test-only changes)? A release MAY still be produced per the versioning rules already in place, but the changelog for it MUST omit those excluded entries and MAY be empty of user-facing notes.
- How does the system handle a release pipeline run where dependency vulnerability scanning finds a known-critical vulnerability? The release MUST be blocked until resolved.
- What happens when a contributor runs the empty CLI skeleton with an unrecognized flag or argument? It MUST fail with a clear, non-zero-exit error rather than silently succeeding.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The repository MUST contain a buildable Go module exposing a single `arc` command-line application built on top of the mandated CLI framework, with a root command only (no business subcommands yet).
- **FR-002**: The `arc` root command MUST support `--help` and `--version`, and MUST print usage guidance when invoked without recognized arguments, consistent with standard CLI guidelines.
- **FR-003**: The command-line entry points MUST be organized under a dedicated command-wiring location, separated from any (currently absent) domain logic location, so future business logic has nowhere else to go but the correct layer.
- **FR-004**: The repository MUST define an automated check that runs on every pull request to build the module and run its test suite with coverage measurement, and MUST report a pass/fail status usable as a required merge gate.
- **FR-005**: The repository MUST define an automated check that runs on every pull request to perform static analysis of the codebase, and MUST report a pass/fail status usable as a required merge gate, independent of the build-and-test check.
- **FR-006**: Test coverage results from pull-request checks MUST be published to a coverage-trend-visible location accessible from the pull request.
- **FR-007**: The repository MUST define an automated pipeline triggered on changes to the default branch that builds the application, runs its tests, and — upon success — automatically determines and creates the next semantic-version tag without manual editing of a version number.
- **FR-008**: The repository MUST define a release configuration, checked into the repository root, that governs how release artifacts (cross-platform binaries and checksums, at minimum) are assembled — with no manually assembled release artifacts permitted.
- **FR-009**: The default-branch pipeline MUST invoke the release configuration automatically after a new version tag is created, publishing the resulting artifacts, and MUST NOT require a maintainer to run the release process from a local machine for an official release.
- **FR-010**: The release configuration's changelog generation MUST exclude non-user-facing commit categories (for example, documentation-only and test-only changes) from published release notes.
- **FR-011**: All automated pipelines MUST pin the language toolchain version explicitly rather than resolving to an unpinned "latest" version at run time.
- **FR-012**: The release pipeline MUST scan third-party dependencies for known-critical vulnerabilities before publishing a release and MUST block publication when one is found.
- **FR-013**: The repository's automated checks and pipelines MUST require no live credentials, live cloud access, or manual local setup beyond what is provisioned in the CI environment in order to pass.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new contributor can go from a fresh clone to a running `arc --help`/`arc --version` output using a single documented build command, in under one minute.
- **SC-002**: 100% of pull requests opened against the default branch automatically receive both a build-and-test result and a static-analysis result, with zero manual check-triggering steps.
- **SC-003**: 100% of merges to the default branch that pass their checks result in a new, correctly incremented semantic-version release with downloadable artifacts, with zero manual release steps.
- **SC-004**: Every published release's notes contain zero entries from excluded (non-user-facing) commit categories.
- **SC-005**: Time from a passing merge on the default branch to published, downloadable release artifacts is under 15 minutes without human intervention.

## Assumptions

- The project is hosted on GitHub and uses GitHub Actions as its CI/CD platform (already implied by "github actions" in the request and by the project constitution).
- The mandated toolchain named in the project constitution and ADRs applies: `github.com/spf13/cobra` for the command framework, GoReleaser for release packaging, `staticcheck` for static analysis, and a coverage-reporting service (e.g., Coveralls) for coverage-trend visibility.
- "Empty cobra application" means a root command with no business subcommands — it exists to prove the build/CI/release pipeline end-to-end, and to give future features a place to attach subcommands, matching the `cmd/`/`internal/` layering from ADR 001.
- The Go module path follows the existing repository (`github.com/fogfish/arcnet-cli`), producing a binary named `arc`.
- Package-manager distribution beyond raw GitHub release artifacts (e.g., a Homebrew tap) is a "SHOULD", not a "MUST", per the constitution, and is therefore treated as out of scope for this bootstrap feature; it may be added later without breaking this spec.
- No existing automated pipelines, release configuration, or command implementation exist yet in this repository, so this feature adds all of them rather than modifying existing ones.
- A minimal end-to-end test exercising the root command's `RunE` (the `--help`/`--version` behavior) is in scope so Story 1 is independently verifiable in CI, matching the project's E2E testing principle even though no business logic exists yet.
- The release pipeline's GoReleaser Action MUST install a GoReleaser major version compatible with the config schema version declared in `.goreleaser.yaml` — the two are not independently choosable and MUST be pinned together, never left to the action's own default resolution (see BUG-001).

**Bugfix**: 2026-07-01 — BUG-001 Added assumption that the GoReleaser Action's installed version and `.goreleaser.yaml`'s declared config schema version must be pinned together (FR-009, FR-013 affected: the release pipeline was failing on every run in CI because the action defaulted to installing GoReleaser 1.x against a `version: 2` config).
