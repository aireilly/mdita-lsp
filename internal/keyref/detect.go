package keyref

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

// shortcutRefRe matches [key] shortcut references, not followed by ( or [.
var shortcutRefRe = regexp.MustCompile(`\[([^\[\]]+)\][^(\[]`)

// fullRefLinkRe matches [text][key] reference-style links, capturing the key portion.
var fullRefLinkRe = regexp.MustCompile(`\[[^\]]*\]\[([^\]]+)\]`)

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
	// A bracketed label inside a fenced block is literal text in the
	// generated <codeblock>, not a key reference.
	lines := strings.Split(document.MaskFencedCode(text), "\n")
	var locs []KeyrefLocation

	for i, line := range lines {
		// Track key positions already captured to avoid duplicates.
		seen := make(map[int]bool)

		// Full reference-style links: [text][key] — capture the key portion.
		for _, m := range fullRefLinkRe.FindAllStringSubmatchIndex(line, -1) {
			labelStart := m[2]
			labelEnd := m[3]
			label := line[labelStart:labelEnd]
			if strings.HasPrefix(label, "^") {
				continue
			}
			seen[labelStart] = true
			locs = append(locs, KeyrefLocation{
				Key:     label,
				Line:    i,
				EndChar: m[1], // end of the full [text][key] construct
			})
		}

		// Shortcut references: [key] not followed by ( or [.
		for _, m := range shortcutRefRe.FindAllStringSubmatchIndex(line, -1) {
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
			if seen[labelStart] {
				// Already captured by fullRefLinkRe.
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     label,
				Line:    i,
				EndChar: labelEnd + 1,
			})
		}

		// HTML data-keyref attributes: <tag data-keyref="key">.
		for _, m := range dataKeyrefRe.FindAllStringSubmatchIndex(line, -1) {
			key := line[m[2]:m[3]]
			locs = append(locs, KeyrefLocation{
				Key:     key,
				Line:    i,
				EndChar: m[1],
			})
		}
	}
	return locs
}

func DetectAtPosition(text string, pos document.Position) *KeyrefAtPos {
	lines := strings.Split(document.MaskFencedCode(text), "\n")
	if pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	col := pos.Character

	// Check [text][key] pattern — cursor must be within the [key] portion.
	for _, m := range fullRefLinkRe.FindAllStringSubmatchIndex(line, -1) {
		keyStart, keyEnd := m[2], m[3]
		if col >= keyStart && col <= keyEnd {
			key := line[keyStart:keyEnd]
			return &KeyrefAtPos{
				Label: key,
				Range: document.Rng(pos.Line, m[0], pos.Line, m[1]),
			}
		}
	}

	// Check [key] shortcut.
	for _, m := range shortcutRefRe.FindAllStringSubmatchIndex(line, -1) {
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

	// Check data-keyref="key" attribute.
	for _, m := range dataKeyrefRe.FindAllStringSubmatchIndex(line, -1) {
		keyStart, keyEnd := m[2], m[3]
		if col >= keyStart && col <= keyEnd {
			key := line[keyStart:keyEnd]
			return &KeyrefAtPos{
				Label: key,
				Range: document.Rng(pos.Line, m[0], pos.Line, m[1]),
			}
		}
	}

	return nil
}
