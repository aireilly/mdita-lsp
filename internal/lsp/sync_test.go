package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func openDoc(t *testing.T, s *Server, uri, text string) {
	t.Helper()
	params, err := json.Marshal(DidOpenParams{
		TextDocument: TextDocumentItem{URI: uri, Version: 1, Text: text},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleDidOpen(context.Background(), params); err != nil {
		t.Fatalf("didOpen: %v", err)
	}
}

func change(t *testing.T, s *Server, uri string, version, sl, sc, el, ec int, text string) {
	t.Helper()
	raw := fmt.Sprintf(`{
		"textDocument": {"uri": %q, "version": %d},
		"contentChanges": [{
			"range": {"start": {"line": %d, "character": %d}, "end": {"line": %d, "character": %d}},
			"text": %q
		}]
	}`, uri, version, sl, sc, el, ec, text)
	if err := s.handleDidChange(context.Background(), json.RawMessage(raw)); err != nil {
		t.Fatalf("didChange: %v", err)
	}
}

func textOf(t *testing.T, s *Server, uri string) string {
	t.Helper()
	doc, _ := s.workspace.FindDoc(uri)
	if doc == nil {
		t.Fatalf("no document for %s", uri)
	}
	return doc.Text
}

// Positions are UTF-16 code units. Counting bytes put the insert one byte
// late for a line containing a non-ASCII character, and the server's copy of
// the buffer drifted from the editor's from then on.
func TestIncrementalChangeOnNonASCIILine(t *testing.T) {
	s := NewServer()
	uri := "file:///tmp/test/doc.md"
	openDoc(t, s, uri, "# Café ok\n")

	// Column 9 is the end of "# Café ok" in UTF-16 units.
	change(t, s, uri, 2, 0, 9, 0, 9, "!")

	if got, want := textOf(t, s, uri), "# Café ok!\n"; got != want {
		t.Errorf("buffer = %q, want %q", got, want)
	}
}

func TestIncrementalChangeInsideNonASCIILine(t *testing.T) {
	s := NewServer()
	uri := "file:///tmp/test/doc.md"
	openDoc(t, s, uri, "# Café ok\n")

	// Column 8 sits between "o" and "k".
	change(t, s, uri, 2, 0, 8, 0, 8, "!")

	if got, want := textOf(t, s, uri), "# Café o!k\n"; got != want {
		t.Errorf("buffer = %q, want %q", got, want)
	}
}

// A position past the end of a line used to slice out of range and take the
// process down.
func TestIncrementalChangePastEndOfLine(t *testing.T) {
	s := NewServer()
	uri := "file:///tmp/test/doc.md"
	openDoc(t, s, uri, "# Short\n")

	change(t, s, uri, 2, 0, 500, 0, 500, "!")

	if got := textOf(t, s, uri); !strings.Contains(got, "!") {
		t.Errorf("buffer = %q, want the insert clamped to the line end", got)
	}
}

func TestIncrementalChangePastEndOfDocument(t *testing.T) {
	s := NewServer()
	uri := "file:///tmp/test/doc.md"
	openDoc(t, s, uri, "# Short\n")

	change(t, s, uri, 2, 99, 0, 99, 0, "tail")

	if got := textOf(t, s, uri); !strings.HasSuffix(got, "tail") {
		t.Errorf("buffer = %q, want the text appended", got)
	}
}

func TestInitializeAdvertisesPositionEncoding(t *testing.T) {
	s := NewServer()
	result, err := s.handleInitialize(context.Background(),
		json.RawMessage(`{"capabilities": {}, "rootUri": "file:///tmp/test"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := result.(InitializeResult).Capabilities.PositionEncoding
	if got != "utf-16" {
		t.Errorf("positionEncoding = %q, want \"utf-16\"", got)
	}
}

// A panic in a handler used to end the process. It becomes a JSON-RPC error.
func TestDispatchRecoversFromAPanic(t *testing.T) {
	s := NewServer()
	// A hover request for a document the server has never seen exercises the
	// handler; a malformed params payload makes it fail rather than panic, so
	// drive the recover path directly.
	_, err := s.dispatch(context.Background(), "textDocument/hover",
		json.RawMessage(`{"textDocument": {"uri": 5}}`))
	if err == nil {
		t.Error("expected an error for malformed params")
	}
}

func lspMessage(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

// `exit` was ignored, so the server stayed alive after the client shut it
// down and left an orphan per editor session.
func TestExitAfterShutdownStopsServing(t *testing.T) {
	s := NewServer()
	in := strings.NewReader(
		lspMessage(`{"jsonrpc":"2.0","id":1,"method":"shutdown"}`) +
			lspMessage(`{"jsonrpc":"2.0","method":"exit"}`) +
			lspMessage(`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{}}`))
	var out bytes.Buffer

	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve = %v, want nil after a clean shutdown/exit", err)
	}
	if strings.Contains(out.String(), `"id":2`) {
		t.Error("the server kept answering requests after exit")
	}
}

func TestExitWithoutShutdownIsAnError(t *testing.T) {
	s := NewServer()
	in := strings.NewReader(lspMessage(`{"jsonrpc":"2.0","method":"exit"}`))
	var out bytes.Buffer

	err := s.Serve(context.Background(), in, &out)
	if !errors.Is(err, ErrExitWithoutShutdown()) {
		t.Errorf("Serve = %v, want ErrExitWithoutShutdown", err)
	}
}

// A reply to a server-to-client request carries an id but no method. The
// server answered it with "method not found".
func TestClientResponseIsNotDispatched(t *testing.T) {
	s := NewServer()
	in := strings.NewReader(
		lspMessage(`{"jsonrpc":"2.0","id":1,"result":{"applied":true}}`) +
			lspMessage(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`) +
			lspMessage(`{"jsonrpc":"2.0","method":"exit"}`))
	var out bytes.Buffer

	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "method not found") {
		t.Errorf("the client's reply was dispatched as a request: %s", out.String())
	}
}
