//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

// locateFirstLine returns the 1-based line number of the first line in raw
// for which pred reports true, or 0 when no line matches.
func locateFirstLine(raw []byte, pred func(line string) bool) int {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	line := 0
	for scanner.Scan() {
		line++
		if pred(scanner.Text()) {
			return line
		}
	}
	return 0
}

// locateFrontMatterDelimiter returns the line of the first "---" delimiter,
// or line 1 when none is found (research.md D3).
func locateFrontMatterDelimiter(raw []byte) int {
	line := locateFirstLine(raw, func(l string) bool {
		return strings.TrimSpace(l) == "---"
	})
	if line == 0 {
		return 1
	}
	return line
}

// locateFrontMatterField returns the line of a "key: value" front-matter
// entry, falling back to the front-matter delimiter line when key is not
// found (research.md D3).
func locateFrontMatterField(raw []byte, key string) int {
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `\s*:`)
	line := locateFirstLine(raw, func(l string) bool {
		return pattern.MatchString(strings.TrimSpace(l))
	})
	if line == 0 {
		return locateFrontMatterDelimiter(raw)
	}
	return line
}

// locateLinkTarget returns the first line containing the literal "[[target"
// occurrence — matches both a bare wikilink and a predicate-qualified
// inline form (research.md D3), or 0 when not found.
func locateLinkTarget(raw []byte, target string) int {
	needle := "[[" + target
	return locateFirstLine(raw, func(l string) bool {
		return strings.Contains(l, needle)
	})
}

// locatePredicateToken returns the first line containing "predicate::", or
// 0 when not found (research.md D3).
func locatePredicateToken(raw []byte, predicate string) int {
	needle := predicate + "::"
	return locateFirstLine(raw, func(l string) bool {
		return strings.Contains(l, needle)
	})
}

// locateBlockLabel returns the first line matching either a "## Title"
// heading or a "**Title**" bold-label paragraph (research.md D3), or 0 when
// not found.
func locateBlockLabel(raw []byte, title string) int {
	heading := "## " + title
	bold := "**" + title + "**"
	return locateFirstLine(raw, func(l string) bool {
		t := strings.TrimSpace(l)
		return t == heading || t == bold
	})
}

var (
	conflictStartPattern = regexp.MustCompile(`^<{7}`)
	conflictMidPattern   = regexp.MustCompile(`^={7}$`)
	conflictEndPattern   = regexp.MustCompile(`^>{7}`)
)

// locateConflictMarker returns the first line beginning an unresolved git
// merge-conflict marker (research.md D13), or 0 when none is found.
func locateConflictMarker(raw []byte) int {
	return locateFirstLine(raw, func(l string) bool {
		return conflictStartPattern.MatchString(l) || conflictMidPattern.MatchString(l) || conflictEndPattern.MatchString(l)
	})
}
