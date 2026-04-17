package folding

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type FoldingRange struct {
	StartLine      int
	StartCharacter int
	EndLine        int
	EndCharacter   int
	Kind           string // "region", "comment", "imports"
}

func GetRanges(doc *document.Document) []FoldingRange {
	var ranges []FoldingRange
	ranges = append(ranges, foldHeadings(doc)...)
	ranges = append(ranges, foldYAMLFrontMatter(doc)...)
	ranges = append(ranges, foldToCMarkers(doc)...)
	return ranges
}

func foldHeadings(doc *document.Document) []FoldingRange {
	headings := doc.Index.Headings()
	if len(headings) == 0 {
		return nil
	}

	lines := strings.Split(doc.Text, "\n")
	lastLine := len(lines) - 1

	var ranges []FoldingRange
	for i, h := range headings {
		startLine := h.Range.Start.Line

		var endLine int
		if i+1 < len(headings) {
			endLine = headings[i+1].Range.Start.Line - 1
		} else {
			endLine = lastLine
		}

		for endLine > startLine && strings.TrimSpace(lines[endLine]) == "" {
			endLine--
		}

		if endLine > startLine {
			ranges = append(ranges, FoldingRange{
				StartLine: startLine,
				EndLine:   endLine,
				Kind:      "region",
			})
		}
	}
	return ranges
}

func foldYAMLFrontMatter(doc *document.Document) []FoldingRange {
	if doc.Meta == nil {
		return nil
	}

	lines := strings.Split(doc.Text, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}

	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" || trimmed == "..." {
			if i > 1 {
				return []FoldingRange{{
					StartLine: 0,
					EndLine:   i,
					Kind:      "region",
				}}
			}
			break
		}
	}
	return nil
}

func foldToCMarkers(doc *document.Document) []FoldingRange {
	lines := strings.Split(doc.Text, "\n")
	startLine := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "<!--toc:start-->") {
			startLine = i
		}
		if strings.Contains(trimmed, "<!--toc:end-->") && startLine >= 0 {
			return []FoldingRange{{
				StartLine: startLine,
				EndLine:   i,
				Kind:      "region",
			}}
		}
	}
	return nil
}
