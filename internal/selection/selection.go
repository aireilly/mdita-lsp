package selection

import "github.com/aireilly/mdita-lsp/internal/document"

type SelectionRange struct {
	Range  document.Range   `json:"range"`
	Parent *SelectionRange  `json:"parent,omitempty"`
}

func GetRanges(doc *document.Document, positions []document.Position) []SelectionRange {
	var results []SelectionRange
	headings := doc.Index.Headings()

	for _, pos := range positions {
		sr := buildSelectionRange(doc, pos, headings)
		results = append(results, sr)
	}
	return results
}

func buildSelectionRange(doc *document.Document, pos document.Position, headings []*document.Heading) SelectionRange {
	lineRange := lineRangeAt(doc, pos)
	sr := SelectionRange{Range: lineRange}

	elem := doc.ElementAt(pos)
	if elem != nil {
		elemRange := elem.Rng()
		if !rangeEquals(elemRange, lineRange) {
			sr = SelectionRange{
				Range:  elemRange,
				Parent: &SelectionRange{Range: lineRange},
			}
		}
	}

	for i := len(headings) - 1; i >= 0; i-- {
		h := headings[i]
		if h.Range.Start.Line > pos.Line {
			continue
		}
		sectionEnd := sectionEndLine(headings, i, doc)
		if pos.Line <= sectionEnd {
			sectionRange := document.Rng(h.Range.Start.Line, 0, sectionEnd, 0)
			outermost := outermostParent(&sr)
			outermost.Parent = &SelectionRange{Range: sectionRange}
			break
		}
	}

	return sr
}

func lineRangeAt(doc *document.Document, pos document.Position) document.Range {
	lines := doc.Lines
	lineStart := 0
	lineEnd := len(doc.Text)
	if pos.Line < len(lines) {
		lineStart = lines[pos.Line]
	}
	if pos.Line+1 < len(lines) {
		lineEnd = lines[pos.Line+1] - 1
	}
	col := lineEnd - lineStart
	if col < 0 {
		col = 0
	}
	return document.Rng(pos.Line, 0, pos.Line, col)
}

func sectionEndLine(headings []*document.Heading, idx int, doc *document.Document) int {
	level := headings[idx].Level
	for i := idx + 1; i < len(headings); i++ {
		if headings[i].Level <= level {
			return headings[i].Range.Start.Line - 1
		}
	}
	totalLines := len(doc.Lines) - 1
	if totalLines < 0 {
		totalLines = 0
	}
	return totalLines
}

func outermostParent(sr *SelectionRange) *SelectionRange {
	for sr.Parent != nil {
		sr = sr.Parent
	}
	return sr
}

func rangeEquals(a, b document.Range) bool {
	return a.Start.Line == b.Start.Line &&
		a.Start.Character == b.Start.Character &&
		a.End.Line == b.End.Line &&
		a.End.Character == b.End.Character
}
