# Quickstart: Validating Lint Conformance Gaps

**Feature**: `024-lint-conformance-gaps`

Runnable validation for every user story in [spec.md](spec.md). Each section is independent — run
only the story you are verifying.

## Prerequisites

```sh
go build -o /tmp/arc ./cmd/arc
export PATH=/tmp:$PATH
git config --global user.email >/dev/null || git config --global user.email you@example.org
```

## Automated gate

```sh
go test ./...                                    # full suite
go test ./internal/app/lint/kernel/... -run Sowa  # C1.1 exhaustive 144-combination table
go test ./internal/core/... -run Identity         # C2.1 forbidden-character scan
```

---

## US1 — `arc apply` blocks unsafe identities (P1)

```sh
mkdir /tmp/g && cd /tmp/g && arc init
cat > /tmp/bad.md <<'PATCH'
---
"@type": patch
document: doc-a
published: 2026-04-12
---
# doc-a

## Source/rescorla-2026-tls13

---
"@id": rescorla-2026-tls13
"@type": Source
title: "TLS 1.3"
published: 2026-04-12
---
# TLS 1.3

A design retrospective.

## Mentions
- mentions:: [[Handshake/Protocol]]
PATCH
```

**Expect** — rejection naming `/` and its position, graph untouched
([C2.3](contracts/identity-charset-contract.md)):

```sh
git rev-parse HEAD > /tmp/before.hash
arc apply /tmp/bad.md
# exit non-zero; error names "/" and a position in "Handshake/Protocol"
git rev-parse HEAD | diff - /tmp/before.hash   # identical — no commit
git status --short                             # empty
```

**Expect** — the same patch with the offending link fixed applies cleanly:

```sh
sed 's#Handshake/Protocol#Handshake Protocol (TLS)#' /tmp/bad.md > /tmp/good.md
arc apply /tmp/good.md
git log --oneline -1   # one new commit
```

---

## US2 — `arc lint` surfaces existing unsafe identities (P2)

```sh
cd /tmp/g
cat > "Entity/Handshake_Protocol.md" <<'NODE'
---
"@id": "Handshake/Protocol"
"@type": Entity
category: [independent, abstract, occurrent, script]
---
# Handshake/Protocol

NODE
```

**Expect** — a `RuleIdentityCharset` violation naming `/` at position 10
([C2.2](contracts/identity-charset-contract.md)):

```sh
arc lint --json | grep -A2 identityCharset
```

**Expect** — no file changed by the lint run itself:

```sh
git status --short   # only the manually-added fixture above, nothing lint touched
```

**Expect** — a legal citekey-style identity passes clean:

```sh
grep -q '"@id": "rescorla-2026-tls13"' Source/*.md && arc lint --json | grep -c identityCharset
# 0 (or count excludes this node)
```

---

## US3 — `arc lint` rejects mixed-taxonomy Sowa categories (P3)

```sh
cd /tmp/g
cat > "Entity/Free Will.md" <<'NODE'
---
"@id": Free Will
"@type": Entity
category: [independent, physical, continuant, purpose]
definition: placeholder
---
# Free Will

NODE
```

**Expect** — a `RuleEntityCategory` violation naming the rejected tuple and suggesting
`[independent, physical, continuant, object]` ([C1.2](contracts/sowa-category-contract.md)):

```sh
arc lint --json | grep -A3 entityCategory
```

**Expect** — correcting the leaf word clears the violation:

```sh
sed -i '' 's/purpose\]/object]/' "Entity/Free Will.md"
arc lint --json | grep -c entityCategory   # 0
```

**Expect** — a wrong-length category keeps its own distinct message
([C1.3](contracts/sowa-category-contract.md)):

```sh
sed -i '' 's/category:.*/category: [independent, abstract, occurrent]/' "Entity/Free Will.md"
arc lint
# category must decode to exactly four Sowa words, found 3
```
