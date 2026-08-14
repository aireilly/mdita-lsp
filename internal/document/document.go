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

	kind := Topic
	if paths.IsMditaMapFile(uri, []string{"mditamap"}) {
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

func resolveTaskSections(doc *Document) {
	isTask := doc.Meta != nil && doc.Meta.Schema == SchemaTask
	if !isTask {
		for _, e := range doc.Elements {
			if h, ok := e.(*Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
					}
				}
			}
		}
	}

	for _, e := range doc.Elements {
		h, ok := e.(*Heading)
		if !ok || h.Level < 2 {
			continue
		}
		// Always check related links regardless of task status
		lowerText := strings.ToLower(h.Text)
		if lowerText == "related information" || lowerText == "related links" {
			h.IsRelLinks = true
		}
		if h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				if c == "related-links" {
					h.IsRelLinks = true
				}
			}
		}

		if !isTask {
			continue
		}
		// Resolve task section from class first, then title
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

func resolveImplicitSections(doc *Document) {
	isTask := doc.Meta != nil && doc.Meta.Schema == SchemaTask
	if !isTask {
		for _, e := range doc.Elements {
			if h, ok := e.(*Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
					}
				}
			}
		}
	}
	if !isTask {
		return
	}

	lines := strings.Split(doc.Text, "\n")
	titleLine := -1
	for _, e := range doc.Elements {
		if h, ok := e.(*Heading); ok && h.IsTitle() {
			titleLine = h.Range.Start.Line
		}
	}
	if titleLine < 0 {
		return
	}

	// Find the first and last top-level ordered list lines (the steps block).
	firstOLLine := -1
	lastOLLine := -1
	for i := titleLine + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		// Stop scanning at a non-title heading (explicit task section heading).
		if indent == 0 && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if indent == 0 && implicitIsOrderedItem(trimmed) {
			if firstOLLine < 0 {
				firstOLLine = i
			}
			lastOLLine = i
		}
	}

	// ImplicitContext: non-empty non-heading content between title and first ordered list.
	if firstOLLine > titleLine+1 {
		contextEnd := firstOLLine - 1
		for contextEnd > titleLine && strings.TrimSpace(lines[contextEnd]) == "" {
			contextEnd--
		}
		hasContent := false
		for i := titleLine + 1; i <= contextEnd; i++ {
			t := strings.TrimSpace(lines[i])
			if t != "" && !strings.HasPrefix(t, "#") {
				hasContent = true
				break
			}
		}
		if hasContent {
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitContext,
				Range: Rng(titleLine+1, 0, contextEnd, len(lines[contextEnd])),
			})
		}
	}

	// Find the true end of the steps block, including nested/continuation lines after lastOLLine.
	stepsBlockEnd := lastOLLine
	if lastOLLine >= 0 {
		for i := lastOLLine + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				continue
			}
			indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
			if indent > 0 {
				stepsBlockEnd = i
			} else {
				break
			}
		}
	}

	// ImplicitResult: non-empty non-heading content after the steps block.
	if lastOLLine >= 0 {
		resultStart := -1
		resultEnd := -1
		for i := stepsBlockEnd + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "#") {
				break
			}
			if resultStart < 0 {
				resultStart = i
			}
			resultEnd = i
		}
		if resultStart >= 0 {
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitResult,
				Range: Rng(resultStart, 0, resultEnd, len(lines[resultEnd])),
			})
		}
	}

	// Detect nested structures (choices, substeps, choicetable) within the steps block.
	if firstOLLine >= 0 {
		doc.ImplicitSections = append(doc.ImplicitSections, detectNestedListSections(lines, firstOLLine)...)
	}
}

// detectNestedListSections scans lines from fromLine looking for indented list items and
// tables nested inside ordered list items, returning ImplicitChoices, ImplicitSubsteps,
// and ImplicitChoicetable sections.
func detectNestedListSections(lines []string, fromLine int) []ImplicitSection {
	var sections []ImplicitSection
	i := fromLine
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Stop at any top-level heading.
		if indent == 0 && strings.HasPrefix(trimmed, "#") {
			break
		}
		// Stop at top-level non-list non-empty content.
		if indent == 0 && trimmed != "" && !implicitIsOrderedItem(trimmed) && !implicitIsUnorderedItem(trimmed) {
			break
		}

		if indent >= 2 {
			switch {
			case implicitIsUnorderedItem(trimmed):
				blockStart := i
				blockEnd := i
				for j := i + 1; j < len(lines); j++ {
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
					Kind:  ImplicitChoices,
					Range: Rng(blockStart, 0, blockEnd, len(lines[blockEnd])),
				})
				i = blockEnd + 1
				continue
			case implicitIsOrderedItem(trimmed):
				blockStart := i
				blockEnd := i
				for j := i + 1; j < len(lines); j++ {
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
					Kind:  ImplicitSubsteps,
					Range: Rng(blockStart, 0, blockEnd, len(lines[blockEnd])),
				})
				i = blockEnd + 1
				continue
			case strings.HasPrefix(trimmed, "|"):
				blockStart := i
				blockEnd := i
				for j := i + 1; j < len(lines); j++ {
					jt := strings.TrimSpace(lines[j])
					if strings.HasPrefix(jt, "|") {
						blockEnd = j
					} else if jt == "" {
						continue
					} else {
						break
					}
				}
				sections = append(sections, ImplicitSection{
					Kind:  ImplicitChoicetable,
					Range: Rng(blockStart, 0, blockEnd, len(lines[blockEnd])),
				})
				i = blockEnd + 1
				continue
			}
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

func taskSectionKindFromTitle(title string) TaskSectionKind {
	switch strings.ToLower(title) {
	case "prerequisites":
		return TaskSectionPrereq
	case "about this task":
		return TaskSectionContext
	case "verification":
		return TaskSectionResult
	case "next steps":
		return TaskSectionPostreq
	default:
		return TaskSectionNone
	}
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
