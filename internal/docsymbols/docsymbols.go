package docsymbols

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type DocSymbol struct {
	Name     string
	Kind     int // 5=class (file), 23=struct (section)
	Range    document.Range
	Children []DocSymbol
}

func GetSymbols(doc *document.Document) []DocSymbol {
	headings := doc.Index.Headings()
	if len(headings) == 0 {
		return nil
	}

	var root []DocSymbol
	var stack []*[]DocSymbol
	var levels []int

	stack = append(stack, &root)
	levels = append(levels, 0)

	for _, h := range headings {
		sym := DocSymbol{
			Name:  h.Text,
			Kind:  23,
			Range: h.Range,
		}
		if h.IsTitle() {
			sym.Kind = 5
		}

		for len(levels) > 1 && h.Level <= levels[len(levels)-1] {
			stack = stack[:len(stack)-1]
			levels = levels[:len(levels)-1]
		}

		parent := stack[len(stack)-1]
		*parent = append(*parent, sym)
		idx := len(*parent) - 1
		stack = append(stack, &(*parent)[idx].Children)
		levels = append(levels, h.Level)
	}

	return root
}

func SearchWorkspace(docs []*document.Document, query string) []DocSymbol {
	query = strings.ToLower(query)
	var results []DocSymbol
	for _, doc := range docs {
		for _, h := range doc.Index.Headings() {
			if strings.Contains(strings.ToLower(h.Text), query) {
				results = append(results, DocSymbol{
					Name:  h.Text,
					Kind:  23,
					Range: h.Range,
				})
			}
		}
	}
	return results
}
