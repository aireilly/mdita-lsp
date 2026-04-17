package keyref

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
)

type KeyEntry struct {
	Href  string
	Title string
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
