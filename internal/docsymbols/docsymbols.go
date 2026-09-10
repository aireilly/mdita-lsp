package docsymbols

import (
	"path"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

// DocSymbol is an LSP DocumentSymbol. The json tags are load-bearing: a
// client decides whether a response holds DocumentSymbol or SymbolInformation
// by testing the first element for lowercase "name", "kind", "range" and
// "selectionRange", then converts the whole array as whichever type it picked.
// Untagged fields serialise as Go names, the test fails, and the client reads
// a location that is not there.
type DocSymbol struct {
	Name           string         `json:"name"`
	Kind           int            `json:"kind"` // 5=class (file), 23=struct (section)
	Range          document.Range `json:"range"`
	SelectionRange document.Range `json:"selectionRange"`
	Children       []DocSymbol    `json:"children,omitempty"`
}

// Location is an LSP Location.
type Location struct {
	URI   string         `json:"uri"`
	Range document.Range `json:"range"`
}

// SymbolInformation is what workspace/symbol answers with. Unlike a
// DocumentSymbol it carries a location, which is what lets the client open the
// file a match lives in.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
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
			Name:           h.Text,
			Kind:           23,
			Range:          h.Range,
			SelectionRange: h.Range,
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

func SearchWorkspace(docs []*document.Document, query string) []SymbolInformation {
	query = strings.ToLower(query)
	var results []SymbolInformation
	for _, doc := range docs {
		container := path.Base(doc.URI)
		for _, h := range doc.Index.Headings() {
			if !strings.Contains(strings.ToLower(h.Text), query) {
				continue
			}
			kind := 23
			if h.IsTitle() {
				kind = 5
			}
			results = append(results, SymbolInformation{
				Name: h.Text,
				Kind: kind,
				Location: Location{
					URI:   doc.URI,
					Range: h.Range,
				},
				ContainerName: container,
			})
		}
	}
	return results
}
