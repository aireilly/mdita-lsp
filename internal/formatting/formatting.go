package formatting

import (
	"strings"
	"unicode/utf8"

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

func Format(doc *document.Document, opts Options) []TextEdit {
	lines := strings.Split(doc.Text, "\n")
	var edits []TextEdit

	edits = append(edits, trimTrailingWhitespace(lines)...)
	edits = append(edits, normalizeHeadingSpacing(lines)...)
	edits = append(edits, ensureTrailingNewline(lines)...)
	edits = append(edits, alignTables(lines)...)

	return edits
}

func trimTrailingWhitespace(lines []string) []TextEdit {
	var edits []TextEdit
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if len(trimmed) != len(line) {
			edits = append(edits, TextEdit{
				Range: document.Range{
					Start: document.Position{Line: i, Character: utf8.RuneCountInString(trimmed)},
					End:   document.Position{Line: i, Character: utf8.RuneCountInString(line)},
				},
				NewText: "",
			})
		}
	}
	return edits
}

func normalizeHeadingSpacing(lines []string) []TextEdit {
	var edits []TextEdit
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		hashes := 0
		for _, ch := range trimmed {
			if ch == '#' {
				hashes++
			} else {
				break
			}
		}
		if hashes == 0 || hashes > 6 {
			continue
		}
		rest := trimmed[hashes:]
		if len(rest) == 0 {
			continue
		}
		if rest[0] != ' ' {
			edits = append(edits, TextEdit{
				Range: document.Range{
					Start: document.Position{Line: i, Character: 0},
					End:   document.Position{Line: i, Character: utf8.RuneCountInString(line)},
				},
				NewText: trimmed[:hashes] + " " + strings.TrimLeft(rest, " \t"),
			})
		} else if strings.HasPrefix(rest, "  ") {
			edits = append(edits, TextEdit{
				Range: document.Range{
					Start: document.Position{Line: i, Character: 0},
					End:   document.Position{Line: i, Character: utf8.RuneCountInString(line)},
				},
				NewText: trimmed[:hashes] + " " + strings.TrimLeft(rest, " \t"),
			})
		}

	}
	return edits
}

func ensureTrailingNewline(lines []string) []TextEdit {
	if len(lines) == 0 {
		return nil
	}
	last := lines[len(lines)-1]
	if last == "" {
		return nil
	}
	lastLine := len(lines) - 1
	lastChar := utf8.RuneCountInString(last)
	return []TextEdit{{
		Range: document.Range{
			Start: document.Position{Line: lastLine, Character: lastChar},
			End:   document.Position{Line: lastLine, Character: lastChar},
		},
		NewText: "\n",
	}}
}

func alignTables(lines []string) []TextEdit {
	var edits []TextEdit
	i := 0
	for i < len(lines) {
		if !isTableRow(lines[i]) {
			i++
			continue
		}
		start := i
		for i < len(lines) && isTableRow(lines[i]) {
			i++
		}
		tableEdits := alignTableBlock(lines, start, i)
		edits = append(edits, tableEdits...)
	}
	return edits
}

func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|")
}

func alignTableBlock(lines []string, start, end int) []TextEdit {
	rows := make([][]string, end-start)
	maxCols := 0
	for i := start; i < end; i++ {
		cells := parseTableCells(lines[i])
		rows[i-start] = cells
		if len(cells) > maxCols {
			maxCols = len(cells)
		}
	}

	if maxCols == 0 {
		return nil
	}

	colWidths := make([]int, maxCols)
	for _, row := range rows {
		for j, cell := range row {
			w := utf8.RuneCountInString(cell)
			if w > colWidths[j] {
				colWidths[j] = w
			}
		}
	}

	var edits []TextEdit
	for ri, row := range rows {
		lineIdx := start + ri
		var b strings.Builder
		b.WriteString("|")
		for j := 0; j < maxCols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			if isSeparatorCell(cell) {
				b.WriteString(" ")
				b.WriteString(strings.Repeat("-", colWidths[j]))
				b.WriteString(" |")
			} else {
				b.WriteString(" ")
				b.WriteString(cell)
				padding := colWidths[j] - utf8.RuneCountInString(cell)
				b.WriteString(strings.Repeat(" ", padding))
				b.WriteString(" |")
			}
		}
		newLine := b.String()
		if newLine != lines[lineIdx] {
			edits = append(edits, TextEdit{
				Range: document.Range{
					Start: document.Position{Line: lineIdx, Character: 0},
					End:   document.Position{Line: lineIdx, Character: utf8.RuneCountInString(lines[lineIdx])},
				},
				NewText: newLine,
			})
		}
	}
	return edits
}

func parseTableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

func isSeparatorCell(cell string) bool {
	stripped := strings.Trim(cell, "-:")
	return len(cell) > 0 && stripped == ""
}
