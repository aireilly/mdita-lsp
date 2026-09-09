package diagnostic

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func checkLinks(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	for _, ml := range doc.Index.MdLinks() {
		if ml.URL == "" && ml.Anchor != "" {
			if d := checkFragment(ml, doc, doc, ""); d != nil {
				diags = append(diags, *d)
			}
		}
		if ml.URL != "" && !strings.HasPrefix(ml.URL, "http://") && !strings.HasPrefix(ml.URL, "https://") {
			if candidates := folder.ResolveLinkCandidates(ml.URL, doc.URI); len(candidates) > 1 {
				diags = append(diags, Diagnostic{
					Range:    ml.Range,
					Severity: SeverityWarning,
					Code:     CodeAmbiguousLink,
					Source:   source,
					Message:  "Link '" + ml.URL + "' matches " + itoa(len(candidates)) + " files; use a path relative to this file",
				})
			}
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
				if d := checkFragment(ml, doc, target, ml.URL); d != nil {
					diags = append(diags, *d)
				}
			}
		}
	}

	return diags
}

// checkFragment validates a link fragment against its target. A fragment of
// the form "topic-id/element-id" is DITA element addressing, which the plug-in
// passes straight through to @href; a plain fragment addresses a heading.
func checkFragment(ml *document.MdLink, src, target *document.Document, url string) *Diagnostic {
	in := ""
	if url != "" {
		in = " in '" + url + "'"
	}

	topicID, elementID, isDita := document.SplitFragment(ml.Anchor)
	if isDita {
		if got := target.TopicID(); got != "" && got != topicID {
			return &Diagnostic{
				Range:    ml.Range,
				Severity: SeverityError,
				Code:     CodeBrokenLink,
				Source:   source,
				Message:  "Link to topic '" + topicID + "'" + in + ", which has the id '" + got + "'",
			}
		}
		if !target.HasElementID(elementID) {
			return &Diagnostic{
				Range:    ml.Range,
				Severity: SeverityError,
				Code:     CodeBrokenLink,
				Source:   source,
				Message:  "Link to non-existent element '" + elementID + "'" + in,
			}
		}
		return nil
	}

	if len(target.Index.HeadingsBySlug(paths.SlugOf(ml.Anchor))) > 0 {
		return nil
	}
	// The plug-in also accepts a bare topic id as a fragment.
	if target.TopicID() == ml.Anchor {
		return nil
	}
	return &Diagnostic{
		Range:    ml.Range,
		Severity: SeverityError,
		Code:     CodeBrokenLink,
		Source:   source,
		Message:  "Link to non-existent heading '#" + ml.Anchor + "'" + in,
	}
}

func checkNonBreakingWhitespace(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, h := range doc.Index.Headings() {
		if containsNBSP(h.Text) {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeNBSP,
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
