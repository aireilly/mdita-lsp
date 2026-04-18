package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestFolderAddRemoveDoc(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	doc := document.New("file:///project/doc.md", 1, "# Test\n")
	f.AddDoc(doc)

	if f.DocCount() != 1 {
		t.Errorf("DocCount = %d, want 1", f.DocCount())
	}

	got := f.DocByURI("file:///project/doc.md")
	if got == nil {
		t.Fatal("DocByURI returned nil")
	}

	f.RemoveDoc("file:///project/doc.md")
	if f.DocCount() != 0 {
		t.Errorf("DocCount after remove = %d, want 0", f.DocCount())
	}
}

func TestFolderDocBySlug(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	doc := document.New("file:///project/my-topic.md", 1, "# My Topic\n")
	f.AddDoc(doc)

	got := f.DocBySlug(paths.SlugOf("my-topic"))
	if got == nil {
		t.Fatal("DocBySlug returned nil")
	}
}

func TestWorkspaceAddRemoveFolder(t *testing.T) {
	ws := New()
	f := NewFolder("file:///project", config.Default())
	ws.AddFolder(f)

	if len(ws.Folders()) != 1 {
		t.Errorf("Folders = %d, want 1", len(ws.Folders()))
	}

	ws.RemoveFolder("file:///project")
	if len(ws.Folders()) != 0 {
		t.Errorf("Folders after remove = %d", len(ws.Folders()))
	}
}

func TestWorkspaceFindDoc(t *testing.T) {
	ws := New()
	f := NewFolder("file:///project", config.Default())
	doc := document.New("file:///project/intro.md", 1, "# Intro\n")
	f.AddDoc(doc)
	ws.AddFolder(f)

	got, folder := ws.FindDoc("file:///project/intro.md")
	if got == nil || folder == nil {
		t.Fatal("FindDoc returned nil")
	}
}

func TestResolveLinkRelativePath(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	target := document.New("file:///project/docs/install.md", 1, "# Install\n")
	source := document.New("file:///project/guide/index.md", 1, "# Guide\n")
	f.AddDoc(target)
	f.AddDoc(source)

	got := f.ResolveLink("../docs/install.md", source.URI)
	if got == nil {
		t.Fatal("ResolveLink returned nil for relative path")
	}
	if got.URI != target.URI {
		t.Errorf("URI = %q, want %q", got.URI, target.URI)
	}
}

func TestResolveLinkSameDir(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	target := document.New("file:///project/install.md", 1, "# Install\n")
	source := document.New("file:///project/guide.md", 1, "# Guide\n")
	f.AddDoc(target)
	f.AddDoc(source)

	got := f.ResolveLink("install.md", source.URI)
	if got == nil {
		t.Fatal("ResolveLink returned nil for same-dir path")
	}
}

func TestResolveLinkNonexistent(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	source := document.New("file:///project/guide.md", 1, "# Guide\n")
	f.AddDoc(source)

	got := f.ResolveLink("nonexistent.md", source.URI)
	if got != nil {
		t.Error("ResolveLink should return nil for nonexistent file")
	}
}

func TestResolveLinkHTTP(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	source := document.New("file:///project/guide.md", 1, "# Guide\n")
	f.AddDoc(source)

	if f.ResolveLink("http://example.com", source.URI) != nil {
		t.Error("ResolveLink should return nil for http URL")
	}
	if f.ResolveLink("https://example.com", source.URI) != nil {
		t.Error("ResolveLink should return nil for https URL")
	}
}

func TestFolderScanFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Doc\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "notes.markdown"), []byte("# Notes\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "map.mditamap"), []byte("# Map\n- [D](doc.md)\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("not md"), 0644)

	cfg := config.Default()
	f := NewFolder(paths.PathToURI(dir), cfg)
	err := f.ScanFiles()
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}
	if f.DocCount() != 3 {
		t.Errorf("DocCount = %d, want 3", f.DocCount())
	}
}
