package keyref

import (
	"testing"
)

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

// A plain topicref in a map defines no key: the plug-in only emits <keydef>
// for reference-style link definitions.
func TestBuildMergedTableIgnoresTopicrefs(t *testing.T) {
	mapText := "# Map\n\n- [Install](install.md)\n- [Config](config.md)\n"
	table := BuildMergedTable([]string{mapText})
	if len(table) != 0 {
		t.Errorf("BuildMergedTable = %v, want no keys", table)
	}
}

func TestBuildMergedTableRefStyleKeydefs(t *testing.T) {
	mapText := "# Map\n\n- [Install](install.md)\n\n[prod-url]: https://example.com\n[install]: install.md \"Installation Guide\"\n"
	table := BuildMergedTable([]string{mapText})

	entry, ok := table["prod-url"]
	if !ok {
		t.Fatal("expected 'prod-url' key from reference-style link")
	}
	if entry.Href != "https://example.com" {
		t.Errorf("Href = %q, want %q", entry.Href, "https://example.com")
	}

	entry, ok = table["install"]
	if !ok {
		t.Fatal("expected 'install' key from reference-style link")
	}
	if entry.Href != "install.md" {
		t.Errorf("Href = %q, want %q", entry.Href, "install.md")
	}
	if entry.Title != "Installation Guide" {
		t.Errorf("Title = %q, want %q", entry.Title, "Installation Guide")
	}
}

func TestBuildMergedTableFirstMapWins(t *testing.T) {
	a := "[k]: a.md\n"
	b := "[k]: b.md\n"
	table := BuildMergedTable([]string{a, b})
	if table["k"].Href != "a.md" {
		t.Errorf("Href = %q, want a.md", table["k"].Href)
	}
}
