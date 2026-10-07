package diagnostic

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestBrokenMapReference(t *testing.T) {
	mapDoc := makeDoc("file:///project/map.mditamap",
		"# My Map", "", "- [Topic](nonexistent.md)")
	f := makeFolder(mapDoc)
	diags := CheckDitamap(mapDoc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenMapTopicref {
			found = true
		}
	}
	if !found {
		t.Error("expected BrokenMapReference diagnostic")
	}
}

func TestValidMapReference(t *testing.T) {
	topic := makeDoc("file:///project/topic.md", "# Topic", "", "Content.")
	mapDoc := makeDoc("file:///project/map.mditamap",
		"# My Map", "", "- [Topic](topic.md)")
	f := makeFolder(topic, mapDoc)
	diags := CheckDitamap(mapDoc, f)

	for _, d := range diags {
		if d.Code == CodeBrokenMapTopicref {
			t.Error("should not report BrokenMapReference for valid ref")
		}
	}
}

func TestCircularMapReference(t *testing.T) {
	map1 := document.New("file:///project/a.mditamap", 1,
		"# Map A\n\n- [Map B](b.mditamap)\n")
	map2 := document.New("file:///project/b.mditamap", 1,
		"# Map B\n\n- [Map A](a.mditamap)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(map1)
	f.AddDoc(map2)

	diags := CheckDitamap(map1, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeCircularMapReference {
			found = true
		}
	}
	if !found {
		t.Error("expected CircularMapReference diagnostic")
	}
}

// Nesting a topicref under another is how a map builds a hierarchy; the
// plug-in does not expect the nested topic's heading level to match the
// nesting depth and emits nothing about it. The server used to report it,
// which was noise on every working map.
func TestNestedTopicrefToH1TopicIsNotReported(t *testing.T) {
	parent := document.New("file:///project/parent.md", 1,
		"# Parent Topic\n\nSome content.\n")
	child := document.New("file:///project/child.md", 1,
		"# Child Topic\n\nChild content.\n")

	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Parent](parent.md)\n  - [Child](child.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(parent)
	f.AddDoc(child)
	f.AddDoc(mapDoc)

	for _, d := range CheckDitamap(mapDoc, f) {
		t.Errorf("unexpected diagnostic %s: %s", d.Code, d.Message)
	}
}

func TestMapRefDiagnosticsSkipNonFileHrefs(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Site](https://example.com/page)\n"+
			"- [Anchor](#my-map)\n"+
			"- [Other map](other.ditamap)\n"+
			"- [Root](/elsewhere.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	for _, d := range CheckDitamap(mapDoc, f) {
		if d.Code == CodeBrokenMapTopicref && !strings.Contains(d.Message, "other.ditamap") {
			t.Errorf("unexpected broken-topicref diagnostic: %s", d.Message)
		}
	}
}

func TestMapRefDiagnosticPointsAtTheTopicref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [One](missing-one.md)\n- [Two](missing-two.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	lines := map[string]int{}
	for _, d := range CheckDitamap(mapDoc, f) {
		if d.Code == CodeBrokenMapTopicref {
			lines[d.Message] = d.Range.Start.Line
		}
	}
	if got := lines["Map references non-existent file: missing-one.md"]; got != 2 {
		t.Errorf("first topicref reported on line %d, want 2", got)
	}
	if got := lines["Map references non-existent file: missing-two.md"]; got != 3 {
		t.Errorf("second topicref reported on line %d, want 3", got)
	}
}
