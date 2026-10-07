package formatting

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func applyEdits(text string, edits []TextEdit) string {
	lines := strings.Split(text, "\n")
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		startLine := e.Range.Start.Line
		endLine := e.Range.End.Line
		startChar := e.Range.Start.Character
		endChar := e.Range.End.Character

		if startLine == endLine {
			line := lines[startLine]
			newLine := line[:document.ByteOffsetForColumn(line, startChar)] +
				e.NewText + line[document.ByteOffsetForColumn(line, endChar):]
			lines[startLine] = newLine
		} else {
			before := lines[startLine][:document.ByteOffsetForColumn(lines[startLine], startChar)]
			after := lines[endLine][document.ByteOffsetForColumn(lines[endLine], endChar):]
			lines[startLine] = before + e.NewText + after
			lines = append(lines[:startLine+1], lines[endLine+1:]...)
		}
	}
	return strings.Join(lines, "\n")
}

// assertNoOverlap fails when two edits touch the same characters. Clients apply
// overlapping edits in an undefined order, which corrupts the buffer.
func assertNoOverlap(t *testing.T, edits []TextEdit) {
	t.Helper()
	seen := make(map[int][2]int)
	for _, e := range edits {
		if e.Range.Start.Line != e.Range.End.Line {
			t.Fatalf("multi-line edit %v is not expected from the formatter", e.Range)
		}
		line := e.Range.Start.Line
		prev, ok := seen[line]
		if ok {
			t.Errorf("two edits on line %d: %v and [%d,%d]", line, prev,
				e.Range.Start.Character, e.Range.End.Character)
		}
		seen[line] = [2]int{e.Range.Start.Character, e.Range.End.Character}
	}
}

func format(t *testing.T, text string) string {
	t.Helper()
	doc := document.New("file:///test.md", 1, text)
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})
	assertNoOverlap(t, edits)
	return applyEdits(doc.Text, edits)
}

func TestTrailingWhitespace(t *testing.T) {
	got := format(t, "# Title  \n\nSome text   \n")
	want := "# Title\n\nSome text\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Two trailing spaces before another line of the same paragraph are a hard
// line break, which the plug-in renders as <?linebreak?>.
func TestHardLineBreakPreserved(t *testing.T) {
	got := format(t, "# Title\n\nfirst line  \nsecond line\n")
	want := "# Title\n\nfirst line  \nsecond line\n"
	if got != want {
		t.Errorf("hard line break was stripped: got %q, want %q", got, want)
	}
}

func TestHardLineBreakNormalizedToTwoSpaces(t *testing.T) {
	got := format(t, "para one     \npara two\n")
	want := "para one  \npara two\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrailingSpacesBeforeBlankLineAreTrimmed(t *testing.T) {
	got := format(t, "last line of para  \n\nnext para\n")
	want := "last line of para\n\nnext para\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHeadingSpacing(t *testing.T) {
	got := format(t, "# Title\n\n##  Section\n")
	if !strings.Contains(got, "## Section") {
		t.Errorf("expected '## Section', got %q", got)
	}
}

// "#Title" has no space after the hash, so CommonMark and the plug-in read it
// as a paragraph. Adding the space would change what the build produces.
func TestHashWithoutSpaceIsLeftAlone(t *testing.T) {
	got := format(t, "#Title\n\ntext\n")
	want := "#Title\n\ntext\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrailingNewline(t *testing.T) {
	got := format(t, "# Title\n\nContent")
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}
}

func TestAlreadyHasTrailingNewline(t *testing.T) {
	doc := document.New("file:///test.md", 1, "# Title\n\nContent\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})
	if len(edits) != 0 {
		t.Errorf("expected no edits, got %v", edits)
	}
}

func TestTableSingleSpacePadding(t *testing.T) {
	got := format(t, "| Name | Value |\n|:---|---:|\n| a | a much longer value |\n")
	want := "| Name | Value |\n| :--- | ---: |\n| a | a much longer value |\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTableCollapsesExistingPadding(t *testing.T) {
	got := format(t, "| a   | bb                  |\n| --- | ------------------- |\n| ccc | d                   |\n")
	want := "| a | bb |\n| --- | --- |\n| ccc | d |\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// An escaped pipe is cell text. Splitting on it added a column and turned the
// table into a paragraph of literal pipes in the build.
func TestTableKeepsEscapedPipe(t *testing.T) {
	src := "| a | b |\n| --- | --- |\n| x \\| y | z |\n"
	got := format(t, src)
	if got != src {
		t.Errorf("escaped pipe was split: got %q, want %q", got, src)
	}
}

// A pipe inside a code span is cell text too.
func TestTableKeepsPipeInCodeSpan(t *testing.T) {
	src := "| a | b |\n| --- | --- |\n| `c|d` | z |\n"
	got := format(t, src)
	if got != src {
		t.Errorf("code-span pipe was split: got %q, want %q", got, src)
	}
}

func TestTableNeverAddsAColumn(t *testing.T) {
	src := "| a | b |\n| --- | --- |\n| x \\| y | z |\n"
	got := format(t, src)
	for i, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if n := len(splitRow(line)); n != 2 {
			t.Errorf("line %d has %d cells, want 2: %q", i, n, line)
		}
	}
}

// A delimiter row whose cell count differs from the header's does not open a
// table (HEADER_SEPARATOR_COLUMN_MATCH), so the block must be left alone.
func TestNonTablePipeBlockIsLeftAlone(t *testing.T) {
	src := "| a | b | c |\n| --- | --- |\n| x | y |\n"
	got := format(t, src)
	if got != src {
		t.Errorf("non-table pipe block was rewritten: got %q, want %q", got, src)
	}
}

func TestPipeBlockWithoutDelimiterRowIsLeftAlone(t *testing.T) {
	src := "| not | a table |\n| still | not |\n"
	got := format(t, src)
	if got != src {
		t.Errorf("got %q, want %q", got, src)
	}
}

func TestNoChanges(t *testing.T) {
	doc := document.New("file:///test.md", 1, "# Title\n\nContent\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})
	if len(edits) != 0 {
		t.Errorf("expected no edits for well-formatted doc, got %d", len(edits))
	}
}

// Trailing whitespace and a table rewrite on the same line used to produce two
// overlapping edits.
func TestTrailingWhitespaceOnTableRowYieldsOneEdit(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"| a   | b |   \n| --- | --- |\n| c | d |\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})
	assertNoOverlap(t, edits)
	got := applyEdits(doc.Text, edits)
	want := "| a | b |\n| --- | --- |\n| c | d |\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Edit columns are UTF-16 code units, so a line with non-ASCII text must not
// shift the end of the replaced range.
func TestEditColumnsAreUTF16(t *testing.T) {
	doc := document.New("file:///test.md", 1, "# Café  \n\ntext\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
	if got := edits[0].Range.End.Character; got != 8 {
		t.Errorf("end character = %d, want 8 (UTF-16 units of '# Café  ')", got)
	}
}

func TestAlignTablesOnlyTouchesTables(t *testing.T) {
	text := "# Title  \n\n| a   | bb |\n|---|---|\n| ccc | d |\n"
	edits := AlignTables(text)
	assertNoOverlap(t, edits)
	for _, e := range edits {
		if e.Range.Start.Line == 0 {
			t.Errorf("save-time formatting touched a non-table line: %v", e)
		}
	}
	got := applyEdits(text, edits)
	if !strings.HasPrefix(got, "# Title  \n") {
		t.Errorf("heading line was changed: %q", got)
	}
}
