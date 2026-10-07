package workspace

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Folder struct {
	RootURI string
	Config  *config.Config

	mu      sync.RWMutex
	docs    map[string]*document.Document
	slugMap map[paths.Slug]*document.Document
}

func NewFolder(rootURI string, cfg *config.Config) *Folder {
	return &Folder{
		RootURI: rootURI,
		Config:  cfg,
		docs:    make(map[string]*document.Document),
		slugMap: make(map[paths.Slug]*document.Document),
	}
}

func (f *Folder) AddDoc(doc *document.Document) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[doc.URI] = doc
	id := doc.DocID(f.RootURI)
	f.slugMap[id.Slug] = doc
}

func (f *Folder) RemoveDoc(uri string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	doc, ok := f.docs[uri]
	if !ok {
		return
	}
	id := doc.DocID(f.RootURI)
	delete(f.slugMap, id.Slug)
	delete(f.docs, uri)
}

func (f *Folder) DocByURI(uri string) *document.Document {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.docs[uri]
}

func (f *Folder) DocBySlug(slug paths.Slug) *document.Document {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.slugMap[slug]
}

func (f *Folder) DocCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.docs)
}

func (f *Folder) AllDocs() []*document.Document {
	f.mu.RLock()
	defer f.mu.RUnlock()
	docs := make([]*document.Document, 0, len(f.docs))
	for _, d := range f.docs {
		docs = append(docs, d)
	}
	return docs
}

func (f *Folder) ScanFiles() error {
	rootPath, err := paths.URIToPath(f.RootURI)
	if err != nil {
		return err
	}
	exts := f.Config.Core.Markdown.FileExtensions
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == ".hg" {
				return filepath.SkipDir
			}
			return nil
		}
		if !paths.IsMarkdownFile(path, exts) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		uri := paths.PathToURI(path)
		doc := document.New(uri, 0, string(data))
		f.AddDoc(doc)
		return nil
	})
}

func (f *Folder) MapTexts() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var texts []string
	for _, d := range f.docs {
		if d.Kind == document.Map {
			texts = append(texts, d.Text)
		}
	}
	return texts
}

// ResolveLinkCandidates returns every document a link could plausibly refer
// to. An exact path match is unambiguous and returns a single candidate;
// otherwise the same-named files elsewhere in the workspace are offered, so a
// diagnostic can suggest the path the author meant.
func (f *Folder) ResolveLinkCandidates(rawURL string, sourceURI string) []*document.Document {
	if d := f.resolveExact(rawURL, sourceURI); d != nil {
		return []*document.Document{d}
	}
	var matches []*document.Document
	for _, d := range f.AllDocs() {
		id := d.DocID(f.RootURI)
		if paths.MatchesURL(id, decodeURL(rawURL)) {
			matches = append(matches, d)
		}
	}
	return matches
}

// ResolveLink resolves a link the way the plug-in resolves an href: against
// the source file's directory, and nowhere else.
//
// It used to fall back to any document with the same file name, so
// "[stem](target.md)" written in sub/ silently resolved to the root
// target.md. Everything in the editor then behaved as if the link worked
// while the build failed with DOTX008E on sub/target.md.
// ResolveLinkCandidates still offers the same-named files, so a diagnostic
// can name them.
func (f *Folder) ResolveLink(rawURL string, sourceURI string) *document.Document {
	return f.resolveExact(rawURL, sourceURI)
}

// resolveExact resolves a link against the source file's directory, which is
// how the plug-in resolves an href.
func (f *Folder) resolveExact(rawURL string, sourceURI string) *document.Document {
	if rawURL == "" || isExternalURL(rawURL) {
		return nil
	}
	srcPath, err := paths.URIToPath(sourceURI)
	if err != nil {
		return nil
	}
	targetPath := filepath.Clean(filepath.Join(filepath.Dir(srcPath), decodeURL(rawURL)))
	return f.DocByURI(paths.PathToURI(targetPath))
}

// decodeURL undoes percent-escapes. The plug-in keeps "my%20file.md" as the
// href and DITA-OT resolves it to "my file.md" on disk.
func decodeURL(rawURL string) string {
	decoded, err := url.PathUnescape(rawURL)
	if err != nil {
		return rawURL
	}
	return decoded
}

func isExternalURL(rawURL string) bool {
	if strings.HasPrefix(rawURL, "//") || strings.HasPrefix(rawURL, "/") {
		return true
	}
	colon := strings.Index(rawURL, ":")
	if colon < 2 {
		return false
	}
	for i := 0; i < colon; i++ {
		c := rawURL[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') &&
			c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	c := rawURL[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func (f *Folder) RootPath() string {
	p, _ := paths.URIToPath(f.RootURI)
	return p
}
