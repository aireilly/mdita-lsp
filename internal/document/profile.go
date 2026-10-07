package document

import "github.com/aireilly/mdita-lsp/internal/paths"

// MditaProfile is the MDITA parser profile a document is read under, or
// NotMdita when the plug-in reads it as full Markdown DITA.
type MditaProfile int

const (
	// NotMdita is Markdown DITA: the md and markdown formats, which get the
	// full extension set.
	NotMdita MditaProfile = iota
	// MditaCore is MDitaReader.CORE_PROFILE.
	MditaCore
	// MditaExtended is MDitaReader.EXTENDED_PROFILE.
	MditaExtended
)

// DeclaredProfile reports the profile the file itself declares.
//
// Two things decide it, in order: a declared $schema, which is what
// DefaultSchemaProvider keys on, and otherwise the file extension, because
// plugin.xml registers MDitaReader for format "mdita" and MarkdownReader for
// md and markdown. A .mdita file with no $schema gets MDitaReader's own
// default, the extended profile.
//
// A .md or .markdown file that declares no MDITA $schema returns NotMdita.
// It can still be MDITA in a build whose map gives it format="mdita", which
// is what core.mdita.apply_to_markdown exists to say; the diagnostics layer
// applies that, because this package does not read configuration.
func (d *Document) DeclaredProfile() MditaProfile {
	if d.Meta != nil {
		switch d.Meta.Schema {
		case SchemaMditaCore:
			return MditaCore
		case SchemaMditaExtended:
			return MditaExtended
		}
	}
	if paths.FormatForURI(d.URI) == paths.FormatMdita {
		return MditaExtended
	}
	return NotMdita
}
