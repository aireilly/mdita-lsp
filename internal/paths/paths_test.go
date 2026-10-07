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

func TestPathToURIEncodesSpecialCharacters(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/my doc.md", "file:///home/user/my%20doc.md"},
		{"/home/user/a#b.md", "file:///home/user/a%23b.md"},
		{"/home/user/plain.md", "file:///home/user/plain.md"},
	}
	for _, tt := range tests {
		if got := PathToURI(tt.path); got != tt.want {
			t.Errorf("PathToURI(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestURIRoundTrip(t *testing.T) {
	for _, path := range []string{
		"/home/user/doc.md",
		"/home/user/my doc.md",
		"/home/user/a#b.md",
		"/home/user/ünïcode.md",
	} {
		uri := PathToURI(path)
		got, err := URIToPath(uri)
		if err != nil {
			t.Fatalf("URIToPath(%q): %v", uri, err)
		}
		if got != path {
			t.Errorf("round trip of %q gave %q (uri %q)", path, got, uri)
		}
	}
}

func TestURIToPathWindowsDrive(t *testing.T) {
	got, err := URIToPath("file:///C:/Users/me/doc.md")
	if err != nil {
		t.Fatalf("URIToPath: %v", err)
	}
	if got != `C:/Users/me/doc.md` && got != `C:\Users\me\doc.md` {
		t.Errorf("URIToPath drive path = %q", got)
	}
}
