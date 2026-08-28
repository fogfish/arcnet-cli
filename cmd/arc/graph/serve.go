//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appconfig "github.com/fogfish/arcnet-cli/internal/app/config"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// serveImplName/serveImplVersion identify this MCP server to a connecting
// client (mcp.Implementation) — not the same version string as the Cobra
// root command's own --version, since an MCP client cares about the server
// implementation, not the binary distribution.
const (
	serveImplName    = "arc"
	serveImplVersion = "0.1.0"
)

// schemaAdvertisement is sent to every connecting client as
// InitializeResult.Instructions (research.md D3, spec FR-008/FR-009) — the
// session-level guidance naming schema and recommending it as the first
// call, independent of schema's own mcp.Tool.Description. research.md D9:
// also names predicate-scoped filtering/traversal, via the triple
// filter.statements shape, as available.
const schemaAdvertisement = `This server exposes a knowledge graph. Call the schema tool first, before any other tool — it returns the graph's full ontology (every predicate and class, with descriptions), which makes every subsequent node_get/node_grep/subgraph_get/context_retrieve/node_match call more accurate. node_grep, subgraph_get, and context_retrieve each accept an optional filter.statements argument (triple source/predicate/target constraints) — a predicate-only statement scopes subgraph_get/context_retrieve's neighbor traversal to a named relation, while a statement also naming source/target narrows which nodes are returned. node_match also accepts a filter.statements argument, but — unlike the other three — it is required: node_match lists every distinct fact ({id, property, value}) that justified each match, never a node's full content.`

// stringOrArray decodes a JSON value that may be either a single string or
// an array of strings into a []string (research.md D7) — used by every
// mcpStatement field so a client can write either shape.
type stringOrArray []string

func (s *stringOrArray) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*s = stringOrArray{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

// mcpStatement is one triple statement in mcpFilter's wire shape
// (research.md D7, contracts/mcp-contract.md): source/predicate/target
// each accept a single string or an array of strings (OR-of-values), with
// an optional sibling *Pattern field for the regexp case, mirroring
// core.Matcher's Values/Patterns split.
type mcpStatement struct {
	Source           stringOrArray `json:"source,omitempty"`
	SourcePattern    stringOrArray `json:"sourcePattern,omitempty"`
	Predicate        stringOrArray `json:"predicate,omitempty"`
	PredicatePattern stringOrArray `json:"predicatePattern,omitempty"`
	Target           stringOrArray `json:"target,omitempty"`
	TargetPattern    stringOrArray `json:"targetPattern,omitempty"`
}

// mcpFilter is node_grep's/subgraph_get's/context_retrieve's optional
// filter argument's wire shape (research.md D7, contracts/mcp-contract.md)
// — a JSON-native list of triple statements, kept private to this file
// since the two drivers' native input shapes (CLI flags vs. MCP JSON) do
// not share a common decoding path.
type mcpFilter struct {
	Statements []mcpStatement `json:"statements,omitempty"`
}

// toMatcher compiles values/patterns into a core.Matcher, returning
// service.ErrInvalidFilterPattern on the first invalid regexp
// (research.md D7).
func toMatcher(values, patterns stringOrArray) (core.Matcher, error) {
	m := core.Matcher{Values: []string(values)}
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return core.Matcher{}, service.ErrInvalidFilterPattern.With(err, p)
		}
		m.Patterns = append(m.Patterns, re)
	}
	return m, nil
}

// toCoreFilter converts f into a core.Filter, compiling every *Pattern
// field via regexp.Compile and returning service.ErrInvalidFilterPattern on
// the first invalid one (research.md D7). A nil f converts to a zero-value
// core.Filter{} (matches every node).
func (f *mcpFilter) toCoreFilter() (core.Filter, error) {
	if f == nil {
		return core.Filter{}, nil
	}

	var out core.Filter
	for _, s := range f.Statements {
		source, err := toMatcher(s.Source, s.SourcePattern)
		if err != nil {
			return core.Filter{}, err
		}
		predicate, err := toMatcher(s.Predicate, s.PredicatePattern)
		if err != nil {
			return core.Filter{}, err
		}
		target, err := toMatcher(s.Target, s.TargetPattern)
		if err != nil {
			return core.Filter{}, err
		}
		out.Statements = append(out.Statements, core.Statement{Source: source, Predicate: predicate, Target: target})
	}
	return out, nil
}

// stringOrArraySchema is stringOrArray's JSON Schema: either a single
// string or an array of strings (research.md D7) — jsonschema.ForType's
// default struct-derived reflection would otherwise type a []string-backed
// field as "array" only, rejecting the single-string convenience form
// before toCoreFilter's custom decoding ever runs.
func stringOrArraySchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		AnyOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
		},
	}
}

// inputSchemaFor derives T's input schema via reflection, substituting
// stringOrArraySchema() for every stringOrArray-typed field so a tool's
// filter.statements argument accepts either wire shape (research.md D7).
func inputSchemaFor[T any]() (*jsonschema.Schema, error) {
	return jsonschema.ForType(reflect.TypeFor[T](), &jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[stringOrArray](): stringOrArraySchema(),
		},
	})
}

// resolveHTTPAddr resolves --http's address argument (research.md D5, spec
// FR-003): a bare port or ":port" (no host) resolves to loopback-only
// "127.0.0.1:<port>"; an explicit host is used exactly as given; a
// syntactically invalid address returns service.ErrHTTPAddr.
func resolveHTTPAddr(addr string) (string, error) {
	if port, err := strconv.Atoi(addr); err == nil {
		return net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), nil
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", service.ErrHTTPAddr.With(err, addr)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port), nil
}

// logCall writes one stderr line per MCP tool call, recording the tool name,
// its key arguments, and its outcome (research.md D9, spec FR-019).
func logCall(tool, args string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "serve: %s %s error: %s\n", tool, args, err.Error())
		return
	}
	fmt.Fprintf(os.Stderr, "serve: %s %s ok\n", tool, args)
}

// renderMatchTable renders matches as node_grep's markdown reply (research.md
// D2, contracts/mcp-contract.md): a fixed header, one row per match, header
// only when matches is empty (spec FR-009).
func renderMatchTable(matches []kernel.Match) string {
	var b strings.Builder
	b.WriteString("| id | type | line | snippet |\n|---|---|---|---|\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", m.ID, m.Type, m.Line, m.Text)
	}
	return b.String()
}

// renderFactTable renders node_match's markdown reply (specs/028-node-
// match-filter contracts/mcp-contract.md): a fixed header, one row per
// matching fact, header only when matches is empty (spec FR-007).
func renderFactTable(matches []kernel.MatchEntry) string {
	var b strings.Builder
	b.WriteString("| id | property | value |\n|---|---|---|\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", m.ID, m.Property, m.Value)
	}
	return b.String()
}

// nodeGetArgs is node_get's input schema.
type nodeGetArgs struct {
	ID string `json:"id" jsonschema:"the node's basename"`
}

// nodeGetHandler fetches one node by id and renders it exactly as
// core.RenderNode already serializes it on disk (research.md D2).
func nodeGetHandler(dir string, index core.Index) func(context.Context, *mcp.CallToolRequest, nodeGetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeGetArgs) (*mcp.CallToolResult, any, error) {
		node, err := appgraph.NodeGet(ctx, fsys.Local{}, dir, args.ID)
		logCall("node_get", fmt.Sprintf("id=%q", args.ID), err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderNode(node, index)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	}
}

// nodeGrepArgs is node_grep's input schema.
type nodeGrepArgs struct {
	Pattern string     `json:"pattern" jsonschema:"regexp pattern to search node content for"`
	Filter  *mcpFilter `json:"filter,omitempty" jsonschema:"optional filter narrowing which nodes are scanned"`
}

// nodeGrepHandler searches node content for pattern, narrowed by an optional
// filter, and renders one markdown table row per matching line.
func nodeGrepHandler(dir string, cfg configkernel.GrepConfig) func(context.Context, *mcp.CallToolRequest, nodeGrepArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeGrepArgs) (*mcp.CallToolResult, any, error) {
		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("node_grep", fmt.Sprintf("pattern=%q", args.Pattern), err)
			return nil, nil, err
		}

		result, err := appgraph.Grep(ctx, fsys.Local{}, filter, args.Pattern, cfg, dir)
		logCall("node_grep", fmt.Sprintf("pattern=%q", args.Pattern), err)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: renderMatchTable(result.Matches)}}}, nil, nil
	}
}

// subgraphGetArgs is subgraph_get's input schema. Filter is new
// (research.md D8) — subgraph_get had no filter argument before this
// feature, so both flat-inclusion narrowing and predicate-scoped traversal
// were unreachable from it.
type subgraphGetArgs struct {
	ID     string     `json:"id" jsonschema:"seed node basename"`
	Depth  *int       `json:"depth,omitempty" jsonschema:"number of hops to traverse from the seed, default 1"`
	Filter *mcpFilter `json:"filter,omitempty" jsonschema:"optional filter; a predicate-only statement scopes traversal, a source/target-naming statement narrows the result"`
}

// subgraphGetHandler extracts the seed plus every node reachable within
// depth hops and renders the result as one patch-exchange document, byte-
// identical to arc subgraph's own stdout for the same seed/depth.
func subgraphGetHandler(dir string, cfg configkernel.SubgraphConfig, index core.Index) func(context.Context, *mcp.CallToolRequest, subgraphGetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args subgraphGetArgs) (*mcp.CallToolResult, any, error) {
		depth := 1
		if args.Depth != nil {
			depth = *args.Depth
		}

		logArgs := fmt.Sprintf("id=%q depth=%d", args.ID, depth)
		if depth < 0 {
			err := service.ErrInvalidDepth.With(errNoCause, strconv.Itoa(depth))
			logCall("subgraph_get", logArgs, err)
			return nil, nil, err
		}

		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("subgraph_get", logArgs, err)
			return nil, nil, err
		}

		result, err := appgraph.Subgraph(ctx, fsys.Local{}, filter, args.ID, depth, cfg, dir, false)
		logCall("subgraph_get", logArgs, err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderPatch(result.Patch, index)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	}
}

// contextRetrieveArgs is context_retrieve's input schema.
type contextRetrieveArgs struct {
	Query  string     `json:"query" jsonschema:"free text, matched literally and case-insensitively"`
	Filter *mcpFilter `json:"filter,omitempty" jsonschema:"optional filter narrowing every retrieval pass"`
	Limit  *int       `json:"limit,omitempty" jsonschema:"maximum number of node objects to return, default 10"`
}

// contextRetrieveHandler runs the three-pass retrieval (content match,
// attribute match, neighbor expansion) and renders the ranked, truncated
// result as one patch-exchange document, byte-shape-identical to
// subgraph_get's own reply construction (contracts/mcp-contract.md).
// contextRetrieveArgs.Limit's nil-resolution mirrors subgraphGetArgs.Depth's
// existing pattern.
func contextRetrieveHandler(dir string, cfgGrep configkernel.GrepConfig, cfgSubgraph configkernel.SubgraphConfig, index core.Index) func(context.Context, *mcp.CallToolRequest, contextRetrieveArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args contextRetrieveArgs) (*mcp.CallToolResult, any, error) {
		limit := 10
		if args.Limit != nil {
			limit = *args.Limit
		}
		logArgs := fmt.Sprintf("query=%q filter=%t limit=%d", args.Query, args.Filter != nil, limit)

		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("context_retrieve", logArgs, err)
			return nil, nil, err
		}

		result, err := appgraph.ContextRetrieve(ctx, fsys.Local{}, filter, args.Query, limit, cfgGrep, cfgSubgraph, dir)
		logCall("context_retrieve", logArgs, err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderPatch(result.Patch, index)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	}
}

// nodeMatchArgs is node_match's input schema. Unlike node_grep/
// subgraph_get/context_retrieve, Filter is REQUIRED (non-pointer, no
// "omitempty") — spec FR-001/FR-005.
type nodeMatchArgs struct {
	Filter mcpFilter `json:"filter" jsonschema:"required filter; at least one statement"`
}

// nodeMatchHandler evaluates filter against every node's own facts and
// renders one markdown table row per distinct fact that satisfied at
// least one statement (contracts/mcp-contract.md). args.Filter is a
// concrete, always-non-nil value here, unlike the other tools' *mcpFilter
// — toCoreFilter is called on &args.Filter; the empty-statements check
// itself happens in service.Match, not here, so the validation lives in
// one place regardless of transport.
func nodeMatchHandler(dir string) func(context.Context, *mcp.CallToolRequest, nodeMatchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeMatchArgs) (*mcp.CallToolResult, any, error) {
		filter, err := args.Filter.toCoreFilter()
		if err != nil {
			logCall("node_match", "", err)
			return nil, nil, err
		}

		result, err := appgraph.Match(ctx, fsys.Local{}, filter, dir)
		logCall("node_match", "", err)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: renderFactTable(result.Matches)}}}, nil, nil
	}
}

// schemaArgs is schema's input schema — empty, since schema takes no
// arguments (spec FR-006).
type schemaArgs struct{}

// renderSchema renders index as schema's markdown reply (research.md D2,
// contracts/mcp-contract.md, data-model.md): a sorted-by-name Predicates
// bullet list (name + description only), followed by a sorted-by-name
// Classes section, one subsection per class with its description and its
// required/optional predicate names ("(none)" for an empty list). Pure
// function, no I/O — mirrors renderMatchTable's existing shape.
func renderSchema(index core.Index) string {
	var b strings.Builder

	b.WriteString("## Predicates\n\n")
	for _, name := range slices.Sorted(maps.Keys(index.Predicates)) {
		fmt.Fprintf(&b, "- **%s**: %s\n", name, index.Predicates[name].Description)
	}

	b.WriteString("\n## Classes\n\n")
	for _, name := range slices.Sorted(maps.Keys(index.Types)) {
		def := index.Types[name]
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", name, def.Description)
		fmt.Fprintf(&b, "Required: %s\n", joinOrNone(def.Required))
		fmt.Fprintf(&b, "Optional: %s\n\n", joinOrNone(def.Optional))
	}

	return b.String()
}

// joinOrNone joins names with ", ", or renders "(none)" for an empty list
// (research.md D2).
func joinOrNone(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

// schemaHandler renders the already-resolved index as schema's reply — the
// operation cannot fail once index is already in hand, so the returned
// error is always nil (kept only because mcp.AddTool's handler signature
// requires an error return). dir mirrors nodeGetHandler/subgraphGetHandler's
// existing factory shape (data-model.md) though schema makes no domain call
// of its own and never reads it.
func schemaHandler(dir string, index core.Index) func(context.Context, *mcp.CallToolRequest, schemaArgs) (*mcp.CallToolResult, any, error) {
	return func(context.Context, *mcp.CallToolRequest, schemaArgs) (*mcp.CallToolResult, any, error) {
		text := renderSchema(index)
		logCall("schema", "", nil)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
	}
}

// buildServer mounts dir, preflights EnsureGraph (spec FR-004), loads
// .arc/config.yml once, and registers node_get/node_grep/subgraph_get/
// context_retrieve/node_match on a new mcp.Server — the same construction
// RunE runs before selecting a transport, factored out so tests can
// exercise the real, registered tool handlers directly over
// mcp.NewInMemoryTransports() (research.md D7).
func buildServer(ctx context.Context, dir string) (*mcp.Server, error) {
	if err := appgraph.EnsureGraph(ctx, fsys.Local{}, dir); err != nil {
		return nil, err
	}

	store, err := (fsys.Local{}).Mount(dir)
	if err != nil {
		return nil, err
	}
	index := resolveIndexOrDefault(store)

	cfgFile, err := appconfig.Load(store)
	if err != nil {
		return nil, err
	}

	subgraphCfg := cfgFile.Subgraph
	if subgraphCfg.DirectCap <= 0 {
		subgraphCfg.DirectCap = defaultSubgraphDirectCap
	}
	if subgraphCfg.BacklinkCap <= 0 {
		subgraphCfg.BacklinkCap = defaultSubgraphBacklinkCap
	}

	server := mcp.NewServer(&mcp.Implementation{Name: serveImplName, Version: serveImplVersion}, &mcp.ServerOptions{Instructions: schemaAdvertisement})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "node_get",
		Description: "Fetch a node's full content by id.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, nodeGetHandler(dir, index))

	nodeGrepSchema, err := inputSchemaFor[nodeGrepArgs]()
	if err != nil {
		return nil, err
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "node_grep",
		Description: "Search node content for lines matching a regexp pattern, optionally narrowed by a filter.statements triple filter (see schema).",
		InputSchema: nodeGrepSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, nodeGrepHandler(dir, cfgFile.Grep))

	subgraphGetSchema, err := inputSchemaFor[subgraphGetArgs]()
	if err != nil {
		return nil, err
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "subgraph_get",
		Description: "Return the fully-resolved subgraph rooted at a node, to a given hop depth, optionally scoped/narrowed by a filter.statements triple filter (see schema) — a predicate-only statement restricts which relations traversal follows.",
		InputSchema: subgraphGetSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, subgraphGetHandler(dir, subgraphCfg, index))

	contextRetrieveSchema, err := inputSchemaFor[contextRetrieveArgs]()
	if err != nil {
		return nil, err
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "context_retrieve",
		Description: "Assemble the full content of every node relevant to a free-text query in one call — content match, attribute match, and neighbor expansion combined, ranked, deduplicated, and truncated to limit; optionally scoped/narrowed by a filter.statements triple filter (see schema).",
		InputSchema: contextRetrieveSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, contextRetrieveHandler(dir, cfgFile.Grep, subgraphCfg, index))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "schema",
		Description: "Return the graph's full ontology — every currently defined class and predicate, with descriptions — so a client knows what vocabulary is available before reading or writing the graph. Recommended as the first tool call of a session.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, schemaHandler(dir, index))

	nodeMatchSchema, err := inputSchemaFor[nodeMatchArgs]()
	if err != nil {
		return nil, err
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "node_match",
		Description: "List every distinct fact ({id, property, value}) on nodes that fully satisfy a required filter.statements triple filter (see schema) — evidence of why each node matched, not the node's full content.",
		InputSchema: nodeMatchSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, nodeMatchHandler(dir))

	return server, nil
}

// NewServeCmd builds the `arc serve` command.
func NewServeCmd() *cobra.Command {
	var httpAddr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run an MCP server exposing the graph to LLM clients.",
		Long: `
arc serve starts a Model Context Protocol (MCP) server exposing six
read-only tools — node_get, node_grep, subgraph_get, context_retrieve,
schema, node_match — the first three backed by the same use-case functions
arc grep/arc subgraph already call; context_retrieve (query + attribute
match plus one-hop neighbor expansion, ranked and truncated to a limit),
schema (the graph's full ontology: every class and predicate, with
descriptions), and node_match (every distinct {id, property, value} fact
justifying a match against a required filter.statements filter) are
MCP-only, with no Cobra command of their own. schema is the recommended
first call of a session — every connecting client is told so via the
server's own session-start guidance. It serves over stdio by default, or
over Streamable HTTP/SSE when --http <addr> is given.
A bare port or :port binds 127.0.0.1 only; an explicit host binds exactly
that host. serve is strictly read-only and never modifies the graph or its
git history.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc serve
	arc serve --http :8080
	arc serve --http 0.0.0.0:8080`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			dir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			server, err := buildServer(ctx, dir)
			if err != nil {
				return err
			}

			if httpAddr == "" {
				return server.Run(ctx, &mcp.StdioTransport{})
			}

			addr, err := resolveHTTPAddr(httpAddr)
			if err != nil {
				return err
			}

			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return service.ErrHTTPAddr.With(err, httpAddr)
			}

			httpServer := &http.Server{
				Handler: mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil),
			}

			go func() {
				<-ctx.Done()
				httpServer.Close()
			}()

			if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&httpAddr, "http", "", "Serve over Streamable HTTP/SSE at [host]:port instead of stdio (bare port/:port binds loopback only)")

	return cmd
}
