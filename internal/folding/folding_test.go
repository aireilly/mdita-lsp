package folding

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestFoldHeadings(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nIntro.\n\n## Section\n\nContent.\n\n## Other\n\nMore.\n")
	ranges := GetRanges(doc)

	if len(ranges) < 2 {
		t.Fatalf("got %d folding ranges, want >= 2", len(ranges))
	}

	if ranges[0].StartLine != 0 {
		t.Errorf("first range start = %d, want 0", ranges[0].StartLine)
	}
}

func TestFoldYAMLFrontMatter(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\nauthor: Test\nsource: Example\n---\n# Title\n")
	ranges := GetRanges(doc)

	foundYAML := false
	for _, r := range ranges {
		if r.StartLine == 0 && r.EndLine == 3 {
			foundYAML = true
		}
	}
	if !foundYAML {
		t.Error("missing YAML front matter folding range")
	}
}

func TestFoldToCMarkers(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n<!--toc:start-->\n- [A](#a)\n- [B](#b)\n<!--toc:end-->\n\n## A\n\n## B\n")
	ranges := GetRanges(doc)

	foundToC := false
	for _, r := range ranges {
		if r.StartLine == 2 && r.EndLine == 5 {
			foundToC = true
		}
	}
	if !foundToC {
		t.Error("missing ToC folding range")
	}
}

func TestNoFoldingForShortDoc(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n")
	ranges := GetRanges(doc)

	for _, r := range ranges {
		if r.Kind == "region" && r.StartLine == r.EndLine {
			t.Error("should not create zero-line folding range")
		}
	}
}
