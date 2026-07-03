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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	confighttp "github.com/fogfish/arcnet-cli/internal/app/config/adapter/http"
)

func TestFetcherSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mergeRules:\n  source: none\n"))
	}))
	defer srv.Close()

	f := confighttp.New()
	body, err := f.Fetch(context.Background(), srv.URL)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.String(string(body)).Contain("mergeRules"))
}

func TestFetcherNon2xxStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := confighttp.New()
	_, err := f.Fetch(context.Background(), srv.URL)

	it.Then(t).ShouldNot(it.Nil(err))
}

func TestFetcherTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := confighttp.New()
	f.Client.Timeout = 1 * time.Millisecond

	_, err := f.Fetch(context.Background(), srv.URL)

	it.Then(t).ShouldNot(it.Nil(err))
}
