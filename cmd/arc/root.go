package main

import (
	"github.com/spf13/cobra"
)

var version = "dev"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arc",
		Short: "arc is the command-line interface for the Arcnet knowledge graph.",
		Long: `
arc is the command-line interface for the Arcnet knowledge graph.

It is currently an empty skeleton: no subcommands exist yet, and running it
with no arguments prints this help text.

See more info https://github.com/fogfish/arcnet-cli
Report issues at https://github.com/fogfish/arcnet-cli/issues`,
		Example: `
	arc --help
	arc --version`,
		Version: version,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	return cmd
}
