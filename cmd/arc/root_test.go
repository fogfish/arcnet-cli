//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"
)

// sut redirects os.Stdout, invokes the command's RunE directly with args,
// and returns the captured output alongside RunE's returned error.
func sut(cmd *cobra.Command, args []string) (string, error) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	err := cmd.RunE(cmd, args)

	w.Close()
	os.Stdout = stdout
	return <-ch, err
}

// run redirects os.Stdout and invokes cmd.Execute() with args set via
// SetArgs, exercising Cobra's own flag-parsing and --help/--version
// short-circuits, which never reach RunE.
func run(cmd *cobra.Command, args []string) (string, error) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmd.SetArgs(args)

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	err := cmd.Execute()

	w.Close()
	os.Stdout = stdout
	return <-ch, err
}

func TestRootNoArgsPrintsHelp(t *testing.T) {
	// Scenario from specs/001-cli-infrastructure/spec.md, US1: arc
	out, err := sut(newRootCmd(), []string{})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain("Usage:"))
}

func TestRootHelpFlag(t *testing.T) {
	// Scenario from specs/001-cli-infrastructure/spec.md, US1: arc --help
	out, err := run(newRootCmd(), []string{"--help"})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain("Usage:")).
		Should(it.String(out).Contain("arc"))
}

func TestRootVersionFlag(t *testing.T) {
	// Scenario from specs/001-cli-infrastructure/spec.md, US1: arc --version
	out, err := run(newRootCmd(), []string{"--version"})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain("arc")).
		Should(it.String(out).Contain(version))
}

func TestRootUnrecognizedFlag(t *testing.T) {
	// Scenario from specs/001-cli-infrastructure/spec.md, US1: arc --bogus
	out, err := run(newRootCmd(), []string{"--bogus"})

	it.Then(t).
		Should(it.Error(out, err).Contain("unknown flag"))
}

func TestRootUnrecognizedSubcommand(t *testing.T) {
	// Scenario from specs/001-cli-infrastructure/spec.md, US1: arc bogus
	out, err := run(newRootCmd(), []string{"bogus"})

	it.Then(t).
		Should(it.Error(out, err).Contain("unknown command"))
}
