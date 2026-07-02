package bios

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"
)

// TestReporterDoneEndsWithCleanNewline is a regression test for BUG-001:
// lipgloss.Style.Render() treats multi-line input as a block and pads every
// line to the block's width, so a trailing "\n" embedded inside the styled
// string is replaced with padding spaces instead of a real line break. The
// next line then lands on the same terminal row, indented by the previous
// line's length. Reporter.Done/.Error MUST render styled text alone and
// write the newline outside the styled span.
func TestReporterDoneEndsWithCleanNewline(t *testing.T) {
	original := SCHEMA
	SCHEMA = SCHEMA_COLOR
	t.Cleanup(func() { SCHEMA = original })

	var buf bytes.Buffer
	reporter := stderrReporter{w: &buf}

	reporter.Done("Checking git availability", 14*time.Millisecond)
	reporter.Done("Preparing git repository", 21*time.Millisecond)

	out := buf.String()

	it.Then(t).
		Should(it.True(strings.HasSuffix(out, "\n"))).
		Should(it.True(!strings.Contains(out, " \n"))).
		Should(it.Equal(2, strings.Count(out, "\n")))
}

func TestReporterErrorEndsWithCleanNewline(t *testing.T) {
	original := SCHEMA
	SCHEMA = SCHEMA_COLOR
	t.Cleanup(func() { SCHEMA = original })

	var buf bytes.Buffer
	reporter := stderrReporter{w: &buf}

	reporter.Error("Committing empty graph", errTest{"exit status 1"})

	out := buf.String()

	it.Then(t).
		Should(it.True(strings.HasSuffix(out, "\n"))).
		Should(it.True(!strings.Contains(out, " \n"))).
		Should(it.Equal(1, strings.Count(out, "\n")))
}

type errTest struct{ msg string }

func (e errTest) Error() string { return e.msg }
