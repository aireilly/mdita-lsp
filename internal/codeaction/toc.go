package codeaction

import (
	"fmt"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

func GenerateToC(doc *document.Document, levels []int) string {
	levelSet := make(map[int]bool)
	for _, l := range levels {
		levelSet[l] = true
	}

	headings := doc.Index.Headings()
	var filtered []*document.Heading
	for _, h := range headings {
		if levelSet[h.Level] {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) == 0 {
		return ""
	}

	minLevel := filtered[0].Level
	for _, h := range filtered {
		if h.Level < minLevel {
			minLevel = h.Level
		}
	}

	var sb strings.Builder
	sb.WriteString("<!--toc:start-->\n")
	for _, h := range filtered {
		indent := strings.Repeat("  ", h.Level-minLevel)
		slug := paths.Slugify(h.Text)
		sb.WriteString(fmt.Sprintf("%s- [%s](#%s)\n", indent, h.Text, slug))
	}
	sb.WriteString("<!--toc:end-->")

	return sb.String()
}
