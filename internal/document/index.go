package document

import "github.com/aireilly/mdita-lsp/internal/paths"

type Index struct {
	headings     []*Heading
	headingSlug  map[paths.Slug][]*Heading
	headingID    map[string][]*Heading
	mdLinks      []*MdLink
	linkDefs     []*LinkDef
	linkDefLabel map[string]*LinkDef
	Meta         *YAMLMetadata
	Features     *BlockFeatures
	ShortDesc    string
}

func BuildIndex(elements []Element, bf *BlockFeatures, meta *YAMLMetadata) *Index {
	idx := &Index{
		headingSlug:  make(map[paths.Slug][]*Heading),
		headingID:    make(map[string][]*Heading),
		linkDefLabel: make(map[string]*LinkDef),
		Features:     bf,
		Meta:         meta,
	}
	if idx.Features == nil {
		idx.Features = &BlockFeatures{}
	}

	for _, e := range elements {
		switch el := e.(type) {
		case *Heading:
			idx.headings = append(idx.headings, el)
			idx.headingSlug[el.Slug] = append(idx.headingSlug[el.Slug], el)
			if el.ID != "" {
				idx.headingID[el.ID] = append(idx.headingID[el.ID], el)
			}
		case *MdLink:
			idx.mdLinks = append(idx.mdLinks, el)
		case *LinkDef:
			idx.linkDefs = append(idx.linkDefs, el)
			idx.linkDefLabel[el.Label] = el
		}
	}
	return idx
}

func (idx *Index) Headings() []*Heading {
	return idx.headings
}

func (idx *Index) HeadingsBySlug(slug paths.Slug) []*Heading {
	return idx.headingSlug[slug]
}

// HeadingsByAnchor looks a link fragment up against the ids the plug-in
// generates, which is what a built link resolves against. A duplicate heading
// gets a "-1" suffix there, so matching on the heading text alone reported
// working links as broken.
func (idx *Index) HeadingsByAnchor(anchor string) []*Heading {
	if hs := idx.headingID[anchor]; len(hs) > 0 {
		return hs
	}
	return idx.headingSlug[paths.SlugOf(anchor)]
}

// HeadingByID returns the heading with this generated id, or nil.
func (idx *Index) HeadingByID(id string) *Heading {
	if hs := idx.headingID[id]; len(hs) > 0 {
		return hs[0]
	}
	return nil
}

func (idx *Index) Title() *Heading {
	for _, h := range idx.headings {
		if h.IsTitle() {
			return h
		}
	}
	return nil
}

func (idx *Index) MdLinks() []*MdLink {
	return idx.mdLinks
}

func (idx *Index) LinkDefs() []*LinkDef {
	return idx.linkDefs
}

func (idx *Index) LinkDefByLabel(label string) *LinkDef {
	return idx.linkDefLabel[label]
}

func (idx *Index) AllLinks() []Element {
	var links []Element
	for _, m := range idx.mdLinks {
		links = append(links, m)
	}
	return links
}
