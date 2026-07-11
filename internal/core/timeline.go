//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import (
	"fmt"
	"strings"
	"time"
)

// TimelinePeriods derives the yearly ("2026") and monthly ("2026-04")
// period codes a published date falls into (CORE §9.4).
func TimelinePeriods(published time.Time) (yearly, monthly string) {
	return published.Format("2006"), published.Format("2006-01")
}

// TimelineEntry renders one CORE §9.4 timeline bullet as a "cites"-tagged
// edge (BUG-002 — a timeline's own chronological reference to a source node
// it contains, reusing the citation predicate rather than a distinct
// "entries" name). The per-entry display annotation (title, authors, date)
// is derived here from the source's own data and never stored on the
// timeline node itself (AST §6.4) — it is discarded on re-parse, kept only
// for human readability.
func TimelineEntry(id, title string, authors []string, published time.Time) string {
	return fmt.Sprintf("- cites:: [[%s]] — *%s* (%s) — %s", id, title, strings.Join(authors, ", "), published.Format("2006-01-02"))
}
