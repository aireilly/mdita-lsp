package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
)

// EffectiveProfile resolves the MDITA profile for a document. An MDITA $schema
// URN selects the profile the way the plug-in's schema provider does; without
// one, the workspace setting applies.
func EffectiveProfile(doc *document.Document, cfg *config.Config) config.Profile {
	if doc.Meta != nil {
		switch doc.Meta.Schema {
		case document.SchemaMditaCore:
			return config.ProfileCore
		case document.SchemaMditaExtended:
			return config.ProfileExtended
		}
	}
	return cfg.Core.Mdita.Profile
}

// isMditaSchema reports whether the document declares an MDITA $schema, which
// means the MDITA parser profiles apply rather than the full Markdown DITA one.
func isMditaSchema(doc *document.Document) bool {
	return doc.Meta != nil &&
		(doc.Meta.Schema == document.SchemaMditaCore || doc.Meta.Schema == document.SchemaMditaExtended)
}

// CheckProfile warns when a document uses markdown constructs that the MDITA
// profile in force does not parse.
func CheckProfile(doc *document.Document, cfg *config.Config) []Diagnostic {
	profile := EffectiveProfile(doc, cfg)
	mdita := isMditaSchema(doc) || profile == config.ProfileCore
	if !mdita {
		return nil
	}

	var diags []Diagnostic
	idx := doc.Index

	// Neither MDITA profile enables the attributes extension, so {.class} and
	// {#id} stay literal text.
	if idx.Features.HasAttributes {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeMditaProfileFeature,
			Source:   source,
			Message:  "Attributes are not available in the MDITA profiles",
		})
	}

	// MDITA topics are limited to a title and second-level sections.
	for _, h := range idx.Headings() {
		if h.Level > 2 {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeMditaProfileFeature,
				Source:   source,
				Message:  "MDITA allows only level 1 titles and level 2 sections",
			})
		}
	}

	if profile != config.ProfileCore {
		return diags
	}

	if idx.Features.HasFootnoteRefs || idx.Features.HasFootnoteDefs {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeCoreProfileFeature,
			Source:   source,
			Message:  "Footnotes are not available in MDITA core profile",
		})
	}

	if idx.Features.HasDefinitionList {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeCoreProfileFeature,
			Source:   source,
			Message:  "Definition lists are not available in MDITA core profile",
		})
	}

	return diags
}
