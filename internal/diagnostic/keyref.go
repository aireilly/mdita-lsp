package diagnostic

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

var shortcutRefRe = regexp.MustCompile(`\[([^\[\]]+)\][^(\[]`)

func CheckKeyrefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	var mapTexts []string
	for _, d := range folder.AllDocs() {
		if d.Kind == document.Map {
			mapTexts = append(mapTexts, d.Text)
		}
	}
	merged := keyref.BuildMergedTable(mapTexts)
	if len(merged) == 0 {
		return nil
	}

	lines := strings.Split(doc.Text, "\n")
	for lineNum, line := range lines {
		matches := shortcutRefRe.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			label := line[match[2]:match[3]]
			if label == "" {
				continue
			}
			if doc.Index.LinkDefByLabel(label) != nil {
				continue
			}
			if _, ok := keyref.Resolve(merged, label); !ok {
				diags = append(diags, Diagnostic{
					Range:    document.Rng(lineNum, match[2], lineNum, match[3]),
					Severity: SeverityWarning,
					Code:     CodeUnresolvedKeyref,
					Source:   source,
					Message:  "Unresolved keyref: " + label,
				})
			}
		}
	}

	return diags
}
