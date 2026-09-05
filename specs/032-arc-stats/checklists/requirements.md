# Specification Quality Checklist: Graph Statistics (`arc stats`)

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-05
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Validation Notes

**Re-validated 2026-09-05 after `/speckit-clarify`** — 5 questions asked and
answered; all 16 items still pass (16/16 → 16/16, no state changes).

Pre-draft decisions (recorded before the spec was first written):

1. **Ingestion basis** — read from the graph's own yearly/monthly ingestion
   period records, not by re-parsing each source node's publication date.
2. **Verbose statistic set** — all four candidate groups included:
   connectivity health (FR-014), degree & hubs (FR-015), schema coverage
   (FR-016), content volume (FR-017).

Clarification session decisions (see spec `## Clarifications`):

3. **Node census** — every node file counts, so the per-type breakdown sums
   to the total (FR-002). Folder layout is explicitly NOT fixed: users add
   folders and move nodes, so FR-001 mandates layout-independent discovery
   and FR-004a forbids deriving any figure from a file's path.
4. **`--json` + `--verbose`** — machine-readable output mirrors verbosity
   (FR-020a); detail fields are omitted entirely, not nulled, so a consumer
   distinguishes "not requested" from "requested and empty". Verbosity gates
   computation, not just rendering (FR-018).
5. **Exit status** — always 0 on a completed scan; non-zero only when no
   report could be produced (FR-024). Health gating stays with `arc lint`.
6. **Broken-link scope** — spans structural edges and inline references,
   distinct targets per node, identical to the conformance check (FR-006).
   The knowing asymmetry against the edge total is not left to be inferred:
   FR-006a requires the report to state both counting rules.
7. **Terminology** — the breakdown is by *type* (what a node declares), and
   *Class* keeps its established meaning as the schema node registering a
   type's vocabulary. The two are held distinct throughout, per Principle II.

Defaults documented rather than spent as questions: ranking truncated to ten
entries (FR-015), degree statistics computed over all nodes including
zero-degree ones, "placeholder" normalized to the codebase's canonical term
**stub**.

Cross-cutting consistency requirements, so the report cannot contradict
itself: FR-004 (type counts sum to node total), FR-012 (predicate counts sum
to edge total), FR-013 (months sum to their year), FR-014 (per-target figures
sum to the broken-link total), FR-006/SC-003 (agreement with the conformance
check).

Terminology stays domain-level throughout — "type", "predicate", "ingestion
period record", "stub node" — rather than naming file layouts or flag
spellings, which belong to `/speckit-plan`.

- Items marked incomplete require spec updates before `/speckit-clarify` or `/speckit-plan`
