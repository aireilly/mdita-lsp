package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/document"
)

func checkMditaCompliance(doc *document.Document) []Diagnostic {
	var diags []Diagnostic

	// Heading hierarchy and footnotes are properties of the markdown, not of
	// the metadata. These checks used to sit behind an early return, so a file
	// without front matter -- which Markdown DITA does not require -- was
	// never checked for the heading skip that fails the build.
	diags = append(diags, checkHeadingHierarchy(doc)...)
	diags = append(diags, checkSectionDepth(doc)...)
	diags = append(diags, checkNestedSection(doc)...)
	diags = append(diags, checkFootnotes(doc)...)

	// Front matter is optional everywhere the plug-in reads: a topic without
	// it is ordinary Markdown DITA, and a map's structure comes from its
	// list. The server used to note its absence on every markdown file in the
	// workspace, READMEs and notes included, for something that is a choice
	// rather than an omission. Code 4 is retired.
	if doc.Meta == nil {
		return diags
	}

	if doc.Meta.SchemaRaw != "" && doc.Meta.Schema == document.SchemaUnknown {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeUnrecognizedSchema,
			Source:   source,
			Message:  "Unrecognized $schema value: " + doc.Meta.SchemaRaw,
		})
	}

	// The plug-in pulls the first paragraph into <shortdesc> when a schema is
	// declared or when the title carries a concept/task/reference outputclass.
	title := doc.Index.Title()
	if title != nil && doc.Index.ShortDesc == "" && expectsShortDesc(doc, title) {
		diags = append(diags, Diagnostic{
			Range:    title.Range,
			Severity: SeverityWarning,
			Code:     CodeMissingShortDesc,
			Source:   source,
			Message:  "Missing short description (paragraph after title)",
		})
	}

	return diags
}

// shortDescTitleTypes are the title outputclasses that pull the following
// paragraph into <shortdesc>.
var shortDescTitleTypes = map[string]bool{"concept": true, "task": true, "reference": true}

func expectsShortDesc(doc *document.Document, title *document.Heading) bool {
	if doc.Meta != nil && doc.Meta.Schema != document.SchemaUnknown && doc.Meta.SchemaRaw != "" {
		return true
	}
	if title.Attributes == nil {
		return false
	}
	for _, c := range title.Attributes.Classes {
		if shortDescTitleTypes[c] {
			return true
		}
	}
	return false
}

// checkHeadingHierarchy mirrors MarkdownParserImpl.validate: walking the
// headings in order, a level more than one above the previous one throws a
// ParseException and the build fails. The level starts at 0, so a document
// whose first heading is an H2 is a skip too. This is an error, not a
// warning: the build does not produce output.
func checkHeadingHierarchy(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	level := 0
	for _, h := range doc.Index.Headings() {
		if h.Level > level+1 {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityError,
				Code:     CodeHeadingHierarchy,
				Source:   source,
				Message: "Heading level raised from " + itoa(level) + " to " + itoa(h.Level) +
					" without an intermediate heading level; the build fails on this",
			})
		}
		level = h.Level
	}
	return diags
}

// checkSectionDepth mirrors the other heading rule in TopicRenderer: a
// heading that becomes a <section> must sit below the topic heading that
// encloses it, or the build fails.
//
// It bites when a nested topic at the same level comes first:
//
//	# Task         topic, level 1
//	## Details     a nested topic, level 2
//	## Procedure   a section at level 2, inside a level 2 topic -- throws
//
// A concept or a reference cannot reach this state, because a heading one
// level below the topic title is a section there rather than a nested topic.
func checkSectionDepth(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	topicLevel := 0
	var topicHeading *document.Heading
	for _, h := range doc.Index.Headings() {
		if !h.Section {
			topicLevel = h.Level
			topicHeading = h
			continue
		}
		if h.Level <= topicLevel {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityError,
				Code:     CodeSectionAfterNestedTopic,
				Source:   source,
				Message:  SectionAfterNestedTopicMessage(doc, h, topicHeading),
			})
		}
	}
	return diags
}

// checkNestedSection mirrors the depth rule TopicRenderer applies inside a
// concept or a reference: the heading one level below the topic title opens a
// <section>, and anything deeper would nest a section, which DITA does not
// allow.
func checkNestedSection(doc *document.Document) []Diagnostic {
	if !document.IsSectionTopic(doc) {
		return nil
	}
	var diags []Diagnostic
	topicLevel := 0
	prevLevel := 0
	for _, h := range doc.Index.Headings() {
		skipped := h.Level > prevLevel+1
		prevLevel = h.Level
		if !h.Section {
			topicLevel = h.Level
			continue
		}
		// A skipped heading level is already reported, and it fails the build
		// before the renderer sees the section.
		if h.Level > topicLevel+1 && !skipped {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityError,
				Code:     CodeNestedSection,
				Source:   source,
				Message:  NestedSectionMessage(doc, h, topicLevel),
			})
		}
	}
	return diags
}

func checkFootnotes(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	bf := doc.Index.Features

	defLabels := make(map[string]bool)
	for _, def := range bf.FootnoteDefLabels {
		defLabels[def.Label] = true
	}

	refLabels := make(map[string]bool)
	for _, ref := range bf.FootnoteRefLabels {
		refLabels[ref.Label] = true
	}

	for _, ref := range bf.FootnoteRefLabels {
		if !defLabels[ref.Label] {
			diags = append(diags, Diagnostic{
				Range:    ref.Range,
				Severity: SeverityWarning,
				Code:     CodeFootnoteRefOrphan,
				Source:   source,
				Message:  "Footnote reference without definition: " + ref.Label,
			})
		}
	}

	for _, def := range bf.FootnoteDefLabels {
		if !refLabels[def.Label] {
			diags = append(diags, Diagnostic{
				Range:    def.Range,
				Severity: SeverityInfo,
				Code:     CodeFootnoteDefOrphan,
				Source:   source,
				Message:  "Footnote definition without reference: " + def.Label,
			})
		}
	}

	return diags
}

func checkTaskTypeInNonTask(doc *document.Document) []Diagnostic {
	if document.IsTaskTopic(doc) {
		return nil
	}

	var diags []Diagnostic
	for _, e := range doc.Elements {
		if h, ok := e.(*document.Heading); ok && h.TaskSection != document.TaskSectionNone {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeTaskTypeInNonTask,
				Source:   source,
				Message:  "Task section heading in a non-task topic (add {.task} to H1 or set $schema to task.xsd)",
			})
		}
	}
	return diags
}
