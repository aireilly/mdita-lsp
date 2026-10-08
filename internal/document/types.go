package document

import "github.com/aireilly/mdita-lsp/internal/paths"

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
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
	SchemaMditaCore
	SchemaMditaExtended
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
	case "urn:oasis:names:tc:mdita:core:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:core:rng:topic.rng":
		return SchemaMditaCore
	case "urn:oasis:names:tc:mdita:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:rng:topic.rng",
		"urn:oasis:names:tc:mdita:extended:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:extended:rng:topic.rng":
		return SchemaMditaExtended
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
	// ID is the @id the plug-in puts on the generated element.
	ID string
	// ExplicitID records an author-written {#id}, which the plug-in keeps
	// verbatim instead of generating one.
	ExplicitID bool
	Slug       paths.Slug
	Range      Range
	// LineRange covers the heading's whole source line, including the hashes.
	// Range starts after them, so it cannot be used for a line replacement.
	LineRange   Range
	Attributes  *ParsedAttribute
	TaskSection TaskSectionKind
	// Section records that the plug-in builds this heading as a <section> of
	// the topic above it rather than opening a nested topic.
	Section bool
	// SectionTopicID is the @id of the topic the section belongs to, empty
	// when that topic is the root topic. Only set when Section is true.
	SectionTopicID string
}

func (h *Heading) Rng() Range    { return h.Range }
func (h *Heading) element()      {}
func (h *Heading) IsTitle() bool { return h.Level == 1 }

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
	ID          string
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

type ParsedAttribute struct {
	Classes   []string
	ID        string
	KeyValues map[string]string
	Range     Range
}

type BlockAttribute struct {
	Attr ParsedAttribute
	Line int
}

type TaskSectionKind int

const (
	TaskSectionNone TaskSectionKind = iota
	TaskSectionPrereq
	TaskSectionContext
	TaskSectionResult
	TaskSectionPostreq
	TaskSectionTroubleshooting
	// TaskSectionSteps marks a steps marker heading ("Procedure"/"Steps"). The
	// heading itself maps to no DITA element; the ordered list that follows it
	// becomes <steps>.
	TaskSectionSteps
)

type ImplicitSectionKind int

const (
	ImplicitContext ImplicitSectionKind = iota
	ImplicitSteps
	ImplicitStepsUnordered
	ImplicitResult
	ImplicitChoices
	ImplicitSubsteps
	ImplicitChoicetable
	// ImplicitBodyList marks a body-level list that is followed by another
	// body-level list, so the plug-in keeps it inside <context> (outputclass
	// body-ol/body-ul) instead of promoting it to <steps>.
	ImplicitBodyList
)

type ImplicitSection struct {
	Kind  ImplicitSectionKind
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
	DefElementID
)

type RefType int

const (
	RefMdLink RefType = iota
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

// ConrefElement represents an HTML element with a data-conref or data-conkeyref
// attribute, as used in DITA content references.
type ConrefElement struct {
	Tag       string
	FilePath  string
	TopicID   string
	ElementID string
	IsKeyref  bool
	KeyName   string
	Range     Range
}

func (c *ConrefElement) Rng() Range { return c.Range }
func (c *ConrefElement) element()   {}

// sectionClasses are the heading outputclasses TopicRenderer maps to a
// <section> or <example> instead of opening a nested topic. The list matches
// the renderer's `sections` map.
var sectionClasses = map[string]bool{
	"section":             true,
	"example":             true,
	"prereq":              true,
	"context":             true,
	"result":              true,
	"postreq":             true,
	"tasktroubleshooting": true,
}

// IsSectionClass reports whether a heading outputclass turns the heading into
// a section rather than a nested topic.
func IsSectionClass(class string) bool {
	return sectionClasses[class]
}

// topicTypeClasses are the heading outputclasses that type a topic. One of
// them on a heading inside a concept or a reference keeps the heading a nested
// topic, which is the only way to nest one there.
var topicTypeClasses = map[string]bool{
	"topic":     true,
	"concept":   true,
	"task":      true,
	"reference": true,
}

// IsTopicTypeClass reports whether a heading outputclass types a topic.
func IsTopicTypeClass(class string) bool {
	return topicTypeClasses[class]
}
