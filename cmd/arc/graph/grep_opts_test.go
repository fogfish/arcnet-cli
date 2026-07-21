//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"errors"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
)

func TestOptsFilterBuildParsesExactAttrValue(t *testing.T) {
	opts := optsFilter{attr: []string{"status=mature"}}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("mature", f.Attrs["status"]))
}

func TestOptsFilterBuildParsesPatternAttrValue(t *testing.T) {
	opts := optsFilter{attr: []string{`title~=^TLS`}}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(f.AttrPatterns["title"].MatchString("TLS 1.3")))
}

func TestOptsFilterBuildRejectsMalformedAttrValue(t *testing.T) {
	opts := optsFilter{attr: []string{"status"}}

	_, err := opts.build()

	it.Then(t).Should(it.True(errors.Is(err, service.ErrInvalidAttrFlag)))
}

func TestOptsFilterBuildComposesTypeTagAttr(t *testing.T) {
	opts := optsFilter{
		typ:  []string{"Entity", "Source"},
		tag:  []string{"cryptography"},
		attr: []string{"status=mature"},
	}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(2, len(f.Types))).
		Should(it.Seq(f.Types).Equal("Entity", "Source")).
		Should(it.Equal(1, len(f.Tags))).
		Should(it.Equal("mature", f.Attrs["status"]))
}
