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

func buildMessage(method string, id *int, params interface{}) string {
	req := map[string]interface{}{
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

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]interface{}{
		"capabilities": map[string]interface{}{},
		"rootUri":      "file:///tmp/lsp-test",
	}))

	input.WriteString(buildMessage("initialized", nil, nil))

	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        "file:///tmp/lsp-test/doc.md",
			"languageId": "markdown",
			"version":    1,
			"text":       "# Hello World\n\n## Section\n\nSome [[#section]] link.\n",
		},
	}))

	input.WriteString(buildMessage("textDocument/hover", intPtr(2), map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": "file:///tmp/lsp-test/doc.md"},
		"position":     map[string]interface{}{"line": 0, "character": 3},
	}))

	input.WriteString(buildMessage("textDocument/documentSymbol", intPtr(3), map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": "file:///tmp/lsp-test/doc.md"},
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

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]interface{}{
		"capabilities": map[string]interface{}{},
		"rootUri":      "file:///tmp/lsp-test-comp",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     "file:///tmp/lsp-test-comp/doc.md",
			"version": 1,
			"text":    "# Title\n\n[[\n",
		},
	}))
	input.WriteString(buildMessage("textDocument/completion", intPtr(2), map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": "file:///tmp/lsp-test-comp/doc.md"},
		"position":     map[string]interface{}{"line": 2, "character": 2},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing completion response")
	}
}

func TestLSPFormatting(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]interface{}{
		"capabilities": map[string]interface{}{},
		"rootUri":      "file:///tmp/lsp-test-fmt",
	}))
	input.WriteString(buildMessage("initialized", nil, nil))
	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     "file:///tmp/lsp-test-fmt/doc.md",
			"version": 1,
			"text":    "#Title  \n\nContent   \n",
		},
	}))
	input.WriteString(buildMessage("textDocument/formatting", intPtr(2), map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": "file:///tmp/lsp-test-fmt/doc.md"},
		"options":      map[string]interface{}{"tabSize": 4, "insertSpaces": true},
	}))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Error("missing formatting response")
	}
	if !strings.Contains(out, "newText") {
		t.Error("expected formatting edits")
	}
}

func TestLSPUnknownMethod(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]interface{}{
		"capabilities": map[string]interface{}{},
		"rootUri":      "file:///tmp/lsp-test-unk",
	}))
	input.WriteString(buildMessage("unknownMethod/foo", intPtr(2), nil))
	input.WriteString(buildMessage("shutdown", intPtr(3), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.Serve(ctx, &input, &output)

	out := output.String()
	if !strings.Contains(out, "method not found") {
		t.Error("expected error for unknown method")
	}
}
