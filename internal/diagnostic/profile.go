package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

// EffectiveProfile resolves the MDITA profile for a document. An MDITA
// $schema URN selects the profile the way the plug-in's schema provider does;
// otherwise the workspace setting applies, and a .mdita file with neither
// falls back to MDitaReader's default, the extended profile.
func EffectiveProfile(doc *document.Document, cfg *config.Config) config.Profile {
	if doc.Meta != nil {
		switch doc.Meta.Schema {
		case document.SchemaMditaCore:
			return config.ProfileCore
		case document.SchemaMditaExtended:
			return config.ProfileExtended
		}
	}
	return config.ProfileVal(cfg.Core.Mdita.Profile)
}

// isMdita reports whether the MDITA parser profiles apply to a document
// rather than the full Markdown DITA one.
//
// plugin.xml registers MDitaReader for format "mdita", so a .mdita file is
// MDITA whether or not it declares a $schema. Requiring the $schema meant a
// level-3 heading in a .mdita file -- fatal for the build -- drew no
// diagnostic at all.
//
// DITA-OT takes the format from the topicref, not the extension, so a
// workspace can author MDITA in .md files; core.mdita.apply_to_markdown says
// so. That used to be bound to the profile value, which meant only
// `profile: core` had any effect -- `profile: extended` switched nothing on
// and could not even pick the profile for a .mdita file.
func isMdita(doc *document.Document, cfg *config.Config) bool {
	if doc.Meta != nil &&
		(doc.Meta.Schema == document.SchemaMditaCore || doc.Meta.Schema == document.SchemaMditaExtended) {
		return true
	}
	if paths.FormatForURI(doc.URI) == paths.FormatMdita {
		return true
	}
	return config.BoolVal(cfg.Core.Mdita.ApplyToMarkdown)
}

// CheckProfile warns when a document uses markdown constructs that the MDITA
// profile in force does not parse.
func CheckProfile(doc *document.Document, cfg *config.Config) []Diagnostic {
	if !config.BoolVal(cfg.Core.Mdita.Enable) || !isMdita(doc, cfg) {
		return nil
	}
	profile := EffectiveProfile(doc, cfg)

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
