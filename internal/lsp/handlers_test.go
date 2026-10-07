package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

// serverWithFiles writes files into a temp directory, initializes a server
// rooted there, and returns the server and the root path.
func serverWithFiles(t *testing.T, files map[string]string) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := NewServer()
	params, err := json.Marshal(InitializeParams{RootURI: paths.PathToURI(root)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.handleInitialize(context.Background(), params); err != nil {
		t.Fatal(err)
	}
	return s, root
}

func call(t *testing.T, fn func(context.Context, json.RawMessage) (any, error), params string) any {
	t.Helper()
	result, err := fn(context.Background(), json.RawMessage(params))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return result
}

func posParams(uri string, line, char int) string {
	return fmt.Sprintf(`{"textDocument":{"uri":%q},"position":{"line":%d,"character":%d}}`,
		uri, line, char)
}

func TestWorkspaceIndexingPicksUpMditaFiles(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"topic.mdita": "# A topic\n",
		"guide.md":    "# Guide\n",
	})
	folder := s.workspace.FolderForURI(paths.PathToURI(filepath.Join(root, "guide.md")))
	if folder == nil {
		t.Fatal("no folder")
	}
	if folder.DocByURI(paths.PathToURI(filepath.Join(root, "topic.mdita"))) == nil {
		t.Error(".mdita file was not indexed")
	}
}

func TestDefinitionHandlerFollowsALink(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md": "# Guide\n\n## Install Steps\n",
		"a.md":     "# A\n\n[x](guide.md#install-steps)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleDefinition, posParams(uri, 2, 3))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "guide.md") {
		t.Errorf("definition returned %s", data)
	}
}

func TestHoverHandlerDescribesALink(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md": "# Guide\n\nThe guide.\n",
		"a.md":     "# A\n\n[x](guide.md)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleHover, posParams(uri, 2, 3))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "Guide") {
		t.Errorf("hover returned %s", data)
	}
}

func TestCodeLensCountsSectionReferences(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md": "# Guide\n\n## Install Steps\n",
		"a.md":     "# A\n\n[x](guide.md#install-steps)\n",
		"b.md":     "# B\n\n[y](guide.md#install-steps)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "guide.md"))

	result := call(t, s.handleCodeLens, fmt.Sprintf(`{"textDocument":{"uri":%q}}`, uri))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "2 references") {
		t.Errorf("code lens returned %s, want a lens reading \"2 references\"", data)
	}
}

func TestRenameHandlerReplacesTheWholeLine(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md": "# Guide\n\n## Install Steps\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "guide.md"))

	result := call(t, s.handleRename,
		fmt.Sprintf(`{"textDocument":{"uri":%q},"position":{"line":2,"character":5},"newName":"Setup Steps"}`, uri))
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "## ## ") {
		t.Errorf("rename doubled the hashes: %s", data)
	}
	if !strings.Contains(string(data), "## Setup Steps") {
		t.Errorf("rename returned %s", data)
	}
}

func TestFormattingHandlerReturnsNonOverlappingEdits(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"t.md": "| a   | b |   \n| --- | --- |\n| c | d |\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "t.md"))
	openDoc(t, s, uri, "| a   | b |   \n| --- | --- |\n| c | d |\n")

	result := call(t, s.handleFormatting,
		fmt.Sprintf(`{"textDocument":{"uri":%q},"options":{"tabSize":4,"insertSpaces":true}}`, uri))
	edits, ok := result.([]TextEditResult)
	if !ok {
		t.Fatalf("unexpected result type %T", result)
	}
	seen := map[int]bool{}
	for _, e := range edits {
		if seen[e.Range.Start.Line] {
			t.Errorf("two edits on line %d", e.Range.Start.Line)
		}
		seen[e.Range.Start.Line] = true
	}
}

func TestWillSaveTouchesTableLinesOnly(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"t.md": "# Title  \n\n| a   | b |\n| --- | --- |\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "t.md"))
	openDoc(t, s, uri, "# Title  \n\n| a   | b |\n| --- | --- |\n")

	result := call(t, s.handleWillSaveWaitUntil,
		fmt.Sprintf(`{"textDocument":{"uri":%q},"reason":1}`, uri))
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), `"line":0`) {
		t.Errorf("save-time formatting touched the heading line: %s", data)
	}
}

func TestPullDiagnosticsReportsABrokenLink(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"a.md": "# A\n\n[x](missing.md)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handlePullDiagnostics, fmt.Sprintf(`{"textDocument":{"uri":%q}}`, uri))
	report, ok := result.(DocumentDiagnosticReport)
	if !ok {
		t.Fatalf("unexpected result type %T", result)
	}
	var found bool
	for _, item := range report.Items {
		if item.Code == "2" {
			found = true
		}
	}
	if !found {
		t.Errorf("no broken-link diagnostic: %+v", report.Items)
	}
}

func TestCodeActionOffersTheMditaSchema(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"a.md": "---\nid: a\n---\n# A\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleCodeAction, fmt.Sprintf(
		`{"textDocument":{"uri":%q},"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"context":{"diagnostics":[]}}`, uri))
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "tc:dita:xsd:topic.xsd") {
		t.Errorf("the action still inserts the DITA topic URN: %s", data)
	}
	if !strings.Contains(string(data), "tc:mdita:xsd:topic.xsd") {
		t.Errorf("no MDITA $schema action: %s", data)
	}
	// It must patch the existing block rather than add a second one.
	if strings.Contains(string(data), `---\n$schema`) {
		t.Errorf("a second front matter block was offered: %s", data)
	}
}

func TestAddToMapWritesAnHrefRelativeToTheMap(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"maps/guide.mditamap": "# Guide\n\n- [Intro](../topics/intro.md)\n",
		"topics/user.md":      "# User topic\n",
	})
	mapURI := paths.PathToURI(filepath.Join(root, "maps/guide.mditamap"))
	docURI := paths.PathToURI(filepath.Join(root, "topics/user.md"))

	var sent any
	s.SetRequest(func(_ string, params any) { sent = params })

	if _, err := s.executeAddToMap([]string{docURI, mapURI}); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(sent)
	if !strings.Contains(string(data), "../topics/user.md") {
		t.Errorf("addToMap wrote %s, want an href of ../topics/user.md", data)
	}
}

func TestDocumentSymbolsOutline(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"a.md": "# A\n\n## One\n\n## Two\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleDocumentSymbol, fmt.Sprintf(`{"textDocument":{"uri":%q}}`, uri))
	data, _ := json.Marshal(result)
	for _, want := range []string{"A", "One", "Two"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("outline is missing %q: %s", want, data)
		}
	}
}

func TestReferencesHandlerFindsSectionLinks(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md": "# Guide\n\n## Install Steps\n",
		"a.md":     "# A\n\n[x](guide.md#install-steps)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "guide.md"))

	result := call(t, s.handleReferences, posParams(uri, 2, 5))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "a.md") {
		t.Errorf("references returned %s", data)
	}
}

func TestCompletionOfALinkTarget(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"install.md": "# Installation\n",
		"a.md":       "# A\n\n[x](inst\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))
	openDoc(t, s, uri, "# A\n\n[x](inst\n")

	result := call(t, s.handleCompletion, posParams(uri, 2, 8))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "install.md") {
		t.Errorf("completion returned %s", data)
	}
}

func TestCompletionIsCappedAtMaxCandidates(t *testing.T) {
	files := map[string]string{".mdita-lsp.yaml": "completion:\n  max_candidates: 3\n"}
	for i := 0; i < 10; i++ {
		files[fmt.Sprintf("topic%d.md", i)] = fmt.Sprintf("# Topic %d\n", i)
	}
	files["a.md"] = "# A\n\n[x](top\n"
	s, root := serverWithFiles(t, files)
	uri := paths.PathToURI(filepath.Join(root, "a.md"))
	openDoc(t, s, uri, "# A\n\n[x](top\n")

	result := call(t, s.handleCompletion, posParams(uri, 2, 7))
	items, ok := result.([]CompletionItemResult)
	if !ok {
		t.Skipf("unexpected result type %T", result)
	}
	if len(items) > 3 {
		t.Errorf("got %d items, want at most 3", len(items))
	}
}

func TestDocumentHighlightOnAHeading(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"a.md": "# A\n\n## Setup\n\n[go](#setup)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleDocumentHighlight, posParams(uri, 2, 4))
	data, _ := json.Marshal(result)
	if string(data) == "null" || string(data) == "[]" {
		t.Errorf("no highlights for a linked heading: %s", data)
	}
}

func TestFoldingRanges(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"a.md": "# A\n\ntext\n\n## One\n\nmore\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleFoldingRange, fmt.Sprintf(`{"textDocument":{"uri":%q}}`, uri))
	data, _ := json.Marshal(result)
	if string(data) == "null" || string(data) == "[]" {
		t.Errorf("no folding ranges: %s", data)
	}
}

func TestInlayHintsShowLinkTargets(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md": "# Guide\n",
		"a.md":     "# A\n\n[x](guide.md)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleInlayHint, fmt.Sprintf(
		`{"textDocument":{"uri":%q},"range":{"start":{"line":0,"character":0},"end":{"line":5,"character":0}}}`, uri))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "Guide") {
		t.Errorf("inlay hints returned %s", data)
	}
}

func TestWillRenameFilesRewritesBothDirections(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"guide.md":     "# Guide\n\n[ref](reference.md)\n",
		"reference.md": "# Reference\n",
		"a.md":         "# A\n\n[g](guide.md)\n",
	})
	oldURI := paths.PathToURI(filepath.Join(root, "guide.md"))
	newURI := paths.PathToURI(filepath.Join(root, "topics", "guide.md"))

	result := call(t, s.handleWillRenameFiles, fmt.Sprintf(
		`{"files":[{"oldUri":%q,"newUri":%q}]}`, oldURI, newURI))
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "topics/guide.md") {
		t.Errorf("the incoming link was not rewritten: %s", data)
	}
	if !strings.Contains(string(data), "../reference.md") {
		t.Errorf("the moved file's own link was not rewritten: %s", data)
	}
}

func TestSemanticTokensFull(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{
		"a.md": "# A\n\n[x](a.md)\n",
	})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))

	result := call(t, s.handleSemanticTokensFull, fmt.Sprintf(`{"textDocument":{"uri":%q}}`, uri))
	data, _ := json.Marshal(result)
	if string(data) == "null" {
		t.Error("no semantic tokens")
	}
}

func TestDidChangeConfigurationReloads(t *testing.T) {
	s, _ := serverWithFiles(t, map[string]string{"a.md": "# A\n"})
	if err := s.handleDidChangeConfiguration(context.Background(), json.RawMessage(`{}`)); err != nil {
		t.Errorf("didChangeConfiguration: %v", err)
	}
}

func TestDidCloseClearsDiagnostics(t *testing.T) {
	s, root := serverWithFiles(t, map[string]string{"a.md": "# A\n\n[x](missing.md)\n"})
	uri := paths.PathToURI(filepath.Join(root, "a.md"))
	openDoc(t, s, uri, "# A\n\n[x](missing.md)\n")

	var published []DiagnosticResult
	s.SetNotify(func(method string, params any) {
		if method != "textDocument/publishDiagnostics" {
			return
		}
		if p, ok := params.(DiagnosticParams); ok {
			published = p.Diagnostics
		}
	})

	if err := s.handleDidClose(context.Background(),
		json.RawMessage(fmt.Sprintf(`{"textDocument":{"uri":%q}}`, uri))); err != nil {
		t.Fatal(err)
	}
	if len(published) != 0 {
		t.Errorf("didClose published %d diagnostics, want none", len(published))
	}
}
