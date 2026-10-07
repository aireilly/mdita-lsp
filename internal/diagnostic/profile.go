package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
)

// EffectiveProfile resolves the MDITA profile for a document: the one the
// file declares through $schema or its extension, and otherwise the extended
// profile for a .md or .markdown topic in a workspace that sets
// core.mdita.apply_to_markdown.
//
// There is no setting for which profile. It had been broken since the
// beginning -- the check read "an MDITA schema, or profile == core", and
// ProfileExtended was the zero value, so only `core` ever did anything --
// and it was the wrong shape besides. A config key talks to the editor
// alone, while $schema talks to the editor and to DITA-OT at once, so a
// setting could claim a profile the build did not use.
func EffectiveProfile(doc *document.Document, cfg *config.Config) document.MditaProfile {
	if declared := doc.DeclaredProfile(); declared != document.NotMdita {
		return declared
	}
	if config.BoolVal(cfg.Core.Mdita.ApplyToMarkdown) {
		return document.MditaExtended
	}
	return document.NotMdita
}

// CheckProfile warns when a document uses markdown constructs that the MDITA
// profile in force does not parse.
func CheckProfile(doc *document.Document, cfg *config.Config) []Diagnostic {
	profile := EffectiveProfile(doc, cfg)
	if !config.BoolVal(cfg.Core.Mdita.Enable) || profile == document.NotMdita {
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
	// MarkdownParserImpl.validate throws on a deeper heading, so the build
	// fails rather than degrading.
	for _, h := range idx.Headings() {
		if h.Level > 2 {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityError,
				Code:     CodeMditaProfileFeature,
				Source:   source,
				Message: "LwDITA does not support a level " + itoa(h.Level) +
					" heading; the build fails on this",
			})
		}
	}

	if profile != document.MditaCore {
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
