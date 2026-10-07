package formatting

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type TextEdit struct {
	Range   document.Range
	NewText string
}

type Options struct {
	TabSize      int
	InsertSpaces bool
}

// Format returns the edits that normalize a document's whitespace, headings,
// and pipe tables.
//
// Every rule runs in one pass over the lines and the result is diffed line by
// line, so a line yields at most one edit. Emitting one edit per rule let the
// trailing-whitespace edit and the heading or table edit for the same line
// overlap, which clients apply in an undefined order.
func Format(doc *document.Document, opts Options) []TextEdit {
	return editsForText(doc.Text)
}

// AlignTables returns only the edits that reformat pipe tables. This is what
// runs on save, where rewriting anything else would be a surprise.
func AlignTables(text string) []TextEdit {
	lines := splitLines(text)
	formatted := make([]string, len(lines))
	copy(formatted, lines)
	applyTables(formatted)
	return diffLines(lines, formatted, false)
}

func editsForText(text string) []TextEdit {
	lines := splitLines(text)
	formatted := make([]string, len(lines))
	copy(formatted, lines)

	for i := range formatted {
		formatted[i] = trimTrailing(formatted, i)
		formatted[i] = normalizeHeading(formatted[i])
	}
	applyTables(formatted)

	return diffLines(lines, formatted, true)
}

func splitLines(text string) []string {
	return strings.Split(text, "\n")
}

// diffLines turns two line slices of the same length into one edit per changed
// line. With addTrailingNewline set, a file whose last line has content also
// gets the newline appended to that line's edit rather than as a separate edit
// at the same position.
func diffLines(old, formatted []string, addTrailingNewline bool) []TextEdit {
	var edits []TextEdit
	last := len(old) - 1

	for i := range old {
		newText := formatted[i]
		trailing := addTrailingNewline && i == last && old[i] != ""
		if trailing {
			newText += "\n"
		}
		if newText == old[i] {
			continue
		}
		edits = append(edits, TextEdit{
			Range: document.Range{
				Start: document.Position{Line: i, Character: 0},
				End:   document.Position{Line: i, Character: document.UTF16Len(old[i])},
			},
			NewText: newText,
		})
	}
	return edits
}

// trimTrailing removes trailing spaces and tabs, but leaves a markdown hard
// line break alone. Two or more trailing spaces before another line of the
// same paragraph are what the plug-in turns into <?linebreak?>; stripping them
// silently joined the lines in the built output.
func trimTrailing(lines []string, i int) string {
	line := lines[i]
	trimmed := strings.TrimRight(line, " \t")
	if trimmed == line {
		return line
	}
	if isHardLineBreak(lines, i, trimmed) {
		return trimmed + "  "
	}
	return trimmed
}

func isHardLineBreak(lines []string, i int, trimmed string) bool {
	if trimmed == "" {
		return false
	}
	if i+1 >= len(lines) || strings.TrimSpace(lines[i+1]) == "" {
		return false
	}
	// Only spaces count; a tab does not make a hard break.
	tail := lines[i][len(trimmed):]
	return len(tail) >= 2 && strings.Trim(tail, " ") == ""
}

// normalizeHeading puts exactly one space between an ATX heading's hashes and
// its text. Indentation and the text itself are left as they are.
func normalizeHeading(line string) string {
	trimmed := strings.TrimSpace(line)
	hashes := 0
	for _, ch := range trimmed {
		if ch != '#' {
			break
		}
		hashes++
	}
	if hashes == 0 || hashes > 6 {
		return line
	}
	rest := trimmed[hashes:]
	if rest == "" {
		return line
	}
	if rest[0] != ' ' && rest[0] != '\t' {
		// "#foo" is a paragraph, not a heading, so leave it alone.
		return line
	}
	body := strings.TrimLeft(rest, " \t")
	if body == "" {
		return line
	}
	return trimmed[:hashes] + " " + body
}

// applyTables rewrites each pipe-table block in place.
func applyTables(lines []string) {
	i := 0
	for i < len(lines) {
		if !isPipeRow(lines[i]) {
			i++
			continue
		}
		start := i
		for i < len(lines) && isPipeRow(lines[i]) {
			i++
		}
		if out := formatTable(lines[start:i]); out != nil {
			copy(lines[start:i], out)
		}
	}
}
