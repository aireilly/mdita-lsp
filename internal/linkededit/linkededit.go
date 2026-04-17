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

	var ranges []document.Range
	ranges = append(ranges, heading.Range)

	for _, wl := range doc.Index.WikiLinks() {
		if wl.Doc == "" && wl.Heading == heading.Text {
			ranges = append(ranges, wl.Range)
		}
	}

	if len(ranges) <= 1 {
		return nil
	}

	return &LinkedEditingRanges{Ranges: ranges}
}
