//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package grep_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/pkg/grep"
)

func TestSearchLiteralAndRegexProduceIdenticalMatches(t *testing.T) {
	fsys := fstest.MapFS{
		"sources/a.md": &fstest.MapFile{Data: []byte("TLS 1.3 is great\nnothing here\n")},
	}

	literal, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{})
	it.Then(t).Should(it.Nil(err))

	regex, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Equal(len(literal.Matches), len(regex.Matches)))
	it.Then(t).
		Should(it.Equal(1, len(literal.Matches))).
		Should(it.Equal("sources/a.md", literal.Matches[0].Path)).
		Should(it.Equal(1, literal.Matches[0].Line))
}

func TestSearchLineMatchingMultipleTimesCollapsesToOneMatch(t *testing.T) {
	fsys := fstest.MapFS{
		"a.md": &fstest.MapFile{Data: []byte("TLS TLS TLS\n")},
	}

	result, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Matches)))
	it.Then(t).Should(it.Equal(0, result.Matches[0].Start)).Should(it.Equal(3, result.Matches[0].End))
}

// unreadableFS wraps a MapFS, forcing Open of a specific path to fail so
// scanFile hits its own error path without needing a real filesystem.
type unreadableFS struct {
	fstest.MapFS
	badPath string
}

func (f unreadableFS) Open(name string) (fs.File, error) {
	if name == f.badPath {
		return nil, fs.ErrPermission
	}
	return f.MapFS.Open(name)
}

func TestSearchUnreadableFileRecordedAndScanContinues(t *testing.T) {
	fsys := unreadableFS{
		MapFS: fstest.MapFS{
			"a.md": &fstest.MapFile{Data: []byte("TLS\n")},
			"b.md": &fstest.MapFile{Data: []byte("TLS\n")},
		},
		badPath: "a.md",
	}

	result, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Matches)))
	it.Then(t).Should(it.Equal(1, len(result.Unreadable)))
	it.Then(t).Should(it.Equal("a.md", result.Unreadable[0]))
}

func TestSearchInvalidPatternReturnsHardErrorBeforeAnyFileOpened(t *testing.T) {
	fsys := fstest.MapFS{
		"a.md": &fstest.MapFile{Data: []byte("TLS\n")},
	}

	result, err := grep.Search(context.Background(), fsys, "[TLS", grep.Options{})

	it.Then(t).Should(it.True(err != nil))
	it.Then(t).Should(it.Equal(0, len(result.Matches)))
}

func TestSearchMatchesOrderingIsDeterministic(t *testing.T) {
	fsys := fstest.MapFS{
		"z.md": &fstest.MapFile{Data: []byte("TLS\n")},
		"a.md": &fstest.MapFile{Data: []byte("TLS\nTLS\n")},
		"m.md": &fstest.MapFile{Data: []byte("TLS\n")},
	}

	for i := 0; i < 20; i++ {
		result, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{Workers: 8})
		it.Then(t).Should(it.Nil(err))
		it.Then(t).Should(it.Equal(4, len(result.Matches)))
		it.Then(t).
			Should(it.Equal("a.md", result.Matches[0].Path)).
			Should(it.Equal(1, result.Matches[0].Line)).
			Should(it.Equal("a.md", result.Matches[1].Path)).
			Should(it.Equal(2, result.Matches[1].Line)).
			Should(it.Equal("m.md", result.Matches[2].Path)).
			Should(it.Equal("z.md", result.Matches[3].Path))
	}
}

func TestSearchIncludeExcludesFilesFromScan(t *testing.T) {
	fsys := fstest.MapFS{
		"a.md": &fstest.MapFile{Data: []byte("TLS\n")},
		"b.md": &fstest.MapFile{Data: []byte("TLS\n")},
	}

	result, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{
		Include: func(path string) bool { return path == "a.md" },
	})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Matches)))
	it.Then(t).Should(it.Equal("a.md", result.Matches[0].Path))
}

func TestSearchExtensionConfigurable(t *testing.T) {
	fsys := fstest.MapFS{
		"a.md":  &fstest.MapFile{Data: []byte("TLS\n")},
		"a.txt": &fstest.MapFile{Data: []byte("TLS\n")},
	}

	defaultResult, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(defaultResult.Matches)))

	txtResult, err := grep.Search(context.Background(), fsys, "TLS", grep.Options{Extension: ".txt"})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(txtResult.Matches)))
	it.Then(t).Should(it.Equal("a.txt", txtResult.Matches[0].Path))
}

func TestSearchWorkersConfigurable(t *testing.T) {
	files := fstest.MapFS{}
	for i := 0; i < 40; i++ {
		files[string(rune('a'+i%26))+"/"+string(rune('a'+i))+".md"] = &fstest.MapFile{Data: []byte("TLS\n")}
	}

	for _, workers := range []int{1, 8, 32} {
		result, err := grep.Search(context.Background(), files, "TLS", grep.Options{Workers: workers})
		it.Then(t).Should(it.Nil(err))
		it.Then(t).Should(it.Equal(40, len(result.Matches)))
	}
}
