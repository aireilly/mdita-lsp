package diagnostic

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

// checkMapRefs reports a topicref whose target is not there.
//
// It used to report every href it could not find in the document index, so an
// external URL, a bare fragment, an image and a .ditamap were all flagged as
// missing files -- and every one of them at line 0, column 0. Only a
// workspace-relative href is checked now, the filesystem settles anything the
// server does not index, and the diagnostic points at the topicref's own line.
func checkMapRefs(m *ditamap.MapStructure, doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	docPath, err := paths.URIToPath(doc.URI)
	if err != nil {
		return nil
	}
	docDir := filepath.Dir(docPath)
	lines := strings.Split(doc.Text, "\n")

	for _, ref := range m.AllRefs() {
		href := ref.Href
		if !isWorkspaceHref(href) {
			continue
		}
		target := filepath.Join(docDir, decodeHref(stripFragment(href)))
		if folder.DocByURI(paths.PathToURI(target)) != nil {
			continue
		}
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			continue
		}
		diags = append(diags, Diagnostic{
			Range:    document.Rng(ref.Line, 0, ref.Line, lineLength(lines, ref.Line)),
			Severity: SeverityError,
			Code:     CodeBrokenMapTopicref,
			Source:   source,
			Message:  "Map references non-existent file: " + href,
		})
	}
	return diags
}

// isWorkspaceHref reports whether an href names a file in the workspace. A
// bare fragment addresses this map, and an href with a scheme or a leading
// slash leaves the source tree.
func isWorkspaceHref(href string) bool {
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "/") {
		return false
	}
	return !hasURIScheme(href)
}

// lineLength gives a diagnostic's range the extent of the line it sits on,
// in UTF-16 units.
func lineLength(lines []string, line int) int {
	if line < 0 || line >= len(lines) {
		return 0
	}
	return document.UTF16Len(lines[line])
}

func stripFragment(href string) string {
	if i := strings.Index(href, "#"); i >= 0 {
		return href[:i]
	}
	return href
}

func decodeHref(href string) string {
	decoded, err := url.PathUnescape(href)
	if err != nil {
		return href
	}
	return decoded
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

	for _, href := range m.AllHrefs() {
		if !isWorkspaceHref(href) {
			continue
		}
		targetPath := filepath.Join(docDir, decodeHref(stripFragment(href)))
		targetURI := paths.PathToURI(targetPath)
		// Following only .mditamap files missed a cycle through a .md map,
		// which a document declares with $schema: …map.xsd. Document.Kind
		// already knows which files are maps, so ask it rather than the
		// extension.
		target := folder.DocByURI(targetURI)
		if target == nil || target.Kind != document.Map {
			continue
		}
		if hasCycle(targetURI, folder, visited) {
			return true
		}
	}

	delete(visited, uri)
	return false
}
