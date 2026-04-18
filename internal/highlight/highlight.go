package highlight

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

const (
	KindText  = 1
	KindRead  = 2
	KindWrite = 3
)

type Highlight struct {
	Range document.Range `json:"range"`
	Kind  int            `json:"kind"`
}

func GetHighlights(doc *document.Document, pos document.Position) []Highlight {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	switch el := elem.(type) {
	case *document.Heading:
		return highlightHeading(doc, el)
	case *document.MdLink:
		return highlightMdLink(doc, el)
	}
	return nil
}

func highlightHeading(doc *document.Document, heading *document.Heading) []Highlight {
	var highlights []Highlight

	highlights = append(highlights, Highlight{
		Range: heading.Range,
		Kind:  KindWrite,
	})

	for _, ml := range doc.Index.MdLinks() {
		if ml.URL == "" && paths.SlugOf(ml.Anchor) == heading.Slug {
			highlights = append(highlights, Highlight{
				Range: ml.Range,
				Kind:  KindRead,
			})
		}
	}

	return highlights
}

func highlightMdLink(doc *document.Document, ml *document.MdLink) []Highlight {
	if ml.URL == "" {
		return nil
	}

	var highlights []Highlight
	for _, other := range doc.Index.MdLinks() {
		if other.URL == ml.URL {
			highlights = append(highlights, Highlight{
				Range: other.Range,
				Kind:  KindRead,
			})
		}
	}
	return highlights
}
