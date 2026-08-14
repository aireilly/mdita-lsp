package document

import (
	"testing"
)

func TestParseHeadings(t *testing.T) {
	text := "# Title\n\n## Section One\n\n### Subsection\n"
	elements, _, _ := Parse(text)

	headings := filterHeadings(elements)
	if len(headings) != 3 {
		t.Fatalf("got %d headings, want 3", len(headings))
	}
	if headings[0].Level != 1 || headings[0].Text != "Title" {
		t.Errorf("heading[0] = {%d, %q}, want {1, \"Title\"}", headings[0].Level, headings[0].Text)
	}
	if headings[1].Level != 2 || headings[1].Text != "Section One" {
		t.Errorf("heading[1] = {%d, %q}, want {2, \"Section One\"}", headings[1].Level, headings[1].Text)
	}
	if headings[2].Level != 3 || headings[2].Text != "Subsection" {
		t.Errorf("heading[2] = {%d, %q}, want {3, \"Subsection\"}", headings[2].Level, headings[2].Text)
	}
}

func TestParseMdLinks(t *testing.T) {
	text := "# Title\n\n[link text](other.md)\n\n[anchor](other.md#section)\n\n[ref][label]\n"
	elements, _, _ := Parse(text)

	mdLinks := filterMdLinks(elements)
	if len(mdLinks) < 2 {
		t.Fatalf("got %d md links, want at least 2", len(mdLinks))
	}
	if mdLinks[0].Text != "link text" || mdLinks[0].URL != "other.md" {
		t.Errorf("ml[0] = {%q, %q}", mdLinks[0].Text, mdLinks[0].URL)
	}
	if mdLinks[1].URL != "other.md" || mdLinks[1].Anchor != "section" {
		t.Errorf("ml[1] = {%q, %q}", mdLinks[1].URL, mdLinks[1].Anchor)
	}
}

func TestParseYAMLFrontMatter(t *testing.T) {
	text := "---\nauthor: John\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\nkeyword: [go, lsp]\n---\n# Title\n"
	elements, _, meta := Parse(text)

	if meta == nil {
		t.Fatal("expected YAML metadata, got nil")
	}
	if meta.Author != "John" {
		t.Errorf("Author = %q, want %q", meta.Author, "John")
	}
	if meta.Schema != SchemaTask {
		t.Errorf("Schema = %v, want SchemaTask", meta.Schema)
	}
	if len(meta.Keywords) != 2 || meta.Keywords[0] != "go" {
		t.Errorf("Keywords = %v, want [go lsp]", meta.Keywords)
	}

	_ = elements
}

func TestParseLinkDefs(t *testing.T) {
	text := "# Title\n\n[label]: https://example.com\n"
	elements, _, _ := Parse(text)

	linkDefs := filterLinkDefs(elements)
	if len(linkDefs) != 1 {
		t.Fatalf("got %d link defs, want 1", len(linkDefs))
	}
	if linkDefs[0].Label != "label" || linkDefs[0].URL != "https://example.com" {
		t.Errorf("ld[0] = {%q, %q}", linkDefs[0].Label, linkDefs[0].URL)
	}
}

func TestParseBlockFeatures(t *testing.T) {
	text := "# Title\n\n1. step one\n2. step two\n\n| col1 | col2 |\n|------|------|\n| a | b |\n"
	_, bf, _ := Parse(text)

	if !bf.HasOrderedList {
		t.Error("expected HasOrderedList = true")
	}
	if !bf.HasTable {
		t.Error("expected HasTable = true")
	}
}

func TestParseAdmonitions(t *testing.T) {
	text := "# Title\n\n!!! note\n    This is a note.\n"
	_, bf, _ := Parse(text)

	if len(bf.Admonitions) != 1 {
		t.Fatalf("got %d admonitions, want 1", len(bf.Admonitions))
	}
	if bf.Admonitions[0].Type != "note" {
		t.Errorf("admonition type = %q, want %q", bf.Admonitions[0].Type, "note")
	}
}

func TestMdLinkRange(t *testing.T) {
	text := "# Title\n\nSome text [link](other.md) more text.\n"
	elements, _, _ := Parse(text)

	mdLinks := filterMdLinks(elements)
	if len(mdLinks) != 1 {
		t.Fatalf("got %d md links, want 1", len(mdLinks))
	}
	ml := mdLinks[0]
	if ml.Range.Start.Line != 2 {
		t.Errorf("start line = %d, want 2", ml.Range.Start.Line)
	}
	if ml.Range.Start.Character != 10 {
		t.Errorf("start char = %d, want 10", ml.Range.Start.Character)
	}
	if ml.Range.End.Line != 2 {
		t.Errorf("end line = %d, want 2", ml.Range.End.Line)
	}
	if ml.Range.End.Character != 26 {
		t.Errorf("end char = %d, want 26", ml.Range.End.Character)
	}
}

func filterHeadings(elems []Element) []*Heading {
	var result []*Heading
	for _, e := range elems {
		if h, ok := e.(*Heading); ok {
			result = append(result, h)
		}
	}
	return result
}

func filterMdLinks(elems []Element) []*MdLink {
	var result []*MdLink
	for _, e := range elems {
		if m, ok := e.(*MdLink); ok {
			result = append(result, m)
		}
	}
	return result
}

func filterLinkDefs(elems []Element) []*LinkDef {
	var result []*LinkDef
	for _, e := range elems {
		if l, ok := e.(*LinkDef); ok {
			result = append(result, l)
		}
	}
	return result
}
