package diagnostic

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

// The messages here are the plug-in's own, word for word, so a diagnostic in
// the editor and the error from the build read the same.

var attrSuffix = regexp.MustCompile(`\s*\{[^}]*\}\s*$`)

// headingSource returns the heading as the author wrote it, hashes and block
// attributes included.
func headingSource(doc *document.Document, h *document.Heading) string {
	line := h.LineRange.Start.Line
	lines := strings.Split(doc.Text, "\n")
	if line < len(lines) {
		return strings.TrimSpace(lines[line])
	}
	return marker(h.Level) + " " + h.Text
}

// headingSourceWithoutAttributes returns the heading as written, without its
// block attributes.
func headingSourceWithoutAttributes(doc *document.Document, h *document.Heading) string {
	return strings.TrimSpace(attrSuffix.ReplaceAllString(headingSource(doc, h), ""))
}

func marker(level int) string { return strings.Repeat("#", level) }

// NestedSectionMessage is the message the plug-in reports for a heading that
// would nest a section inside a concept or a reference.
func NestedSectionMessage(doc *document.Document, h *document.Heading, topicLevel int) string {
	return `"` + headingSource(doc, h) + `" can't go here: DITA sections don't nest. ` +
		`Make it a "` + marker(topicLevel+1) + `" section, or move it to its own topic.`
}

// SectionAfterNestedTopicMessage is the message the plug-in reports for a
// section heading at the level of a heading that already opened a nested
// topic.
func SectionAfterNestedTopicMessage(doc *document.Document, h, parent *document.Heading) string {
	heading := headingSource(doc, h)
	plain := headingSourceWithoutAttributes(doc, h)
	above := `the heading above it`
	if parent != nil {
		above = `"` + headingSourceWithoutAttributes(doc, parent) + `"`
	}
	return `"` + heading + `" can't be a section here: ` + above +
		` above it opened a nested topic. Add {.section} to ` + above +
		`, or remove it from "` + plain + `".`
}

// SectionLinkMessage is the message for a link that addresses a section
// without the topic id the build needs.
func SectionLinkMessage(url, qualified string) string {
	return "Link '" + url + "' addresses a section, which the build resolves only " +
		"with the topic id: '" + qualified + "'"
}
