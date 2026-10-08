package codeaction

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/diagnostic"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

var sectionClassAttr = regexp.MustCompile(`\s*\{\s*\.section\s*\}`)

// fixNestedSectionActions offers to lift a heading that would nest a section
// in a concept or a reference up to the level that opens one.
func fixNestedSectionActions(doc *document.Document, rng document.Range) []CodeAction {
	if !document.IsSectionTopic(doc) {
		return nil
	}
	var actions []CodeAction
	topicLevel := 0
	prevLevel := 0
	for _, h := range doc.Index.Headings() {
		skipped := h.Level > prevLevel+1
		prevLevel = h.Level
		if !h.Section {
			topicLevel = h.Level
			continue
		}
		if h.Level <= topicLevel+1 || skipped || !rangesOverlap(rng, h.Range) {
			continue
		}
		fixed := strings.Repeat("#", topicLevel+1)
		actions = append(actions, CodeAction{
			Title:  "Change to " + fixed,
			Kind:   "quickfix",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   h.LineRange,
				NewText: fixed + " " + headingRest(doc, h),
			},
			Diagnostics: []DiagnosticInfo{{
				Range:    h.Range,
				Severity: 1,
				Code:     diagnostic.CodeNestedSection,
				Source:   source,
				Message:  diagnostic.NestedSectionMessage(doc, h, topicLevel),
			}},
		})
	}
	return actions
}

// fixSectionAfterNestedTopicActions offers both ways out of the ordering trap:
// make the heading above a section too, or stop this one being one.
func fixSectionAfterNestedTopicActions(doc *document.Document, rng document.Range) []CodeAction {
	var actions []CodeAction
	topicLevel := 0
	var parent *document.Heading
	for _, h := range doc.Index.Headings() {
		if !h.Section {
			topicLevel = h.Level
			parent = h
			continue
		}
		if h.Level > topicLevel || parent == nil || !rangesOverlap(rng, h.Range) {
			continue
		}
		diag := DiagnosticInfo{
			Range:    h.Range,
			Severity: 1,
			Code:     diagnostic.CodeSectionAfterNestedTopic,
			Source:   source,
			Message:  diagnostic.SectionAfterNestedTopicMessage(doc, h, parent),
		}
		actions = append(actions,
			CodeAction{
				Title:  "Add {.section} to every heading at this level",
				Kind:   "quickfix",
				DocURI: doc.URI,
				Edit: &TextEdit{
					Range:   parent.LineRange,
					NewText: strings.TrimRight(lineOf(doc, parent), " ") + " {.section}",
				},
				Diagnostics: []DiagnosticInfo{diag},
			},
			CodeAction{
				Title:  "Remove {.section} from this heading",
				Kind:   "quickfix",
				DocURI: doc.URI,
				Edit: &TextEdit{
					Range:   h.LineRange,
					NewText: strings.TrimRight(sectionClassAttr.ReplaceAllString(lineOf(doc, h), ""), " "),
				},
				Diagnostics: []DiagnosticInfo{diag},
			},
		)
	}
	return actions
}

// qualifySectionLinkActions offers to add the topic id a link to a section
// needs.
func qualifySectionLinkActions(doc *document.Document, rng document.Range, folder *workspace.Folder) []CodeAction {
	var actions []CodeAction
	for _, ml := range doc.Index.MdLinks() {
		if ml.Anchor == "" || ml.URL == "" || !rangesOverlap(rng, ml.Range) {
			continue
		}
		target := folder.ResolveLink(ml.URL, doc.URI)
		if target == nil || !document.IsSectionTopic(target) {
			continue
		}
		if _, _, isDita := document.SplitFragment(ml.Anchor); isDita {
			continue
		}
		hs := target.Index.HeadingsByAnchor(ml.Anchor)
		if len(hs) == 0 || !hs[0].Section {
			continue
		}
		qualified := ml.URL + "#" + target.SectionAddress(hs[0])
		actions = append(actions, CodeAction{
			Title:  "Add the topic id to the section link",
			Kind:   "quickfix",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   ml.Range,
				NewText: "[" + ml.Text + "](" + qualified + ")",
			},
			Diagnostics: []DiagnosticInfo{{
				Range:    ml.Range,
				Severity: 2,
				Code:     diagnostic.CodeSectionLinkNeedsTopicID,
				Source:   source,
				Message:  diagnostic.SectionLinkMessage(ml.URL+"#"+ml.Anchor, qualified),
			}},
		})
	}
	return actions
}

// lineOf returns the heading's source line, hashes and attributes included.
func lineOf(doc *document.Document, h *document.Heading) string {
	lines := strings.Split(doc.Text, "\n")
	if h.LineRange.Start.Line < len(lines) {
		return strings.TrimRight(lines[h.LineRange.Start.Line], "\r")
	}
	return strings.Repeat("#", h.Level) + " " + h.Text
}

// headingRest returns everything after the hashes, so an edit that changes the
// level keeps the author's block attributes.
func headingRest(doc *document.Document, h *document.Heading) string {
	return strings.TrimLeft(strings.TrimLeft(lineOf(doc, h), "#"), " ")
}
