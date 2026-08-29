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
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appconfig "github.com/fogfish/arcnet-cli/internal/app/config"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
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

// sessionInstructionsPurpose is the fixed opening sentence of
// sessionInstructions() (research.md D5) — the server's overall purpose and
// its recommended first call, independent of any one tool's own
// mcp.Tool.Description.
const sessionInstructionsPurpose = "This server exposes a knowledge graph read-only, over six tools: node_get, node_grep, subgraph_get, context_retrieve, schema, node_match. Call schema first, before any other tool."

// sessionInstructions composes the server's InitializeResult.Instructions
// string (research.md D5, spec FR-007/FR-008/FR-009) from the fixed purpose
// sentence plus every tool's own WorkflowNote constant, in tool-
// registration order — replacing the single hand-authored
// schemaAdvertisement paragraph so workflow/tool-preference guidance stays
// colocated with the tool file it describes.
func sessionInstructions() string {
	return strings.Join([]string{
		sessionInstructionsPurpose,
		schemaWorkflowNote,
		nodeGetWorkflowNote,
		nodeGrepWorkflowNote,
		subgraphGetWorkflowNote,
		contextRetrieveWorkflowNote,
		nodeMatchWorkflowNote,
	}, " ")
}

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
	Source           stringOrArray `json:"source,omitempty" jsonschema:"exact source node id(s) the triple must have — single string or array of strings (OR-of-values); omit for any source"`
	SourcePattern    stringOrArray `json:"sourcePattern,omitempty" jsonschema:"regexp(s) the triple's source node id must match — single string or array of strings (OR-of-patterns); omit for any source"`
	Predicate        stringOrArray `json:"predicate,omitempty" jsonschema:"exact predicate name(s) the triple must have — single string or array of strings; omit for any predicate"`
	PredicatePattern stringOrArray `json:"predicatePattern,omitempty" jsonschema:"regexp(s) the triple's predicate name must match"`
	Target           stringOrArray `json:"target,omitempty" jsonschema:"exact target value(s)/node id(s) the triple must have — single string or array of strings"`
	TargetPattern    stringOrArray `json:"targetPattern,omitempty" jsonschema:"regexp(s) the triple's target value must match"`
}

// mcpFilter is node_grep's/subgraph_get's/context_retrieve's optional
// filter argument's wire shape (research.md D7, contracts/mcp-contract.md)
// — a JSON-native list of triple statements, kept private to this file
// since the two drivers' native input shapes (CLI flags vs. MCP JSON) do
// not share a common decoding path.
type mcpFilter struct {
	Statements []mcpStatement `json:"statements,omitempty" jsonschema:"list of triple constraints, ANDed together; an absent or empty list matches every node. A statement naming only predicate scopes neighbor traversal to that relation instead of narrowing results (subgraph_get/context_retrieve only) — see each tool's description"`
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

// must panics on a non-nil error — used at package-load time to build each
// tool's InputSchema from a static struct tag, where an error can only be a
// programming mistake caught on the first `go build`/test run, never a
// runtime condition (research.md D2).
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// withExamples attaches example values to field's generated schema property
// (research.md D3) — the `jsonschema` struct tag has no syntax for
// `examples`, so this mutates the reflected *jsonschema.Schema's per-property
// Examples slice, the same post-processing pattern stringOrArraySchema()
// already uses for type substitution. Panics if field is not a property of
// s (a maintainer-visible programming error, never a runtime condition).
func withExamples(s *jsonschema.Schema, field string, examples ...any) *jsonschema.Schema {
	prop, ok := s.Properties[field]
	if !ok {
		panic(fmt.Sprintf("withExamples: no property %q on schema", field))
	}
	prop.Examples = append(prop.Examples, examples...)
	return s
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

	server := mcp.NewServer(&mcp.Implementation{Name: serveImplName, Version: serveImplVersion}, &mcp.ServerOptions{Instructions: sessionInstructions()})

	mcp.AddTool(server, nodeGetTool, nodeGetHandler(dir, index))
	mcp.AddTool(server, nodeGrepTool, nodeGrepHandler(dir, cfgFile.Grep))
	mcp.AddTool(server, subgraphGetTool, subgraphGetHandler(dir, subgraphCfg, index))
	mcp.AddTool(server, contextRetrieveTool, contextRetrieveHandler(dir, cfgFile.Grep, subgraphCfg, index))
	mcp.AddTool(server, schemaTool, schemaHandler(dir, index))
	mcp.AddTool(server, nodeMatchTool, nodeMatchHandler(dir))

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
