package diagnostic

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/vocabulary"
)

var validAdmonitionTypes = map[string]bool{
	"note": true, "tip": true, "warning": true, "caution": true,
	"danger": true, "attention": true, "important": true, "notice": true,
	"fastpath": true, "remember": true, "restriction": true, "trouble": true,
}

func checkMditaCompliance(doc *document.Document) []Diagnostic {
	var diags []Diagnostic

	if doc.Meta == nil {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeMissingYamlFrontMatter,
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

	title := doc.Index.Title()
	if title != nil && doc.Index.ShortDesc == "" {
		diags = append(diags, Diagnostic{
			Range:    title.Range,
			Severity: SeverityWarning,
			Code:     CodeMissingShortDescription,
			Source:   source,
			Message:  "Missing short description (paragraph after title)",
		})
	}

	diags = append(diags, checkHeadingHierarchy(doc)...)
	diags = append(diags, checkSchemaSpecific(doc)...)
	diags = append(diags, checkExtendedFeatures(doc)...)
	diags = append(diags, checkAdmonitions(doc)...)
	diags = append(diags, checkFootnotes(doc)...)
	diags = append(diags, checkTaskSections(doc)...)
	if doc.Index.Features.HasAttributes {
		diags = append(diags, checkInlineAttributes(doc)...)
	}

	return diags
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
				Code:     CodeInvalidHeadingHierarchy,
				Source:   source,
				Message:  "Invalid heading hierarchy: skipped heading level",
			})
		}
	}
	return diags
}

func checkSchemaSpecific(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	if doc.Meta == nil {
		return diags
	}
	bf := doc.Index.Features

	switch doc.Meta.Schema {
	case document.SchemaTask:
		if !bf.HasOrderedList {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityWarning,
				Code:     CodeTaskMissingProcedure,
				Source:   source,
				Message:  "Task topic is missing a procedure (ordered list)",
			})
		}

	case document.SchemaConcept:
		if bf.HasOrderedList {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeConceptHasProcedure,
				Source:   source,
				Message:  "Concept topic contains an ordered list — consider using task schema",
			})
		}

	case document.SchemaReference:
		if !bf.HasTable && !bf.HasDefinitionList {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeReferenceMissingTable,
				Source:   source,
				Message:  "Reference topic is missing a table or definition list",
			})
		}
	}

	if doc.Kind == document.Map {
		hasNonLinkContent := bf.HasOrderedList || bf.HasTable || bf.HasDefinitionList
		if hasNonLinkContent {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeMapHasBodyContent,
				Source:   source,
				Message:  "Map contains body content beyond topic references",
			})
		}
	}

	return diags
}

func checkExtendedFeatures(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	if doc.Meta == nil || doc.Meta.Schema != document.SchemaMditaCoreTopic {
		return diags
	}
	bf := doc.Index.Features

	if bf.HasDefinitionList {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Definition lists are an extended profile feature",
		})
	}
	if bf.HasFootnoteRefs || bf.HasFootnoteDefs {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Footnotes are an extended profile feature",
		})
	}
	if bf.HasStrikethrough {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Strikethrough is an extended profile feature",
		})
	}
	if bf.HasAttributes {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Generic attributes are an extended profile feature",
		})
	}
	if len(bf.Admonitions) > 0 {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Admonitions are an extended profile feature",
		})
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
				Code:     CodeFootnoteRefWithoutDef,
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
				Code:     CodeFootnoteDefWithoutRef,
				Source:   source,
				Message:  "Footnote definition without reference: " + def.Label,
			})
		}
	}

	return diags
}

func checkAdmonitions(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	bf := doc.Index.Features
	for _, adm := range bf.Admonitions {
		if !validAdmonitionTypes[strings.ToLower(adm.Type)] {
			diags = append(diags, Diagnostic{
				Range:    adm.Range,
				Severity: SeverityWarning,
				Code:     CodeUnknownAdmonitionType,
				Source:   source,
				Message:  "Unknown admonition type: " + adm.Type,
			})
		}
	}
	return diags
}

func checkTaskSections(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	seen := make(map[document.TaskSectionKind]bool)
	lastOrder := 0

	for _, h := range doc.Index.Headings() {
		if h.TaskSection == document.TaskSectionNone {
			continue
		}
		if seen[h.TaskSection] {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityError,
				Code:     CodeDuplicateTaskSection,
				Source:   source,
				Message:  "Duplicate task section: \"" + h.Text + "\"",
			})
			continue
		}
		seen[h.TaskSection] = true

		order := taskSectionOrder(h.TaskSection)
		if order > 0 && order < lastOrder {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeTaskSectionOutOfOrder,
				Source:   source,
				Message:  "Task section \"" + h.Text + "\" should appear earlier",
			})
		}
		if order > lastOrder {
			lastOrder = order
		}
	}
	return diags
}

func taskSectionOrder(kind document.TaskSectionKind) int {
	switch kind {
	case document.TaskSectionPrereq:
		return 1
	case document.TaskSectionContext:
		return 2
	case document.TaskSectionResult:
		return 4
	case document.TaskSectionPostreq:
		return 5
	case document.TaskSectionTroubleshooting:
		return 6
	default:
		return 0
	}
}

func checkInlineAttributes(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, ia := range doc.InlineAttrs {
		for _, class := range ia.Attr.Classes {
			elem, ok := vocabulary.LookupDomainElementByName(class)
			if !ok {
				_, isStep := vocabulary.LookupStepElementByName(class)
				if !isStep {
					diags = append(diags, Diagnostic{
						Range:    ia.Attr.Range,
						Severity: SeverityWarning,
						Code:     CodeUnknownOutputclass,
						Source:   source,
						Message:  "Unknown outputclass \"{." + class + "}\"",
					})
				}
				continue
			}
			if elem.ParentKind != ia.TargetKind {
				diags = append(diags, Diagnostic{
					Range:    ia.Attr.Range,
					Severity: SeverityWarning,
					Code:     CodeDomainClassWrongParent,
					Source:   source,
					Message:  "Domain class \"" + class + "\" expects " + elem.ParentKind + ", not " + ia.TargetKind,
				})
			}
			if class == "menucascade" && !strings.Contains(ia.TargetText, " > ") {
				diags = append(diags, Diagnostic{
					Range:    ia.Attr.Range,
					Severity: SeverityWarning,
					Code:     CodeMenucascadeMissingSeparator,
					Source:   source,
					Message:  "menucascade requires \" > \" separator between menu items",
				})
			}
		}
	}
	return diags
}
