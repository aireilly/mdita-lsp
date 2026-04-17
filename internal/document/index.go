package document

import "github.com/aireilly/mdita-lsp/internal/paths"

type Index struct {
	headings     []*Heading
	headingSlug  map[paths.Slug][]*Heading
	wikiLinks    []*WikiLink
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
		case *WikiLink:
			idx.wikiLinks = append(idx.wikiLinks, el)
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

func (idx *Index) Title() *Heading {
	for _, h := range idx.headings {
		if h.IsTitle() {
			return h
		}
	}
	return nil
}

func (idx *Index) WikiLinks() []*WikiLink {
	return idx.wikiLinks
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
	for _, w := range idx.wikiLinks {
		links = append(links, w)
	}
	for _, m := range idx.mdLinks {
		links = append(links, m)
	}
	return links
}
