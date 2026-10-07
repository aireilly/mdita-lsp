package formatting

import "strings"

// splitRow splits a pipe-table row into cells the way a GFM parser does: a
// pipe escaped with a backslash and a pipe inside a code span are cell text,
// not a cell boundary. Splitting on every pipe is what made the formatter add
// a column to a table whose cells held an escaped pipe or a code span
// containing one.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "|") {
		return nil
	}
	s = s[1:]

	var cells []string
	var cur strings.Builder
	backticks := 0 // length of the run opening the current code span, 0 outside one

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if ch == '\\' && i+1 < len(runes) {
			cur.WriteRune(ch)
			cur.WriteRune(runes[i+1])
			i++
			continue
		}

		if ch == '`' {
			run := 1
			for i+run < len(runes) && runes[i+run] == '`' {
				run++
			}
			switch backticks {
			case 0:
				backticks = run
			case run:
				backticks = 0
			}
			for j := 0; j < run; j++ {
				cur.WriteRune('`')
			}
			i += run - 1
			continue
		}

		if ch == '|' && backticks == 0 {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}

		cur.WriteRune(ch)
	}

	// Text after the last pipe is a trailing cell; an empty tail means the row
	// ended with the closing pipe and has no extra cell.
	tail := strings.TrimSpace(cur.String())
	if tail != "" || backticks != 0 {
		cells = append(cells, tail)
	}
	return cells
}

// isDelimiterRow reports whether the row is a GFM delimiter row: every cell is
// dashes with optional leading and trailing colons.
func isDelimiterRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !isSeparatorCell(c) {
			return false
		}
	}
	return true
}

func isSeparatorCell(cell string) bool {
	body := strings.TrimPrefix(cell, ":")
	body = strings.TrimSuffix(body, ":")
	if body == "" {
		return false
	}
	return strings.Trim(body, "-") == ""
}

// normalizeSeparatorCell reduces a delimiter-row cell to three dashes while
// keeping any alignment colons.
func normalizeSeparatorCell(cell string) string {
	left := strings.HasPrefix(cell, ":")
	right := strings.HasSuffix(cell, ":") && len(cell) > 1
	switch {
	case left && right:
		return ":---:"
	case left:
		return ":---"
	case right:
		return "---:"
	default:
		return "---"
	}
}

// isPipeRow reports whether a line could be part of a pipe table. A table row
// starts with a pipe; the closing pipe is optional in GFM.
func isPipeRow(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// formatTable rewrites the lines of one pipe table, returning nil when the
// block is not a table the plug-in would parse as one.
//
// The column count comes from the delimiter row, which is what decides the
// table's width. Cells beyond it are left where they are rather than widening
// every other row, because the plug-in discards extra columns (the reader sets
// DISCARD_EXTRA_COLUMNS) and adding an empty column to the source changed what
// the build produced.
func formatTable(lines []string) []string {
	if len(lines) < 2 {
		return nil
	}

	header := splitRow(lines[0])
	delim := splitRow(lines[1])
	if len(header) == 0 || !isDelimiterRow(delim) {
		return nil
	}
	// The readers set HEADER_SEPARATOR_COLUMN_MATCH, so a delimiter row whose
	// cell count differs from the header's does not open a table at all.
	if len(delim) != len(header) {
		return nil
	}

	out := make([]string, len(lines))
	for i, line := range lines {
		cells := splitRow(line)
		if len(cells) == 0 {
			out[i] = line
			continue
		}
		if i == 1 {
			for j := range cells {
				cells[j] = normalizeSeparatorCell(cells[j])
			}
		}
		var b strings.Builder
		b.WriteString("|")
		for _, cell := range cells {
			b.WriteString(" ")
			b.WriteString(cell)
			b.WriteString(" |")
		}
		out[i] = b.String()
	}
	return out
}
