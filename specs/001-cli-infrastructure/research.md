# Phase 0 Research: CLI Development Infrastructure Bootstrap

## Reference implementation

The user pointed to [`github.com/fogfish/iq`](https://github.com/fogfish/iq/tree/main) as the pattern to mirror for CI and release automation. Its three workflows and `.goreleaser.yaml` were fetched directly and used as the concrete baseline (adjusted only for this repo's module path, binary name, and constitution requirements):

- `.github/workflows/check-test.yml` — PR-triggered `go build` + `go test -coverprofile` + Coveralls upload.
- `.github/workflows/check-code.yml` — PR-triggered `staticcheck` via `dominikh/staticcheck-action`.
- `.github/workflows/build.yml` — push-to-`main`-triggered build/test, automatic SemVer tag via `reecetech/version-increment`, then `goreleaser/goreleaser-action` release.
- `.goreleaser.yaml` — `CGO_ENABLED=0` builds for linux/windows/darwin (windows/arm64 excluded), binary-only archives, `checksums.txt`, changelog excluding `^docs:`/`^test:` commits, and a Homebrew tap (`brews:`) block.

## Decision: Go toolchain version

- **Decision**: Pin Go `1.26` in `go.mod` and in every `actions/setup-go` step.
- **Rationale**: Constitution Principle XIII/Mandatory Tooling requires the CI Go version be pinned, never "latest". The reference project pins `1.25`, but this repository targets `1.26` to match the toolchain actually installed in this environment (1.26.2), keeping local builds and CI builds on the same major.minor line; the pinning mechanics (explicit `go-version` string, never floating) are what carries over from the reference project, not the exact number.
- **Alternatives considered**: Matching the reference project's `1.25` pin exactly — rejected once the target version was explicitly changed to `1.26`.

## Decision: CLI framework and command scope

- **Decision**: `github.com/spf13/cobra` root command only, binary name `arc`, module `github.com/fogfish/arcnet-cli`. No subcommands are added in this feature.
- **Rationale**: Constitution Principle IX mandates Cobra as the sole framework; the feature spec (User Story 1) explicitly scopes this feature to an empty, buildable skeleton, not any business subcommand.
- **Alternatives considered**: None — Cobra is a hard constitutional mandate (no `urfave/cli`, no hand-rolled `flag` wrapper permitted).

## Decision: Terminal styling dependency

- **Decision**: Do **not** add `github.com/charmbracelet/lipgloss` in this feature.
- **Rationale**: Constitution Principle X mandates lipgloss for "all colored/styled terminal output." An empty root command emits only Cobra's own built-in plain-text help/usage/version output — there is no styled output to implement yet, so adding the dependency now would be an unused import with nothing to style. It will be introduced in the first feature that actually renders colored/styled output.
- **Alternatives considered**: Adding it preemptively as scaffolding — rejected per the project's own "don't add for hypothetical future requirements" norm; it is trivial to add exactly when the first styled command needs it.

## Decision: Test framework and E2E pattern

- **Decision**: `go test ./...` with `github.com/fogfish/it/v2` (`it.Then(t).Should(...)`) as the sole assertion library. One colocated E2E test, `cmd/arc/root_test.go`, using a shared `sut()` helper that pipes `os.Stdout` and invokes the root command's `RunE` directly, covering the `--help` and `--version` acceptance scenarios from the spec.
- **Rationale**: Constitution Principles VI and VIII mandate `it/v2` and the colocated `cmd/<command>_test.go` + `sut()` pattern specifically so E2E tests exercise the exact production `RunE` Cobra dispatches to.
- **Alternatives considered**: A separate `tests/e2e/` tree spawning the compiled binary as a subprocess — rejected; the constitution explicitly calls this out as the pattern being replaced.

## Decision: Coverage reporting

- **Decision**: `shogo82148/actions-goveralls@v1` uploading `profile.cov`, with `continue-on-error: true` (matching the reference project), in both the PR test workflow and the `main`-branch build workflow.
- **Rationale**: Constitution Mandatory Tooling names Coveralls (via this action, "or equivalent") explicitly as the coverage-reporting service; `continue-on-error` keeps a Coveralls outage from blocking merges/releases, since coverage visibility is a trend signal, not a merge gate itself (the gate is `go test` passing).
- **Alternatives considered**: Codecov — rejected, not what the constitution/reference project names; would require a different token/setup for no added value here.

## Decision: Static analysis gate

- **Decision**: `dominikh/staticcheck-action@v1.3.1` with `install-go: false` (Go already set up by the preceding `actions/setup-go` step), on `pull_request` `opened`/`synchronize`, as a separate required `check` workflow from the test workflow.
- **Rationale**: Constitution Mandatory Tooling requires `staticcheck` to run in a dedicated workflow and block merge independent of the test workflow (`check-code` vs `check-test`).
- **Alternatives considered**: `golangci-lint` — rejected; the constitution names `staticcheck` specifically, not a meta-linter.

## Decision: Versioning and release automation

- **Decision**: On push to `main`: build + test (as in the PR workflow) run again, then `reecetech/version-increment@2023.10.2` computes the next SemVer tag (patch by default, minor/major via a `[x.Y.z]`/`[X.y.z]` commit-message marker exactly as the reference project does), the tag is pushed, then `goreleaser/goreleaser-action@v5` runs `release`.
- **Rationale**: Constitution Principle XIV mandates automatic SemVer tag increment plus a GoReleaser release on the `build` workflow, with breaking-change major bumps signaled explicitly rather than inferred — the commit-marker convention from the reference project satisfies this without extra tooling.
- **Alternatives considered**: A dedicated "release-please"/changesets-style bot — rejected as unnecessary process weight for a bootstrap feature with zero user-facing commands yet; the simpler marker convention matches the reference project and constitution's minimum bar.

## Decision: GoReleaser configuration shape

- **Decision**: Mirror the reference `.goreleaser.yaml` almost exactly: `go mod tidy` pre-hook, `CGO_ENABLED=0`, `goos: [linux, windows, darwin]` with `windows/arm64` ignored, binary-only archives, `checksums.txt`, ascending changelog excluding `^docs:`/`^test:`, and a `brews:` tap block pointed at this repo (owner `fogfish`, name `arcnet-cli`, binary `arc`).
- **Rationale**: The user explicitly asked to use the reference project as the setup pattern; constitution Principle XIII mandates GoReleaser plus (where the ecosystem has a package-manager convention) a Homebrew formula whose smoke test invokes `<binary> --version`.
- **Alternatives considered**: Dropping the `brews:` block since it's a "SHOULD" not "MUST" — rejected in favor of matching the reference project 1:1 as instructed, since Homebrew is a valid, low-cost convention for this ecosystem and the constitution explicitly anticipates it.

## Decision: Vulnerability scanning

- **Decision**: Add a `govulncheck` step to the release path (`build` workflow, before the GoReleaser step), even though it did not appear in the fetched reference workflows.
- **Rationale**: Constitution Principle XIV / Mandatory Tooling states `govulncheck` (or equivalent) MUST scan dependencies before every release and MUST block on known-critical vulnerabilities — this is a binding constitutional gate, not optional, so it is added even though the reference project's example didn't include it.
- **Alternatives considered**: Skipping it to match the reference project exactly — rejected; the constitution takes precedence over the reference example where the two diverge, and the spec's edge cases (FR-012) already require this gate.

## Open questions resolved

No `[NEEDS CLARIFICATION]` markers remained in the spec; the only two implementation choices requiring a decision (lipgloss inclusion timing, govulncheck placement) are resolved above with rationale rather than deferred.
