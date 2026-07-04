//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"testing"

	"github.com/fogfish/it/v2"
)

const locateFixture = `---
kind: entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.

## Mentions
- mentions:: [[foo-2026-x]]
- relatesTo:: [[Other Node]]
`

func TestLocateFrontMatterDelimiter(t *testing.T) {
	line := locateFrontMatterDelimiter([]byte(locateFixture))
	it.Then(t).Should(it.Equal(1, line))
}

func TestLocateFrontMatterDelimiterAbsentFallsBackToLine1(t *testing.T) {
	line := locateFrontMatterDelimiter([]byte("no front matter here\n"))
	it.Then(t).Should(it.Equal(1, line))
}

func TestLocateFrontMatterField(t *testing.T) {
	line := locateFrontMatterField([]byte(locateFixture), "category")
	it.Then(t).Should(it.Equal(4, line))
}

func TestLocateFrontMatterFieldAbsentFallsBackToDelimiter(t *testing.T) {
	line := locateFrontMatterField([]byte(locateFixture), "id")
	it.Then(t).Should(it.Equal(1, line))
}

func TestLocateLinkTarget(t *testing.T) {
	line := locateLinkTarget([]byte(locateFixture), "Other Node")
	it.Then(t).Should(it.Equal(12, line))
}

func TestLocateLinkTargetNotFound(t *testing.T) {
	line := locateLinkTarget([]byte(locateFixture), "Nonexistent")
	it.Then(t).Should(it.Equal(0, line))
}

func TestLocatePredicateToken(t *testing.T) {
	line := locatePredicateToken([]byte(locateFixture), "relatesTo")
	it.Then(t).Should(it.Equal(12, line))
}

func TestLocateBlockLabelHeading(t *testing.T) {
	line := locateBlockLabel([]byte(locateFixture), "Mentions")
	it.Then(t).Should(it.Equal(10, line))
}

func TestLocateBlockLabelBold(t *testing.T) {
	raw := "---\nkind: entity\n---\n# X\n\n**Cites**\n- cites:: [[Y]]\n"
	line := locateBlockLabel([]byte(raw), "Cites")
	it.Then(t).Should(it.Equal(6, line))
}

func TestLocateConflictMarker(t *testing.T) {
	raw := "<<<<<<< HEAD\nkind: entity\n=======\nkind: entity\n>>>>>>> feature\n"
	line := locateConflictMarker([]byte(raw))
	it.Then(t).Should(it.Equal(1, line))
}

func TestLocateConflictMarkerAbsent(t *testing.T) {
	line := locateConflictMarker([]byte(locateFixture))
	it.Then(t).Should(it.Equal(0, line))
}
