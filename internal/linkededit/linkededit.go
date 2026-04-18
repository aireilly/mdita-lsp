package linkededit

import "github.com/aireilly/mdita-lsp/internal/document"

type LinkedEditingRanges struct {
	Ranges []document.Range `json:"ranges"`
}

func GetLinkedRanges(doc *document.Document, pos document.Position) *LinkedEditingRanges {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}

	return &LinkedEditingRanges{Ranges: []document.Range{heading.Range}}
}
