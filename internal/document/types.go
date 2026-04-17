package document

import "github.com/aireilly/mdita-lsp/internal/paths"

type Range struct {
	Start Position
	End   Position
}

type Position struct {
	Line      int
	Character int
}

func Rng(sl, sc, el, ec int) Range {
	return Range{Start: Position{Line: sl, Character: sc}, End: Position{Line: el, Character: ec}}
}

type DocKind int

const (
	Topic DocKind = iota
	Map
)

func (k DocKind) String() string {
	switch k {
	case Topic:
		return "topic"
	case Map:
		return "map"
	default:
		return "unknown"
	}
}

type DitaSchema int

const (
	SchemaTopic DitaSchema = iota
	SchemaConcept
	SchemaTask
	SchemaReference
	SchemaMap
	SchemaMditaTopic
	SchemaMditaCoreTopic
	SchemaMditaExtendedTopic
	SchemaUnknown
)

func DitaSchemaFromString(s string) DitaSchema {
	switch s {
	case "urn:oasis:names:tc:dita:xsd:topic.xsd",
		"urn:oasis:names:tc:dita:rng:topic.rng":
		return SchemaTopic
	case "urn:oasis:names:tc:dita:xsd:concept.xsd",
		"urn:oasis:names:tc:dita:rng:concept.rng":
		return SchemaConcept
	case "urn:oasis:names:tc:dita:xsd:task.xsd",
		"urn:oasis:names:tc:dita:rng:task.rng":
		return SchemaTask
	case "urn:oasis:names:tc:dita:xsd:reference.xsd",
		"urn:oasis:names:tc:dita:rng:reference.rng":
		return SchemaReference
	case "urn:oasis:names:tc:dita:xsd:map.xsd",
		"urn:oasis:names:tc:dita:rng:map.rng":
		return SchemaMap
	case "urn:oasis:names:tc:mdita:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:rng:topic.rng":
		return SchemaMditaTopic
	case "urn:oasis:names:tc:mdita:core:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:core:rng:topic.rng":
		return SchemaMditaCoreTopic
	case "urn:oasis:names:tc:mdita:extended:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:extended:rng:topic.rng":
		return SchemaMditaExtendedTopic
	default:
		return SchemaUnknown
	}
}

type Element interface {
	Rng() Range
	element()
}

type Heading struct {
	Level int
	Text  string
	ID    string
	Slug  paths.Slug
	Range Range
}

func (h *Heading) Rng() Range { return h.Range }
func (h *Heading) element()   {}
func (h *Heading) IsTitle() bool { return h.Level == 1 }

type WikiLink struct {
	Doc     string
	Heading string
	Title   string
	Range   Range
}

func (w *WikiLink) Rng() Range { return w.Range }
func (w *WikiLink) element()   {}

type MdLink struct {
	Text   string
	URL    string
	Anchor string
	IsRef  bool
	Range  Range
}

func (m *MdLink) Rng() Range { return m.Range }
func (m *MdLink) element()   {}

type LinkDef struct {
	Label string
	URL   string
	Range Range
}

func (l *LinkDef) Rng() Range { return l.Range }
func (l *LinkDef) element()   {}

type YAMLMetadata struct {
	Author      string
	Source      string
	Publisher   string
	Permissions string
	Audience    string
	Category    string
	Keywords    []string
	ResourceID  string
	Schema      DitaSchema
	SchemaRaw   string
	OtherMeta   map[string]string
	Range       Range
}

type FootnoteLabel struct {
	Label string
	Range Range
}

type BlockFeatures struct {
	HasOrderedList    bool
	HasUnorderedList  bool
	HasTable          bool
	HasDefinitionList bool
	HasFootnoteRefs   bool
	HasFootnoteDefs   bool
	HasStrikethrough  bool
	HasAttributes     bool
	FootnoteRefLabels []FootnoteLabel
	FootnoteDefLabels []FootnoteLabel
	Admonitions       []Admonition
}

type Admonition struct {
	Type  string
	Range Range
}

type SymKind int

const (
	DefKind SymKind = iota
	RefKind
)

func (k SymKind) String() string {
	if k == DefKind {
		return "def"
	}
	return "ref"
}

type DefType int

const (
	DefDoc DefType = iota
	DefTitle
	DefHeading
	DefLinkDef
)

type RefType int

const (
	RefWikiLink RefType = iota
	RefMdLink
	RefKeyref
)

type Symbol struct {
	Kind    SymKind
	DefType DefType
	RefType RefType
	Name    string
	Slug    paths.Slug
	DocURI  string
	Range   Range
}
