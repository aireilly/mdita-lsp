package diagnostic

import (
	"path/filepath"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

// CheckConrefs validates all conref and conkeyref elements in a document,
// reporting diagnostics when referenced files or keys cannot be resolved.
func CheckConrefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	for _, el := range doc.Elements {
		ce, ok := el.(*document.ConrefElement)
		if !ok {
			continue
		}
		if ce.IsKeyref {
			diags = append(diags, checkConkeyref(ce, folder)...)
		} else {
			diags = append(diags, checkConref(ce, doc, folder)...)
		}
	}
	return diags
}

func checkConref(ce *document.ConrefElement, doc *document.Document, folder *workspace.Folder) []Diagnostic {
	srcPath, err := paths.URIToPath(doc.URI)
	if err != nil {
		return nil
	}
	srcDir := filepath.Dir(srcPath)
	targetPath := filepath.Join(srcDir, ce.FilePath)
	targetURI := paths.PathToURI(targetPath)
	targetDoc := folder.DocByURI(targetURI)

	if targetDoc == nil {
		return []Diagnostic{{
			Range:    ce.Range,
			Severity: SeverityWarning,
			Code:     CodeConrefTargetMissing,
			Source:   source,
			Message:  "Conref target not found: " + ce.FilePath,
		}}
	}

	if ce.ElementID == "" {
		return nil
	}

	for _, el := range targetDoc.Elements {
		if h, ok := el.(*document.Heading); ok && h.ID == ce.ElementID {
			return nil
		}
	}

	return []Diagnostic{{
		Range:    ce.Range,
		Severity: SeverityWarning,
		Code:     CodeConrefElementMissing,
		Source:   source,
		Message:  "Conref element not found: " + ce.ElementID + " in " + ce.FilePath,
	}}
}

func checkConkeyref(ce *document.ConrefElement, folder *workspace.Folder) []Diagnostic {
	table := keyref.BuildMergedTable(folder.MapTexts())
	if len(table) == 0 {
		return nil
	}
	_, ok := keyref.Resolve(table, ce.KeyName)
	if !ok {
		return []Diagnostic{{
			Range:    ce.Range,
			Severity: SeverityWarning,
			Code:     CodeConkeyrefKeyMissing,
			Source:   source,
			Message:  "Conkeyref key not found: " + ce.KeyName,
		}}
	}
	return nil
}
