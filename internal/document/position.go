package document

import "unicode/utf8"

// LSP positions count UTF-16 code units, not bytes and not runes. The server
// advertises positionEncoding "utf-16", the one encoding every client must
// support, and converts at the boundary. Counting bytes instead put the edit
// for "append ! at column 9 of '# Café ok'" one byte late and desynced the
// server's copy of the buffer from the editor's.

// UTF16Len returns the length of s in UTF-16 code units.
func UTF16Len(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

// UTF16Column converts a byte offset within line to a UTF-16 column. An offset
// past the end of the line clamps to the line's length.
func UTF16Column(line string, byteOffset int) int {
	if byteOffset > len(line) {
		byteOffset = len(line)
	}
	if byteOffset < 0 {
		byteOffset = 0
	}
	return UTF16Len(line[:byteOffset])
}

// ByteOffsetForColumn converts a UTF-16 column within line to a byte offset.
// A column past the end of the line clamps to the line's length, so a position
// the client sends for a line it has already changed cannot slice out of range.
func ByteOffsetForColumn(line string, column int) int {
	if column <= 0 {
		return 0
	}
	units := 0
	for i, r := range line {
		if units >= column {
			return i
		}
		units++
		if r > 0xFFFF {
			units++
		}
		// A column landing inside a surrogate pair rounds up to the end of the
		// rune rather than splitting it.
		if units > column {
			return i + utf8.RuneLen(r)
		}
	}
	return len(line)
}

// OffsetFromPosition converts an LSP position to a byte offset in text. Both
// the line and the column are clamped, so an out-of-range position from the
// client yields a valid offset instead of a panic.
func OffsetFromPosition(text string, lineMap []int, pos Position) int {
	if len(lineMap) == 0 {
		return 0
	}
	line := pos.Line
	if line < 0 {
		return 0
	}
	if line >= len(lineMap) {
		return len(text)
	}
	start := lineMap[line]
	end := len(text)
	if line+1 < len(lineMap) {
		end = lineMap[line+1] - 1 // exclude the newline
		if end < start {
			end = start
		}
	}
	return start + ByteOffsetForColumn(text[start:end], pos.Character)
}
