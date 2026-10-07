package references

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestFindRefsToHeading(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nContent.\n")
	doc2 := document.New("file:///project/a.md", 1, "# A\n\n[link](intro.md)\n")
	doc3 := document.New("file:///project/b.md", 1, "# B\n\n[link](intro.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	f.AddDoc(doc3)

	g := symbols.NewGraph()
	for _, d := range f.AllDocs() {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	heading := doc1.Index.Title()
	locs := FindRefs(doc1, heading.Range.Start, f, g)
	if len(locs) < 2 {
		t.Errorf("FindRefs returned %d, want >= 2", len(locs))
	}
}

func TestFindRefsNoRefs(t *testing.T) {
	doc := document.New("file:///project/lonely.md", 1, "# Lonely\n\nNo refs.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	heading := doc.Index.Title()
	locs := FindRefs(doc, heading.Range.Start, f, g)
	if len(locs) != 0 {
		t.Errorf("FindRefs returned %d, want 0", len(locs))
	}
}

func TestFindRefsNonHeading(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	locs := FindRefs(doc, document.Position{Line: 2, Character: 3}, f, g)
	if locs != nil {
		t.Errorf("FindRefs on non-heading returned %d locs", len(locs))
	}
}

func TestFindRefsMdLink(t *testing.T) {
	doc1 := document.New("file:///project/kitchen-sink.md", 1, "# Kitchen sink\n\nContent.\n")
	doc2 := document.New("file:///project/prereqs.md", 1, "# Prereqs\n\n[Kitchen sink](kitchen-sink.md)\n")
	doc3 := document.New("file:///project/guide.md", 1, "# Guide\n\n[Also links](kitchen-sink.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	f.AddDoc(doc3)

	g := symbols.NewGraph()
	for _, d := range f.AllDocs() {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	links := doc2.Index.MdLinks()
	if len(links) == 0 {
		t.Fatal("no md links parsed")
	}
	locs := FindRefs(doc2, links[0].Range.Start, f, g)
	if len(locs) < 2 {
		t.Errorf("FindRefs for MdLink returned %d, want >= 2", len(locs))
	}
}

func TestCountRefsCountsLinksToTheTopic(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n")
	doc2 := document.New("file:///project/a.md", 1, "# A\n\n[link](intro.md)\n")

	f := workspace.NewFolder("file:///project", config.Default())
	f.AddDoc(doc1)
	f.AddDoc(doc2)

	if got := CountRefs(doc1.Index.Title(), doc1, f); got != 1 {
		t.Errorf("CountRefs for the title = %d, want 1", got)
	}
}

// A link to a section used to be counted against the topic's H1, leaving the
// section itself with an empty reference list and a code lens of 0.
func TestSectionReferencesAreCountedAgainstTheSection(t *testing.T) {
	target := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## Install Steps\n\ntext\n")
	a := document.New("file:///project/a.md", 1, "# A\n\n[x](guide.md#install-steps)\n")
	b := document.New("file:///project/b.md", 1, "# B\n\n[y](guide.md#install-steps)\n")
	c := document.New("file:///project/c.md", 1, "# C\n\n[z](guide.md#install-steps)\n")

	f := workspace.NewFolder("file:///project", config.Default())
	for _, d := range []*document.Document{target, a, b, c} {
		f.AddDoc(d)
	}

	var section *document.Heading
	for _, h := range target.Index.Headings() {
		if h.Level == 2 {
			section = h
		}
	}
	if section == nil {
		t.Fatal("no level 2 heading parsed")
	}

	if got := CountRefs(section, target, f); got != 3 {
		t.Errorf("section references = %d, want 3", got)
	}
	if got := CountRefs(target.Index.Title(), target, f); got != 0 {
		t.Errorf("title references = %d, want 0; the links address the section", got)
	}
}
