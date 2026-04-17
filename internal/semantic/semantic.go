package semantic

import (
	"sort"

	"github.com/aireilly/mdita-lsp/internal/document"
)

const (
	TokenTypeWikiLink = 0
	TokenTypeRefLink  = 1
)

var TokenTypes = []string{"class", "class"}

type token struct {
	line   int
	char   int
	length int
	typ    int
}

func Encode(doc *document.Document) []uint32 {
	var tokens []token

	for _, wl := range doc.Index.WikiLinks() {
		r := wl.Range
		if r.Start.Line == r.End.Line {
			tokens = append(tokens, token{
				line:   r.Start.Line,
				char:   r.Start.Character,
				length: r.End.Character - r.Start.Character,
				typ:    TokenTypeWikiLink,
			})
		}
	}

	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].line != tokens[j].line {
			return tokens[i].line < tokens[j].line
		}
		return tokens[i].char < tokens[j].char
	})

	var data []uint32
	prevLine := 0
	prevChar := 0
	for _, tok := range tokens {
		deltaLine := tok.line - prevLine
		deltaChar := tok.char
		if deltaLine == 0 {
			deltaChar = tok.char - prevChar
		}
		data = append(data, uint32(deltaLine), uint32(deltaChar), uint32(tok.length), uint32(tok.typ), 0)
		prevLine = tok.line
		prevChar = tok.char
	}
	return data
}
