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
		case *document.ConrefElement:
			return resolveConref(el, doc, folder)
		}
	}

	if kr := keyref.DetectAtPosition(doc.Text, pos); kr != nil {
		return resolveKeyref(kr, doc, folder)
	}

	return nil
}

func resolveMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) []Location {
	if ml.URL == "" && ml.Anchor != "" {
		if loc := resolveFragment(ml.Anchor, doc); loc != nil {
			return []Location{*loc}
		}
		return nil
	}

	if ml.URL != "" {
		target := folder.ResolveLink(ml.URL, doc.URI)
		if target != nil {
			if ml.Anchor != "" {
				if loc := resolveFragment(ml.Anchor, target); loc != nil {
					return []Location{*loc}
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

// resolveFragment locates the heading a link fragment addresses, accepting
// both the "topic-id/element-id" DITA form and a plain heading slug.
func resolveFragment(anchor string, target *document.Document) *Location {
	_, elementID, isDita := document.SplitFragment(anchor)
	if isDita {
		for _, e := range target.Elements {
			if h, ok := e.(*document.Heading); ok && h.ID == elementID {
				return &Location{URI: target.URI, Range: h.Range}
			}
		}
		return nil
	}
	for _, h := range target.Index.HeadingsBySlug(paths.SlugOf(anchor)) {
		return &Location{URI: target.URI, Range: h.Range}
	}
	return nil
}

func resolveConref(ce *document.ConrefElement, doc *document.Document, folder *workspace.Folder) []Location {
	if ce.IsKeyref {
		return resolveConkeyref(ce, folder)
	}

	srcPath, err := paths.URIToPath(doc.URI)
	if err != nil {
		return nil
	}
	srcDir := filepath.Dir(srcPath)
	targetPath := filepath.Join(srcDir, ce.FilePath)
	targetURI := paths.PathToURI(targetPath)
	targetDoc := folder.DocByURI(targetURI)
	if targetDoc == nil {
		return nil
	}

	if ce.ElementID != "" {
		for _, el := range targetDoc.Elements {
			if h, ok := el.(*document.Heading); ok && h.ID == ce.ElementID {
				return []Location{{URI: targetURI, Range: h.Range}}
			}
		}
	}

	title := targetDoc.Index.Title()
	if title != nil {
		return []Location{{URI: targetURI, Range: title.Range}}
	}
	return []Location{{URI: targetURI, Range: document.Rng(0, 0, 0, 0)}}
}

func resolveConkeyref(ce *document.ConrefElement, folder *workspace.Folder) []Location {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, ce.KeyName)
	if !ok || entry.Href == "" {
		return nil
	}
	for _, mapDoc := range folder.AllDocs() {
		if mapDoc.Kind != document.Map {
			continue
		}
		mapPath, err := paths.URIToPath(mapDoc.URI)
		if err != nil {
			continue
		}
		mapDir := filepath.Dir(mapPath)
		targetPath := filepath.Join(mapDir, entry.Href)
		targetURI := paths.PathToURI(targetPath)
		target := folder.DocByURI(targetURI)
		if target != nil {
			title := target.Index.Title()
			if title != nil {
				return []Location{{URI: targetURI, Range: title.Range}}
			}
			return []Location{{URI: targetURI, Range: document.Rng(0, 0, 0, 0)}}
		}
	}
	return nil
}

func resolveKeyref(kr *keyref.KeyrefAtPos, doc *document.Document, folder *workspace.Folder) []Location {
	// Prefer navigating directly to the LinkDef in the map file.
	for _, mapDoc := range folder.AllDocs() {
		if mapDoc.Kind != document.Map {
			continue
		}
		for _, el := range mapDoc.Elements {
			if ld, ok := el.(*document.LinkDef); ok && ld.Label == kr.Label {
				return []Location{{URI: mapDoc.URI, Range: ld.Range}}
			}
		}
	}

	// Fall back: href-based keydef — navigate to the referenced topic file.
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, kr.Label)
	if !ok {
		return nil
	}
	if entry.Href == "" {
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
