package diagnostic

import (
	"net/url"
	"os"
	"path/filepath"
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
			continue
		}
		if ml.URL == "" || !isCheckableLink(ml.URL) {
			continue
		}

		target := folder.ResolveLink(ml.URL, doc.URI)
		if target == nil {
			if existsOnDisk(ml.URL, doc.URI) {
				// A resource the server does not index -- an image, a .ditamap,
				// a PDF -- is still a valid link target when the file is there.
				continue
			}
			// Naming the same-named files elsewhere turns "broken link" into
			// something the author can act on; the plug-in resolves an href
			// against the source file's directory and nowhere else.
			if candidates := folder.ResolveLinkCandidates(ml.URL, doc.URI); len(candidates) > 0 {
				diags = append(diags, Diagnostic{
					Range:    ml.Range,
					Severity: SeverityError,
					Code:     CodeAmbiguousLink,
					Source:   source,
					Message: "Link '" + ml.URL + "' does not resolve from this file. " +
						itoa(len(candidates)) + " file(s) with that name exist elsewhere; " +
						"use a path relative to this file, such as '" +
						suggestPath(candidates[0], doc) + "'",
				})
				continue
			}
			diags = append(diags, Diagnostic{
				Range:    ml.Range,
				Severity: SeverityError,
				Code:     CodeBrokenLink,
				Source:   source,
				Message:  "Link to non-existent file '" + ml.URL + "'",
			})
			continue
		}

		if ml.Anchor != "" {
			if d := checkFragment(ml, doc, target, ml.URL); d != nil {
				diags = append(diags, *d)
			}
		}
	}

	return diags
}

// suggestPath returns the path from the source file to a candidate target.
func suggestPath(target, src *document.Document) string {
	srcPath, err := paths.URIToPath(src.URI)
	if err != nil {
		return ""
	}
	targetPath, err := paths.URIToPath(target.URI)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(filepath.Dir(srcPath), targetPath)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// isCheckableLink reports whether a link URL names a file in the workspace
// that the server can resolve.
//
// A URL with a scheme leaves the workspace. A root-relative URL does too: the
// plug-in gives "/target.md" scope="external" and never resolves it against
// the source tree, so reporting it as a missing file was wrong.
func isCheckableLink(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	if strings.HasPrefix(rawURL, "//") || strings.HasPrefix(rawURL, "/") {
		return false
	}
	if hasURIScheme(rawURL) {
		return false
	}
	return true
}

// hasURIScheme reports whether the URL starts with "scheme:". A Windows-style
// "C:" is not treated as a scheme because a single letter is not a valid one.
func hasURIScheme(rawURL string) bool {
	colon := strings.Index(rawURL, ":")
	if colon < 2 {
		return false
	}
	for i := 0; i < colon; i++ {
		c := rawURL[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '+', c == '-', c == '.':
		default:
			return false
		}
	}
	// A scheme must start with a letter.
	c := rawURL[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// existsOnDisk reports whether a link target resolves to a file the server
// does not index. Percent-escapes are decoded first: the plug-in keeps
// "my%20file.md" as an href and the build resolves it to "my file.md".
func existsOnDisk(rawURL, sourceURI string) bool {
	decoded, err := url.PathUnescape(rawURL)
	if err != nil {
		decoded = rawURL
	}
	srcPath, err := paths.URIToPath(sourceURI)
	if err != nil {
		return false
	}
	target := filepath.Join(filepath.Dir(srcPath), decoded)
	info, err := os.Stat(target)
	return err == nil && !info.IsDir()
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

	if len(target.Index.HeadingsByAnchor(ml.Anchor)) > 0 {
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
	return strings.ContainsRune(s, ' ')
}
