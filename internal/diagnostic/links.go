package diagnostic

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func checkLinks(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	for _, wl := range doc.Index.WikiLinks() {
		if wl.Doc == "" && wl.Heading != "" {
			slug := paths.SlugOf(wl.Heading)
			if len(doc.Index.HeadingsBySlug(slug)) == 0 {
				diags = append(diags, Diagnostic{
					Range:    wl.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent heading '" + wl.Heading + "'",
				})
			}
			continue
		}

		if wl.Doc != "" {
			slug := paths.SlugOf(wl.Doc)
			target := folder.DocBySlug(slug)
			if target == nil {
				diags = append(diags, Diagnostic{
					Range:    wl.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent document '" + wl.Doc + "'",
				})
			} else if wl.Heading != "" {
				hslug := paths.SlugOf(wl.Heading)
				if len(target.Index.HeadingsBySlug(hslug)) == 0 {
					diags = append(diags, Diagnostic{
						Range:    wl.Range,
						Severity: SeverityError,
						Code:     CodeBrokenLink,
						Source:   source,
						Message:  "Link to non-existent heading '" + wl.Heading + "' in '" + wl.Doc + "'",
					})
				}
			}
		}
	}

	for _, ml := range doc.Index.MdLinks() {
		if ml.URL == "" && ml.Anchor != "" {
			slug := paths.SlugOf(ml.Anchor)
			if len(doc.Index.HeadingsBySlug(slug)) == 0 {
				diags = append(diags, Diagnostic{
					Range:    ml.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent heading '#" + ml.Anchor + "'",
				})
			}
		}
		if ml.URL != "" && !strings.HasPrefix(ml.URL, "http://") && !strings.HasPrefix(ml.URL, "https://") {
			target := folder.ResolveLink(ml.URL, doc.URI)
			if target == nil {
				diags = append(diags, Diagnostic{
					Range:    ml.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent file '" + ml.URL + "'",
				})
			} else if ml.Anchor != "" {
				hslug := paths.SlugOf(ml.Anchor)
				if len(target.Index.HeadingsBySlug(hslug)) == 0 {
					diags = append(diags, Diagnostic{
						Range:    ml.Range,
						Severity: SeverityError,
						Code:     CodeBrokenLink,
						Source:   source,
						Message:  "Link to non-existent heading '#" + ml.Anchor + "' in '" + ml.URL + "'",
					})
				}
			}
		}
	}

	return diags
}

func checkNonBreakingWhitespace(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, h := range doc.Index.Headings() {
		if containsNBSP(h.Text) {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeNonBreakingWhitespace,
				Source:   source,
				Message:  "Heading contains non-breaking whitespace",
			})
		}
	}
	return diags
}

func containsNBSP(s string) bool {
	return strings.ContainsRune(s, '\u00A0')
}
