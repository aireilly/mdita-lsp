package docsymbols

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestDocumentSymbols(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n### Sub\n\n## Other\n")
	syms := GetSymbols(doc)

	if len(syms) == 0 {
		t.Fatal("expected symbols")
	}
	if syms[0].Name != "Title" {
		t.Errorf("syms[0].Name = %q", syms[0].Name)
	}
}

func TestDocumentSymbolsNesting(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n### Sub\n\n## Other\n")
	syms := GetSymbols(doc)

	if len(syms) != 1 {
		t.Fatalf("expected 1 root symbol, got %d", len(syms))
	}
	if syms[0].Kind != 5 {
		t.Errorf("title kind = %d, want 5 (class)", syms[0].Kind)
	}
	if len(syms[0].Children) != 2 {
		t.Errorf("expected 2 children of title, got %d", len(syms[0].Children))
	}
	if len(syms[0].Children) >= 1 && len(syms[0].Children[0].Children) != 1 {
		t.Errorf("Section should have 1 child (Sub), got %d", len(syms[0].Children[0].Children))
	}
}

func TestDocumentSymbolsNoHeadings(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "Just text.\n")
	syms := GetSymbols(doc)
	if syms != nil {
		t.Errorf("expected nil for no headings, got %v", syms)
	}
}

func TestWorkspaceSymbols(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
		document.New("file:///project/b.md", 1, "# Beta\n"),
	}
	syms := SearchWorkspace(docs, "alp")
	if len(syms) != 1 || syms[0].Name != "Alpha" {
		t.Errorf("SearchWorkspace = %v", syms)
	}
}

func TestWorkspaceSymbolsNoMatch(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
	}
	syms := SearchWorkspace(docs, "xyz")
	if len(syms) != 0 {
		t.Errorf("expected 0 matches, got %d", len(syms))
	}
}

func TestWorkspaceSymbolsCaseInsensitive(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
	}
	syms := SearchWorkspace(docs, "ALPHA")
	if len(syms) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(syms))
	}
}

// The wire format is the contract with the client, and a struct without json
// tags satisfies every in-process assertion while breaking it. These tests
// look at the marshalled bytes for that reason.

func TestDocumentSymbolWireFormat(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n")
	syms := GetSymbols(doc)

	raw, err := json.Marshal(syms)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 root symbol, got %d", len(decoded))
	}

	// A client tells DocumentSymbol from SymbolInformation by testing the
	// first element for these four keys. Miss one and it converts the whole
	// response as the wrong type.
	for _, key := range []string{"name", "kind", "range", "selectionRange"} {
		if _, ok := decoded[0][key]; !ok {
			t.Errorf("missing %q in %s", key, raw)
		}
	}

	if strings.Contains(string(raw), `"Name"`) {
		t.Errorf("Go field names leaked into the wire format: %s", raw)
	}
}

func TestDocumentSymbolSelectionRangeWithinRange(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n")
	syms := GetSymbols(doc)

	if len(syms) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(syms))
	}
	sym := syms[0]
	if sym.SelectionRange.Start != sym.Range.Start || sym.SelectionRange.End != sym.Range.End {
		t.Errorf("selectionRange %+v must sit inside range %+v", sym.SelectionRange, sym.Range)
	}
}

func TestWorkspaceSymbolWireFormat(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/alpha.md", 1, "# Alpha\n\n## Alpha detail\n"),
	}
	syms := SearchWorkspace(docs, "alpha")

	if len(syms) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(syms))
	}

	raw, err := json.Marshal(syms)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded []struct {
		Name     string `json:"name"`
		Kind     int    `json:"kind"`
		Location struct {
			URI   string         `json:"uri"`
			Range document.Range `json:"range"`
		} `json:"location"`
		ContainerName string `json:"containerName"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Without a location the client cannot open the file a match lives in.
	if decoded[0].Location.URI != "file:///project/alpha.md" {
		t.Errorf("location.uri = %q, want the document URI; raw: %s", decoded[0].Location.URI, raw)
	}
	if decoded[0].Name != "Alpha" {
		t.Errorf("name = %q", decoded[0].Name)
	}
	if decoded[0].Kind != 5 {
		t.Errorf("kind = %d, want 5 for a title", decoded[0].Kind)
	}
	if decoded[0].ContainerName != "alpha.md" {
		t.Errorf("containerName = %q", decoded[0].ContainerName)
	}
}
