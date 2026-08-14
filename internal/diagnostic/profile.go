package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
)

// CheckProfile warns when a document uses MDITA extended profile features
// while the workspace is configured for the core profile.
func CheckProfile(doc *document.Document, cfg *config.Config) []Diagnostic {
	if cfg.Core.Mdita.Profile != config.ProfileCore {
		return nil
	}

	var diags []Diagnostic

	idx := doc.Index
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

	if idx.Features.HasAttributes {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeCoreProfileFeature,
			Source:   source,
			Message:  "Attributes are not available in MDITA core profile",
		})
	}

	return diags
}
