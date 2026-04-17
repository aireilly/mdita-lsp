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
	case *document.WikiLink:
		return highlightWikiLink(doc, el)
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

	for _, wl := range doc.Index.WikiLinks() {
		if wl.Doc == "" && paths.SlugOf(wl.Heading) == heading.Slug {
			highlights = append(highlights, Highlight{
				Range: wl.Range,
				Kind:  KindRead,
			})
		}
	}

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

func highlightWikiLink(doc *document.Document, wl *document.WikiLink) []Highlight {
	var highlights []Highlight

	if wl.Doc == "" && wl.Heading != "" {
		slug := paths.SlugOf(wl.Heading)
		for _, h := range doc.Index.Headings() {
			if h.Slug == slug {
				highlights = append(highlights, Highlight{
					Range: h.Range,
					Kind:  KindWrite,
				})
			}
		}
		for _, other := range doc.Index.WikiLinks() {
			if other.Doc == "" && paths.SlugOf(other.Heading) == slug {
				highlights = append(highlights, Highlight{
					Range: other.Range,
					Kind:  KindRead,
				})
			}
		}
	} else if wl.Doc != "" {
		slug := paths.SlugOf(wl.Doc)
		for _, other := range doc.Index.WikiLinks() {
			if paths.SlugOf(other.Doc) == slug {
				highlights = append(highlights, Highlight{
					Range: other.Range,
					Kind:  KindRead,
				})
			}
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
