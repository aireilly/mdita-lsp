// Package conref provides parsing and resolution of DITA content references
// (conref and conkeyref) expressed as HTML data attributes in MDITA documents.
package conref

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

var conrefRe = regexp.MustCompile(`<(\w+)\s[^>]*data-conref="([^"]+)"[^>]*>`)
var conkeyrefRe = regexp.MustCompile(`<(\w+)\s[^>]*data-conkeyref="([^"]+)"[^>]*>`)

// Parse scans text for HTML elements carrying data-conref or data-conkeyref
// attributes and returns a ConrefElement for each match found.
func Parse(text string) []*document.ConrefElement {
	var refs []*document.ConrefElement
	lines := strings.Split(text, "\n")

	for lineNum, line := range lines {
		for _, m := range conrefRe.FindAllStringSubmatchIndex(line, -1) {
			tag := line[m[2]:m[3]]
			value := line[m[4]:m[5]]
			ce := parseConrefValue(value)
			ce.Tag = tag
			ce.Range = document.Rng(lineNum, m[0], lineNum, m[1])
			refs = append(refs, ce)
		}

		for _, m := range conkeyrefRe.FindAllStringSubmatchIndex(line, -1) {
			tag := line[m[2]:m[3]]
			value := line[m[4]:m[5]]
			ce := parseConkeyrefValue(value)
			ce.Tag = tag
			ce.IsKeyref = true
			ce.Range = document.Rng(lineNum, m[0], lineNum, m[1])
			refs = append(refs, ce)
		}
	}
	return refs
}

func parseConrefValue(value string) *document.ConrefElement {
	ce := &document.ConrefElement{}
	hashIdx := strings.Index(value, "#")
	if hashIdx < 0 {
		ce.FilePath = value
		return ce
	}
	ce.FilePath = value[:hashIdx]
	rest := value[hashIdx+1:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		ce.TopicID = rest
		return ce
	}
	ce.TopicID = rest[:slashIdx]
	ce.ElementID = rest[slashIdx+1:]
	return ce
}

func parseConkeyrefValue(value string) *document.ConrefElement {
	ce := &document.ConrefElement{}
	slashIdx := strings.Index(value, "/")
	if slashIdx < 0 {
		ce.KeyName = value
		return ce
	}
	ce.KeyName = value[:slashIdx]
	ce.ElementID = value[slashIdx+1:]
	return ce
}

// Resolve returns the target document URI for a conref element.
// For conrefs it resolves the file path relative to the source document.
// For conkeyrefs it resolves the key name through the workspace key table,
// then navigates to the target file.
func Resolve(ce *document.ConrefElement, doc *document.Document, folder *workspace.Folder) (string, error) {
	if ce.IsKeyref {
		return resolveConkeyref(ce, folder)
	}
	return resolveConref(ce, doc, folder)
}

func resolveConref(ce *document.ConrefElement, doc *document.Document, folder *workspace.Folder) (string, error) {
	srcPath, err := paths.URIToPath(doc.URI)
	if err != nil {
		return "", err
	}
	srcDir := filepath.Dir(srcPath)
	targetPath := filepath.Join(srcDir, ce.FilePath)
	targetURI := paths.PathToURI(targetPath)
	target := folder.DocByURI(targetURI)
	if target == nil {
		return "", nil
	}
	return targetURI, nil
}

func resolveConkeyref(ce *document.ConrefElement, folder *workspace.Folder) (string, error) {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, ce.KeyName)
	if !ok || entry.Href == "" {
		return "", nil
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
			return targetURI, nil
		}
	}
	return "", nil
}
