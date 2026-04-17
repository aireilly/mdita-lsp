package workspace

import (
	"strings"
	"sync"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type Workspace struct {
	mu      sync.RWMutex
	folders map[string]*Folder
}

func New() *Workspace {
	return &Workspace{
		folders: make(map[string]*Folder),
	}
}

func (ws *Workspace) AddFolder(f *Folder) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.folders[f.RootURI] = f
}

func (ws *Workspace) RemoveFolder(rootURI string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.folders, rootURI)
}

func (ws *Workspace) Folders() []*Folder {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	folders := make([]*Folder, 0, len(ws.folders))
	for _, f := range ws.folders {
		folders = append(folders, f)
	}
	return folders
}

func (ws *Workspace) FolderByURI(rootURI string) *Folder {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.folders[rootURI]
}

func (ws *Workspace) FindDoc(uri string) (*document.Document, *Folder) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for _, f := range ws.folders {
		if doc := f.DocByURI(uri); doc != nil {
			return doc, f
		}
	}
	return nil, nil
}

func (ws *Workspace) FolderForURI(uri string) *Folder {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for _, f := range ws.folders {
		if strings.HasPrefix(uri, f.RootURI) {
			return f
		}
	}
	return nil
}
