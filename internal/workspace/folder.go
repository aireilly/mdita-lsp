package workspace

import (
	"os"
	"path/filepath"
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

func (f *Folder) RootPath() string {
	p, _ := paths.URIToPath(f.RootURI)
	return p
}
