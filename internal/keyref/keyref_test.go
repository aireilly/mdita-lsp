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

func TestBuildMergedTableYAMLKeys(t *testing.T) {
	mapText := "---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n  version: \"4.15\"\n  docs-url: \"https://docs.example.com\"\n---\n# Map\n\n- [Install](install.md)\n"
	table := BuildMergedTable([]string{mapText})

	// stem-based key from TopicRef
	if _, ok := table["install"]; !ok {
		t.Error("expected 'install' key from TopicRef")
	}

	// YAML text keydef
	entry, ok := table["product-name"]
	if !ok {
		t.Fatal("expected 'product-name' key from YAML keys:")
	}
	if entry.Value != "Red Hat OpenShift" {
		t.Errorf("Value = %q, want %q", entry.Value, "Red Hat OpenShift")
	}
	if entry.Title != "Red Hat OpenShift" {
		t.Errorf("Title = %q, want %q", entry.Title, "Red Hat OpenShift")
	}
	if entry.Href != "" {
		t.Errorf("Href = %q, want empty for text keydef", entry.Href)
	}

	// YAML URL keydef
	urlEntry, ok := table["docs-url"]
	if !ok {
		t.Fatal("expected 'docs-url' key from YAML keys:")
	}
	if urlEntry.Href != "https://docs.example.com" {
		t.Errorf("Href = %q, want %q", urlEntry.Href, "https://docs.example.com")
	}
	if urlEntry.Value != "" {
		t.Errorf("Value = %q, want empty for URL keydef", urlEntry.Value)
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

func TestBuildMergedTableRefStylePrecedence(t *testing.T) {
	mapText := "---\nkeys:\n  my-key: \"from YAML\"\n---\n# Map\n\n- [Install](install.md)\n\n[my-key]: https://from-refstyle.com\n"
	table := BuildMergedTable([]string{mapText})

	entry := table["my-key"]
	// YAML keys take precedence over reference-style links
	if entry.Value != "from YAML" {
		t.Errorf("YAML key should take precedence: got Value=%q Href=%q", entry.Value, entry.Href)
	}
}

func TestBuildMergedTableYAMLKeysPrecedence(t *testing.T) {
	mapText := "---\nkeys:\n  install: \"Installation Guide\"\n---\n# Map\n\n- [Install](install.md)\n"
	table := BuildMergedTable([]string{mapText})

	entry := table["install"]
	// YAML key takes precedence over stem-based
	if entry.Value != "Installation Guide" {
		t.Errorf("YAML key should take precedence: Value = %q", entry.Value)
	}
}
