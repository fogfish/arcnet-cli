# Quickstart: Graph Schema as a First-Class Citizen

Validates spec.md's three user stories end-to-end against a real local graph. No network access is required anywhere in this feature (research.md D5).

## Prerequisites

- `git` on `PATH`
- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- An empty target directory

## Scenario 1 — `arc init` seeds a first-class, versioned schema (spec.md User Story 1)

```sh
$ arc init graph && cd graph
✅ Initialized empty knowledge graph at /path/to/graph (commit a1b2c3d)

$ find _schema -type f | sort
_schema/nodes/entity.md
_schema/nodes/resource.md
_schema/nodes/source.md
_schema/nodes/timeline.md
_schema/predicates/broader.md
_schema/predicates/cites.md
_schema/predicates/conformsTo.md
_schema/predicates/hasPart.md
_schema/predicates/isCitedBy.md
_schema/predicates/isPartOf.md
_schema/predicates/isReplacedBy.md
_schema/predicates/mentionedIn.md
_schema/predicates/mentions.md
_schema/predicates/narrower.md
_schema/predicates/related.md
_schema/predicates/replaces.md
_schema/predicates/requires.md

$ cat _schema/nodes/entity.md
---
kind: schema
merge: union
---
# entity

A concept or subject mentioned across sources, mergeable across contributions.

$ ls _meta 2>&1
ls: _meta: No such file or directory

$ git log --oneline -1
a1b2c3d graph(init): empty knowledge graph
```

**Expected outcome**: every core node kind and predicate is present as its own readable, committed document; no `_meta/` folder and no merge-rule content in `.arc/config.yml` exist.

## Scenario 2 — Applying a patch with a network access outage still succeeds (spec.md User Story 1, Acceptance Scenario 5)

There is no fetch to fail (research.md D5) — `arc init` behaves identically with or without network access:

```sh
$ HTTP_PROXY=http://localhost:1 HTTPS_PROXY=http://localhost:1 arc init graph2
✅ Initialized empty knowledge graph at /path/to/graph2 (commit ...)
```

## Scenario 3 — `arc apply` auto-registers a previously-unseen node kind and predicate (spec.md User Story 2)

```sh
$ cat > note.md <<'EOF'
---
kind: patch
document: first-note
published: 2026-07-04
title: A note that introduces a new kind
EOF

# hypothesis
## first-note
```yaml
category: draft
```
A working hypothesis, related to TLS.

**Related**
- related:: [[Transport Layer Security]]
EOF

$ arc apply note.md
✅ first-note applied — hypothesis: +1 created (commit d4e5f6a)
   ⚠ "hypothesis" is not a recognized node kind for this graph — applied using the default "union" merge behavior

$ cat _schema/nodes/hypothesis.md
---
kind: schema
merge: union
---
# hypothesis

$ git show --stat d4e5f6a | grep hypothesis
 _schema/nodes/hypothesis.md | 4 ++++
 hypotheses/first-note.md    | 6 ++++++
```

**Expected outcome**: the new kind's schema document is created in the *same* commit as the patch's own content — `git show --stat` on that one commit lists both files.

## Scenario 4 — Re-applying the same kind no longer warns (spec.md User Story 2, Acceptance Scenario 3 / User Story 3, Acceptance Scenario 3)

```sh
$ cat > note2.md <<'EOF'
---
kind: patch
document: second-note
published: 2026-07-05
title: A second hypothesis, no warning this time
EOF

# hypothesis
## second-note
```yaml
category: draft
```
Another working hypothesis.
EOF

$ arc apply note2.md
✅ second-note applied — hypothesis: +1 created (commit ...)
```

**Expected outcome**: no unrecognized-kind warning — the kind was registered by Scenario 3's application; `_schema/nodes/hypothesis.md` is unchanged (not duplicated or overwritten, spec FR-011).

## Scenario 5 — Editing a schema document's `merge` behavior changes future applies (spec.md User Story 3, Acceptance Scenario 3)

```sh
$ sed -i '' 's/merge: union/merge: union-first-writer/' _schema/nodes/hypothesis.md
$ git add _schema/nodes/hypothesis.md && git commit -m "graph(schema): hypothesis is union-first-writer"

$ arc apply note3.md   # a third hypothesis patch, same kind
✅ third-note applied — hypothesis: +1 merged (commit ...)
```

**Expected outcome**: the hand-edited `merge: union-first-writer` is the behavior `arc apply` actually uses — confirmed by inspecting the merge outcome against a fixture where `union-first-writer`'s empty-field-fill behavior is observably different from `union`'s.

## Scenario 6 — `arc lint` never flags `_schema/` documents (spec.md Clarifications Q1/Q3)

```sh
$ arc lint
✅ 4 nodes checked, 4 passing, 0 failing
```

**Expected outcome**: the `_schema/nodes/hypothesis.md` and every other schema document are absent from the checked-node count entirely — confirmed by `find . -name '*.md' -not -path './_schema/*' | wc -l` matching the reported count exactly.

## Verifying schema documents are excluded from graph-wide basename uniqueness (Clarifications Q3)

```sh
$ echo '---
kind: entity
category: [independent, abstract, occurrent, script]
---
# hypothesis
A namesake entity, unrelated to the schema kind of the same name.' > entities/hypothesis.md

$ arc lint
✅ 5 nodes checked, 5 passing, 0 failing
```

**Expected outcome**: an ordinary content node named `hypothesis` (in `entities/`) coexisting with the schema document `_schema/nodes/hypothesis.md` is not reported as a basename collision.
