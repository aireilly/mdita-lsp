package hover

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func GetHover(doc *document.Document, pos document.Position, folder *workspace.Folder) string {
	if h := hoverYAMLKey(doc, pos); h != "" {
		return h
	}

	elem := doc.ElementAt(pos)
	if elem != nil {
		switch el := elem.(type) {
		case *document.WikiLink:
			return hoverWikiLink(el, doc, folder)
		case *document.MdLink:
			return hoverMdLink(el, doc, folder)
		case *document.Heading:
			return "**" + el.Text + "** (level " + itoa(el.Level) + ")"
		}
	}

	if kr := keyref.DetectAtPosition(doc.Text, pos); kr != nil {
		return hoverKeyref(kr, folder)
	}

	return ""
}

var yamlKeyDocs = map[string]string{
	"$schema":     "DITA topic type schema. Values: `urn:oasis:names:tc:mdita:xsd:topic.xsd` (core), `urn:oasis:names:tc:mdita:extended:rng:topic.rng` (extended)",
	"author":      "Topic author name",
	"source":      "Original source of the content",
	"publisher":   "Publisher of the content",
	"permissions":  "Access permissions for this topic",
	"audience":    "Intended audience for this topic",
	"category":    "Topic category for classification",
	"keyword":     "Keywords for indexing and search (comma-separated or YAML list)",
	"resourceid":  "Unique resource identifier for cross-references",
}

func hoverYAMLKey(doc *document.Document, pos document.Position) string {
	if doc.Meta == nil {
		return ""
	}
	if !posInYAMLRange(pos, doc.Meta.Range) {
		return ""
	}

	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	colonIdx := strings.Index(line, ":")
	if colonIdx < 0 {
		return ""
	}
	key := strings.TrimSpace(line[:colonIdx])
	if key == "---" || key == "" {
		return ""
	}

	if desc, ok := yamlKeyDocs[key]; ok {
		value := strings.TrimSpace(line[colonIdx+1:])
		result := "**" + key + "** (MDITA metadata)\n\n" + desc
		if value != "" {
			result += "\n\nCurrent value: `" + value + "`"
		}
		return result
	}
	return ""
}

func posInYAMLRange(pos document.Position, r document.Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	return true
}

func hoverWikiLink(wl *document.WikiLink, doc *document.Document, folder *workspace.Folder) string {
	if wl.Doc == "" && wl.Heading != "" {
		slug := paths.SlugOf(wl.Heading)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return "**" + h.Text + "**"
		}
		return ""
	}

	targetSlug := paths.SlugOf(wl.Doc)
	target := folder.DocBySlug(targetSlug)
	if target == nil {
		return ""
	}

	title := target.Index.Title()
	if title != nil {
		result := "**" + title.Text + "**"
		if wl.Heading != "" {
			hslug := paths.SlugOf(wl.Heading)
			for _, h := range target.Index.HeadingsBySlug(hslug) {
				result += " > " + h.Text
			}
		}
		return result
	}
	return ""
}

func hoverKeyref(kr *keyref.KeyrefAtPos, folder *workspace.Folder) string {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, kr.Label)
	if !ok {
		return ""
	}
	result := "**" + kr.Label + "** (keyref)"
	if entry.Title != "" {
		result += "\n\nTarget: " + entry.Title + " (" + entry.Href + ")"
	} else {
		result += "\n\nTarget: " + entry.Href
	}
	return result
}

func hoverMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) string {
	if ml.URL == "" && ml.Anchor != "" {
		slug := paths.SlugOf(ml.Anchor)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return "**" + h.Text + "**"
		}
	}
	if ml.URL != "" {
		target := resolveRelativeLink(ml.URL, doc, folder)
		if target != nil {
			title := target.Index.Title()
			if title != nil {
				result := "**" + title.Text + "** (" + ml.URL + ")"
				if ml.Anchor != "" {
					hslug := paths.SlugOf(ml.Anchor)
					for _, h := range target.Index.HeadingsBySlug(hslug) {
						result += " > " + h.Text
					}
				}
				return result
			}
		}
	}
	return ml.URL
}

func resolveRelativeLink(url string, doc *document.Document, folder *workspace.Folder) *document.Document {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return nil
	}
	srcPath, _ := paths.URIToPath(doc.URI)
	srcDir := filepath.Dir(srcPath)
	targetPath := filepath.Clean(filepath.Join(srcDir, url))
	targetURI := paths.PathToURI(targetPath)
	if d := folder.DocByURI(targetURI); d != nil {
		return d
	}
	for _, d := range folder.AllDocs() {
		id := d.DocID(folder.RootURI)
		if paths.MatchesURL(id, url) {
			return d
		}
	}
	return nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
