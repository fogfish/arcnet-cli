# MCP Contract: Tool Metadata (this feature)

This is the wire contract a connecting MCP client observes after this
feature ships — `mcp.Tool.Name`/`.Description`/`.InputSchema` for all six
tools, plus the server's `Instructions` string. Tool *behavior* (what each
tool actually does, what data it reads, what it returns) is unchanged from
today; only the metadata describing that behavior is added to or improved.
This file is the source of truth `tasks.md`/implementation drafts the exact
description/example strings from — it is the "draft all necessary
descriptions" deliverable the plan produces.

## Shared: `filter` argument field-level documentation (research.md D4)

Added once, on `mcpStatement` in `serve.go` (currently undocumented):

| Field | `jsonschema` description |
|---|---|
| `source` | exact source node id(s) the triple must have — single string or array of strings (OR-of-values); omit for any source |
| `sourcePattern` | regexp(s) the triple's source node id must match — single string or array of strings (OR-of-patterns); omit for any source |
| `predicate` | exact predicate name(s) the triple must have — single string or array of strings; omit for any predicate |
| `predicatePattern` | regexp(s) the triple's predicate name must match |
| `target` | exact target value(s)/node id(s) the triple must have — single string or array of strings |
| `targetPattern` | regexp(s) the triple's target value must match |

And on `mcpFilter` itself: `statements` — "list of triple constraints, ANDed together; an absent or empty list matches every node. A statement naming only `predicate` scopes neighbor traversal to that relation instead of narrowing results (subgraph_get/context_retrieve only) — see each tool's description."

## `node_get`

**File**: `serve_tool_node_get.go`

**Description** (was: `"Fetch a node's full content by id."`):
> Fetch one node's full stored content by its id. Returns the node exactly as stored: its front-matter attributes and body, including its outgoing relations, rendered as markdown.

**Parameters**:

| Field | Tag | Example(s) |
|---|---|---|
| `id` | `jsonschema:"the node's basename (filename without extension), e.g. the value returned as \"id\" by node_grep/node_match"` | `"tls-1-3"` |

**Workflow note** (`nodeGetWorkflowNote`): "Use node_get once you already have a specific node id — from node_grep, node_match, subgraph_get, or a prior context_retrieve result — and need that one node's complete content."

## `node_grep`

**File**: `serve_tool_node_grep.go`

**Description** (was: `"Search node content for lines matching a regexp pattern, optionally narrowed by a filter.statements triple filter (see schema)."`):
> Search every node's content for lines matching a regexp pattern, optionally narrowed to a subset of nodes by `filter.statements`. Returns a markdown table, one row per matching line: `id`, `type`, `line` (1-based line number), `snippet` (the matching line's text).

**Parameters**:

| Field | Tag | Example(s) |
|---|---|---|
| `pattern` | `jsonschema:"regexp pattern to search node content for"` | `"TODO"`, `"func\\s+\\w+\\("` |
| `filter` | `jsonschema:"optional filter narrowing which nodes are scanned; omit to scan every node"` | see below |

**Filter examples** (`withExamples(s, "filter", ...)`):
1. Exact-value narrowing: `{"statements":[{"predicate":"type","target":"Source"}]}` — only scan nodes of type `Source`.
2. Pattern narrowing: `{"statements":[{"predicate":"title","targetPattern":"^TLS"}]}` — only scan nodes whose `title` starts with "TLS".

**Workflow note** (`nodeGrepWorkflowNote`): "Prefer node_grep when you need the exact matching line(s) of a node's raw content (e.g. to quote a passage); prefer context_retrieve when you want whole, ranked node objects instead of individual line matches."

## `subgraph_get`

**File**: `serve_tool_subgraph_get.go`

**Description** (was: `"Return the fully-resolved subgraph rooted at a node, to a given hop depth, optionally scoped/narrowed by a filter.statements triple filter (see schema) — a predicate-only statement restricts which relations traversal follows."`): kept, append output sentence.
> ...Returns a patch-exchange document: the seed node plus every node reached within `depth` hops, each rendered with its own attributes and relations, in the same format `arc subgraph` produces on the command line.

**Parameters**:

| Field | Tag | Example(s) |
|---|---|---|
| `id` | `jsonschema:"seed node basename to expand outward from"` | `"tls-1-3"` |
| `depth` | `jsonschema:"number of hops to traverse from the seed, default 1"` | `1`, `2` |
| `filter` | `jsonschema:"optional filter; a predicate-only statement scopes which relations traversal follows, a statement also naming source/target narrows which reached nodes are kept in the result"` | see below |

**Filter examples**:
1. Predicate-only (scopes traversal): `{"statements":[{"predicate":"cites"}]}` — only follow `cites` edges outward/inward.
2. Narrowing (also names target): `{"statements":[{"predicate":"type","target":"Source"}]}` — traverse normally, keep only `Source`-typed nodes in the result.

**Workflow note** (`subgraphGetWorkflowNote`): "Prefer subgraph_get when you already know the seed node and want its resolved neighborhood as one connected document; prefer context_retrieve when you don't yet know a good seed and want to start from a free-text query instead."

## `context_retrieve`

**File**: `serve_tool_context_retrieve.go`

**Description** (was: `"Assemble the full content of every node relevant to a free-text query in one call — content match, attribute match, and neighbor expansion combined, ranked, deduplicated, and truncated to limit; optionally scoped/narrowed by a filter.statements triple filter (see schema)."`): kept, append output sentence.
> ...Returns a patch-exchange document (same shape as subgraph_get) containing the ranked, deduplicated, limit-truncated set of relevant nodes.

**Parameters**:

| Field | Tag | Example(s) |
|---|---|---|
| `query` | `jsonschema:"free text, matched literally and case-insensitively against node content and attributes"` | `"post-quantum key exchange"` |
| `filter` | `jsonschema:"optional filter narrowing every retrieval pass (content match, attribute match, and neighbor expansion all respect it)"` | see below |
| `limit` | `jsonschema:"maximum number of node objects to return, default 10"` | `10`, `25` |

**Filter examples**:
1. `{"statements":[{"predicate":"type","target":"Source"}]}` — only ever consider `Source`-typed nodes across all three retrieval passes.
2. `{"statements":[{"predicate":"tags","target":["cryptography","protocols"]}]}` — OR-of-values: nodes tagged either `cryptography` or `protocols`.

**Workflow note** (`contextRetrieveWorkflowNote`): "Prefer context_retrieve as the default way to gather everything relevant to a topic in one call; fall back to node_grep when you specifically need line-level text matches, or to subgraph_get when you already have a seed id and want its exact neighborhood rather than a ranked, query-driven set."

## `schema`

**File**: `serve_tool_schema.go`

**Description** (unchanged text, already states output; keep as-is): `"Return the graph's full ontology — every currently defined class and predicate, with descriptions — so a client knows what vocabulary is available before reading or writing the graph. Recommended as the first tool call of a session."`

**Parameters**: none (`schemaArgs{}`).

**Workflow note** (`schemaWorkflowNote`): "Always call schema first in a new session — every other tool's filter.statements predicate names and node `type` values are only meaningful once you know the graph's actual vocabulary."

## `node_match`

**File**: `serve_tool_node_match.go`

**Description** (was: `"List every distinct fact ({id, property, value}) on nodes that fully satisfy a required filter.statements triple filter (see schema) — evidence of why each node matched, not the node's full content."`): kept, append output sentence.
> ...Returns a markdown table, one row per distinct fact: `id`, `property`, `value`.

**Parameters**:

| Field | Tag | Example(s) |
|---|---|---|
| `filter` | `jsonschema:"required filter; at least one statement — an empty filter is rejected since node_match's output is evidence of *why* nodes matched, and every node vacuously matches an empty filter"` | see below |

**Filter examples** (both non-trivial, since empty is rejected):
1. `{"statements":[{"predicate":"type","target":"Source"}]}`
2. `{"statements":[{"predicate":"title","targetPattern":"^TLS"}]}`

**Workflow note** (`nodeMatchWorkflowNote`): "Use node_match instead of node_grep/context_retrieve when you need to know *which facts* justified a match (for citation or explanation), not the node's content itself."

## Server-level `Instructions` (research.md D5)

`sessionInstructions()` composes, in tool-registration order:

1. Fixed purpose sentence: "This server exposes a knowledge graph read-only, over six tools: node_get, node_grep, subgraph_get, context_retrieve, schema, node_match. Call schema first, before any other tool."
2. `schemaWorkflowNote`
3. `nodeGetWorkflowNote`
4. `nodeGrepWorkflowNote`
5. `subgraphGetWorkflowNote`
6. `contextRetrieveWorkflowNote`
7. `nodeMatchWorkflowNote`

Joined with a single space into one string, assigned to `mcp.ServerOptions.Instructions` — client-visible content is a superset of today's `schemaAdvertisement`, now traceable one sentence per tool file.
