package rename

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type PrepareResult struct {
	Range document.Range
	Text  string
}

type TextEdit struct {
	URI     string
	Range   document.Range
	NewText string
}

func Prepare(doc *document.Document, pos document.Position) *PrepareResult {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}
	switch el := elem.(type) {
	case *document.Heading:
		return &PrepareResult{
			Range: el.Range,
			Text:  el.Text,
		}
	}
	return nil
}

// DoRename renames a heading and repoints the links that address it.
//
// The heading's own range starts after the hashes, so writing
// "## New title" into it produced "## ## New title"; the whole source line is
// replaced instead. Renaming a heading also changes the id the plug-in
// generates for it, which breaks every "#old-anchor" link in the workspace, so
// those fragments are rewritten in the same edit.
//
// The symbol graph is not consulted: a heading reference is a link with a
// fragment, which the folder resolves directly.
func DoRename(doc *document.Document, pos document.Position, newName string, folder *workspace.Folder, _ *symbols.Graph) []TextEdit {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}
	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}

	newName = strings.TrimSpace(newName)
	if newName == "" {
		return nil
	}

	edits := []TextEdit{{
		URI:     doc.URI,
		Range:   heading.LineRange,
		NewText: headingPrefix(heading.Level) + newName,
	}}

	if heading.ExplicitID {
		// An author-written {#id} survives the rename, so no link changes.
		return edits
	}

	oldAnchor := heading.ID
	newAnchor := newAnchorFor(doc, heading, newName)
	if oldAnchor == "" || newAnchor == "" || oldAnchor == newAnchor {
		return edits
	}

	if folder == nil {
		return edits
	}
	for _, other := range folder.AllDocs() {
		for _, ml := range other.Index.MdLinks() {
			if ml.Anchor != oldAnchor {
				continue
			}
			if !addressesDoc(ml, other, doc, folder) {
				continue
			}
			edits = append(edits, TextEdit{
				URI:     other.URI,
				Range:   ml.Range,
				NewText: "[" + ml.Text + "](" + ml.URL + "#" + newAnchor + ")",
			})
		}
	}
	return edits
}

// newAnchorFor works out the id the renamed heading will get, keeping the
// duplicate suffixes the other headings in the document already claim.
func newAnchorFor(doc *document.Document, heading *document.Heading, newName string) string {
	headings := doc.Index.Headings()
	texts := make([]string, len(headings))
	idx := -1
	for i, h := range headings {
		texts[i] = h.Text
		if h == heading {
			idx = i
			texts[i] = newName
		}
	}
	if idx < 0 {
		return ""
	}
	return paths.AnchorIDs(texts)[idx]
}

// addressesDoc reports whether a link in src points at target. A link with no
// URL addresses its own document.
func addressesDoc(ml *document.MdLink, src, target *document.Document, folder *workspace.Folder) bool {
	if ml.URL == "" {
		return src.URI == target.URI
	}
	resolved := folder.ResolveLink(ml.URL, src.URI)
	return resolved != nil && resolved.URI == target.URI
}

func headingPrefix(level int) string {
	return strings.Repeat("#", level) + " "
}
