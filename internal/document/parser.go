package document

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

var admonitionRegex = regexp.MustCompile(`(?m)^!!!\s+(\w+)`)
var footnoteRefRegex = regexp.MustCompile(`\[\^([^\]]+)\][^:]`)
var footnoteDefRegex = regexp.MustCompile(`(?m)^\[\^([^\]]+)\]:`)

func Parse(source string) ([]Element, *BlockFeatures, *YAMLMetadata) {
	src := []byte(source)

	var meta *YAMLMetadata
	mdContent := source
	yamlEnd := 0

	if strings.HasPrefix(source, "---\n") || strings.HasPrefix(source, "---\r\n") {
		closeIdx := strings.Index(source[4:], "\n---")
		if closeIdx < 0 {
			closeIdx = strings.Index(source[4:], "\n...")
		}
		if closeIdx >= 0 {
			yamlBlock := source[4 : 4+closeIdx]
			meta = parseYAMLMeta(yamlBlock)
			yamlEnd = 4 + closeIdx + 4
			if yamlEnd < len(source) && source[yamlEnd-1] == '\r' {
				yamlEnd++
			}
		}
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			&wikiLinkExtension{},
			extension.NewTable(),
			extension.Strikethrough,
			extension.DefinitionList,
			extension.Footnote,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	var elements []Element
	bf := &BlockFeatures{}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			headingText := extractText(node, src)
			id := ""
			if idAttr, ok := node.AttributeString("id"); ok {
				if idBytes, ok := idAttr.([]byte); ok {
					id = string(idBytes)
				}
			}
			if id == "" {
				id = paths.Slugify(headingText)
			}
			elements = append(elements, &Heading{
				Level: node.Level,
				Text:  headingText,
				ID:    id,
				Slug:  paths.SlugOf(headingText),
				Range: nodeRange(node, src),
			})

		case *wikiLinkNode:
			elements = append(elements, &WikiLink{
				Doc:     node.Doc,
				Heading: node.Heading,
				Title:   node.Title,
				Range:   nodeRange(node, src),
			})

		case *ast.Link:
			url := string(node.Destination)
			linkText := extractText(node, src)
			anchor := ""
			if idx := strings.Index(url, "#"); idx >= 0 {
				anchor = url[idx+1:]
				url = url[:idx]
			}
			if !isExternalURL(url) {
				elements = append(elements, &MdLink{
					Text:   linkText,
					URL:    url,
					Anchor: anchor,
					Range:  nodeRange(node, src),
				})
			}

		case *ast.List:
			if node.IsOrdered() {
				bf.HasOrderedList = true
			} else {
				bf.HasUnorderedList = true
			}

		case *gmast.Table:
			bf.HasTable = true

		case *gmast.Strikethrough:
			bf.HasStrikethrough = true

		case *gmast.DefinitionList:
			bf.HasDefinitionList = true

		case *gmast.FootnoteLink:
			bf.HasFootnoteRefs = true

		case *gmast.FootnoteBacklink:
			bf.HasFootnoteDefs = true
		}

		return ast.WalkContinue, nil
	})

	elements = append(elements, parseLinkDefs(mdContent, yamlEnd)...)
	bf.Admonitions = parseAdmonitions(source)
	bf.FootnoteRefLabels = parseFootnoteRefs(source)
	bf.FootnoteDefLabels = parseFootnoteDefs(source)

	_ = mdContent
	return elements, bf, meta
}

func parseYAMLMeta(yamlContent string) *YAMLMetadata {
	meta := &YAMLMetadata{OtherMeta: make(map[string]string)}

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return meta
	}

	for key, val := range raw {
		sval := ""
		switch v := val.(type) {
		case string:
			sval = v
		}

		switch key {
		case "author":
			meta.Author = sval
		case "source":
			meta.Source = sval
		case "publisher":
			meta.Publisher = sval
		case "permissions":
			meta.Permissions = sval
		case "audience":
			meta.Audience = sval
		case "category":
			meta.Category = sval
		case "resourceid":
			meta.ResourceID = sval
		case "$schema":
			meta.SchemaRaw = sval
			meta.Schema = DitaSchemaFromString(sval)
		case "keyword":
			meta.Keywords = parseKeywords(val)
		default:
			if sval != "" {
				meta.OtherMeta[key] = sval
			}
		}
	}
	return meta
}

func parseKeywords(val interface{}) []string {
	switch v := val.(type) {
	case string:
		return []string{v}
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

var linkDefRegex = regexp.MustCompile(`(?m)^\[([^\]]+)\]:\s+(.+)$`)

func parseLinkDefs(source string, offset int) []Element {
	var elements []Element
	matches := linkDefRegex.FindAllStringSubmatchIndex(source[offset:], -1)
	for _, m := range matches {
		label := source[offset+m[2] : offset+m[3]]
		url := strings.TrimSpace(source[offset+m[4] : offset+m[5]])
		rng := rangeFromOffset(source, offset+m[0], offset+m[1])
		elements = append(elements, &LinkDef{
			Label: label,
			URL:   url,
			Range: rng,
		})
	}
	return elements
}

func parseAdmonitions(source string) []Admonition {
	var admonitions []Admonition
	matches := admonitionRegex.FindAllStringSubmatchIndex(source, -1)
	for _, m := range matches {
		adType := source[m[2]:m[3]]
		rng := rangeFromOffset(source, m[0], m[1])
		admonitions = append(admonitions, Admonition{
			Type:  strings.ToLower(adType),
			Range: rng,
		})
	}
	return admonitions
}

func extractText(n ast.Node, src []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			sb.Write(t.Segment.Value(src))
		} else if c.HasChildren() {
			sb.WriteString(extractText(c, src))
		}
	}
	return sb.String()
}

func nodeRange(n ast.Node, src []byte) Range {
	// Inline nodes don't have Lines(), so use parent
	if n.Type() == ast.TypeInline {
		parent := n.Parent()
		if parent != nil {
			return nodeRange(parent, src)
		}
		return Range{}
	}

	lines := n.Lines()
	if lines.Len() > 0 {
		first := lines.At(0)
		last := lines.At(lines.Len() - 1)
		return rangeFromOffset(string(src), first.Start, last.Stop)
	}
	return Range{}
}

func rangeFromOffset(source string, start, end int) Range {
	sl, sc := offsetToLineCol(source, start)
	el, ec := offsetToLineCol(source, end)
	return Rng(sl, sc, el, ec)
}

func offsetToLineCol(source string, offset int) (int, int) {
	line := 0
	col := 0
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

func parseFootnoteRefs(source string) []FootnoteLabel {
	var labels []FootnoteLabel
	matches := footnoteRefRegex.FindAllStringSubmatchIndex(source, -1)
	for _, m := range matches {
		label := source[m[2]:m[3]]
		rng := rangeFromOffset(source, m[0], m[1])
		labels = append(labels, FootnoteLabel{Label: label, Range: rng})
	}
	return labels
}

func parseFootnoteDefs(source string) []FootnoteLabel {
	var labels []FootnoteLabel
	matches := footnoteDefRegex.FindAllStringSubmatchIndex(source, -1)
	for _, m := range matches {
		label := source[m[2]:m[3]]
		rng := rangeFromOffset(source, m[0], m[1])
		labels = append(labels, FootnoteLabel{Label: label, Range: rng})
	}
	return labels
}

func isExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "mailto:") ||
		strings.HasPrefix(url, "ftp://")
}
