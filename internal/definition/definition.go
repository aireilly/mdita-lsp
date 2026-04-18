package definition

import (
	"path/filepath"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Location struct {
	URI   string
	Range document.Range
}

func GotoDef(doc *document.Document, pos document.Position, folder *workspace.Folder) []Location {
	elem := doc.ElementAt(pos)
	if elem != nil {
		switch el := elem.(type) {
		case *document.MdLink:
			return resolveMdLink(el, doc, folder)
		}
	}

	if kr := keyref.DetectAtPosition(doc.Text, pos); kr != nil {
		return resolveKeyref(kr, doc, folder)
	}

	return nil
}

func resolveMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) []Location {
	if ml.URL == "" && ml.Anchor != "" {
		slug := paths.SlugOf(ml.Anchor)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return []Location{{URI: doc.URI, Range: h.Range}}
		}
		return nil
	}

	if ml.URL != "" {
		target := folder.ResolveLink(ml.URL, doc.URI)
		if target != nil {
			if ml.Anchor != "" {
				hslug := paths.SlugOf(ml.Anchor)
				for _, h := range target.Index.HeadingsBySlug(hslug) {
					return []Location{{URI: target.URI, Range: h.Range}}
				}
			}
			title := target.Index.Title()
			if title != nil {
				return []Location{{URI: target.URI, Range: title.Range}}
			}
			return []Location{{URI: target.URI, Range: document.Rng(0, 0, 0, 0)}}
		}
	}

	return nil
}

func resolveKeyref(kr *keyref.KeyrefAtPos, doc *document.Document, folder *workspace.Folder) []Location {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, kr.Label)
	if !ok {
		return nil
	}

	for _, mapDoc := range folder.AllDocs() {
		if mapDoc.Kind != document.Map {
			continue
		}
		mapPath, _ := paths.URIToPath(mapDoc.URI)
		mapDir := filepath.Dir(mapPath)
		targetPath := filepath.Join(mapDir, entry.Href)
		targetURI := paths.PathToURI(targetPath)
		target := folder.DocByURI(targetURI)
		if target != nil {
			title := target.Index.Title()
			if title != nil {
				return []Location{{URI: target.URI, Range: title.Range}}
			}
			return []Location{{URI: target.URI, Range: document.Rng(0, 0, 0, 0)}}
		}
	}
	return nil
}
