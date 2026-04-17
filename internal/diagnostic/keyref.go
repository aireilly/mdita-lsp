package diagnostic

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

var shortcutRefRe = regexp.MustCompile(`\[([^\[\]]+)\](?:[^(\[])`)

func CheckKeyrefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	tables := buildKeyTables(folder)
	if len(tables) == 0 {
		return nil
	}

	merged := mergeKeyTables(tables)

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

func buildKeyTables(folder *workspace.Folder) []keyref.KeyTable {
	var tables []keyref.KeyTable
	for _, doc := range folder.AllDocs() {
		if doc.Kind != document.Map {
			continue
		}
		m, err := ditamap.ParseMap(doc.Text)
		if err != nil {
			continue
		}
		table := keyref.ExtractKeys(m)
		if len(table) > 0 {
			tables = append(tables, table)
		}
	}
	return tables
}

func mergeKeyTables(tables []keyref.KeyTable) keyref.KeyTable {
	merged := make(keyref.KeyTable)
	for _, t := range tables {
		for k, v := range t {
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
		}
	}
	return merged
}
