package paths

import (
	"testing"
)

func TestURIToPath(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"file:///home/user/doc.md", "/home/user/doc.md"},
		{"file:///home/user/my%20doc.md", "/home/user/my doc.md"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			got, err := URIToPath(tt.uri)
			if err != nil {
				t.Fatalf("URIToPath(%q) error: %v", tt.uri, err)
			}
			if got != tt.want {
				t.Errorf("URIToPath(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestPathToURI(t *testing.T) {
	got := PathToURI("/home/user/doc.md")
	want := "file:///home/user/doc.md"
	if got != want {
		t.Errorf("PathToURI = %q, want %q", got, want)
	}
}

func TestRelPath(t *testing.T) {
	got := RelPath("/home/user/project", "/home/user/project/docs/file.md")
	if got != "docs/file.md" {
		t.Errorf("RelPath = %q, want %q", got, "docs/file.md")
	}
}

func TestDocIDFromURI(t *testing.T) {
	id := DocIDFromURI("file:///home/user/project/docs/intro.md", "file:///home/user/project")
	if id.RelPath != "docs/intro.md" {
		t.Errorf("DocID.RelPath = %q, want %q", id.RelPath, "docs/intro.md")
	}
	if id.Stem != "intro" {
		t.Errorf("DocID.Stem = %q, want %q", id.Stem, "intro")
	}
}

func TestIsMditaMapFile(t *testing.T) {
	tests := []struct {
		path string
		exts []string
		want bool
	}{
		{"foo.mditamap", []string{"mditamap"}, true},
		{"foo.md", []string{"mditamap"}, false},
		{"foo.ditamap", []string{"mditamap", "ditamap"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsMditaMapFile(tt.path, tt.exts); got != tt.want {
				t.Errorf("IsMditaMapFile(%q, %v) = %v, want %v", tt.path, tt.exts, got, tt.want)
			}
		})
	}
}
