package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type PartialKind int

const (
	PartialInlineLink PartialKind = iota
	PartialInlineAnchor
	PartialRefLink
	PartialYamlKey
	PartialKeyref
	PartialHeadingText
	PartialAttrClass
	PartialBlockAttr
)

type PartialElement struct {
	Kind          PartialKind
	Input         string
	DocPart       string
	Range         document.Range
	HasCloseBrace bool
}

func DetectPartial(text string, pos document.Position) *PartialElement {
	lines := strings.Split(text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	col := pos.Character
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	if inYamlBlock(lines, pos.Line) {
		key := strings.TrimSpace(prefix)
		if !strings.Contains(key, ":") {
			return &PartialElement{
				Kind:  PartialYamlKey,
				Input: key,
			}
		}
		return nil
	}

	// Detect block attribute completion ({key=" pattern on standalone line)
	trimmed := strings.TrimSpace(prefix)
	if strings.HasPrefix(trimmed, "{") && !strings.Contains(trimmed, "}") && !strings.Contains(line, "#") {
		input := strings.TrimPrefix(trimmed, "{")
		startChar := strings.Index(line, "{") + 1
		endChar := col
		return &PartialElement{
			Kind:  PartialBlockAttr,
			Input: input,
			Range: document.Range{
				Start: document.Position{Line: pos.Line, Character: startChar},
				End:   document.Position{Line: pos.Line, Character: endChar},
			},
		}
	}

	// Detect attribute class completion ({. pattern) - check before heading text
	if idx := strings.LastIndex(prefix, "{."); idx >= 0 {
		if !strings.Contains(prefix[idx:], "}") {
			input := prefix[idx+2:]
			startChar := idx + 2
			hasClose := false
			if col < len(line) {
				suffix := line[col:]
				if bi := strings.Index(suffix, "}"); bi >= 0 && strings.TrimSpace(suffix[:bi]) == "" {
					hasClose = true
				}
			}
			return &PartialElement{
				Kind:          PartialAttrClass,
				Input:         input,
				HasCloseBrace: hasClose,
				Range: document.Range{
					Start: document.Position{Line: pos.Line, Character: startChar},
					End:   document.Position{Line: pos.Line, Character: col},
				},
			}
		}
	}

	// Detect heading text completion (## prefix)
	trimmed = strings.TrimSpace(prefix)
	if strings.HasPrefix(trimmed, "##") && !strings.Contains(prefix, "[") {
		after := strings.TrimPrefix(trimmed, "##")
		after = strings.TrimPrefix(after, " ")
		return &PartialElement{
			Kind:  PartialHeadingText,
			Input: after,
		}
	}

	if idx := strings.LastIndex(prefix, "]("); idx >= 0 {
		if !strings.Contains(prefix[idx:], ")") {
			content := prefix[idx+2:]
			if hashIdx := strings.Index(content, "#"); hashIdx >= 0 {
				return &PartialElement{
					Kind:    PartialInlineAnchor,
					DocPart: content[:hashIdx],
					Input:   content[hashIdx+1:],
				}
			}
			return &PartialElement{
				Kind:  PartialInlineLink,
				Input: content,
			}
		}
	}

	if idx := strings.LastIndex(prefix, "["); idx >= 0 {
		if !strings.Contains(prefix[idx:], "]") && !strings.HasPrefix(prefix[idx:], "[[") {
			content := prefix[idx+1:]
			if !strings.HasPrefix(content, "^") {
				suffix := line[col:]
				endCol := col
				if len(suffix) > 0 && suffix[0] == ']' {
					endCol = col + 1
				}
				return &PartialElement{
					Kind:  PartialKeyref,
					Input: content,
					Range: document.Range{
						Start: document.Position{Line: pos.Line, Character: idx},
						End:   document.Position{Line: pos.Line, Character: endCol},
					},
				}
			}
		}
	}

	return nil
}

func inYamlBlock(lines []string, lineNum int) bool {
	if lineNum == 0 {
		return false
	}
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}
	for i := 1; i < lineNum; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" || trimmed == "..." {
			return false
		}
	}
	return true
}
