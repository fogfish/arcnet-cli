# Contract: `arc` Root Command Interface

The user-facing contract this feature must satisfy (verified by the E2E test in `cmd/arc/root_test.go`).

| Invocation | Expected exit code | Expected output (stdout unless noted) |
|---|---|---|
| `arc` (no args) | 0 | Usage/help text (Cobra default root behavior) |
| `arc --help` / `arc -h` | 0 | Usage/help text including `Short`/`Long`/`Example` |
| `arc --version` / `arc -v` | 0 | A single version line (e.g., `arc version <semver>`) |
| `arc <unrecognized-flag>` | non-zero | Error message on stderr identifying the unknown flag, plus usage |
| `arc <unrecognized-subcommand>` | non-zero | Error message on stderr identifying the unknown command, plus usage |

**Stability**: Per constitution Principle XIV, only `--json`/`--plain` output is a stable scripting contract. This feature introduces neither, so the human-readable help/usage/version text above MAY change in future minor/patch releases without a major version bump.
