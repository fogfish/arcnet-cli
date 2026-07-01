# Quickstart: Validate the CLI Infrastructure Bootstrap

## Prerequisites

- Go 1.26 installed locally (matching the pinned CI version — see [research.md](research.md))
- `git`, and for local GoReleaser dry-runs, the `goreleaser` CLI (optional; CI is authoritative)

## 1. Build and run the skeleton (User Story 1)

```bash
go build -o arc ./cmd/arc
./arc --help
./arc --version
```

Expected: `--help` prints usage text and exits 0; `--version` prints a single version line and exits 0. See [contracts/cli-contract.md](contracts/cli-contract.md) for the full table.

## 2. Run the same checks CI runs (User Story 2)

```bash
go build ./...
go test -v -coverprofile=profile.cov $(go list ./... | grep -v /examples/)
staticcheck ./...
```

Expected: all three commands exit 0, matching what `check-test.yml` and `check-code.yml` will report on a pull request. See [contracts/ci-release-contract.md](contracts/ci-release-contract.md).

## 3. Validate the E2E test suite

```bash
go test ./cmd/arc/... -run TestRoot -v
```

Expected: passes, covering the `--help`/`--version` acceptance scenarios (spec User Story 1, Scenarios 2–3) via the `sut()` helper invoking `RunE` directly.

## 4. Dry-run the release config (optional, local only)

```bash
goreleaser release --snapshot --clean
ls dist/
```

Expected: `dist/` contains per-platform binaries plus `checksums.txt`, with no artifacts actually published (snapshot mode). This validates `.goreleaser.yaml` without needing a real tag or GitHub token.

## 5. Confirm the real pipeline (post-merge, in CI)

1. Open a pull request touching any file — confirm both `test` and `check` workflow runs appear and must pass before merge (User Story 2, Scenarios 1–3).
2. Merge to `main` — confirm the `build` workflow creates a new version tag and a GitHub Release with cross-platform artifacts (User Story 3, Scenarios 1–2).
3. Inspect the release notes — confirm no `docs:`/`test:` commits appear (User Story 3, Scenario 3).
