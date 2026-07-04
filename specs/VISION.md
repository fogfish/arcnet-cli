# VISION — `arc` CLI

**Scope:** `arc` is the command-line tool for knowledge graph management. `arc` owns bootstrapping, ingestion, navigation, validation, repair, and serving the graph as a knowledge service.

`arc` is **extension-agnostic**: domain profiles (ARCNET-DOMAIN-*) add new `kind` values and predicates, but `arc` handles them transparently through the core mechanisms — kind-keyed merge operations, the predicate registry, and the alias table. No special subcommand exists for any extension.

---

## Graph Root

`arc` locates the graph root by walking up the directory tree from the current working directory until it finds a `.arc/` directory, exactly as git locates `.git/`. All commands operate on the graph at the located root; if no `.arc/` directory is found, `arc` exits with an error. The `--graph <dir>` flag overrides discovery and pins the root explicitly.

The `.arc/` directory is the single location for all arc-managed state that is not part of the versioned graph: the binary index, cached metadata, and future tooling state. It is created by `arc init`, excluded from git via `.gitignore`, and never committed.

---

## Filtering

Several commands accept a filter to narrow the set of nodes they operate on. Filters are composable: all flags present in a single invocation are ANDed together. Where a flag is repeatable, the repeated values combine as stated below.

**Kind filter**

`--kind <kind>` restricts results to nodes whose `kind` field equals `<kind>`. The flag is repeatable; multiple `--kind` values form an OR set — a node matches if its kind equals any of the listed values. Any kind value is accepted, including extension kinds introduced by domain profiles.

**Tag filter**

`--tag <tag>` restricts results to nodes whose `tags` array contains `<tag>`. The flag is repeatable with AND semantics — all listed tags must be present on the node.

**Attribute filter**

`--attr <name>=<value>` restricts results to nodes where the front-matter attribute `<name>` equals `<value>`. For scalar attributes the comparison is case-insensitive string equality. For array attributes the check is membership — `<value>` must be an element of the array. Applies to any attribute of any kind: `ref`, `status`, `maturity`, `class`, `category`, `published`, or any attribute introduced by a domain profile.

`--attr <name>~=<pattern>` restricts results to nodes where the front-matter attribute `<name>` matches the regexp `<pattern>`. For array attributes the pattern is tested against each element; the node matches if any element matches.

The `--attr` flag is repeatable with AND semantics — all specified attribute conditions must hold simultaneously.

**MCP filter object**

MCP tools that accept a filter receive it as a single JSON object parameter `filter`. The schema mirrors the CLI flags:

```json
{
  "kind":         ["source", "entity"],
  "tags":         ["cryptography", "protocols"],
  "attrs":        { "status": "backlog", "ref": "standard" },
  "attrPatterns": { "title": "TLS.*", "category": "independent" }
}
```

All fields are optional. `kind` is an array with OR semantics. `tags` is an array with AND semantics. `attrs` is a map of exact-match conditions, all ANDed. `attrPatterns` is a map of regexp-match conditions, all ANDed. An absent or empty `filter` object matches all nodes.

---

## Phase 1 — Bootstrap and Init

A new or cloned graph is ready to receive patches. Graph as git repository is initialized as part of the graph itself (CORE §11: "Git MUST be used").

- [x] `arc init [<dir>]` — initialize a new knowledge graph: create the canonical folder layout (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`); write stub files `_meta/predicates.md` and `_meta/aliases.md`; create `.arc/` for arc-managed state (see Graph Root); run `git init` and create `.gitkeep` for empty folders; write `.gitignore` excluding `.arc/`; stage everything and produce the initial commit `graph(init): empty knowledge graph` (CORE §11)
- [ ] `arc clone <url> [<dir>]` — clone an existing graph repository, build the local index (`arc index build`), and verify the working tree passes the Phase 6 lint before returning; report any pre-existing violations
- [ ] `arc pull` — fetch and merge remote changes (`git pull`), then run `arc index update` to bring the local index in sync; the standard workflow for contributors sharing a graph repository
- [ ] `arc push` — push local commits to the remote (`git push`); `arc` verifies the index is not stale before pushing so the remote reflects a consistent state

---

## Phase 2 — Patch Apply, Retract, and Reapply

The central operation: ingest a patch produced by the proprietary tool into the graph. Retraction is the clean inverse — it removes a document's entire contribution so a new patch for the same document can be applied from scratch.

- [x] `arc apply <patch.md>` — apply a patch file to the graph (CORE §12.3): parse the patch manifest (`kind: patch`, `document`, `published`, `stats`); check idempotency and skip with a clear message if `sources/<id>.md` is already tracked (CORE §11.2); for each H1/H2 node section reconstruct the node object (ARCNET-AST §4); **create** new node files when the basename does not exist; **merge** into existing files per the kind's declared merge operation — `none` for `source`, `union` for `entity`, `union first-writer` for `resource`, and per-profile operation for domain/extension kinds (CORE §10); derive and append timeline entries from the source's `published` date (CORE §9.4); produce exactly one git commit with the mandatory subject, stats, and `Source-Id:` trailer (CORE §11.3); update the local index cache (Phase 4) atomically within the same filesystem transaction
- [ ] `arc apply --dry-run <patch.md>` — parse and validate the patch; print the full diff of what would be created or merged; make no writes and no git operations
- [ ] `arc apply --batch <dir>` — apply every `*.md` patch in a directory in published-date order, skipping already-ingested sources; each patch is still exactly one commit
- [ ] `arc retract <source-id>` — remove a document's entire contribution from the graph: locate the original ingest commit via `git log --grep=Source-Id: <source-id>`; run `git revert <commit>` to undo it; git's three-way merge correctly un-merges shared nodes (CORE §11.2); produce a `graph(retract): <source-id>` commit; update the index cache
- [ ] `arc retract --dry-run <source-id>` — locate the ingest commit and print the full set of files that would be reverted; make no writes and no git operations
- [ ] `arc reapply <source-id> <new-patch.md>` — retract the existing ingestion of `<source-id>` and immediately apply the new patch in sequence; the two operations produce two commits (`graph(retract):` then `graph(ingest):`); intended for the case where a document's patch is regenerated with different extraction parameters and the old nodes must be fully replaced rather than merged
- [ ] `arc reapply --dry-run <source-id> <new-patch.md>` — show the retract diff followed by the apply diff; make no writes and no git operations

**Merge conflicts**

When the `union` merge encounters a divergent scalar field on an existing node (CORE §10 — first-writer wins, later divergent value flagged `needsReview`), `arc` writes the conflict directly into the node file using git-style conflict markers so it remains human-readable and diff-friendly:

```
definition: <<<<<<< rescorla-2026-tls13
A cryptographic protocol that establishes an authenticated channel.
=======
A network security protocol standardized in RFC 8446.
>>>>>>> chen-2026-pqkex
```

The markers record the two source-ids so the reader knows which patch introduced each value. The node file is staged with the conflict present; the ingest commit proceeds and records the unresolved state. Resolution is always a deliberate act:

- [ ] `arc conflicts` — list all node files that contain unresolved conflict markers; print `<kind>  <id>  <conflicted-fields>`
- [ ] `arc resolve <basename> --accept existing|incoming` — resolve all conflicted fields in the node by keeping one side; remove the conflict markers; commit as `graph(resolve): <basename>`; alternatively the user edits the file manually to craft a merged value, then runs `arc resolve <basename>` with no flag to commit the hand-edited result

---

## Phase 3 — Stats and Info

A user can inspect the graph and measure its state without navigating edge structure.

- [ ] `arc show <basename>` — render a node's front-matter and body to stdout in a readable form
- [ ] `arc list [<filter>]` — list matching nodes as `<kind>  <id>  <title>`, one per line; supports the full filter syntax (see Filtering)
- [ ] `arc timeline [<period>]` — display the timeline index; `<period>` narrows to `YYYY` or `YYYY-MM` (CORE §9.4)
- [ ] `arc since <date|commit>` — list nodes added or modified since the given ISO-8601 date or git commit hash; print `<kind>  <id>  <title>  <commit-date>`; gives an agent or user a temporal diff of the graph — "what did the graph learn since my last session?"
- [ ] `arc popular [<filter>]` — rank matching nodes by in-degree (number of incoming edges) descending; print `<rank>  <in-degree>  <kind>  <id>  <title>`; surfaces the most central concepts in the graph — the equivalent of a Wikipedia "most-linked-to" page list; accepts the full filter syntax (see Filtering) to rank within a kind or tag
- [ ] `arc orphans [<filter>]` — list matching nodes that have zero incoming edges (nothing in the graph links to them); these are stubs, isolated ingestions, or knowledge gaps; print `<kind>  <id>  <title>`; accepts the full filter syntax
- [ ] `arc citations <source-id> [--depth <n>]` — show the citation sub-network rooted at a source node: the sources it cites (via `cites::` edges), the sources that cite it (via `isCitedBy::` backlinks), and recursively to depth N (default 1); print as an indented tree; reveals which documents form a citation cluster and which resources are most authoritative in the corpus
- [ ] `arc log` — list all `graph(ingest):` commits in reverse chronological order as `<hash>  <date>  <source-id>  <title>`
- [ ] `arc history <basename>` — git log for a single node file, following renames (`git log --follow -- <path>`) (CORE §11.2)
- [ ] `arc locate <source-id>` — find the commit that ingested a source (`git log --grep=<id>`); print hash, date, and subject (CORE §11.2)
- [ ] `arc stats` — summary table: node count by kind, total edges, broken link count, source ingestion rate by year

---

## Phase 4 — Index and Cache

Fast graph navigation depends on never re-parsing every `.md` file on every command. A local index is built once and kept in sync, making all subsequent operations O(1) or O(log n) per query.

- [ ] `arc index build` — parse the entire graph, extract all node objects (ARCNET-AST §4), and write a compact binary index under `.arc/index` (see Graph Root) containing: id → file path mapping; backlink index (target-id → list of (source-id, predicate) pairs); kind index (kind → sorted id list); tag index (tag → id list); attribute index (attribute name+value → id list, covering all scalar and array front-matter fields); predicate index ((predicate, target) → id list); and the git HEAD commit hash for staleness detection
- [ ] `arc index status` — compare the stored HEAD hash against `git rev-parse HEAD`; print `up-to-date` or `stale (N commits behind)`
- [ ] `arc index update` — apply incremental changes since the stored HEAD: re-parse only the files modified in the intervening commits and update all index partitions
- [ ] automatic index freshness check — every `arc` command checks index staleness before executing; if stale it runs `arc index update` transparently and proceeds
- [ ] `arc index build --watch` — stay resident and rebuild incrementally on filesystem events (inotify / FSEvents); intended for long editing sessions

---

## Phase 5 — MCP Server

Expose the full graph navigation surface as a [Model Context Protocol](https://modelcontextprotocol.io) server so any MCP-compatible LLM client can consume the knowledge graph as a set of structured tools. Backed by the Phase 4 index; all tool responses are fast. The server is bidirectional: read tools let an agent navigate the graph, and the write tool lets an agent append to it by streaming a patch it constructed itself.

- [ ] `arc serve` — start an MCP server (stdio transport by default; `--http <port>` for SSE) exposing these tools:
  - `node_get(id)` → full node object (ARCNET-AST §4): attrs, text, edges, links
  - `node_list(filter?)` → array of `{id, kind, title}` for nodes matching the filter object (see Filtering — MCP filter object)
  - `node_grep(pattern, filter?)` → list of `{id, kind, line, snippet}` for nodes whose content matches a regexp pattern, optionally pre-filtered by the filter object
  - `node_edges(id)` → outgoing edges: `[{predicate, target}]` from `edges` + `links`
  - `node_backlinks(id)` → incoming edges: `[{source, predicate}]` from the backlink index
  - `graph_path(from, to)` → shortest directed edge path between two ids, or empty if none
  - `graph_stats()` → the same summary as `arc stats`
  - `timeline_get(period?)` → timeline entries for a year/month period
  - `context_retrieve(query, filter?, limit?)` → the primary RAG tool: runs the same three-pass retrieval as `arc context` — grep match, attribute match, neighbor expansion — and returns the top-N node objects with full content (attrs + text + edges + links); designed to let an agent build its working context in a single tool call without iterating through grep results and fetching each node separately; `limit` defaults to 10
  - `subgraph_get(id, depth?)` → return the fully-resolved subgraph rooted at `id` to `depth` hops (default 1): a flat array of complete node objects for the seed and every reachable neighbor; covers the same operation as `arc subgraph` for agent context expansion mid-conversation
  - `patch_apply(content)` → accept a complete patch document as a string (CORE §12 format), validate it, apply it to the graph, and return the resulting commit hash and stats; this is the write entry-point for LLM agents that generate a patch inline and stream it directly to `arc` without writing a file
  - `patch_validate(content)` → dry-run: parse the patch, run the Phase 6 lint rules against it, and return a list of violations without writing anything; intended for an agent to self-check a patch before committing it
- [ ] `arc serve --readonly` — disable `patch_apply` and `patch_validate`; expose only the navigation tools; safe for untrusted or read-only clients

---

## Phase 6 — Linting

A user can verify the graph against every conformance rule in the CORE spec.

- [x] `arc lint` — run the full CORE §14 checklist across every node and report violations with file path and line number for each: valid YAML front-matter and `kind` field; unique basenames (CORE §3.2); every `[[link]]` resolves to an existing basename; `source` citekey `id` equals its basename (CORE §6.2); `entity` four-word decoded Sowa `category` (CORE §9.2.1); derived nodes link back to at least one `source` (CORE §3.4); predicates are camelCase and registered in `_meta/predicates.md` (CORE §7.3); citations use a registered `cito:`-aligned predicate (CORE §8); each document is exactly one `graph(ingest):` commit (CORE §11.1); extension kind conformance per the kind's profile checklist
- [ ] `arc lint <basename>` — validate a single node in isolation; useful in a pre-commit hook
- [ ] `arc lint --fix` — auto-correct violations that have a safe deterministic fix (e.g. missing `id` field equal to basename, incorrect citekey casing); flag the rest
- [ ] yaml configurable lint rules for extensions

---

## Phase 7 — Repair and Re-linking

A knowledge graph grows patch by patch. When patch D1 was produced, entity E did not yet exist in the graph — so node A's text mentions E by name as plain prose but carries no `[[E]]` link and no `mentions::` edge. Later, patch D2 introduces E. The gap is invisible to the graph: A and E are structurally disconnected even though A clearly talks about E. Re-linking closes that gap retroactively, turning latent text co-occurrence into explicit graph edges. This is the primary connectivity multiplier of the `arc` tool.

**Inline re-linking (primary use case)**

- [ ] `arc relink` — scan every node's `text` and `notes` fields for plain-text occurrences of any entity basename or alias registered in `_meta/aliases.md`, then for each match: add `[[EntityBasename]]` markup at the point of occurrence in the prose (or append a `mentions:: [[EntityBasename]]` edge where the kind uses a headed `## Mentions` block); append the corresponding `mentionedIn:: [[SourceNode]]` backlink on the entity node; insert `[[Canonical|alias-text]]` when the match is an alias so the displayed text is preserved; skip nodes that already carry `[[EntityBasename]]` or a `mentions::` edge to that entity (idempotent); commit all changes as `graph(relink): add N inline mentions across M nodes`
- [ ] `arc relink --dry-run` — print every proposed match as `<node-id>  ·  "<matched text>"  →  [[EntityBasename]]`; make no writes
- [ ] `arc relink --interactive` — confirm or skip each proposed match individually before writing; useful when entity names are short and ambiguous
- [ ] `arc relink [<filter>]` — restrict the set of nodes scanned to those matching the filter (see Filtering); e.g. `--kind source` to link only source abstracts, or `--attr maturity=mature` to restrict to mature thoughts
- [ ] `arc relink --entity <basename>` — re-link a single newly-added entity across all existing nodes; intended to be run immediately after a patch that introduces a new entity

**Synonym merging**

- [ ] `arc relink merge` — handle the case where two distinct basenames represent the same concept (e.g. entity `TLS` and entity `Transport Layer Security` arrived in separate patches): build a synonym graph from `aliases` fields and `_meta/aliases.md`; detect connected components where two basenames collapse to the same canonical subject; rewrite every `[[Alias]]` link across the graph to `[[Canonical]]`; union the alias node's edges into the canonical node; update `_meta/aliases.md`; delete the duplicate file; commit as `graph(relink): merge N synonym entities into canonical forms`
- [ ] `arc relink merge --dry-run` — print proposed merges without writing

**Dead-edge repair**

- [ ] `arc repair` — for every dangling wikilink (link target has no `.md` file): create a minimal `resource` stub so the link resolves (default), or `--to <canonical>` to redirect the link, or `--delete` to remove the edge; commit as `graph(repair): resolve N dangling links`
- [ ] `arc repair --dry-run` — list all dangling links and their proposed resolution

---

## Phase 8 — Query and Navigation

Graph exploration backed by the Phase 4 index for structured queries, and direct file scanning for content search. The index handles graph topology; `grep` handles text.

**Content search**

- [x] `arc grep [<filter>] <pattern>` — scan nodes matching the filter (see Filtering) for lines matching the regexp `<pattern>`; print `<kind>  <id>  <line-number>  <matched line>`, one match per output line; without a filter, scans every node file; suitable for piping to standard tools

**Structured graph queries (index-backed)**

- [ ] `arc edges <basename>` — list all outgoing edges: `<predicate>  →  <target>`
- [ ] `arc backlinks <basename>` — list all nodes with an incoming edge to this basename, grouped by predicate; backed by the backlink index (O(1) per node)
- [ ] `arc path <from> <to>` — breadth-first shortest directed path through the edge graph; print the step chain `<node> —<predicate>→ <node>`
- [ ] `arc neighbors <basename> [--depth <n>]` — expand the ego-graph to depth N; print all reachable nodes and the edges between them (default depth 1)

**Context assembly (RAG and agent memory)**

- [ ] `arc subgraph <basename> [--depth <n>] [<filter>]` — extract a self-contained subgraph: the seed node plus all nodes reachable within N hops (default 1), optionally filtered by kind or attributes on the reached nodes; the filter applies to the expanded nodes, not the seed; output uses the patch exchange format (CORE §12.2) as the serialization: nodes are grouped by kind under `# <Kind>` headings, each node under `## <basename>`, front-matter in a fenced YAML block, body verbatim below — human-readable, LLM-friendly, and round-trippable back into `arc apply`
- [ ] `arc context <query> [<filter>]` — multi-signal retrieval that returns a ranked set of nodes most relevant to `<query>`, serialized in the same patch exchange format as `arc subgraph` so the output is ready to inject into an LLM prompt; the optional filter (see Filtering) narrows the candidate set before retrieval; **note: the specific retrieval algorithm — how signals are combined and ranked — is not specified here and must be developed through experimentation; the interface is fixed but the implementation strategy is an open research question**

---

## Phase 9 — Export and Interoperability

A user can extract graph content in machine-readable or portable forms.

- [ ] `arc export json [<filter>]` — export nodes matching the filter (see Filtering) as the ARCNET-AST JSON model (ARCNET-AST §4–§6); without a filter, exports all nodes; suitable for downstream tooling and LLM context injection
- [ ] `arc export dot [<filter>]` — emit a GraphViz DOT file of the directed edge graph for nodes matching the filter, nodes labelled by basename, edges labelled by predicate
- [ ] `arc export patch <source-id>` — re-serialize an ingested source and its derived nodes back into the patch exchange format (CORE §12); useful for sharing a subgraph

---

## Open Issues

The following gaps are known and deferred; they do not block earlier phases but must be resolved before `arc` can be considered complete.

- **Machine-readable output (#3)** — All commands produce human-readable text. A `--json` output flag is needed on `arc list`, `arc stats`, `arc edges`, `arc backlinks`, `arc popular`, and `arc orphans` to make `arc` composable in scripts and pipelines without relying solely on MCP.
- **Domain profile registration (#4)** — `arc lint` references "extension kind conformance per the kind's profile checklist" but there is no mechanism to tell `arc` which profiles are active or where to load their rules. A profile registration scheme (e.g., `_meta/profiles.md` or entries in `.arc/config`) is required before lint can validate extension kinds.
- **Concurrency and locking (#8)** — The `.arc/` index is a shared mutable resource. Simultaneous `arc apply` invocations or a running `arc serve` alongside a CLI write command can corrupt it. A lockfile protocol and safe index-update semantics under concurrent access are unspecified.
- **Predicate and alias registry management (#9)** — There is no CLI for adding a new predicate to `_meta/predicates.md` or a new alias to `_meta/aliases.md`. Both files are currently edited by hand or updated implicitly by `arc relink merge`. Explicit `arc predicate add` and `arc alias add` commands are needed for safe, validated curation.
