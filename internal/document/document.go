package document

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Document struct {
	URI      string
	Version  int
	Text     string
	Lines    []int
	Elements []Element
	Symbols  []Symbol
	Index    *Index
	Meta     *YAMLMetadata
	Kind     DocKind
}

func New(uri string, version int, text string) *Document {
	elements, bf, meta := Parse(text)
	idx := BuildIndex(elements, bf, meta)
	idx.Meta = meta

	kind := Topic
	if paths.IsMditaMapFile(uri, []string{"mditamap"}) {
		kind = Map
	}

	title := idx.Title()
	if title != nil && idx.ShortDesc == "" {
		idx.ShortDesc = findShortDesc(text, title)
	}

	doc := &Document{
		URI:      uri,
		Version:  version,
		Text:     text,
		Lines:    buildLineMap(text),
		Elements: elements,
		Index:    idx,
		Meta:     meta,
		Kind:     kind,
	}
	doc.Symbols = extractSymbols(doc)
	return doc
}

func (d *Document) ApplyChange(version int, newText string) *Document {
	return New(d.URI, version, newText)
}

func (d *Document) DocID(rootURI string) paths.DocID {
	return paths.DocIDFromURI(d.URI, rootURI)
}

func (d *Document) Defs() []Symbol {
	var defs []Symbol
	for _, s := range d.Symbols {
		if s.Kind == DefKind {
			defs = append(defs, s)
		}
	}
	return defs
}

func (d *Document) Refs() []Symbol {
	var refs []Symbol
	for _, s := range d.Symbols {
		if s.Kind == RefKind {
			refs = append(refs, s)
		}
	}
	return refs
}

func (d *Document) ElementAt(pos Position) Element {
	for _, e := range d.Elements {
		r := e.Rng()
		if posInRange(pos, r) {
			return e
		}
	}
	return nil
}

func posInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

func buildLineMap(text string) []int {
	lines := []int{0}
	for i, ch := range text {
		if ch == '\n' {
			lines = append(lines, i+1)
		}
	}
	return lines
}

func findShortDesc(text string, title *Heading) string {
	lines := strings.Split(text, "\n")
	titleLine := title.Range.Start.Line
	for i := titleLine + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return ""
		}
		return line
	}
	return ""
}

func extractSymbols(doc *Document) []Symbol {
	var syms []Symbol

	syms = append(syms, Symbol{
		Kind:    DefKind,
		DefType: DefDoc,
		Name:    doc.URI,
		DocURI:  doc.URI,
	})

	for _, e := range doc.Elements {
		switch el := e.(type) {
		case *Heading:
			dt := DefHeading
			if el.IsTitle() {
				dt = DefTitle
			}
			syms = append(syms, Symbol{
				Kind:    DefKind,
				DefType: dt,
				Name:    el.Text,
				Slug:    el.Slug,
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *WikiLink:
			syms = append(syms, Symbol{
				Kind:    RefKind,
				RefType: RefWikiLink,
				Name:    el.Doc,
				Slug:    paths.SlugOf(el.Doc),
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *MdLink:
			syms = append(syms, Symbol{
				Kind:    RefKind,
				RefType: RefMdLink,
				Name:    el.URL,
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *LinkDef:
			syms = append(syms, Symbol{
				Kind:    DefKind,
				DefType: DefLinkDef,
				Name:    el.Label,
				DocURI:  doc.URI,
				Range:   el.Range,
			})
		}
	}
	return syms
}
