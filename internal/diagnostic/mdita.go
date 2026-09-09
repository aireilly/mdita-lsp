package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/document"
)

func checkMditaCompliance(doc *document.Document) []Diagnostic {
	var diags []Diagnostic

	if doc.Meta == nil {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeMissingFrontMatter,
			Source:   source,
			Message:  "Missing YAML front matter",
		})
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

	diags = append(diags, checkHeadingHierarchy(doc)...)
	diags = append(diags, checkFootnotes(doc)...)

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

func checkHeadingHierarchy(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	headings := doc.Index.Headings()
	for i := 1; i < len(headings); i++ {
		prev := headings[i-1].Level
		curr := headings[i].Level
		if curr > prev+1 {
			diags = append(diags, Diagnostic{
				Range:    headings[i].Range,
				Severity: SeverityWarning,
				Code:     CodeHeadingHierarchy,
				Source:   source,
				Message:  "Invalid heading hierarchy: skipped heading level",
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
