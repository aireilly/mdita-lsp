package document

import (
	"strings"
	"testing"
)

func TestMaskFencedCodePreservesOffsets(t *testing.T) {
	src := "intro\n\n```\n[^1] and [in-code]\n```\n\nafter\n"
	got := MaskFencedCode(src)
	if len(got) != len(src) {
		t.Fatalf("length changed: %d -> %d", len(src), len(got))
	}
	if strings.Contains(got, "[^1]") || strings.Contains(got, "[in-code]") {
		t.Errorf("fenced content was not masked: %q", got)
	}
	if !strings.Contains(got, "intro") || !strings.Contains(got, "after") {
		t.Errorf("content outside the fence was masked: %q", got)
	}
}

func TestMaskFencedCodeLeavesPlainTextAlone(t *testing.T) {
	src := "a [^1] footnote\n\n[^1]: the note\n"
	if got := MaskFencedCode(src); got != src {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestMaskFencedCodeHandlesTildes(t *testing.T) {
	src := "~~~yaml\nkey: [value]\n~~~\n"
	got := MaskFencedCode(src)
	if strings.Contains(got, "[value]") {
		t.Errorf("tilde fence not masked: %q", got)
	}
}

func TestFootnoteInFencedBlockIsNotReported(t *testing.T) {
	doc := New("file:///test.md", 1, "# T\n\n```\n[^1] sample\n```\n")
	if n := len(doc.Index.Features.FootnoteRefLabels); n != 0 {
		t.Errorf("found %d footnote refs inside a fenced block, want 0", n)
	}
}
