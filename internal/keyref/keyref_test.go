package keyref

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
)

func TestExtractKeys(t *testing.T) {
	m := &ditamap.MapStructure{
		Title: "Product Docs",
		TopicRefs: []ditamap.TopicRef{
			{Href: "install.md", Title: "Installation"},
			{Href: "config.md", Title: "Configuration",
				Children: []ditamap.TopicRef{
					{Href: "config-advanced.md", Title: "Advanced Config"},
				}},
		},
	}

	table := ExtractKeys(m)
	if len(table) != 3 {
		t.Fatalf("ExtractKeys = %d keys, want 3", len(table))
	}

	entry, ok := table["install"]
	if !ok {
		t.Fatal("missing key 'install'")
	}
	if entry.Href != "install.md" || entry.Title != "Installation" {
		t.Errorf("install entry = %+v", entry)
	}
}

func TestResolveKeyref(t *testing.T) {
	table := KeyTable{
		"install": {Href: "install.md", Title: "Installation"},
	}

	entry, ok := Resolve(table, "install")
	if !ok {
		t.Fatal("Resolve returned false")
	}
	if entry.Href != "install.md" {
		t.Errorf("Href = %q", entry.Href)
	}

	_, ok = Resolve(table, "nonexistent")
	if ok {
		t.Error("Resolve should return false for unknown key")
	}
}

func TestExtractKeysFromSlug(t *testing.T) {
	m := &ditamap.MapStructure{
		TopicRefs: []ditamap.TopicRef{
			{Href: "getting-started.md", Title: "Getting Started"},
		},
	}
	table := ExtractKeys(m)

	_, ok := table["getting-started"]
	if !ok {
		t.Error("expected key 'getting-started' derived from href stem")
	}
}

func TestBuildMergedTableRefStyleKeydefs(t *testing.T) {
	mapText := "# Map\n\n- [Install](install.md)\n\n[prod-url]: https://example.com\n[prod-name]: Red Hat OpenShift\n"
	table := BuildMergedTable([]string{mapText})

	// href-based keydef from TopicRef
	if _, ok := table["install"]; !ok {
		t.Error("expected 'install' key from TopicRef")
	}

	// reference-style URL keydef
	entry, ok := table["prod-url"]
	if !ok {
		t.Fatal("expected 'prod-url' key from reference-style link")
	}
	if entry.Href != "https://example.com" {
		t.Errorf("Href = %q, want %q", entry.Href, "https://example.com")
	}

	// reference-style text keydef
	entry2, ok := table["prod-name"]
	if !ok {
		t.Fatal("expected 'prod-name' key from reference-style link")
	}
	if entry2.Value != "Red Hat OpenShift" {
		t.Errorf("Value = %q, want %q", entry2.Value, "Red Hat OpenShift")
	}
}
