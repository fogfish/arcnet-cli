
Confirmed the assumption: specs/CHANGELOG.md:72 plans arc retract as a plain git revert, banking on "git's three-way merge correctly un-merges shared nodes." But internal/core/merge.go shows Merge isn't a textual merge at all — it's semantic reconciliation (paragraph-fuzzy dedup via Jaccard shingles, attrs union/first-writer-wins, conflict markers embedded as text) with no per-field provenance retained. Once a node has absorbed a second source, there's no line-level diff left for git to invert correctly — that's exactly the metadata/content loss you're seeing.

Recommendation: swap "revert via git" for "rebuild via replay." You already tag commits with Source-Id and know patchPath per ingest — so for a given node, walk the commits that touched it, drop the one being retracted, and refold Merge from a zero Node over the remaining contributions in original order, then commit the recomputed result. Main tradeoff: this needs an index (or git log --grep=Source-Id scan) of which patches touched which node, and Merge has to stay deterministic/order-stable for replay to reproduce the right state — more machinery than a bare git revert, but it's correct even when a node has been merged into many times since, whereas git revert only ever worked for nodes untouched since the commit being undone.


# 016 — Domain profile support (optional, deferred)
/speckit-specify text:


Confirm and, where needed, extend arc so that a graph adopting a domain profile beyond ARCNET-CORE's four built-in types (source/entity/resource/timeline) — for example ARCNET-DOMAIN-ARTICLE's "hypothesis"/"aporia" types or ARCNET-DOMAIN-CORE-THOUGHT's "thought" type — works correctly with every arc command, using only the schema mechanism (spec 011) to learn about the new types and their predicates, with no arc code changes required per profile.

A user should be able to: register a new type and its predicates purely by adding _schema/types/ and _schema/predicates/ documents to their graph (by hand, or via a future profile-seeding convenience this feature may add); apply a patch containing nodes of that new type via arc apply; have arc lint validate those nodes' Requires/Optional conformance; have arc grep/arc subgraph filter and export nodes of that type, exactly as it does for source/entity/resource/timeline today.

This feature is primarily a verification/hardening pass rather than new functionality, since specs 010-014 should make the type system fully open by construction — the deliverable is confirming that claim with real DOMAIN-ARTICLE/DOMAIN-CORE-THOUGHT fixture graphs and closing any gap found (e.g. a place in arc that still assumes exactly four types, or assumes a type-specific text-predicate mapping that isn't schema-driven).

Optionally, in scope if time permits: an `arc init --profile article` (or similar) convenience that seeds a graph with a chosen domain profile's _schema/ documents in addition to CORE's, so a user doesn't have to hand-author them.
/speckit-plan text:


Technical approach: primarily an audit + fixture-driven verification spec, run only after 010-014 are stable. Build two test fixture graphs under testdata/ — one exercising ARCNET-DOMAIN-ARTICLE (hypothesis/aporia, including the source-type extension adding proposes/raises) and one exercising ARCNET-DOMAIN-CORE-THOUGHT (thought, including the generatedThought backlink) — and run the full command surface (init, apply, lint, grep, subgraph) against each, per DOMAIN-ARTICLE §7 / DOMAIN-CORE-THOUGHT §5's own conformance checklists.

Known risk areas to specifically check, based on today's codebase: the hardcoded type->text-predicate lookup table introduced as a stopgap in spec 010 (flag if it wasn't fully replaced by the Schema Index in spec 011 as planned — if so, this is the spec that must close that gap, since a profile's "claim"/"tension" text predicates won't be in any hardcoded table); rules_identity.go's checkSourceCitekey and checkEntityCategory (currently hardcoded to node.Kind == "source"/"entity" specifically — confirm these stay correct for profile types that don't need equivalent checks, and that no profile type accidentally triggers them); citoPredicates-derived logic (spec 014 should have already generalized this, verify).

Optional --profile convenience (internal/app/ctrl/service/init.go, internal/app/schema/service): if included, define profile seed data as Go-embedded Markdown fixtures (matching the ARCNET-DOMAIN-ARTICLE.md/ARCNET-DOMAIN-CORE-THOUGHT.md worked examples' own _schema/ nodes) selected by a new `--profile` flag on `arc init`; keep this additive-only and behind an explicit flag so default `arc init` behavior (CORE-only) is unchanged.

Testing: E2E only, against the two new fixture graphs; no new unit-level logic is expected if 010-014 did their job — a failing test here should usually point back at a gap in one of those specs rather than require new arc code.

Constraints: this spec should be scoped small and treated as low-priority/deferred relative to 010-015 — recommend not starting it until there's a concrete need for one of these profiles in a real graph.
Let me know which of these you want to actually run through /speckit-specify — I'd suggest starting with 010, since 011-015 all build on it.


//TODO:
(schema from patch)
schema: attributes with link to schema patch document


C1 rdfs:subClassOf C2