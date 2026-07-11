//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestTimelinePeriods(t *testing.T) {
	published := time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC)

	yearly, monthly := core.TimelinePeriods(published)

	it.Then(t).
		Should(it.Equal("2026", yearly)).
		Should(it.Equal("2026-04", monthly))
}

func TestTimelineEntry(t *testing.T) {
	published := time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC)

	entry := core.TimelineEntry("rescorla-2026-tls13", "TLS 1.3: Design and Rationale", []string{"Eric Rescorla"}, published)

	it.Then(t).Should(it.Equal(
		"- cites:: [[rescorla-2026-tls13]] — *TLS 1.3: Design and Rationale* (Eric Rescorla) — 2026-04-12",
		entry,
	))
}
