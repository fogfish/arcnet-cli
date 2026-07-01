# Changelog

# 2026-07-01

/speckit-specify setup the infrastructure for development of cli called `arc`. The infrastructure includes (1) an empty cobra application; (2) github actions to test, check and release application; (3) goreleaser configuration and github actions integrations.

/speckit-plan setup the infrastructure following the mandatory libraries defined by the constitution.md. Use https://github.com/fogfish/iq/tree/main as an example on how to setup GitHub Action and GoReleasing for testing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-test.yml, linting (staticcheck) https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-code.yml and releasing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/build.yml https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.goreleaser.yaml

