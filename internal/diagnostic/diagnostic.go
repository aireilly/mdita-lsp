package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Severity int

const (
	SeverityError   Severity = 1
	SeverityWarning Severity = 2
	SeverityInfo    Severity = 3
)

const (
	CodeAmbiguousLink                   = "1"
	CodeBrokenLink                      = "2"
	CodeNonBreakingWhitespace           = "3"
	CodeMissingYamlFrontMatter          = "4"
	CodeMissingShortDescription         = "5"
	CodeInvalidHeadingHierarchy         = "6"
	CodeUnrecognizedSchema              = "7"
	CodeTaskMissingProcedure            = "8"
	CodeConceptHasProcedure             = "9"
	CodeReferenceMissingTable           = "10"
	CodeMapHasBodyContent               = "11"
	CodeExtendedFeatureInCoreProfile    = "12"
	CodeFootnoteRefWithoutDef           = "13"
	CodeFootnoteDefWithoutRef           = "14"
	CodeUnknownAdmonitionType           = "15"
	CodeUnresolvedKeyref                = "16"
	CodeBrokenMapReference              = "17"
	CodeCircularMapReference            = "18"
	CodeInconsistentMapHeadingHierarchy = "19"
)

type Diagnostic struct {
	Range    document.Range
	Severity Severity
	Code     string
	Source   string
	Message  string
}

const source = "mdita-lsp"

func Check(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	cfg := folder.Config
	if cfg.Diagnostics.MditaCompliance && cfg.Core.Mdita.Enable {
		diags = append(diags, checkMditaCompliance(doc)...)
	}

	diags = append(diags, checkLinks(doc, folder)...)
	diags = append(diags, checkNonBreakingWhitespace(doc)...)

	if cfg.Diagnostics.DitamapValidation {
		diags = append(diags, CheckDitamap(doc, folder)...)
	}

	return diags
}
