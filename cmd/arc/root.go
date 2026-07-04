//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/cmd/arc/ctrl"
	"github.com/fogfish/arcnet-cli/cmd/arc/graph"
	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

var version = "dev"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arc",
		Short: "arc is the command-line interface for the Arcnet knowledge graph.",
		Long: `
arc is the command-line interface for the Arcnet knowledge graph.

See more info https://github.com/fogfish/arcnet-cli
Report issues at https://github.com/fogfish/arcnet-cli/issues`,
		Example: `
	arc --help
	arc --version
	arc init`,
		Version: version,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			bios.SelectSchema(bios.Color, os.Stdout)
		},
	}

	flags := cmd.PersistentFlags()
	flags.BoolVarP(&bios.Quiet, "quiet", "q", false, "Suppress progress output; errors still shown")
	flags.BoolVarP(&bios.Verbose, "verbose", "v", false, "Show additional diagnostic detail")
	flags.BoolVar(&bios.JSON, "json", false, "Machine-readable structured output")
	flags.BoolVarP(&bios.Color, "color", "C", false, "Force-enable color (auto-detected otherwise)")

	cmd.AddCommand(ctrl.NewInitCmd())
	cmd.AddCommand(graph.NewApplyCmd())
	cmd.AddCommand(lint.NewLintCmd())

	return cmd
}
