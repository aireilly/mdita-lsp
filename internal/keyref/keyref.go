package keyref

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
	"github.com/aireilly/mdita-lsp/internal/document"
)

type KeyEntry struct {
	Href  string
	Title string
	Value string
}

type KeyTable map[string]KeyEntry

func ExtractKeys(m *ditamap.MapStructure) KeyTable {
	table := make(KeyTable)
	extractKeysFromRefs(m.TopicRefs, table)
	return table
}

func extractKeysFromRefs(refs []ditamap.TopicRef, table KeyTable) {
	for _, ref := range refs {
		if ref.Href != "" {
			key := stemFromHref(ref.Href)
			table[key] = KeyEntry{
				Href:  ref.Href,
				Title: ref.Title,
			}
		}
		extractKeysFromRefs(ref.Children, table)
	}
}

func stemFromHref(href string) string {
	base := filepath.Base(href)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func Resolve(table KeyTable, key string) (KeyEntry, bool) {
	entry, ok := table[key]
	return entry, ok
}

func AllKeys(table KeyTable) []string {
	keys := make([]string, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	return keys
}

func isURLValue(s string) bool {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	for _, ext := range []string{".md", ".dita", ".html", ".xml"} {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

func BuildMergedTable(mapTexts []string) KeyTable {
	merged := make(KeyTable)
	for _, text := range mapTexts {
		m, err := ditamap.ParseMap(text)
		if err != nil {
			continue
		}
		table := ExtractKeys(m)
		for k, v := range table {
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
		}

		meta := document.ParseYAMLMeta(text)
		if meta != nil && meta.Keys != nil {
			for k, v := range meta.Keys {
				if isURLValue(v) {
					merged[k] = KeyEntry{Href: v}
				} else {
					merged[k] = KeyEntry{Value: v, Title: v}
				}
			}
		}
	}
	return merged
}
