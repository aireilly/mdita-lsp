package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func buildMessage(method string, id *int, params any) string {
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if id != nil {
		req["id"] = *id
	}
	if params != nil {
		req["params"] = params
	}
	body, _ := json.Marshal(req)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func intPtr(i int) *int { return &i }

func TestFullLSPLifecycle(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test",
	}))

	input.WriteString(buildMessage("initialized", nil, nil))

	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":        "file:///tmp/lsp-test/doc.md",
			"languageId": "markdown",
			"version":    1,
			"text":       "# Hello World\n\n## Section\n\nSome [[#section]] link.\n",
		},
	}))

	input.WriteString(buildMessage("textDocument/hover", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test/doc.md"},
		"position":     map[string]any{"line": 0, "character": 3},
	}))

	input.WriteString(buildMessage("textDocument/documentSymbol", intPtr(3), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test/doc.md"},
	}))

	input.WriteString(buildMessage("shutdown", intPtr(4), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Serve(ctx, &input, &output)
	if err != nil && !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("Serve error: %v", err)
	}

	out := output.String()
	if !strings.Contains(out, "\"id\":1") {
		t.Error("missing initialize response")
	}
	if !strings.Contains(out, "completionProvider") {
		t.Error("missing capabilities in initialize response")
	}
	if !strings.Contains(out, "mdita-lsp") {
		t.Error("missing serverInfo.name in initialize response")
	}
	if !strings.Contains(out, "linkedEditingRangeProvider") {
		t.Error("missing linkedEditingRangeProvider capability")
	}
	if !strings.Contains(out, "inlayHintProvider") {
		t.Error("missing inlayHintProvider capability")
	}
	if !strings.Contains(out, "documentFormattingProvider") {
		t.Error("missing documentFormattingProvider capability")
	}
	if !strings.Contains(out, "executeCommandProvider") {
		t.Error("missing executeCommandProvider capability")
	}
}

func TestLSPCompletion(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-comp",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-comp/doc.md",
			"version": 1,
			"text":    "# Title\n\n[[\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/completion", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-comp/doc.md"},
		"position":     map[string]any{"line": 2, "character": 2},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing completion response")
	}
}

func TestLSPFormatting(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-fmt",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-fmt/doc.md",
			"version": 1,
			"text":    "#Title  \n\nContent   \n",
		},
	}))
	input.WriteString(buildMessage("textDocument/formatting", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-fmt/doc.md"},
		"options":      map[string]any{"tabSize": 4, "insertSpaces": true},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing formatting response")
	}
	if !strings.Contains(out, "newText") {
		t.Error("expected formatting edits")
	}
}

func TestLSPPullDiagnostics(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-diag",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-diag/doc.md",
			"version": 1,
			"text":    "# Title\n\n[[nonexistent]].\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/diagnostic", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-diag/doc.md"},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing pull diagnostics response")
	}
	if !strings.Contains(out, "\"kind\":\"full\"") {
		t.Error("expected full diagnostic report")
	}
}

func TestLSPUnknownMethod(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-unk",
	}))
	input.WriteString(buildMessage("unknownMethod/foo", intPtr(2), nil))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "method not found") {
		t.Error("expected error for unknown method")
	}
	if !strings.Contains(out, "-32601") {
		t.Error("expected LSP error code -32601 (MethodNotFound)")
	}
}

func TestLSPDocumentHighlight(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-hl",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-hl/doc.md",
			"version": 1,
			"text":    "# Title\n\n## Section\n\nSee [[#Section]].\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/documentHighlight", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-hl/doc.md"},
		"position":     map[string]any{"line": 2, "character": 3},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing highlight response")
	}
	if !strings.Contains(out, "\"kind\"") {
		t.Error("expected highlight results with kind field")
	}
}

func TestLSPCodeAction(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-ca",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-ca/doc.md",
			"version": 1,
			"text":    "# Title\n\n## Section\n\nContent.\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/codeAction", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-ca/doc.md"},
		"range": map[string]any{
			"start": map[string]any{"line": 0, "character": 0},
			"end":   map[string]any{"line": 5, "character": 0},
		},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing code action response")
	}
	if !strings.Contains(out, "Generate table of contents") {
		t.Error("expected ToC code action")
	}
}

func TestLSPDefinition(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-def",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-def/doc.md",
			"version": 1,
			"text":    "# Title\n\n## Section\n\nSee [[#Section]].\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/definition", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-def/doc.md"},
		"position":     map[string]any{"line": 4, "character": 7},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing definition response")
	}
}

func TestLSPSemanticTokens(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-sem",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-sem/doc.md",
			"version": 1,
			"text":    "# Title\n\n[[other]] link.\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/semanticTokens/full", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-sem/doc.md"},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing semantic tokens response")
	}
	if !strings.Contains(out, "\"data\"") {
		t.Error("expected data field in semantic tokens response")
	}
}

func TestLSPFoldingRange(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]any{
		"capabilities": map[string]any{},
		"rootUri":      "file:///tmp/lsp-test-fold",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri":     "file:///tmp/lsp-test-fold/doc.md",
			"version": 1,
			"text":    "# Title\n\nParagraph.\n\n## Section\n\nContent.\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/foldingRange", intPtr(2), map[string]any{
		"textDocument": map[string]any{"uri": "file:///tmp/lsp-test-fold/doc.md"},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing folding range response")
	}
	if !strings.Contains(out, "startLine") {
		t.Error("expected folding ranges with startLine")
	}
}
