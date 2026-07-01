# Contract: CI/Release Pipeline Interface

The automation contract this feature must satisfy, mapped to spec User Stories 2 and 3.

## Pull Request Checks

| Workflow | Trigger | Required for merge | Produces |
|---|---|---|---|
| `check-test.yml` (`test`) | `pull_request: opened, synchronize` | Yes | `go build ./...` + `go test -coverprofile` result; coverage uploaded to Coveralls |
| `check-code.yml` (`check`) | `pull_request: opened, synchronize` | Yes | `staticcheck` pass/fail, independent of `test` |

## Default-Branch Pipeline

| Workflow | Trigger | Steps (in order) |
|---|---|---|
| `build.yml` (`build`) | `push: branches: [main]` | 1) `go build`/`go test` + coverage upload 2) determine SemVer increment from commit message marker 3) create + push version tag 4) `govulncheck` (blocks on known-critical) 5) GoReleaser `release` |

## Release Output Contract

- Cross-platform archives: `linux`, `darwin`, `windows` (amd64/arm64, excluding `windows/arm64`), binary-only format.
- `checksums.txt` accompanying every release.
- Changelog excludes commits matching `^docs:` or `^test:`.
- Homebrew formula (`brews:`) published to the tap, whose test invocation is `arc --version`.

## Failure Contract

- Any required PR check failing MUST block merge (GitHub branch protection is the enforcement mechanism, configured out-of-band from the workflow YAML itself — noted as an operational setup step, not a file this feature can encode).
- A `govulncheck` finding of known-critical severity MUST fail the `build` job before the GoReleaser step runs, so no artifacts are published for that commit.
