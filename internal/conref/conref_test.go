package conref

import (
	"testing"
)

func TestParseConref(t *testing.T) {
	text := `<p data-conref="shared.md#topic/warning-para">fallback</p>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].FilePath != "shared.md" {
		t.Errorf("FilePath = %q, want %q", refs[0].FilePath, "shared.md")
	}
	if refs[0].TopicID != "topic" {
		t.Errorf("TopicID = %q, want %q", refs[0].TopicID, "topic")
	}
	if refs[0].ElementID != "warning-para" {
		t.Errorf("ElementID = %q, want %q", refs[0].ElementID, "warning-para")
	}
	if refs[0].IsKeyref {
		t.Error("should not be a keyref")
	}
}

func TestParseConkeyref(t *testing.T) {
	text := `<span data-conkeyref="warnings/disk-full">fallback</span>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].KeyName != "warnings" {
		t.Errorf("KeyName = %q, want %q", refs[0].KeyName, "warnings")
	}
	if refs[0].ElementID != "disk-full" {
		t.Errorf("ElementID = %q, want %q", refs[0].ElementID, "disk-full")
	}
	if !refs[0].IsKeyref {
		t.Error("should be a keyref")
	}
}

func TestParseNoConref(t *testing.T) {
	text := `<p>Regular paragraph with no conref.</p>`
	refs := Parse(text)
	if len(refs) != 0 {
		t.Errorf("got %d conrefs, want 0", len(refs))
	}
}

func TestParseConrefNoElement(t *testing.T) {
	text := `<p data-conref="notes.md#concept">fallback</p>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].FilePath != "notes.md" {
		t.Errorf("FilePath = %q, want %q", refs[0].FilePath, "notes.md")
	}
	if refs[0].TopicID != "concept" {
		t.Errorf("TopicID = %q, want %q", refs[0].TopicID, "concept")
	}
	if refs[0].ElementID != "" {
		t.Errorf("ElementID = %q, want empty", refs[0].ElementID)
	}
}

func TestParseConrefFileOnly(t *testing.T) {
	text := `<div data-conref="shared.md">fallback</div>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].FilePath != "shared.md" {
		t.Errorf("FilePath = %q, want %q", refs[0].FilePath, "shared.md")
	}
	if refs[0].TopicID != "" {
		t.Errorf("TopicID = %q, want empty", refs[0].TopicID)
	}
}

func TestParseConkeyrefKeyOnly(t *testing.T) {
	text := `<span data-conkeyref="mykey">fallback</span>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].KeyName != "mykey" {
		t.Errorf("KeyName = %q, want %q", refs[0].KeyName, "mykey")
	}
	if refs[0].ElementID != "" {
		t.Errorf("ElementID = %q, want empty", refs[0].ElementID)
	}
}
