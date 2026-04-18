package document

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestIndexHeadingsBySlug(t *testing.T) {
	elements := []Element{
		&Heading{Level: 1, Text: "Title", Slug: paths.SlugOf("Title")},
		&Heading{Level: 2, Text: "Section One", Slug: paths.SlugOf("Section One")},
		&Heading{Level: 2, Text: "Section Two", Slug: paths.SlugOf("Section Two")},
	}
	idx := BuildIndex(elements, nil, nil)

	got := idx.HeadingsBySlug(paths.SlugOf("section-one"))
	if len(got) != 1 {
		t.Fatalf("HeadingsBySlug(section-one) returned %d, want 1", len(got))
	}
	if got[0].Text != "Section One" {
		t.Errorf("got heading %q", got[0].Text)
	}
}

func TestIndexTitle(t *testing.T) {
	elements := []Element{
		&Heading{Level: 1, Text: "My Doc Title", Slug: paths.SlugOf("My Doc Title")},
		&Heading{Level: 2, Text: "Section", Slug: paths.SlugOf("Section")},
	}
	idx := BuildIndex(elements, nil, nil)

	title := idx.Title()
	if title == nil || title.Text != "My Doc Title" {
		t.Errorf("Title() = %v, want heading 'My Doc Title'", title)
	}
}

func TestIndexNoTitle(t *testing.T) {
	elements := []Element{
		&Heading{Level: 2, Text: "Section", Slug: paths.SlugOf("Section")},
	}
	idx := BuildIndex(elements, nil, nil)

	if idx.Title() != nil {
		t.Error("expected nil title when no H1")
	}
}

func TestIndexShortDescription(t *testing.T) {
	idx := BuildIndex(nil, nil, nil)
	idx.ShortDesc = "This is the short description."
	if idx.ShortDesc != "This is the short description." {
		t.Error("short description not set")
	}
}

func TestIndexAllHeadings(t *testing.T) {
	elements := []Element{
		&Heading{Level: 1, Text: "T", Slug: paths.SlugOf("T")},
		&Heading{Level: 2, Text: "S", Slug: paths.SlugOf("S")},
	}
	idx := BuildIndex(elements, nil, nil)
	if len(idx.Headings()) != 2 {
		t.Errorf("Headings() returned %d, want 2", len(idx.Headings()))
	}
}
