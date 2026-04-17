package lsp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestInitializeResponse(t *testing.T) {
	s := NewServer()
	params := json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`)

	result, err := s.handleInitialize(context.Background(), params)
	if err != nil {
		t.Fatalf("Initialize error: %v", err)
	}

	var initResult InitializeResult
	data, _ := json.Marshal(result)
	json.Unmarshal(data, &initResult)

	if initResult.Capabilities.CompletionProvider == nil {
		t.Error("missing completion provider")
	}
	if !initResult.Capabilities.DefinitionProvider {
		t.Error("missing definition provider")
	}
	if !initResult.Capabilities.HoverProvider {
		t.Error("missing hover provider")
	}
	if !initResult.Capabilities.ReferencesProvider {
		t.Error("missing references provider")
	}
	if initResult.Capabilities.RenameProvider == nil {
		t.Error("missing rename provider")
	}
}

func TestDidOpenAndDiagnostics(t *testing.T) {
	s := NewServer()

	s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`))

	openParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test/doc.md",
			"languageId": "markdown",
			"version": 1,
			"text": "# Title\n\n[[nonexistent]]\n"
		}
	}`)

	err := s.handleDidOpen(context.Background(), openParams)
	if err != nil {
		t.Fatalf("DidOpen error: %v", err)
	}

	doc, _ := s.workspace.FindDoc("file:///tmp/test/doc.md")
	if doc == nil {
		t.Fatal("document not found after open")
	}
}

func TestCompletion(t *testing.T) {
	s := NewServer()
	s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`))

	s.handleDidOpen(context.Background(), json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test/intro.md",
			"languageId": "markdown",
			"version": 1,
			"text": "# Introduction\n\nContent.\n"
		}
	}`))

	s.handleDidOpen(context.Background(), json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test/doc.md",
			"languageId": "markdown",
			"version": 1,
			"text": "# Doc\n\n[[int"
		}
	}`))

	result, err := s.handleCompletion(context.Background(), json.RawMessage(`{
		"textDocument": {"uri": "file:///tmp/test/doc.md"},
		"position": {"line": 2, "character": 5}
	}`))
	if err != nil {
		t.Fatalf("Completion error: %v", err)
	}
	items, ok := result.([]CompletionItemResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if len(items) == 0 {
		t.Error("expected completion items")
	}
}
