package diagnostic

import (
	"path/filepath"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func CheckDitamap(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	if doc.Kind != document.Map {
		return nil
	}

	m, err := ditamap.ParseMap(doc.Text)
	if err != nil {
		return nil
	}

	var diags []Diagnostic
	diags = append(diags, checkMapRefs(m, doc, folder)...)
	diags = append(diags, checkCircularRefs(doc, folder)...)
	return diags
}

func checkMapRefs(m *ditamap.MapStructure, doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	docPath, _ := paths.URIToPath(doc.URI)
	docDir := filepath.Dir(docPath)

	for _, href := range m.AllHrefs() {
		targetPath := filepath.Join(docDir, href)
		targetURI := paths.PathToURI(targetPath)
		if folder.DocByURI(targetURI) == nil {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityError,
				Code:     CodeBrokenMapReference,
				Source:   source,
				Message:  "Map references non-existent file: " + href,
			})
		}
	}
	return diags
}

func checkCircularRefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	visited := make(map[string]bool)
	if hasCycle(doc.URI, folder, visited) {
		return []Diagnostic{{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityError,
			Code:     CodeCircularMapReference,
			Source:   source,
			Message:  "Circular map reference detected",
		}}
	}
	return nil
}

func hasCycle(uri string, folder *workspace.Folder, visited map[string]bool) bool {
	if visited[uri] {
		return true
	}
	visited[uri] = true

	doc := folder.DocByURI(uri)
	if doc == nil || doc.Kind != document.Map {
		delete(visited, uri)
		return false
	}

	m, err := ditamap.ParseMap(doc.Text)
	if err != nil {
		delete(visited, uri)
		return false
	}

	docPath, _ := paths.URIToPath(uri)
	docDir := filepath.Dir(docPath)
	mapExts := folder.Config.Core.Mdita.MapExtensions

	for _, href := range m.AllHrefs() {
		if !paths.IsMditaMapFile(href, mapExts) {
			continue
		}
		targetPath := filepath.Join(docDir, href)
		targetURI := paths.PathToURI(targetPath)
		if hasCycle(targetURI, folder, visited) {
			return true
		}
	}

	delete(visited, uri)
	return false
}
