package keyref

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

var dataKeyrefRe = regexp.MustCompile(`<\w+\s[^>]*data-keyref="([^"]+)"[^>]*>`)

// DataKeyref holds a resolved key and its source range for an HTML data-keyref attribute.
type DataKeyref struct {
	Key   string
	Range document.Range
}

// DetectDataKeyrefs scans text for all <tag data-keyref="key"> occurrences.
func DetectDataKeyrefs(text string) []DataKeyref {
	var results []DataKeyref
	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		for _, m := range dataKeyrefRe.FindAllStringSubmatchIndex(line, -1) {
			key := line[m[2]:m[3]]
			results = append(results, DataKeyref{
				Key:   key,
				Range: document.Rng(lineNum, m[0], lineNum, m[1]),
			})
		}
	}
	return results
}
