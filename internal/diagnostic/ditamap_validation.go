package diagnostic

import (
	"path/filepath"
	"strconv"

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
	diags = append(diags, checkMapHeadingHierarchy(m, doc, folder)...)
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

func checkMapHeadingHierarchy(m *ditamap.MapStructure, doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	docPath, _ := paths.URIToPath(doc.URI)
	docDir := filepath.Dir(docPath)
	walkTopicRefHierarchy(m.TopicRefs, 0, docDir, folder, &diags)
	return diags
}

func walkTopicRefHierarchy(refs []ditamap.TopicRef, depth int, docDir string, folder *workspace.Folder, diags *[]Diagnostic) {
	for _, ref := range refs {
		if ref.Href == "" {
			continue
		}
		targetPath := filepath.Join(docDir, ref.Href)
		targetURI := paths.PathToURI(targetPath)
		targetDoc := folder.DocByURI(targetURI)
		if targetDoc == nil {
			continue
		}
		title := targetDoc.Index.Title()
		if title == nil {
			continue
		}
		expectedLevel := depth + 1
		if title.Level != expectedLevel && depth > 0 {
			*diags = append(*diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeInconsistentMapHeadingHierarchy,
				Source:   source,
				Message:  "Topic " + ref.Href + " has heading level " + itoa(title.Level) + " but map nesting suggests level " + itoa(expectedLevel),
			})
		}
		walkTopicRefHierarchy(ref.Children, depth+1, docDir, folder, diags)
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
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
