# `v011-graph` — a hand-built ARCNET-CORE v0.11 fixture

Every node here carries **exactly** the predicates ARCNET-CORE v0.11 requires of its
type, and no more. It is written by hand, never generated from `arc init` output or
from `arc apply`: a lint expectation asserted against the tool's own output agrees
with the tool by construction and passes vacuously (`CORE-FIX.md` §5.7,
`specs/023-core-vocabulary-conformance` T011).

The tree holds **content nodes only**. A consumer seeds `_schema/` from
`schema.Seed()` and initializes git itself, so the fixture never carries a stale copy
of the built-in vocabulary.

| Node | Shape under test |
| --- | --- |
| `Source/kolesnikov-2026-vocabulary.md` | all four of §11.2's required predicates — `title`, `published`, `abstract` (leading prose), `mentions` |
| `Entity/Merge Vocabulary.md` | §11.3's `category`, `definition` (leading prose), `mentionedIn` — and **no** `published`/`created` |
| `Reference/RFC 8446.md` | §11.6 v0.11's required `title` **alone** |
| `timeline/yearly/2026.md` | §11.5's required `cites` **alone** |
| `timeline/monthly/2026-04.md` | `cites` plus the optional `granularity`, `period`, `heading` |

`README.md` is not a node: consumers copy `*.md` under the node folders only.
