package lsp

import (
	"context"
	"encoding/json"
	"strings"
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

func TestInitializeHasServerInfo(t *testing.T) {
	s := NewServer()
	s.SetVersion("1.2.3")
	result, err := s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test-si"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(result)
	var ir InitializeResult
	json.Unmarshal(data, &ir)
	if ir.ServerInfo == nil {
		t.Fatal("missing serverInfo")
	}
	if ir.ServerInfo.Name != "mdita-lsp" {
		t.Errorf("expected name mdita-lsp, got %s", ir.ServerInfo.Name)
	}
	if ir.ServerInfo.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", ir.ServerInfo.Version)
	}
}

func TestInitializeHasDiagnosticProvider(t *testing.T) {
	s := NewServer()
	result, err := s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test-dp"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(result)
	var ir InitializeResult
	json.Unmarshal(data, &ir)
	if ir.Capabilities.DiagnosticProvider == nil {
		t.Error("missing diagnosticProvider")
	}
	if !ir.Capabilities.DiagnosticProvider.InterFileDependencies {
		t.Error("expected interFileDependencies=true")
	}
}

func TestWillCreateFilesPopulatesFrontMatter(t *testing.T) {
	s := NewServer()
	s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test-wcf"
	}`))

	result, err := s.handleWillCreateFiles(context.Background(), json.RawMessage(`{
		"files": [{"uri": "file:///tmp/test-wcf/my-topic.md"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected workspace edit")
	}
	data, _ := json.Marshal(result)
	out := string(data)
	if !strings.Contains(out, "$schema") {
		t.Error("expected YAML front matter with $schema")
	}
	if !strings.Contains(out, "my-topic") {
		t.Error("expected title derived from filename")
	}
}

func TestMultiChangeIncremental(t *testing.T) {
	s := NewServer()
	s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test-mc"
	}`))

	s.handleDidOpen(context.Background(), json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test-mc/doc.md",
			"version": 1,
			"text": "# Title\n\nLine A\nLine B\n"
		}
	}`))

	s.handleDidChange(context.Background(), json.RawMessage(`{
		"textDocument": {"uri": "file:///tmp/test-mc/doc.md", "version": 2},
		"contentChanges": [
			{"range": {"start": {"line": 2, "character": 5}, "end": {"line": 2, "character": 6}}, "text": "1"},
			{"range": {"start": {"line": 3, "character": 5}, "end": {"line": 3, "character": 6}}, "text": "2"}
		]
	}`))

	doc, _ := s.workspace.FindDoc("file:///tmp/test-mc/doc.md")
	if doc == nil {
		t.Fatal("document not found")
	}
	if doc.Text != "# Title\n\nLine 1\nLine 2\n" {
		t.Errorf("multi-change sync failed, got %q", doc.Text)
	}
}
