//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package lint

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

type humanRulesPrinter struct{}

// Show lists every rule's name and description, one per line, in
// kernel.RuleDefinitions' fixed declaration order (FR-012).
func (humanRulesPrinter) Show(defs []kernel.RuleDefinition) ([]byte, error) {
	var buf []byte
	for _, def := range defs {
		buf = append(buf, fmt.Sprintf("%s — %s\n", def.Rule, def.Description)...)
	}
	return buf, nil
}

var rulesRenderer = bios.Registry[[]kernel.RuleDefinition]{Human: humanRulesPrinter{}}

// NewLintRulesCmd builds the `arc lint rules` command.
func NewLintRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "List every conformance rule arc lint implements.",
		Long: `
arc lint rules lists every rule arc lint checks, each with its exact
--skip-compatible name and a human-readable description. This is reference
data: it requires no initialized graph and performs no filesystem walk.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc lint rules
	arc lint rules --json`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := rulesRenderer.Resolve(bios.ResolveMode())
			out, err := printer.Show(kernel.RuleDefinitions)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(os.Stdout, string(out)); err != nil {
				return err
			}
			return nil
		},
	}
}
