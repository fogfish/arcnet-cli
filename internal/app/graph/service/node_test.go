//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
)

func TestNodeGetReturnsMatchingNodeFullContent(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"entities/TLS.md": entityNode("TLS", "SSL"),
	})

	node, err := service.NodeGet(context.Background(), mounter, "/graph", "TLS")

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal("TLS", node.ID))
}

func TestNodeGetUnknownIDReturnsErrSeedNotFound(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"entities/TLS.md": entityNode("TLS"),
	})

	_, err := service.NodeGet(context.Background(), mounter, "/graph", "No Such Node")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSeedNotFound)))
}

func TestNodeGetNotAGraphReturnsErrNotAGraphBeforeLookup(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	_, err := service.NodeGet(context.Background(), mounter, "/graph", "TLS")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestEnsureGraphValidGraphReturnsNil(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"entities/TLS.md": entityNode("TLS"),
	})

	err := service.EnsureGraph(context.Background(), mounter, "/graph")

	it.Then(t).Should(it.Nil(err))
}

func TestEnsureGraphNotAGraphReturnsErrNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	err := service.EnsureGraph(context.Background(), mounter, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}
