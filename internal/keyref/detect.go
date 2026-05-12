package keyref

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

var shortcutRefRe = regexp.MustCompile(`\[([^\[\]]+)\][^(\[]`)
var doubleCurlyRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_.\-]+)\}\}`)

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
	fenced := fencedCodeLines(text)
	var locs []KeyrefLocation
	for i, line := range lines {
		// existing [key] detection
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

		// {{key}} detection — skip fenced code blocks
		if fenced[i] {
			continue
		}
		dcMatches := doubleCurlyRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range dcMatches {
			matchStart := m[0]
			keyStart := m[2]
			keyEnd := m[3]
			if isInInlineCode(line, matchStart) {
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     line[keyStart:keyEnd],
				Line:    i,
				EndChar: m[1],
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

	// existing [key] detection
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

	// {{key}} detection
	fenced := fencedCodeLines(text)
	if !fenced[pos.Line] {
		dcMatches := doubleCurlyRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range dcMatches {
			matchStart := m[0]
			matchEnd := m[1]
			keyStart := m[2]
			keyEnd := m[3]
			if isInInlineCode(line, matchStart) {
				continue
			}
			if pos.Character >= matchStart && pos.Character < matchEnd {
				return &KeyrefAtPos{
					Label: line[keyStart:keyEnd],
					Range: document.Rng(pos.Line, matchStart, pos.Line, matchEnd),
				}
			}
		}
	}

	return nil
}

func DetectAllDoubleCurly(text string) []KeyrefLocation {
	lines := strings.Split(text, "\n")
	fenced := fencedCodeLines(text)
	var locs []KeyrefLocation
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		matches := doubleCurlyRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			matchStart := m[0]
			keyStart := m[2]
			keyEnd := m[3]
			if isInInlineCode(line, matchStart) {
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     line[keyStart:keyEnd],
				Line:    i,
				EndChar: m[1],
			})
		}
	}
	return locs
}
