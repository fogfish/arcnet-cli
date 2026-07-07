# Changelog


## 2026-07-07

/speckit-specify Rewrite arc's internal graph node representation to match ARCNET-CORE v0.7 / ARCNET-AST v0.6's predicate-first data model, replacing the current pre-0.5 shape. Study the specifications:
* https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md
* https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md

Today every graph node file uses a "kind" front-matter field and an "id" (or, as a fallback, "title" or "period") to establish identity. The new model requires every node to declare "@id" and "@type" explicitly as quoted YAML keys in front-matter, with "@id" always equal to the file's basename — no fallback to title or period is permitted anymore.

Today a node's prose is split into exactly two fixed slots ("Text" and "Notes"). The new model requires an open, predicate-keyed set of prose fields (e.g. a source's "abstract", an entity's "definition", a resource's "relevance", a hypothesis's "claim") — any number of named prose predicates, not just two.

Today a node's front-matter scalar attributes (tags, category, authors, etc.) are stored internally as plain values. The new model requires every attribute to be stored as an ordered list of values, since a predicate's cardinality (single vs. multi-valued) is no longer assumed from its shape — a single-valued attribute is still a one-element list.

Today outgoing links are split into two different containers depending on whether they were originally written as a flat bulleted list or grouped under a "## Heading"/bold-label block. The new model requires these to be unified into one single ordered list of links per node; whether a link renders flat or grouped under a heading when the node is written back to Markdown must be decided at render time from the target predicate's own declared role, not fixed by how the source document happened to write it.

Consumers of arc (arc apply, arc lint, arc grep, arc subgraph, arc serve, and anyone who runs `arc subgraph --json`) must be able to read and write graph files in this new shape. No needs to support the old format but — arc must not silently corrupt or misread an old-format graph (exit with error is sufficient).

Round-tripping a node (Markdown -> in-memory model -> Markdown) must reproduce the original file's content and connectivity losslessly, matching ARCNET-AST's own conformance checklist (§10).

Out of scope for this feature: per-predicate merge behavior changes (that's a separate feature), schema-node parsing of role/merge/description (that's a separate feature), and CLI-visible flag/command changes.


/speckit-plan Technical approach: this is entirely an internal/core change — internal/core/ast.go, internal/core/markdown.go, and their tests. No internal/app/<use-case> package needs new business logic yet; downstream packages (graph, lint, schema) will only need field-name/type updates to keep compiling, which is fine to land in the same PR since Go won't compile otherwise.

Node type changes (internal/core/ast.go):
- Replace `Kind Kind` with `Type string` (mirrors "@type"); rename `ID string` stays but is now strictly the basename, no fallback derivation.
- Replace `Attrs map[string]any` with `Attrs map[string][]Predicate`, where `Predicate` is a new struct `{ Value any; Target string; Alias string }` — exactly one of Value/Target set per AST §7.
- Replace `Text string` + `Notes string` with `Texts map[string]string`, keyed by predicate name (e.g. "abstract", "definition", "claim", "notes", "text" as the generic fallback per CORE §10.2).
- Replace `Edges []Link` + `Links map[string]LinkBlock` with a single `Edges []Link` (drop the `LinkBlock`/grouping-title storage entirely — AST §3 invariant 4: "grouping is derived, not stored").
- Keep `HRefs []Link` as-is (already matches AST §6.3 conceptually).

Parser changes (internal/core/markdown.go):
- deriveNodeID: delete the id/title/period fallback chain; read "@id" (quoted-key YAML) only, error if absent.
- Front-matter decoding: read "@type" instead of "kind"; wrap every remaining front-matter scalar into `[]Predicate{{Value: v}}` (arrays pass through as multiple Predicate entries).
- Body walking (walkNodeBody): today's logic already distinguishes "leading prose paragraphs" from "a bare list" from "heading/bold-label + list" blocks — keep that structural parsing (it doesn't need schema knowledge to recognize shapes), but (a) route body prose paragraphs into `Texts` keyed by a small type->text-predicate lookup table seeded with CORE/DOMAIN-ARTICLE/DOMAIN-CORE-THOUGHT's known text predicates (source->abstract, entity->definition, resource->relevance, hypothesis->claim, aporia->tension, thought->claim, generic fallback "text"/"notes"), and (b) merge what used to be two containers (bare-list edges, heading-grouped links) into one flat `Edges` slice, no longer keeping the grouping title.
- Note the real dependency on spec 011: the type->text-predicate lookup table above is a stopgap hardcoded map for this spec; spec 011's Schema Index is the eventual source of truth and should replace the hardcoded table once it lands — call this out explicitly as a documented TODO/seam, not a silent duplication.

Renderer changes (RenderNode/RenderPatch): for now, render Edges as a single flat bulleted list (correct per CORE for role="edge" predicates) — full grouped-heading rendering driven by predicate role is deliberately deferred to spec 013 to keep this spec's diff reviewable; document this scoping explicitly in the plan's Constraints section so it isn't mistaken for a completeness gap.

Migration/back-compat: add a clear, tested error path (not a panic, not silent misparse) when a node's front-matter has neither "@id" nor legacy "id" — prefer failing loudly. The support of old-format ("kind"/"id") MUST NOT BE implemented, the compatibility or migration is not a concern at this phase.

Testing: exhaustive table-driven round-trip tests in internal/core/ast_test.go and internal/core/markdown_test.go covering every CORE §11 worked example (source, entity, resource, timeline) and at least one DOMAIN-ARTICLE example (hypothesis with derivedFrom/assumes/addresses) to prove Texts/Attrs/Edges shapes survive Markdown -> model -> Markdown unchanged, per AST §10's checklist. Follow constitution Principle VI (TDD) — write these before the implementation.

Constraints: no new third-party dependency; keep goldmark/goldmark-meta/yaml.v3 as the codec stack; Node/Patch's json tags stay additive-compatible where reasonably possible, but flag explicitly in Complexity Tracking that this feature IS a breaking change to the `arc subgraph --json` contract's Node shape (kind->type, attrs shape change, edges/links merge) — that break is unavoidable and should be called out to the user as a versioning/communication concern, not hidden.


# 2026-07-05

/speckit-specify encode timestamp attribute for graph nodes. The patch document carries on the timestamp `published`. This timestamp has to propogate to each newly created node (except stub on) in the graph. Then, it adds a new attribute for each newly created node `indexed` with ISO8601 timestamp at seconds resolution. The `indexed` timestamp is identical for all nodes in the patch. In node has been merged then `updated` with ISO8601 timestamp at seconds resolution. Both `indexed` and `updated` carries on identical timestamp for the single patch document. All node in the graph carries on `published` and `indexed`. All node been merged carries on `updated`. The `published` attribute is exported out.   

/speckit-plan defined `published` attribute at `Node` type level, making it de-facto core standard attributed. Modify `apply` command to mandage `published`, `indexed` and `updated` attributes. Make sure that `published` and `indexed` has not overrwiten once created at the node level.

---

/speckit-specify `arc serve` — start an MCP server (stdio transport by default; `--http <port>` for SSE) exposing these tools:
  - `node_get(id)` → full node object (ARCNET-AST §4): attrs, text, edges, links
  - `node_grep(pattern, filter?)` → list of `{id, kind, line, snippet}` for nodes whose content matches a regexp pattern, optionally pre-filtered by the filter object
  - `subgraph_get(id, depth?)` → return the fully-resolved subgraph rooted at `id` to `depth` hops (default 1): a flat array of complete node objects for the seed and every reachable neighbor; covers the same operation as `arc subgraph` for agent context expansion mid-conversation

/speckit-plan mcp server is an frontend to existing domains. Use exising `internal/app/graph` and implement only "wiring" of these tools. Reply data in markdown format to for MCP client.


# 2026-07-04

/speckit-specify `arc subgraph <basename> [--depth <n>] [<filter>]` — extract a self-contained subgraph: the seed node plus all nodes reachable within N hops (default 1), optionally filtered by kind or attributes on the reached nodes; the filter applies to the expanded nodes, not the seed; output uses the patch exchange format (CORE §12.2) as the serialization: nodes are grouped by kind under `# <Kind>` headings, each node under `## <basename>`, front-matter in a fenced YAML block, body verbatim below — human-readable, LLM-friendly, and round-trippable back into `arc apply`

/speckit-plan implement grap grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). Do not use the color system for output. The graph serialization to patch format is part of the core `internal/core`

---

/speckit-specify `arc grep [<filter>] <pattern>` — scan nodes matching the filter (see Filtering) for lines matching the regexp `<pattern>`; print `<kind>  <id>  <line-number>  <matched line>`, one match per output line; without a filter, scans every node file; suitable for piping to standard tools.

/speckit-plan implement grap grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). Use colors to highite matched text if color mode is enabled. If macthed line longer that 80 chars (configurable via `.arc/config`) the ellipse before and after to fit the match roughtly to one terminal line. 

However, make a file-system grep utility as a reusable, performance optimized packaged at `internal/pkg/grep`. The utility must:
* Use parallel walker of directory traversal.
* Use parallel file processing.
* Use a bounded worker pool (number of workes configurable via `.arc/config` default is 8) and close files after processed.
* Use buffered reads (bufio.Reader).
* Buffer reuse with sync.Pool (minimize memory allocation).
* Be configured for particular file extension (*.md by default).
* Literal search with bytes.Contains when possible.
* Regex only when the query actually requires it.
* Treat files as plain text within the lib.


---

/speckit-specify Make a schema as a first class citizen of the graph. Instead of `_meta` and `.arc/config` a new folder `_schema` is defined. The folder contains subfolders: (a) `nodes/` contains a document per node kind (e.g. entity.md) and `predicates/` contains a documents per predicate (e.g. related.md). Each of them has `id` equal to file base name (equal to name of this entity) and `kind: schema`. The nodes document also contains a `merge` attribute. It substitude `.arc/config` behaviour. The schema is created by `arc init` for core specification (see https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md). The schema is extended by `arc apply` when new node kind or predicate is discovered in the graph. 

/speckit-plan schema as own domain `internal/app/schema`. Remove "merge" configurability from `.arc/config` but keep the config infrastructure alive, just remove the github downloader, it is not relevant anymore. Integrate `schema` domain with `apply` and `init`. Isolate ALL ARCNET-CORE abstractions, definitions, const and invariants within `schema` domain. It MUST BE a single entity in the app that has dependencies to https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md specification. 


# 2026-07-03

/speckit-specify `arc lint` — run the full CORE §14 checklist across every node and report violations with file path and line number for each: valid YAML front-matter and `kind` field; unique basenames (CORE §3.2); every `[[link]]` resolves to an existing basename; `source` citekey `id` equals its basename (CORE §6.2); `entity` four-word decoded Sowa `category` (CORE §9.2.1); derived nodes link back to at least one `source` (CORE §3.4); predicates are camelCase and registered in `_meta/predicates.md` (CORE §7.3); citations use a registered `cito:`-aligned predicate (CORE §8); each document is exactly one `graph(ingest):` commit (CORE §11.1); extension kind conformance per the kind's profile checklist and graph nodes does not have any active merge conflicts.

/speckit-plan liner is own domain `internal/app/lint`. Also maintain same hierarchy in `cmd/arc/lint`. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). `arc lint` in the normal verbosity mode shows only nodes with issues. `arc lint -v` in the verbose node shows status for each node. In the end it shows the overall graph status.

# 2026-07-02

/speckit-specify `arc apply <patch.md>` — apply a patch file to the graph (CORE §12.3): parse the patch manifest (`kind: patch`, `document`, `published`, `stats`); check idempotency and skip with a clear message if `sources/<id>.md` is already tracked (CORE §11.2); for each H1/H2 node section reconstruct the node object (ARCNET-AST §4 ); **create** new node files when the basename does not exist; **merge** into existing files per the kind's declared merge operation — `none` for `source`, `union` for `entity`, `union first-writer` for `resource`, and per-profile operation for domain/extension kinds (CORE §10); derive and append timeline entries from the source's `published` date (CORE §9.4); produce exactly one git commit with the mandatory subject, stats, and `Source-Id:` trailer (CORE §11.3); update the local index cache (Phase 4) atomically within the same filesystem transaction

See specifications:
* CORE https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md
* ARCNET-AST https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md

/speckit-plan graph I/O is own domain `internal/app/graph`. Also maintain same hierarchy in `cmd/arc/graph`. Define the graph AST https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md as core domain (`internal/core`). Parse the patch using `github.com/yuin/goldmark` markdown parser into AST and then use AST to patch the graph itself. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). Establish YAML based config at `.arc/config.yml` (as part of `arc init`), this version of the config defines only merge rules per kind, the config management is implemented at `internal/app/config`. The default merge rules for all supported profiles is defined at `github.com/fogfish/arcnet-spec`. 

---

/speckit-specify `arc init [<dir>]` — initialize a new knowledge graph: create the canonical folder layout (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`); write stub files `_meta/predicates.md` and `_meta/aliases.md`; create `.arc/` for arc-managed state (see Graph Root from VISION.md); run `git init` and create `.gitkeep` for empty folders; write `.gitignore` excluding `.arc/`; stage everything and produce the initial commit `graph(init): empty knowledge graph` (CORE §11 https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md)

/speckit-plan graph management (control plane) is own domain `internal/app/ctrl`. Also maintain same hierarchy in `cmd/arc/ctrl`. Integrate `git` as first class concept via invocation of command-line tool, informing user of `arc` about the git tool progress. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux0design-system.md).       

# 2026-07-01

/speckit-specify setup the infrastructure for development of cli called `arc`. The infrastructure includes (1) an empty cobra application; (2) github actions to test, check and release application; (3) goreleaser configuration and github actions integrations.

/speckit-plan setup the infrastructure following the mandatory libraries defined by the constitution.md. Use https://github.com/fogfish/iq/tree/main as an example on how to setup GitHub Action and GoReleasing for testing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-test.yml, linting (staticcheck) https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-code.yml and releasing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/build.yml https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.goreleaser.yaml

