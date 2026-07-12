//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package bios

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fogfish/faults"
	"github.com/mattn/go-isatty"
)

// ErrConfirmRefused is Confirm's error when os.Stdin is not a terminal —
// constitution Principle IX's destructive-operation gate refuses rather
// than hanging or silently proceeding when confirmation cannot actually be
// obtained interactively (research.md D10). The caller decides whether to
// call Confirm at all, based on its own --force/-f flag.
const ErrConfirmRefused = faults.Type("confirmation required but stdin is not a terminal — rerun with --force")

// errNoCause is passed to a faults.Type.With for guard conditions that are
// not caused by an underlying Go error, mirroring internal/core's own
// precedent (internal/core/markdown.go's errNoCause).
var errNoCause = errors.New("")

// Confirm prompts prompt + " [y/N] " on os.Stderr and reads one line from
// os.Stdin, reporting true only for an explicit y/yes answer
// (case-insensitive) — the first destructive-operation confirmation gate
// in this codebase (research.md D10, ADR 002's own CLIG checklist item,
// Constitution Principle IX). TTY-gated: when os.Stdin is not a terminal,
// it refuses immediately via ErrConfirmRefused instead of hanging or
// silently proceeding.
func Confirm(prompt string) (bool, error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return false, ErrConfirmRefused.With(errNoCause)
	}

	fmt.Fprint(os.Stderr, prompt+" [y/N] ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false, nil
	}

	return parseConfirmAnswer(scanner.Text()), nil
}

// parseConfirmAnswer reports whether line is an explicit y/yes answer
// (case-insensitive, surrounding whitespace ignored) — split out from
// Confirm so this parsing rule is unit-testable without a real terminal.
func parseConfirmAnswer(line string) bool {
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
