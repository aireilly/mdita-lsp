package document

import (
	"regexp"
	"strings"
)

var (
	blockAttrRegex = regexp.MustCompile(`(?m)^\{([^}]+)\}\s*$`)
	attrClassRegex = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9_-]*)`)
	attrIDRegex    = regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]*)`)
	attrKVRegex    = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_-]*)="([^"]*)"`)
)

func ParseAttrString(s string) ParsedAttribute {
	attr := ParsedAttribute{KeyValues: make(map[string]string)}
	for _, m := range attrClassRegex.FindAllStringSubmatch(s, -1) {
		attr.Classes = append(attr.Classes, m[1])
	}
	if m := attrIDRegex.FindStringSubmatch(s); m != nil {
		attr.ID = m[1]
	}
	for _, m := range attrKVRegex.FindAllStringSubmatch(s, -1) {
		attr.KeyValues[m[1]] = m[2]
	}
	return attr
}

func ScanBlockAttributes(source string) []BlockAttribute {
	var attrs []BlockAttribute
	matches := blockAttrRegex.FindAllStringSubmatchIndex(source, -1)
	for _, m := range matches {
		attrStr := source[m[2]:m[3]]
		line := strings.Count(source[:m[0]], "\n")
		parsed := ParseAttrString(attrStr)
		parsed.Range = rangeFromOffset(source, m[0], m[1])
		attrs = append(attrs, BlockAttribute{
			Attr: parsed,
			Line: line,
		})
	}
	return attrs
}
