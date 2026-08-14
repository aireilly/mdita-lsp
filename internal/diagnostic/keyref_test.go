package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestUnresolvedKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/install.md", 1,
		"# Install\n\nSee [nonexistent-key].\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)

	diags := CheckKeyrefs(topicDoc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeUnresolvedKeyref {
			found = true
		}
	}
	if !found {
		t.Error("expected UnresolvedKeyref diagnostic")
	}
}

func TestResolvedKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/guide.md", 1,
		"# Guide\n\nSee [install].\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)

	diags := CheckKeyrefs(topicDoc, f)
	for _, d := range diags {
		if d.Code == CodeUnresolvedKeyref {
			t.Errorf("should not report UnresolvedKeyref for valid key, got: %s", d.Message)
		}
	}
}
