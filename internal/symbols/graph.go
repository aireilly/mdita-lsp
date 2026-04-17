package symbols

import (
	"strings"
	"sync"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Graph struct {
	mu   sync.RWMutex
	defs map[string][]document.Symbol
	refs map[string][]document.Symbol

	defsBySlug map[paths.Slug][]document.Symbol
	docSlugs   map[paths.Slug]string
}

func NewGraph() *Graph {
	return &Graph{
		defs:       make(map[string][]document.Symbol),
		refs:       make(map[string][]document.Symbol),
		defsBySlug: make(map[paths.Slug][]document.Symbol),
		docSlugs:   make(map[paths.Slug]string),
	}
}

func (g *Graph) AddDefs(docURI string, defs []document.Symbol) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.removeDefs(docURI)
	g.defs[docURI] = defs

	for _, d := range defs {
		if d.Slug != "" {
			g.defsBySlug[d.Slug] = append(g.defsBySlug[d.Slug], d)
		}
		if d.DefType == document.DefDoc {
			stem := docStemSlug(docURI)
			g.docSlugs[stem] = docURI
		}
	}
}

func (g *Graph) AddRefs(docURI string, refs []document.Symbol) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.refs[docURI] = refs
}

func (g *Graph) RemoveDoc(docURI string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removeDefs(docURI)
	delete(g.refs, docURI)
}

func (g *Graph) removeDefs(docURI string) {
	old := g.defs[docURI]
	for _, d := range old {
		if d.Slug != "" {
			filtered := filterByDoc(g.defsBySlug[d.Slug], docURI)
			if len(filtered) == 0 {
				delete(g.defsBySlug, d.Slug)
			} else {
				g.defsBySlug[d.Slug] = filtered
			}
		}
		if d.DefType == document.DefDoc {
			stem := docStemSlug(docURI)
			delete(g.docSlugs, stem)
		}
	}
	delete(g.defs, docURI)
}

func (g *Graph) ResolveRef(ref document.Symbol) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.defsBySlug[ref.Slug]
}

func (g *Graph) ResolveDocRef(slug paths.Slug) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()

	docURI, ok := g.docSlugs[slug]
	if !ok {
		return nil
	}
	for _, d := range g.defs[docURI] {
		if d.DefType == document.DefTitle || d.DefType == document.DefDoc {
			return []document.Symbol{d}
		}
	}
	return nil
}

func (g *Graph) FindRefs(def document.Symbol) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []document.Symbol
	for _, refs := range g.refs {
		for _, r := range refs {
			if r.Slug == def.Slug {
				result = append(result, r)
			}
		}
	}
	return result
}

func (g *Graph) DefsByDoc(docURI string) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.defs[docURI]
}

func (g *Graph) AllDefs() []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var all []document.Symbol
	for _, defs := range g.defs {
		all = append(all, defs...)
	}
	return all
}

func filterByDoc(syms []document.Symbol, docURI string) []document.Symbol {
	var result []document.Symbol
	for _, s := range syms {
		if s.DocURI != docURI {
			result = append(result, s)
		}
	}
	return result
}

func docStemSlug(uri string) paths.Slug {
	lastSlash := strings.LastIndex(uri, "/")
	name := uri[lastSlash+1:]
	dotIdx := strings.LastIndex(name, ".")
	if dotIdx > 0 {
		name = name[:dotIdx]
	}
	return paths.SlugOf(name)
}
