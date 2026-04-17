package document

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type wikiLinkNode struct {
	ast.BaseInline
	Doc     string
	Heading string
	Title   string
}

var kindWikiLink = ast.NewNodeKind("WikiLink")

func (n *wikiLinkNode) Kind() ast.NodeKind { return kindWikiLink }

func (n *wikiLinkNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type wikiLinkParser struct{}

func (p *wikiLinkParser) Trigger() []byte {
	return []byte{'['}
}

func (p *wikiLinkParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 2 || line[0] != '[' || line[1] != '[' {
		return nil
	}

	end := -1
	for i := 2; i < len(line)-1; i++ {
		if line[i] == ']' && line[i+1] == ']' {
			end = i
			break
		}
		if line[i] == '\n' {
			return nil
		}
	}
	if end < 0 {
		return nil
	}

	content := string(line[2:end])
	node := &wikiLinkNode{}

	docPart := content
	if idx := indexOf(content, '|'); idx >= 0 {
		node.Title = content[idx+1:]
		docPart = content[:idx]
	}
	if idx := indexOf(docPart, '#'); idx >= 0 {
		node.Heading = docPart[idx+1:]
		node.Doc = docPart[:idx]
	} else {
		node.Doc = docPart
	}

	consumed := end + 2
	block.Advance(consumed)
	return node
}

func indexOf(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
	}
	return -1
}

type wikiLinkExtension struct{}

func (e *wikiLinkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&wikiLinkParser{}, 199),
		),
	)
}
