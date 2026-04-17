package keyref

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

var shortcutRefRe = regexp.MustCompile(`\[([^\[\]]+)\][^(\[]`)

type KeyrefAtPos struct {
	Label string
	Range document.Range
}

type KeyrefLocation struct {
	Key     string
	Line    int
	EndChar int
}

func DetectAll(text string) []KeyrefLocation {
	lines := strings.Split(text, "\n")
	var locs []KeyrefLocation
	for i, line := range lines {
		matches := shortcutRefRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			bracketStart := m[0]
			labelStart := m[2]
			labelEnd := m[3]
			if bracketStart > 0 && line[bracketStart-1] == '[' {
				continue
			}
			label := line[labelStart:labelEnd]
			if strings.HasPrefix(label, "^") {
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     label,
				Line:    i,
				EndChar: labelEnd + 1,
			})
		}
	}
	return locs
}

func DetectAtPosition(text string, pos document.Position) *KeyrefAtPos {
	lines := strings.Split(text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	matches := shortcutRefRe.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		bracketStart := m[0]
		labelStart := m[2]
		labelEnd := m[3]

		if bracketStart > 0 && line[bracketStart-1] == '[' {
			continue
		}

		label := line[labelStart:labelEnd]
		if strings.HasPrefix(label, "^") {
			continue
		}

		if pos.Character >= labelStart && pos.Character <= labelEnd {
			return &KeyrefAtPos{
				Label: label,
				Range: document.Rng(pos.Line, labelStart, pos.Line, labelEnd),
			}
		}
	}
	return nil
}
