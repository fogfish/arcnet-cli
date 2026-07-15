//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package http_test

import (
	"context"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/http"
)

func TestFetchSuccess(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	client := http.New(5 * time.Second)
	body, err := client.Fetch(context.Background(), srv.URL)
	it.Then(t).Should(it.Nil(err))
	defer body.Close()

	raw, rerr := io.ReadAll(body)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal("hello world", string(raw)))
}

func TestFetchNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNotFound)
	}))
	defer srv.Close()

	client := http.New(5 * time.Second)
	body, err := client.Fetch(context.Background(), srv.URL)

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).
		Should(it.String(err.Error()).Contain(srv.URL)).
		Should(it.String(err.Error()).Contain("404"))
	it.Then(t).Should(it.True(body == nil))
}

func TestFetchContextTimeout(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("too slow"))
	}))
	defer srv.Close()

	client := http.New(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	body, err := client.Fetch(ctx, srv.URL)

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.String(err.Error()).Contain(srv.URL))
	it.Then(t).Should(it.True(body == nil))
}
