package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}    `json:"result,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	scanner.Split(scanLSPMessages)

	s.SetNotify(func(method string, params interface{}) {
		notif := Notification{
			JSONRPC: "2.0",
			Method:  method,
			Params:  params,
		}
		writeMessage(out, notif)
	})

	for scanner.Scan() {
		body := scanner.Bytes()
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		if req.ID != nil {
			result, err := s.dispatch(ctx, req.Method, req.Params)
			resp := Response{JSONRPC: "2.0", ID: req.ID}
			if err != nil {
				resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
			} else {
				resp.Result = result
			}
			writeMessage(out, resp)
		} else {
			s.dispatchNotification(ctx, req.Method, req.Params)
		}
	}
	return scanner.Err()
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "initialize":
		return s.handleInitialize(ctx, params)
	case "textDocument/completion":
		return s.handleCompletion(ctx, params)
	case "textDocument/definition":
		return s.handleDefinition(ctx, params)
	case "textDocument/hover":
		return s.handleHover(ctx, params)
	case "textDocument/references":
		return s.handleReferences(ctx, params)
	case "textDocument/rename":
		return s.handleRename(ctx, params)
	case "textDocument/codeAction":
		return s.handleCodeAction(ctx, params)
	case "textDocument/codeLens":
		return s.handleCodeLens(ctx, params)
	case "textDocument/documentSymbol":
		return s.handleDocumentSymbol(ctx, params)
	case "textDocument/semanticTokens/full":
		return s.handleSemanticTokensFull(ctx, params)
	case "shutdown":
		return nil, nil
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (s *Server) dispatchNotification(ctx context.Context, method string, params json.RawMessage) {
	switch method {
	case "initialized":
		// no-op
	case "textDocument/didOpen":
		s.handleDidOpen(ctx, params)
	case "textDocument/didChange":
		s.handleDidChange(ctx, params)
	case "textDocument/didClose":
		s.handleDidClose(ctx, params)
	case "exit":
		// handled by caller
	}
}

func writeMessage(w io.Writer, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	_, err = io.WriteString(w, header)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func scanLSPMessages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	headerEnd := strings.Index(string(data), "\r\n\r\n")
	if headerEnd < 0 {
		if atEOF {
			return 0, nil, fmt.Errorf("incomplete header")
		}
		return 0, nil, nil
	}

	header := string(data[:headerEnd])
	contentLength := 0
	for _, line := range strings.Split(header, "\r\n") {
		if strings.HasPrefix(line, "Content-Length: ") {
			cl := strings.TrimPrefix(line, "Content-Length: ")
			contentLength, _ = strconv.Atoi(strings.TrimSpace(cl))
		}
	}

	if contentLength == 0 {
		return 0, nil, fmt.Errorf("missing Content-Length")
	}

	totalLen := headerEnd + 4 + contentLength
	if len(data) < totalLen {
		if atEOF {
			return 0, nil, fmt.Errorf("incomplete message")
		}
		return 0, nil, nil
	}

	body := data[headerEnd+4 : totalLen]
	return totalLen, body, nil
}
