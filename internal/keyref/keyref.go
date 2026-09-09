package keyref

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

// KeyEntry is one DITA key definition. The plug-in generates a <keydef> from a
// reference-style link definition in a map file, so a key always carries an
// href and optionally the link title, which becomes the navtitle.
type KeyEntry struct {
	Href  string
	Title string
}

type KeyTable map[string]KeyEntry

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

// BuildMergedTable collects the key definitions from every map in the
// workspace. Only reference-style link definitions define keys; a plain
// topicref in a map defines no key.
func BuildMergedTable(mapTexts []string) KeyTable {
	merged := make(KeyTable)
	for _, text := range mapTexts {
		elements, _, _ := document.Parse(text)
		for _, elem := range elements {
			ld, ok := elem.(*document.LinkDef)
			if !ok {
				continue
			}
			if _, exists := merged[ld.Label]; exists {
				continue
			}
			href, title := splitLinkDefURL(ld.URL)
			merged[ld.Label] = KeyEntry{Href: href, Title: title}
		}
	}
	return merged
}

// splitLinkDefURL separates the href from the optional quoted title of a
// reference-style link definition, as in `[key]: install.md "Installation"`.
func splitLinkDefURL(raw string) (href, title string) {
	raw = strings.TrimSpace(raw)
	for _, q := range []byte{'"', '\''} {
		if idx := strings.IndexByte(raw, q); idx > 0 {
			if end := strings.LastIndexByte(raw, q); end > idx {
				return strings.TrimSpace(raw[:idx]), raw[idx+1 : end]
			}
		}
	}
	return raw, ""
}
