//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package lint

import (
	"encoding/json"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// arc lint rules
// spec.md US2 Scenario 2.1: every rule arc lint implements is listed with
// its name and description.
func TestLintRulesListsEveryRuleWithDescription(t *testing.T) {
	out, err := sut(NewLintRulesCmd(), nil)

	it.Then(t).Should(it.Nil(err))
	for _, def := range kernel.RuleDefinitions {
		it.Then(t).
			Should(it.String(out).Contain(string(def.Rule))).
			Should(it.String(out).Contain(def.Description))
	}
}

// arc lint rules
// spec.md US2 Scenario 2.2: works with no graph present, in any directory.
func TestLintRulesWorksOutsideGraph(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewLintRulesCmd(), nil)

	it.Then(t).Should(it.Nil(err))
}

// arc lint rules / arc lint --skip
// spec.md US2 Scenario 2.3: every name arc lint rules lists is accepted by
// --skip's own validation — no "unknown rule" refusal.
func TestLintRulesNameAcceptedBySkip(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	for _, def := range kernel.RuleDefinitions {
		cmd := NewLintCmd()
		it.Then(t).Should(it.Nil(cmd.Flags().Set("skip", string(def.Rule))))

		_, err := sut(cmd, nil)

		if err != nil {
			it.Then(t).ShouldNot(it.String(err.Error()).Contain("unknown rule"))
		}
	}
}

// arc lint rules
// spec.md US2 Scenario 2.4: output is deterministic across repeated runs.
func TestLintRulesDeterministicOutput(t *testing.T) {
	first, err := sut(NewLintRulesCmd(), nil)
	it.Then(t).Should(it.Nil(err))

	second, err := sut(NewLintRulesCmd(), nil)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Equal(first, second))
}

// arc lint rules --json
// --json contract: a bare JSON array of RuleDefinition, matching
// kernel.RuleDefinitions' length and order.
func TestLintRulesJSONOutput(t *testing.T) {
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewLintRulesCmd(), nil)
	it.Then(t).Should(it.Nil(err))

	var defs []struct {
		Rule        string `json:"rule"`
		Description string `json:"description"`
	}
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &defs)))
	it.Then(t).Should(it.Equal(len(kernel.RuleDefinitions), len(defs)))

	for i, def := range kernel.RuleDefinitions {
		it.Then(t).
			Should(it.Equal(string(def.Rule), defs[i].Rule)).
			Should(it.Equal(def.Description, defs[i].Description))
	}
}
