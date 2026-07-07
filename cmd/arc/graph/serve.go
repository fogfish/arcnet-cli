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
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

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

// mcpFilter is node_grep's optional filter argument's wire shape (research.md
// D4, data-model.md in specs/008-arc-serve-mcp) — a JSON-native counterpart
// to grep.go's own optsFilter, kept private to this file since the two
// drivers' native input shapes (CLI flags vs. MCP JSON) do not share a
// common decoding path.
type mcpFilter struct {
	Kind         []string          `json:"kind,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Attrs        map[string]string `json:"attrs,omitempty"`
	AttrPatterns map[string]string `json:"attrPatterns,omitempty"`
}

// toCoreFilter converts f into a core.Filter, compiling every AttrPatterns
// value via regexp.Compile and returning service.ErrInvalidFilterPattern on
// the first invalid one (research.md D4). A nil f converts to a zero-value
// core.Filter{} (matches every node).
func (f *mcpFilter) toCoreFilter() (core.Filter, error) {
	if f == nil {
		return core.Filter{}, nil
	}

	out := core.Filter{Tags: f.Tags, Kinds: append([]string(nil), f.Kind...)}
	if len(f.Attrs) > 0 {
		out.Attrs = f.Attrs
	}
	for name, pattern := range f.AttrPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return core.Filter{}, service.ErrInvalidFilterPattern.With(err, pattern)
		}
		if out.AttrPatterns == nil {
			out.AttrPatterns = map[string]*regexp.Regexp{}
		}
		out.AttrPatterns[name] = re
	}
	return out, nil
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
	b.WriteString("| id | kind | line | snippet |\n|---|---|---|---|\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", m.ID, m.Type, m.Line, m.Text)
	}
	return b.String()
}

// nodeGetArgs is node_get's input schema.
type nodeGetArgs struct {
	ID string `json:"id" jsonschema:"the node's basename"`
}

// nodeGetHandler fetches one node by id and renders it exactly as
// core.RenderNode already serializes it on disk (research.md D2).
func nodeGetHandler(dir string) func(context.Context, *mcp.CallToolRequest, nodeGetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args nodeGetArgs) (*mcp.CallToolResult, any, error) {
		node, err := appgraph.NodeGet(ctx, fsys.Local{}, dir, args.ID)
		logCall("node_get", fmt.Sprintf("id=%q", args.ID), err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderNode(node)
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

// subgraphGetArgs is subgraph_get's input schema.
type subgraphGetArgs struct {
	ID    string `json:"id" jsonschema:"seed node basename"`
	Depth *int   `json:"depth,omitempty" jsonschema:"number of hops to traverse from the seed, default 1"`
}

// subgraphGetHandler extracts the seed plus every node reachable within
// depth hops and renders the result as one patch-exchange document, byte-
// identical to arc subgraph's own stdout for the same seed/depth.
func subgraphGetHandler(dir string, cfg configkernel.SubgraphConfig) func(context.Context, *mcp.CallToolRequest, subgraphGetArgs) (*mcp.CallToolResult, any, error) {
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

		result, err := appgraph.Subgraph(ctx, fsys.Local{}, core.Filter{}, args.ID, depth, cfg, dir, false)
		logCall("subgraph_get", logArgs, err)
		if err != nil {
			return nil, nil, err
		}

		text, err := core.RenderPatch(result.Patch)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	}
}

// buildServer mounts dir, preflights EnsureGraph (spec FR-004), loads
// .arc/config.yml once, and registers node_get/node_grep/subgraph_get on a
// new mcp.Server — the same construction RunE runs before selecting a
// transport, factored out so tests can exercise the real, registered tool
// handlers directly over mcp.NewInMemoryTransports() (research.md D7).
func buildServer(ctx context.Context, dir string) (*mcp.Server, error) {
	if err := appgraph.EnsureGraph(ctx, fsys.Local{}, dir); err != nil {
		return nil, err
	}

	store, err := (fsys.Local{}).Mount(dir)
	if err != nil {
		return nil, err
	}

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

	server := mcp.NewServer(&mcp.Implementation{Name: serveImplName, Version: serveImplVersion}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "node_get",
		Description: "Fetch a node's full content by id.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, nodeGetHandler(dir))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "node_grep",
		Description: "Search node content for lines matching a regexp pattern, optionally narrowed by a filter.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, nodeGrepHandler(dir, cfgFile.Grep))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "subgraph_get",
		Description: "Return the fully-resolved subgraph rooted at a node, to a given hop depth.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, subgraphGetHandler(dir, subgraphCfg))

	return server, nil
}

// NewServeCmd builds the `arc serve` command.
func NewServeCmd() *cobra.Command {
	var httpAddr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run an MCP server exposing the graph to LLM clients.",
		Long: `
arc serve starts a Model Context Protocol (MCP) server exposing exactly
three read-only tools — node_get, node_grep, subgraph_get — backed by the
same use-case functions arc grep/arc subgraph already call. It serves over
stdio by default, or over Streamable HTTP/SSE when --http <addr> is given.
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
