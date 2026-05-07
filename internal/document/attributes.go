package document

import (
	"regexp"
	"strings"
)

var (
	inlineBoldAttrRegex      = regexp.MustCompile(`\*\*([^*]+)\*\*\{([^}]+)\}`)
	inlineBoldUnderAttrRegex = regexp.MustCompile(`__([^_]+)__\{([^}]+)\}`)
	inlineCodeAttrRegex      = regexp.MustCompile("`([^`]+)`\\{([^}]+)\\}")
	inlineItalicAttrRegex    = regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*\{([^}]+)\}`)
	blockAttrRegex           = regexp.MustCompile(`(?m)^\{([^}]+)\}\s*$`)
	attrClassRegex           = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9_-]*)`)
	attrIDRegex              = regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]*)`)
	attrKVRegex              = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_-]*)="([^"]*)"`)
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

func ScanInlineAttributes(source string) []InlineAttribute {
	var attrs []InlineAttribute
	lines := strings.Split(source, "\n")
	for lineNum, line := range lines {
		attrs = append(attrs, scanLineInline(line, lineNum, inlineBoldAttrRegex, "bold")...)
		attrs = append(attrs, scanLineInline(line, lineNum, inlineBoldUnderAttrRegex, "bold")...)
		attrs = append(attrs, scanLineInline(line, lineNum, inlineCodeAttrRegex, "code")...)
		attrs = append(attrs, scanLineInline(line, lineNum, inlineItalicAttrRegex, "italic")...)
	}
	return attrs
}

func scanLineInline(line string, lineNum int, re *regexp.Regexp, kind string) []InlineAttribute {
	var attrs []InlineAttribute
	matches := re.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		text := line[m[2]:m[3]]
		attrStr := line[m[4]:m[5]]
		braceStart := strings.LastIndex(line[m[0]:m[1]], "{")
		col := m[0] + braceStart
		parsed := ParseAttrString(attrStr)
		parsed.Range = Rng(lineNum, col, lineNum, m[1])
		attrs = append(attrs, InlineAttribute{
			Attr:       parsed,
			TargetKind: kind,
			TargetText: text,
			Line:       lineNum,
			Col:        col,
		})
	}
	return attrs
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
