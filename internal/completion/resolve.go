package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func Resolve(label string, data map[string]string, folder *workspace.Folder) string {
	if data == nil {
		return ""
	}

	switch data["kind"] {
	case "keyref":
		return resolveKeyrefDocs(label, folder)
	}
	return ""
}

func resolveKeyrefDocs(key string, folder *workspace.Folder) string {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := table[key]
	if !ok {
		return ""
	}

	var parts []string
	if entry.Title != "" {
		parts = append(parts, "**"+entry.Title+"**")
	}
	parts = append(parts, "href: `"+entry.Href+"`")
	return strings.Join(parts, "\n\n")
}
