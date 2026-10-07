package references

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Location struct {
	URI   string
	Range document.Range
}

func FindRefs(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []Location {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	switch el := elem.(type) {
	case *document.Heading:
		return findHeadingRefs(el, doc, folder, graph)
	case *document.MdLink:
		return findMdLinkRefs(el, doc, folder)
	}
	return nil
}

func findHeadingRefs(heading *document.Heading, doc *document.Document, folder *workspace.Folder, graph *symbols.Graph) []Location {
	return FindHeadingLocations(heading, doc, folder)
}

// FindHeadingLocations returns every link that addresses a heading.
//
// The symbol graph could not answer this: a link's ref symbol carried the URL
// but no slug, so a graph lookup by slug found nothing and an H2 linked three
// times reported zero references, with the links counted against the H1
// instead. Resolving each link's target document and fragment gets both the
// section count and the topic count right.
func FindHeadingLocations(heading *document.Heading, doc *document.Document, folder *workspace.Folder) []Location {
	if folder == nil {
		return nil
	}
	var locs []Location
	for _, d := range folder.AllDocs() {
		for _, ml := range d.Index.MdLinks() {
			target := d
			if ml.URL != "" {
				target = folder.ResolveLink(ml.URL, d.URI)
			}
			if target == nil || target.URI != doc.URI {
				continue
			}
			if ml.Anchor == "" {
				// A link to the topic addresses its title.
				if heading.IsTitle() {
					locs = append(locs, Location{URI: d.URI, Range: ml.Range})
				}
				continue
			}
			if anchorMatches(ml.Anchor, heading, doc) {
				locs = append(locs, Location{URI: d.URI, Range: ml.Range})
			}
		}
	}
	return locs
}

// anchorMatches reports whether a fragment addresses this heading, allowing
// for the "topic-id/element-id" form the plug-in passes through to @href.
func anchorMatches(anchor string, heading *document.Heading, doc *document.Document) bool {
	_, elementID, isDita := document.SplitFragment(anchor)
	if isDita {
		anchor = elementID
	}
	if heading.ID != "" && anchor == heading.ID {
		return true
	}
	return heading.IsTitle() && doc.TopicID() == anchor
}

func findMdLinkRefs(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) []Location {
	target := folder.ResolveLink(ml.URL, doc.URI)
	if target == nil {
		return nil
	}
	return findDocRefs(target.URI, folder)
}

func findDocRefs(targetURI string, folder *workspace.Folder) []Location {
	var locs []Location
	for _, d := range folder.AllDocs() {
		for _, ml := range d.Index.MdLinks() {
			if resolved := folder.ResolveLink(ml.URL, d.URI); resolved != nil && resolved.URI == targetURI {
				locs = append(locs, Location{URI: d.URI, Range: ml.Range})
			}
		}
	}
	return locs
}

func CountRefs(heading *document.Heading, doc *document.Document, folder *workspace.Folder) int {
	return len(FindHeadingLocations(heading, doc, folder))
}
