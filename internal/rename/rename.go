package rename

import (
	"github.com/aireilly/mdita-lsp/internal/document"
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

func DoRename(doc *document.Document, pos document.Position, newName string, folder *workspace.Folder, graph *symbols.Graph) []TextEdit {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}
	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}

	var edits []TextEdit

	edits = append(edits, TextEdit{
		URI:     doc.URI,
		Range:   heading.Range,
		NewText: headingPrefix(heading.Level) + newName,
	})

	return edits
}

func headingPrefix(level int) string {
	s := ""
	for i := 0; i < level; i++ {
		s += "#"
	}
	return s + " "
}
