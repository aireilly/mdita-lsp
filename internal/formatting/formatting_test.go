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
			runes := []rune(line)
			newLine := string(runes[:startChar]) + e.NewText + string(runes[endChar:])
			lines[startLine] = newLine
		} else {
			startRunes := []rune(lines[startLine])
			endRunes := []rune(lines[endLine])
			before := string(startRunes[:startChar])
			after := string(endRunes[endChar:])
			lines[startLine] = before + e.NewText + after
			lines = append(lines[:startLine+1], lines[endLine+1:]...)
		}
	}
	return strings.Join(lines, "\n")
}

func TestTrailingWhitespace(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"# Title  \n\nSome text   \n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})

	result := applyEdits(doc.Text, edits)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if strings.TrimRight(line, " \t") != line {
			t.Errorf("line %d still has trailing whitespace: %q", i, line)
		}
	}
}

func TestHeadingSpacing(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"#Title\n\n##  Section\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})

	result := applyEdits(doc.Text, edits)
	if !strings.Contains(result, "# Title") {
		t.Errorf("expected '# Title', got %q", result)
	}
	if !strings.Contains(result, "## Section") {
		t.Errorf("expected '## Section', got %q", result)
	}
}

func TestTrailingNewline(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"# Title\n\nContent")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})

	result := applyEdits(doc.Text, edits)
	if !strings.HasSuffix(result, "\n") {
		t.Error("expected trailing newline")
	}
}

func TestAlreadyHasTrailingNewline(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"# Title\n\nContent\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})

	for _, e := range edits {
		if e.NewText == "\n" && e.Range.Start.Line == 2 {
			t.Error("should not add extra trailing newline")
		}
	}
}

func TestTableAlignment(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"| Name | Value |\n| --- | --- |\n| a | long value |\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})

	result := applyEdits(doc.Text, edits)
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) < 3 {
				continue
			}
			if !strings.HasPrefix(parts[1], " ") || !strings.HasSuffix(parts[1], " ") {
				t.Errorf("expected padded cells, got %q", line)
			}
		}
	}
}

func TestNoChanges(t *testing.T) {
	doc := document.New("file:///test.md", 1,
		"# Title\n\nContent\n")
	edits := Format(doc, Options{TabSize: 4, InsertSpaces: true})

	if len(edits) != 0 {
		t.Errorf("expected no edits for well-formatted doc, got %d", len(edits))
	}
}
