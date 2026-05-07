package ditamap

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type RelCell struct {
	TopicRefs []TopicRef
}

type RelRow struct {
	Cells []RelCell
}

type RelTable struct {
	Header []RelCell
	Rows   []RelRow
	Line   int
}

type TopicRef struct {
	Href     string
	Title    string
	Children []TopicRef
	IsMapRef bool
}

type MapStructure struct {
	Title     string
	TopicRefs []TopicRef
	RelTables []RelTable
}

func ParseMap(source string) (*MapStructure, error) {
	src := []byte(source)
	md := goldmark.New(
		goldmark.WithExtensions(extension.NewTable()),
	)
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	m := &MapStructure{}

	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Heading:
			if n.Level == 1 && m.Title == "" {
				m.Title = extractText(n, src)
			}
		case *ast.List:
			m.TopicRefs = append(m.TopicRefs, parseListItems(n, src)...)
		case *gmast.Table:
			rt := parseRelTable(n, src)
			m.RelTables = append(m.RelTables, rt)
		}
	}

	return m, nil
}

func parseListItems(list *ast.List, src []byte) []TopicRef {
	var refs []TopicRef
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		ref := TopicRef{}
		for ic := item.FirstChild(); ic != nil; ic = ic.NextSibling() {
			switch n := ic.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				link := findLink(n, src)
				if link != nil {
					ref.Href = string(link.Destination)
					ref.Title = extractText(link, src)
					ref.IsMapRef = isMapRefHref(ref.Href)
				}
			case *ast.List:
				ref.Children = append(ref.Children, parseListItems(n, src)...)
			}
		}
		if ref.Href != "" || ref.Title != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func findLink(n ast.Node, src []byte) *ast.Link {
	var link *ast.Link
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if l, ok := child.(*ast.Link); ok {
			link = l
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return link
}

func extractText(n ast.Node, src []byte) string {
	var result []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			result = append(result, t.Segment.Value(src)...)
		} else if c.HasChildren() {
			result = append(result, []byte(extractText(c, src))...)
		}
	}
	return string(result)
}

func (m *MapStructure) AllHrefs() []string {
	var hrefs []string
	collectHrefs(m.TopicRefs, &hrefs)
	return hrefs
}

func collectHrefs(refs []TopicRef, out *[]string) {
	for _, ref := range refs {
		if ref.Href != "" {
			*out = append(*out, ref.Href)
		}
		collectHrefs(ref.Children, out)
	}
}

func parseRelTable(table *gmast.Table, src []byte) RelTable {
	rt := RelTable{}

	for c := table.FirstChild(); c != nil; c = c.NextSibling() {
		switch row := c.(type) {
		case *gmast.TableHeader:
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*gmast.TableCell); ok {
					rt.Header = append(rt.Header, parseCellRefs(tc, src))
				}
			}
		case *gmast.TableRow:
			rr := RelRow{}
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*gmast.TableCell); ok {
					rr.Cells = append(rr.Cells, parseCellRefs(tc, src))
				}
			}
			rt.Rows = append(rt.Rows, rr)
		}
	}
	return rt
}

func parseCellRefs(cell *gmast.TableCell, src []byte) RelCell {
	rc := RelCell{}
	_ = ast.Walk(cell, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			ref := TopicRef{
				Href:  string(link.Destination),
				Title: extractText(link, src),
			}
			ref.IsMapRef = isMapRefHref(ref.Href)
			rc.TopicRefs = append(rc.TopicRefs, ref)
		}
		return ast.WalkContinue, nil
	})
	return rc
}

func isMapRefHref(href string) bool {
	return strings.HasSuffix(href, ".ditamap") || strings.HasSuffix(href, ".mditamap")
}
