package lsp

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestApplyIncrementalChange(t *testing.T) {
	text := "Hello World\nLine two\n"
	lineMap := []int{0, 12}

	result := applyIncrementalChange(text, lineMap, document.Range{
		Start: document.Position{Line: 0, Character: 5},
		End:   document.Position{Line: 0, Character: 11},
	}, " Go")

	if result != "Hello Go\nLine two\n" {
		t.Errorf("got %q", result)
	}
}

func TestApplyIncrementalInsert(t *testing.T) {
	text := "AB\n"
	lineMap := []int{0}

	result := applyIncrementalChange(text, lineMap, document.Range{
		Start: document.Position{Line: 0, Character: 1},
		End:   document.Position{Line: 0, Character: 1},
	}, "X")

	if result != "AXB\n" {
		t.Errorf("got %q", result)
	}
}

func TestApplyIncrementalDelete(t *testing.T) {
	text := "ABCD\n"
	lineMap := []int{0}

	result := applyIncrementalChange(text, lineMap, document.Range{
		Start: document.Position{Line: 0, Character: 1},
		End:   document.Position{Line: 0, Character: 3},
	}, "")

	if result != "AD\n" {
		t.Errorf("got %q", result)
	}
}

func TestApplyFullChange(t *testing.T) {
	text := "old content"
	lineMap := []int{0}

	result := applyIncrementalChange(text, lineMap, document.Range{
		Start: document.Position{Line: 0, Character: 0},
		End:   document.Position{Line: 0, Character: 11},
	}, "new content")

	if result != "new content" {
		t.Errorf("got %q", result)
	}
}
