package inlayhint

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
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
		case *document.WikiLink:
			if el.Doc != "" {
				target := folder.DocBySlug(paths.SlugOf(el.Doc))
				if target != nil {
					if t := target.Index.Title(); t != nil && t.Text != el.Doc {
						hints = append(hints, InlayHint{
							Position: document.Position{
								Line:      el.Range.End.Line,
								Character: el.Range.End.Character,
							},
							Label: " → " + t.Text,
							Kind:  KindType,
						})
					}
				}
			}
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
