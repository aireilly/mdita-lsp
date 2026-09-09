package document

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Document struct {
	URI              string
	Version          int
	Text             string
	Lines            []int
	Elements         []Element
	Symbols          []Symbol
	Index            *Index
	Meta             *YAMLMetadata
	Kind             DocKind
	BlockAttrs       []BlockAttribute
	ImplicitSections []ImplicitSection
}

func New(uri string, version int, text string) *Document {
	elements, bf, meta := Parse(text)
	idx := BuildIndex(elements, bf, meta)
	idx.Meta = meta

	// A file is a map when its extension says so, or when it declares the DITA
	// map schema — the plug-in reads a Markdown DITA map from a .md file.
	kind := Topic
	if paths.IsMditaMapFile(uri, []string{"mditamap"}) || (meta != nil && meta.Schema == SchemaMap) {
		kind = Map
	}

	title := idx.Title()
	if title != nil && idx.ShortDesc == "" {
		idx.ShortDesc = findShortDesc(text, title)
	}

	blockAttrs := ScanBlockAttributes(text)
	if len(blockAttrs) > 0 {
		bf.HasAttributes = true
	}

	doc := &Document{
		URI:        uri,
		Version:    version,
		Text:       text,
		Lines:      buildLineMap(text),
		Elements:   elements,
		Index:      idx,
		Meta:       meta,
		Kind:       kind,
		BlockAttrs: blockAttrs,
	}
	doc.Symbols = extractSymbols(doc)
	resolveTaskSections(doc)
	resolveImplicitSections(doc)
	return doc
}

func (d *Document) ApplyChange(version int, newText string) *Document {
	return New(d.URI, version, newText)
}

func (d *Document) DocID(rootURI string) paths.DocID {
	return paths.DocIDFromURI(d.URI, rootURI)
}

func (d *Document) Defs() []Symbol {
	var defs []Symbol
	for _, s := range d.Symbols {
		if s.Kind == DefKind {
			defs = append(defs, s)
		}
	}
	return defs
}

func (d *Document) Refs() []Symbol {
	var refs []Symbol
	for _, s := range d.Symbols {
		if s.Kind == RefKind {
			refs = append(refs, s)
		}
	}
	return refs
}

func (d *Document) ElementAt(pos Position) Element {
	for _, e := range d.Elements {
		r := e.Rng()
		if posInRange(pos, r) {
			return e
		}
	}
	return nil
}

func posInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

func BuildLineMap(text string) []int {
	return buildLineMap(text)
}

func buildLineMap(text string) []int {
	lines := []int{0}
	for i, ch := range text {
		if ch == '\n' {
			lines = append(lines, i+1)
		}
	}
	return lines
}

func findShortDesc(text string, title *Heading) string {
	lines := strings.Split(text, "\n")
	titleLine := title.Range.Start.Line
	for i := titleLine + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return ""
		}
		return line
	}
	return ""
}

func extractSymbols(doc *Document) []Symbol {
	var syms []Symbol

	syms = append(syms, Symbol{
		Kind:    DefKind,
		DefType: DefDoc,
		Name:    doc.URI,
		DocURI:  doc.URI,
	})

	for _, e := range doc.Elements {
		switch el := e.(type) {
		case *Heading:
			dt := DefHeading
			if el.IsTitle() {
				dt = DefTitle
			}
			syms = append(syms, Symbol{
				Kind:    DefKind,
				DefType: dt,
				Name:    el.Text,
				Slug:    el.Slug,
				DocURI:  doc.URI,
				Range:   el.Range,
			})
			if el.ID != "" {
				syms = append(syms, Symbol{
					Kind:    DefKind,
					DefType: DefElementID,
					Name:    el.ID,
					DocURI:  doc.URI,
					Range:   el.Range,
				})
			}

		case *MdLink:
			syms = append(syms, Symbol{
				Kind:    RefKind,
				RefType: RefMdLink,
				Name:    el.URL,
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *LinkDef:
			syms = append(syms, Symbol{
				Kind:    DefKind,
				DefType: DefLinkDef,
				Name:    el.Label,
				DocURI:  doc.URI,
				Range:   el.Range,
			})
		}
	}
	return syms
}

// IsTaskTopic reports whether the plug-in specializes this topic as a DITA task,
// either because the H1 carries a {.task} outputclass or because $schema names
// the task schema.
func IsTaskTopic(doc *Document) bool {
	if doc.Meta != nil && doc.Meta.Schema == SchemaTask {
		return true
	}
	for _, e := range doc.Elements {
		if h, ok := e.(*Heading); ok && h.IsTitle() && h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				if c == "task" {
					return true
				}
			}
		}
	}
	return false
}

func resolveTaskSections(doc *Document) {
	if !IsTaskTopic(doc) {
		return
	}

	for _, e := range doc.Elements {
		h, ok := e.(*Heading)
		if !ok || h.Level < 2 {
			continue
		}
		// Resolve task section from outputclass first, then from the heading title.
		if h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				h.TaskSection = taskSectionKindFromClass(c)
				if h.TaskSection != TaskSectionNone {
					break
				}
			}
		}
		if h.TaskSection == TaskSectionNone {
			h.TaskSection = taskSectionKindFromTitle(h.Text)
		}
	}
}

type blockKind int

const (
	blkOther blockKind = iota
	blkHeading
	blkOrdered
	blkUnordered
)

type topBlock struct {
	kind blockKind
	// forcedBody marks the leading part of an ordered list that the plug-in
	// splits at a numbering reset: that part stays in <context> as a body-ol.
	forcedBody bool
	start      int
	end        int
	level      int
}

// scanTopBlocks splits the lines after the topic title into top-level blocks:
// headings, body-level ordered and unordered lists, and everything else.
// Indented lines and blank lines are folded into the block they follow.
func scanTopBlocks(lines []string, from int) []topBlock {
	var blocks []topBlock
	cur := -1
	pendingBlank := 0

	closeCur := func() { cur = -1; pendingBlank = 0 }

	for i := from; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			pendingBlank++
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent > 0 && cur >= 0 {
			blocks[cur].end = i
			pendingBlank = 0
			continue
		}

		switch {
		case indent == 0 && headingLevel(trimmed) > 0:
			closeCur()
			blocks = append(blocks, topBlock{kind: blkHeading, start: i, end: i, level: headingLevel(trimmed)})
		case indent == 0 && implicitIsOrderedItem(trimmed):
			if cur >= 0 && blocks[cur].kind == blkOrdered {
				blocks[cur].end = i
				pendingBlank = 0
				continue
			}
			closeCur()
			blocks = append(blocks, topBlock{kind: blkOrdered, start: i, end: i})
			cur = len(blocks) - 1
		case indent == 0 && implicitIsUnorderedItem(trimmed):
			if cur >= 0 && blocks[cur].kind == blkUnordered {
				blocks[cur].end = i
				pendingBlank = 0
				continue
			}
			closeCur()
			blocks = append(blocks, topBlock{kind: blkUnordered, start: i, end: i})
			cur = len(blocks) - 1
		default:
			// A blank line ends a run of plain body content, so each paragraph,
			// table, or code block is its own block.
			if cur >= 0 && blocks[cur].kind == blkOther && pendingBlank == 0 {
				blocks[cur].end = i
				continue
			}
			closeCur()
			blocks = append(blocks, topBlock{kind: blkOther, start: i, end: i})
			cur = len(blocks) - 1
		}
		pendingBlank = 0
	}
	return blocks
}

// splitNumberingResets reproduces the plug-in's handling of a body-level
// ordered list whose numbering restarts at 1: the leading part becomes a
// body-ol inside <context> and the remainder becomes <steps>.
func splitNumberingResets(body []topBlock, lines []string) []topBlock {
	var out []topBlock
	for _, b := range body {
		if b.kind != blkOrdered {
			out = append(out, b)
			continue
		}
		split := numberingResetLine(lines, b)
		if split < 0 {
			out = append(out, b)
			continue
		}
		head := b
		head.forcedBody = true
		head.end = split - 1
		for head.end > head.start && strings.TrimSpace(lines[head.end]) == "" {
			head.end--
		}
		tail := b
		tail.start = split
		out = append(out, head, tail)
	}
	return out
}

// numberingResetLine returns the line of the first top-level item that restarts
// the numbering at 1, or -1 when the list numbers monotonically.
func numberingResetLine(lines []string, b topBlock) int {
	prev := -1
	for i := b.start; i <= b.end && i < len(lines); i++ {
		line := lines[i]
		if len(line) != len(strings.TrimLeft(line, " \t")) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !implicitIsOrderedItem(trimmed) {
			continue
		}
		n := 0
		for j := 0; j < len(trimmed) && trimmed[j] >= '0' && trimmed[j] <= '9'; j++ {
			n = n*10 + int(trimmed[j]-'0')
		}
		if prev > 1 && n == 1 {
			return i
		}
		prev = n
	}
	return -1
}

func headingLevel(trimmed string) int {
	n := 0
	for n < len(trimmed) && trimmed[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0
	}
	if n < len(trimmed) && trimmed[n] != ' ' {
		return 0
	}
	return n
}

// resolveImplicitSections reproduces the implicit task structure the plug-in
// derives from a task topic body: content before the steps list becomes
// <context>, content after it becomes <result>, a body-level ordered list
// becomes <steps> and a body-level unordered list becomes <steps-unordered>,
// a body-level list that is followed by another body-level list stays in
// <context> as a body-ol/body-ul, and lists or tables nested inside a step
// become <substeps>, <choices>, or <choicetable>.
func resolveImplicitSections(doc *Document) {
	if !IsTaskTopic(doc) {
		return
	}

	lines := strings.Split(doc.Text, "\n")
	titleLine := -1
	stepsMarkerLines := make(map[int]bool)
	for _, e := range doc.Elements {
		h, ok := e.(*Heading)
		if !ok {
			continue
		}
		if h.IsTitle() {
			titleLine = h.Range.Start.Line
		}
		if h.TaskSection == TaskSectionSteps {
			stepsMarkerLines[h.Range.Start.Line] = true
		}
	}
	if titleLine < 0 {
		return
	}

	blocks := scanTopBlocks(lines, titleLine+1)

	// The paragraph directly after a typed title is pulled into <shortdesc>,
	// so it is not part of the task body.
	if len(blocks) > 0 && blocks[0].kind == blkOther {
		blocks = blocks[1:]
	}

	// A heading other than a steps marker opens an explicit section, so the
	// blocks that follow it are not body-level any more. A steps marker heading
	// closes the open section and hands the body back to implicit handling.
	inSection := false
	var body []topBlock
	for _, b := range blocks {
		if b.kind == blkHeading {
			inSection = !stepsMarkerLines[b.start]
			continue
		}
		if !inSection {
			body = append(body, b)
		}
	}

	body = splitNumberingResets(body, lines)

	// The last body-level list is the steps list; every earlier one is a
	// body-ol/body-ul that the plug-in keeps inside <context>.
	stepsIdx := -1
	for i, b := range body {
		if (b.kind == blkOrdered || b.kind == blkUnordered) && !b.forcedBody {
			stepsIdx = i
		}
	}

	for i, b := range body {
		switch {
		case b.forcedBody:
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitBodyList,
				Range: Rng(b.start, 0, b.end, len(lines[b.end])),
			})
		case i == stepsIdx && b.kind == blkOrdered:
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitSteps,
				Range: Rng(b.start, 0, b.end, len(lines[b.end])),
			})
		case i == stepsIdx && b.kind == blkUnordered:
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitStepsUnordered,
				Range: Rng(b.start, 0, b.end, len(lines[b.end])),
			})
		case b.kind == blkOrdered || b.kind == blkUnordered:
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitBodyList,
				Range: Rng(b.start, 0, b.end, len(lines[b.end])),
			})
		case stepsIdx < 0 || i < stepsIdx:
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitContext,
				Range: Rng(b.start, 0, b.end, len(lines[b.end])),
			})
		default:
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitResult,
				Range: Rng(b.start, 0, b.end, len(lines[b.end])),
			})
		}
	}

	if stepsIdx >= 0 {
		doc.ImplicitSections = append(doc.ImplicitSections,
			detectNestedListSections(lines, body[stepsIdx].start, body[stepsIdx].end)...)
	}
}

// detectNestedListSections scans a steps block for indented list items and
// tables nested inside its steps, returning ImplicitChoices, ImplicitSubsteps,
// and ImplicitChoicetable sections.
func detectNestedListSections(lines []string, fromLine, toLine int) []ImplicitSection {
	var sections []ImplicitSection
	i := fromLine
	for i <= toLine && i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		if indent >= 2 {
			var kind ImplicitSectionKind
			switch {
			case implicitIsUnorderedItem(trimmed):
				kind = ImplicitChoices
			case implicitIsOrderedItem(trimmed):
				kind = ImplicitSubsteps
			case strings.HasPrefix(trimmed, "|"):
				kind = ImplicitChoicetable
			default:
				i++
				continue
			}
			blockEnd := i
			for j := i + 1; j <= toLine && j < len(lines); j++ {
				jt := strings.TrimSpace(lines[j])
				ji := len(lines[j]) - len(strings.TrimLeft(lines[j], " \t"))
				if jt == "" {
					continue
				}
				if ji >= 2 {
					blockEnd = j
				} else {
					break
				}
			}
			sections = append(sections, ImplicitSection{
				Kind:  kind,
				Range: Rng(i, 0, blockEnd, len(lines[blockEnd])),
			})
			i = blockEnd + 1
			continue
		}
		i++
	}
	return sections
}

// implicitIsOrderedItem reports whether s (already trimmed) is an ordered list item
// such as "1. " or "12. ".
func implicitIsOrderedItem(s string) bool {
	if len(s) < 3 {
		return false
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i > 0 && i < len(s)-1 && s[i] == '.' && s[i+1] == ' '
}

// implicitIsUnorderedItem reports whether s (already trimmed) is an unordered list item
// starting with "- ", "* ", or "+ ".
func implicitIsUnorderedItem(s string) bool {
	return len(s) >= 2 && (s[0] == '-' || s[0] == '*' || s[0] == '+') && s[1] == ' '
}

// DefaultImplicitTaskSectionTitles mirrors the plug-in's
// IMPLICIT_TASK_SECTION_TITLES defaults: the heading titles that map to task
// section elements. The "steps" entry lists the marker headings whose ordered
// list becomes <steps>.
func DefaultImplicitTaskSectionTitles() map[string][]string {
	return map[string][]string{
		"prereq":  {"prerequisites"},
		"context": {"about this task"},
		"steps":   {"procedure", "steps"},
		"result":  {"verification"},
		"postreq": {"next steps"},
	}
}

// implicitTaskSectionTitles maps a lowercased heading title to its task
// section. The plug-in lets a DITA-OT integrator override the lists through
// the http://lwdita.org/sax/properties/implicit-task-sections/* properties, so
// SetImplicitTaskSectionTitles offers the same override.
var implicitTaskSectionTitles = buildTitleIndex(DefaultImplicitTaskSectionTitles())

// SetImplicitTaskSectionTitles replaces the heading-title-to-section mapping.
// Call it once while loading configuration, before documents are parsed.
func SetImplicitTaskSectionTitles(titles map[string][]string) {
	merged := DefaultImplicitTaskSectionTitles()
	for section, list := range titles {
		if len(list) > 0 {
			merged[section] = list
		}
	}
	implicitTaskSectionTitles = buildTitleIndex(merged)
}

func buildTitleIndex(titles map[string][]string) map[string]TaskSectionKind {
	index := make(map[string]TaskSectionKind)
	for section, list := range titles {
		kind := taskSectionKindFromClass(section)
		if section == "steps" {
			kind = TaskSectionSteps
		}
		if kind == TaskSectionNone {
			continue
		}
		for _, title := range list {
			index[strings.ToLower(title)] = kind
		}
	}
	return index
}

func taskSectionKindFromTitle(title string) TaskSectionKind {
	return implicitTaskSectionTitles[strings.ToLower(title)]
}

func taskSectionKindFromClass(class string) TaskSectionKind {
	switch class {
	case "prereq":
		return TaskSectionPrereq
	case "context":
		return TaskSectionContext
	case "result":
		return TaskSectionResult
	case "postreq":
		return TaskSectionPostreq
	case "tasktroubleshooting":
		return TaskSectionTroubleshooting
	default:
		return TaskSectionNone
	}
}
