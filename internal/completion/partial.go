package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type PartialKind int

const (
	PartialWikiLink PartialKind = iota
	PartialWikiHeading
	PartialInlineLink
	PartialInlineAnchor
	PartialRefLink
	PartialYamlKey
	PartialKeyref
)

type PartialElement struct {
	Kind    PartialKind
	Input   string
	DocPart string
	Range   document.Range
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

	if idx := strings.LastIndex(prefix, "[["); idx >= 0 {
		if !strings.Contains(prefix[idx:], "]]") {
			content := prefix[idx+2:]
			if hashIdx := strings.Index(content, "#"); hashIdx >= 0 {
				return &PartialElement{
					Kind:    PartialWikiHeading,
					DocPart: content[:hashIdx],
					Input:   content[hashIdx+1:],
				}
			}
			return &PartialElement{
				Kind:  PartialWikiLink,
				Input: content,
			}
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
				return &PartialElement{
					Kind:  PartialKeyref,
					Input: content,
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
