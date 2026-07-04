# Quickstart: `arc grep`

Validates spec.md's three user stories end-to-end against a real local graph. `arc grep` needs no network access and makes no filesystem or git-history changes (spec FR-010, SC-006) — every scenario below can be re-run repeatedly with identical results.

## Prerequisites

- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- A graph created by `arc init` with at least one document ingested via `arc apply` (see `specs/003-apply-patch/quickstart.md` Scenario 1 for a ready-made setup)

## Scenario 1 — Find every occurrence of a term across the whole graph (spec.md User Story 1)

```sh
$ arc grep TLS
source  rescorla-2026-tls13  1  TLS 1.3 is the latest version of the Transport Layer Security protocol.
entity  Transport Layer Security  1  TLS is the successor to SSL.
$ echo $?
0
```

A pattern with no matches anywhere in the graph:

```sh
$ arc grep NoSuchTermAnywhere
$ echo $?
1
```

**Expected outcome**: every occurrence is reported with its owning node's `kind`/`id` and the exact line number, one match per output line (contracts/cli-contract.md); a pattern matching nothing produces no stdout lines and a non-zero exit, distinguishing "ran, found nothing" from a crash (research.md D12).

A node whose content matches on more than one line reports each line separately:

```sh
$ arc grep protocol
source  rescorla-2026-tls13  1  TLS 1.3 is the latest version of the Transport Layer Security protocol.
source  rescorla-2026-tls13  4  This protocol replaces earlier, now-deprecated versions.
```

## Scenario 2 — Narrow the search to a subset of nodes (spec.md User Story 2)

```sh
$ arc grep --kind entity TLS
entity  Transport Layer Security  1  TLS is the successor to SSL.

$ arc grep --tag cryptography TLS
entity  Transport Layer Security  1  TLS is the successor to SSL.

$ arc grep --kind entity --attr status=mature TLS
entity  Transport Layer Security  1  TLS is the successor to SSL.
```

**Expected outcome**: only matches from nodes satisfying every applied condition are reported (research.md D8, VISION.md Filtering); combining `--kind`/`--tag`/`--attr` narrows further (AND across groups). A filter matching zero nodes behaves exactly like a pattern matching nothing:

```sh
$ arc grep --kind resource TLS
$ echo $?
1
```

## Scenario 3 — Pipe search results into other command-line tools (spec.md User Story 3)

```sh
$ arc grep TLS | wc -l
2

$ arc grep TLS | cut -f1
source
entity

$ arc grep TLS | awk '{print $2, $3}'
rescorla-2026-tls13 1
Transport Layer Security 1
```

**Expected outcome**: no header, footer, or summary line is mixed into the output (spec FR-006/FR-007), so standard line-counting and field-extraction tools work without any special-casing — this holds whether or not the terminal supports color, since truncation/highlighting are gated on the same signal as color (research.md D11) and never alter the piped bytes.

## Scenario 4 — Color highlighting and long-line fitting on an interactive terminal (user's explicit UX requirement)

```sh
$ arc grep --color TLS
source  rescorla-2026-tls13  1  \x1b[1mTLS\x1b[0m 1.3 is the latest version of the Transport Layer Security protocol.
```

(the escape sequences above render as the matched substring `TLS` in bold/color on a real terminal — `research.md` D11, ADR 002 DS-05's `SCHEMA.Match`)

```sh
$ arc grep --color "static RSA"
source  rescorla-2026-tls13  42  …emoves support for [static RSA] key exchange, replacing it with ephemeral…
```

(a line longer than the configured `maxLineWidth` — default 80, `.arc/config.yml` `grep.maxLineWidth` — is ellipsis-fitted around the match; brackets above stand in for the actual bold/color rendering)

```sh
$ arc grep --color --verbose "static RSA"
source  rescorla-2026-tls13  42  TLS 1.3 removes support for static RSA key exchange, replacing it with ephemeral Diffie-Hellman key agreement for every handshake.
```

**Expected outcome**: `--verbose` shows the full, untruncated line (still colorized); piping either invocation through `cat` (a non-TTY) reproduces Scenario 3's plain, untruncated, unstyled output regardless of `--color` on the underlying terminal, since color is auto-disabled off-TTY per ADR 002 DS-05 unless forced.

## Configuring workers and line width (research.md D10)

```sh
$ cat >> .arc/config.yml <<'EOF'
grep:
  workers: 16
  maxLineWidth: 120
EOF

$ arc grep TLS
```

**Expected outcome**: the search still returns identical matches (worker count and display width are performance/presentation knobs only, per data-model.md — they never change which lines are reported, only how many run concurrently and how a long line is fitted for display).

## Verifying read-only behavior (spec SC-006)

```sh
$ git status --short > /tmp/before.txt
$ arc grep TLS
$ git status --short > /tmp/after.txt
$ diff /tmp/before.txt /tmp/after.txt
# empty — arc grep changed nothing
```
