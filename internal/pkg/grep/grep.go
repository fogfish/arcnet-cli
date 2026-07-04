//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package grep is a reusable, dependency-free, fs.FS-based content-search
// library (ADR 001's "evolution of domain logic" phase 2, first occupant of
// the internal/pkg tier). It has no dependency on internal/core or any
// internal/app/<use-case> and never imports os (constitution Principle
// VII) — every filesystem operation goes through the caller-supplied
// fs.FS/fs.ReadDirFS.
package grep

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Match is one reported occurrence of a search pattern on a single line
// within a single file.
type Match struct {
	// Path is the fs.FS-relative path (fs.ValidPath form).
	Path string
	// Line is the 1-based line number within Path.
	Line int
	// Text is the full line text, no trailing newline.
	Text string
	// Start is the byte offset within Text where the (first) match begins.
	Start int
	// End is the byte offset within Text where the (first) match ends.
	End int
}

// Options configures a Search.
type Options struct {
	// Extension is the required file suffix, e.g. ".md"; empty defaults to
	// ".md".
	Extension string
	// Workers bounds the concurrent pool size (research.md D3); <= 0
	// defaults to 8.
	Workers int
	// Include, when non-nil, is consulted before a file matching Extension
	// is submitted for content scanning; nil means "scan every file
	// matching Extension".
	Include func(path string) bool
}

// Result is Search's return value.
type Result struct {
	// Matches is sorted by (Path, Line) before return.
	Matches []Match
	// Unreadable holds paths that could not be opened/read; the scan
	// continued for the rest.
	Unreadable []string
}

// Search walks fsys from its root, concurrently scanning every file whose
// name has Options.Extension (and, if set, passes Options.Include) for
// lines matching pattern (regexp semantics). error is non-nil only when
// pattern fails to compile as a regexp, the root directory itself fails to
// list, or ctx is canceled — never for a single file's read failure (that
// is recorded in Result.Unreadable instead).
func Search(ctx context.Context, fsys fs.FS, pattern string, opts Options) (Result, error) {
	ext := opts.Extension
	if ext == "" {
		ext = ".md"
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = 8
	}

	m, err := newMatcher(pattern)
	if err != nil {
		return Result{}, err
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var matches []Match
	var unreadable []string
	var rootErr error
	var cancelErr error

	// spawn dispatches fn as a new goroutine without blocking the caller —
	// the semaphore is acquired *inside* the goroutine, so a directory
	// listing's own dispatch loop never waits on its children's slots,
	// which is what keeps this deadlock-free even at Workers == 1
	// (research.md D3).
	spawn := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fn()
		}()
	}

	var walk func(dir string)
	walk = func(dir string) {
		if err := ctx.Err(); err != nil {
			mu.Lock()
			if cancelErr == nil {
				cancelErr = err
			}
			mu.Unlock()
			return
		}

		entries, err := fs.ReadDir(fsys, dir)
		if err != nil {
			mu.Lock()
			if dir == "." {
				rootErr = err
			} else {
				unreadable = append(unreadable, dir)
			}
			mu.Unlock()
			return
		}

		for _, e := range entries {
			full := e.Name()
			if dir != "." {
				full = path.Join(dir, e.Name())
			}

			if e.IsDir() {
				d := full
				spawn(func() { walk(d) })
				continue
			}

			if !strings.HasSuffix(full, ext) {
				continue
			}
			if opts.Include != nil && !opts.Include(full) {
				continue
			}

			p := full
			spawn(func() {
				if err := ctx.Err(); err != nil {
					mu.Lock()
					if cancelErr == nil {
						cancelErr = err
					}
					mu.Unlock()
					return
				}

				found, err := scanFile(fsys, p, m)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					unreadable = append(unreadable, p)
					return
				}
				matches = append(matches, found...)
			})
		}
	}

	walk(".")
	wg.Wait()

	if rootErr != nil {
		return Result{}, rootErr
	}
	if cancelErr != nil {
		return Result{}, cancelErr
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Path != matches[j].Path {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Line < matches[j].Line
	})

	return Result{Matches: matches, Unreadable: unreadable}, nil
}

// matcher classifies pattern once, up front (research.md D4): a
// metacharacter-free pattern dispatches to a fast bytes.Contains-based
// literal match; anything else compiles a *regexp.Regexp once, not per file
// or per line.
type matcher interface {
	find(line []byte) (start, end int, ok bool)
}

type literalMatcher struct{ needle []byte }

func (m literalMatcher) find(line []byte) (int, int, bool) {
	idx := bytes.Index(line, m.needle)
	if idx < 0 {
		return 0, 0, false
	}
	return idx, idx + len(m.needle), true
}

type regexMatcher struct{ re *regexp.Regexp }

func (m regexMatcher) find(line []byte) (int, int, bool) {
	loc := m.re.FindIndex(line)
	if loc == nil {
		return 0, 0, false
	}
	return loc[0], loc[1], true
}

func newMatcher(pattern string) (matcher, error) {
	if regexp.QuoteMeta(pattern) == pattern {
		return literalMatcher{needle: []byte(pattern)}, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return regexMatcher{re: re}, nil
}

// readerPool holds *bufio.Reader values reused across files (research.md
// D5) to minimize allocation under heavy concurrent scanning.
var readerPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, 64*1024) },
}

// scanFile reads path line-by-line, collapsing a line matching more than
// once into a single Match (the first occurrence's span), and closes the
// file as soon as its own scan finishes.
func scanFile(fsys fs.FS, name string, m matcher) ([]Match, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := readerPool.Get().(*bufio.Reader)
	r.Reset(f)
	defer func() {
		r.Reset(nil)
		readerPool.Put(r)
	}()

	var matches []Match
	line := 0
	for {
		raw, err := r.ReadBytes('\n')
		if len(raw) > 0 {
			line++
			text := bytes.TrimSuffix(bytes.TrimSuffix(raw, []byte("\n")), []byte("\r"))
			if start, end, ok := m.find(text); ok {
				matches = append(matches, Match{
					Path:  name,
					Line:  line,
					Text:  string(text),
					Start: start,
					End:   end,
				})
			}
		}
		if err != nil {
			if err == io.EOF {
				return matches, nil
			}
			return matches, err
		}
	}
}
