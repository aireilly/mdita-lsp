package lsp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestHandleWillSaveWaitUntilTableAlignment(t *testing.T) {
	s := &Server{
		workspace: workspace.New(),
	}

	cfg := config.Default()
	folder := workspace.NewFolder("file:///test", cfg)
	s.workspace.AddFolder(folder)

	misalignedTable := "| a | bb |\n|---|---|\n| ccc | d |\n"
	doc := document.New("file:///test/doc.md", 1, misalignedTable)
	folder.AddDoc(doc)

	params := WillSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test/doc.md",
		},
		Reason: 1,
	}

	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	result, err := s.handleWillSaveWaitUntil(context.Background(), rawParams)
	if err != nil {
		t.Fatalf("handleWillSaveWaitUntil failed: %v", err)
	}

	edits, ok := result.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", result)
	}

	if len(edits) == 0 {
		t.Error("expected table alignment edits, got none")
	}

	if len(edits) > 0 {
		edit := edits[0]
		if _, ok := edit["range"]; !ok {
			t.Error("expected 'range' in edit")
		}
		if _, ok := edit["newText"]; !ok {
			t.Error("expected 'newText' in edit")
		}
	}
}

func TestHandleWillSaveWaitUntilConfigDisabled(t *testing.T) {
	s := &Server{
		workspace: workspace.New(),
	}

	cfg := config.Default()
	disabled := false
	cfg.Core.Mdita.FormatTablesOnSave = &disabled

	folder := workspace.NewFolder("file:///test", cfg)
	s.workspace.AddFolder(folder)

	misalignedTable := "| a | bb |\n|---|---|\n| ccc | d |\n"
	doc := document.New("file:///test/doc.md", 1, misalignedTable)
	folder.AddDoc(doc)

	params := WillSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test/doc.md",
		},
		Reason: 1,
	}

	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	result, err := s.handleWillSaveWaitUntil(context.Background(), rawParams)
	if err != nil {
		t.Fatalf("handleWillSaveWaitUntil failed: %v", err)
	}

	edits, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(edits) != 0 {
		t.Error("expected no edits when format tables on save is disabled")
	}
}

func TestHandleWillSaveWaitUntilNoDocument(t *testing.T) {
	s := &Server{
		workspace: workspace.New(),
	}

	cfg := config.Default()
	folder := workspace.NewFolder("file:///test", cfg)
	s.workspace.AddFolder(folder)

	params := WillSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test/nonexistent.md",
		},
		Reason: 1,
	}

	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	result, err := s.handleWillSaveWaitUntil(context.Background(), rawParams)
	if err != nil {
		t.Fatalf("handleWillSaveWaitUntil failed: %v", err)
	}

	edits, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(edits) != 0 {
		t.Error("expected no edits for nonexistent document")
	}
}

func TestHandleWillSaveWaitUntilAlignedTable(t *testing.T) {
	s := &Server{
		workspace: workspace.New(),
	}

	cfg := config.Default()
	folder := workspace.NewFolder("file:///test", cfg)
	s.workspace.AddFolder(folder)

	alignedTable := "| a   | bb  |\n| --- | --- |\n| ccc | d   |\n"
	doc := document.New("file:///test/doc.md", 1, alignedTable)
	folder.AddDoc(doc)

	params := WillSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test/doc.md",
		},
		Reason: 1,
	}

	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	result, err := s.handleWillSaveWaitUntil(context.Background(), rawParams)
	if err != nil {
		t.Fatalf("handleWillSaveWaitUntil failed: %v", err)
	}

	edits, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(edits) != 0 {
		t.Error("expected no edits for already-aligned table")
	}
}
