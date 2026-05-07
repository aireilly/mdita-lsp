package inlayhint

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/vocabulary"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

const (
	KindType      = 1
	KindParameter = 2
)

type InlayHint struct {
	Position document.Position `json:"position"`
	Label    string            `json:"label"`
	Kind     int               `json:"kind"`
}

func GetHints(doc *document.Document, rng document.Range, folder *workspace.Folder) []InlayHint {
	table := keyref.BuildMergedTable(folder.MapTexts())
	var hints []InlayHint

	for _, elem := range doc.Elements {
		elemRange := elem.Rng()
		if elemRange.End.Line < rng.Start.Line || elemRange.Start.Line > rng.End.Line {
			continue
		}

		switch el := elem.(type) {
		case *document.MdLink:
			if el.URL == "" {
				continue
			}
			if label := mdLinkHintLabel(el, doc, folder); label != "" {
				hints = append(hints, InlayHint{
					Position: document.Position{
						Line:      el.Range.End.Line,
						Character: el.Range.End.Character,
					},
					Label: " → " + label,
					Kind:  KindType,
				})
			}
		}
	}

	hints = append(hints, keyrefHints(doc, rng, table)...)
	hints = append(hints, domainHints(doc, rng)...)
	return hints
}

func mdLinkHintLabel(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) string {
	target := folder.ResolveLink(ml.URL, doc.URI)
	if target == nil {
		return ""
	}
	if t := target.Index.Title(); t != nil && t.Text != ml.Text {
		return t.Text
	}
	return ""
}

func keyrefHints(doc *document.Document, rng document.Range, table keyref.KeyTable) []InlayHint {
	defs := keyref.DetectAll(doc.Text)
	var hints []InlayHint
	for _, d := range defs {
		if d.Line < rng.Start.Line || d.Line > rng.End.Line {
			continue
		}
		entry, ok := table[d.Key]
		if !ok {
			continue
		}
		label := entry.Href
		if entry.Title != "" {
			label = entry.Title
		}
		hints = append(hints, InlayHint{
			Position: document.Position{
				Line:      d.Line,
				Character: d.EndChar,
			},
			Label: " → " + label,
			Kind:  KindType,
		})
	}
	return hints
}

func domainHints(doc *document.Document, rng document.Range) []InlayHint {
	var hints []InlayHint
	for _, ia := range doc.InlineAttrs {
		if ia.Line < rng.Start.Line || ia.Line > rng.End.Line {
			continue
		}
		for _, class := range ia.Attr.Classes {
			if elem, ok := vocabulary.LookupDomainElement(class); ok {
				hints = append(hints, InlayHint{
					Position: document.Position{
						Line:      ia.Line,
						Character: ia.Attr.Range.End.Character,
					},
					Label: " → <" + elem.DITAElement + ">",
					Kind:  KindType,
				})
			}
		}
	}
	return hints
}
