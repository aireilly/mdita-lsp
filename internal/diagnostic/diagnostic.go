package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/config"
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
	CodeAmbiguousLink      = "1"
	CodeBrokenLink         = "2"
	CodeNBSP               = "3"
	CodeMissingFrontMatter = "4"
	CodeMissingShortDesc   = "5"
	CodeHeadingHierarchy   = "6"
	CodeUnrecognizedSchema = "7"
	CodeFootnoteRefOrphan  = "8"
	CodeFootnoteDefOrphan  = "9"
	CodeUnresolvedKeyref   = "10"
	CodeBrokenMapTopicref  = "11"
	// CodeMapHeadingHierarchy is retired. Nesting a topicref under another
	// does not require the nested topic's heading level to match the depth,
	// and the plug-in says nothing about it, so the server reported noise on
	// every working map. The number is not reused.
	CodeMapHeadingHierarchy     = "12"
	CodeCoreProfileFeature      = "13"
	CodeConrefTargetMissing     = "14"
	CodeConrefElementMissing    = "15"
	CodeConkeyrefKeyMissing     = "16"
	CodeConkeyrefElementMissing = "17"
	CodeTaskTypeInNonTask       = "18"
	// CodeCircularMapReference is outside the 1-18 spec range;
	// kept for circular dependency detection in ditamaps.
	CodeCircularMapReference = "19"
	// CodeMditaProfileFeature flags markdown that neither MDITA profile parses.
	CodeMditaProfileFeature = "20"
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
	if config.BoolVal(cfg.Diagnostics.MditaCompliance) {
		diags = append(diags, checkMditaCompliance(doc)...)
		diags = append(diags, CheckProfile(doc, cfg)...)
		diags = append(diags, checkTaskTypeInNonTask(doc)...)
	}

	if config.BoolVal(cfg.Diagnostics.LinkValidation) {
		diags = append(diags, checkLinks(doc, folder)...)
	}

	if config.BoolVal(cfg.Diagnostics.NbspDetection) {
		diags = append(diags, checkNonBreakingWhitespace(doc)...)
	}

	if doc.Kind == document.Map && config.BoolVal(cfg.Diagnostics.DitamapValidation) {
		diags = append(diags, CheckDitamap(doc, folder)...)
	}

	if config.BoolVal(cfg.Diagnostics.KeyrefResolution) {
		diags = append(diags, CheckKeyrefs(doc, folder)...)
	}

	diags = append(diags, CheckConrefs(doc, folder)...)

	return diags
}
