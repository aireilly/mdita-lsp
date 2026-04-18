package paths

import (
	"net/url"
	"path/filepath"
	"strings"
)

type DocID struct {
	URI     string
	RelPath string
	Stem    string
	Slug    Slug
}

func URIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	decoded, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func PathToURI(path string) string {
	return "file://" + path
}

func RelPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

func DocIDFromURI(uri, rootURI string) DocID {
	path, _ := URIToPath(uri)
	rootPath, _ := URIToPath(rootURI)
	rel := RelPath(rootPath, path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return DocID{
		URI:     uri,
		RelPath: rel,
		Stem:    stem,
		Slug:    SlugOf(stem),
	}
}

func MatchesURL(id DocID, url string) bool {
	return id.RelPath == url || id.Stem+".md" == url || id.Stem+".markdown" == url
}

func IsMditaMapFile(path string, mapExts []string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, me := range mapExts {
		if strings.EqualFold(ext, me) {
			return true
		}
	}
	return false
}

func IsMarkdownFile(path string, mdExts []string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, me := range mdExts {
		if strings.EqualFold(ext, me) {
			return true
		}
	}
	return false
}

func IsMarkdownURI(uri string) bool {
	path, err := URIToPath(uri)
	if err != nil {
		return false
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	return ext == "md" || ext == "markdown" || ext == "mditamap"
}
