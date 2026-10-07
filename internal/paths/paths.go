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

// URIToPath converts a file URI to a local path. It handles the Windows forms
// the spec allows -- file:///C:/dir and the UNC file://host/share -- as well as
// percent-escapes such as %20 for a space.
func URIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	decoded, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", err
	}
	if u.Host != "" && u.Host != "localhost" {
		// UNC path: file://server/share/file -> \\server\share\file
		return `\\` + u.Host + filepath.FromSlash(decoded), nil
	}
	// A Windows drive path arrives as /C:/dir; drop the leading slash.
	if len(decoded) > 2 && decoded[0] == '/' && decoded[2] == ':' && isDriveLetter(decoded[1]) {
		decoded = decoded[1:]
		return filepath.FromSlash(decoded), nil
	}
	return decoded, nil
}

// PathToURI converts a local path to a file URI. Each segment is
// percent-encoded, so a path containing a space or a '#' produces a URI the
// client can parse back to the same path.
func PathToURI(path string) string {
	slashed := filepath.ToSlash(path)

	// UNC path: \\server\share\file -> file://server/share/file
	if strings.HasPrefix(slashed, "//") {
		rest := strings.TrimPrefix(slashed, "//")
		host, tail, _ := strings.Cut(rest, "/")
		return "file://" + host + "/" + encodePath(tail)
	}

	// Windows drive path: C:/dir -> file:///C:/dir
	if len(slashed) > 1 && slashed[1] == ':' && isDriveLetter(slashed[0]) {
		drive := slashed[:2]
		tail := strings.TrimPrefix(slashed[2:], "/")
		return "file:///" + drive + "/" + encodePath(tail)
	}

	return "file://" + "/" + encodePath(strings.TrimPrefix(slashed, "/"))
}

func encodePath(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

func isDriveLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
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

// MatchesURL reports whether a link URL could address this document by file
// name alone. It applies only to a bare file name: the plug-in resolves a URL
// containing a separator against the source file's directory, so falling back
// to a same-named file elsewhere in the tree hid links that fail the build.
func MatchesURL(id DocID, url string) bool {
	if id.RelPath == url {
		return true
	}
	if strings.ContainsRune(url, '/') {
		return false
	}
	for _, ext := range SourceExtensions {
		if id.Stem+"."+ext == url {
			return true
		}
	}
	return false
}

// SourceExtensions are the file extensions the plug-in registers a parser for.
var SourceExtensions = []string{"md", "markdown", "mdita", "mditamap"}

// Format is the DITA-OT @format a file is parsed as. DITA-OT takes the format
// from the topicref, but the extension is the convention every build follows
// and the only signal the server has for a file it sees on its own.
type Format int

const (
	FormatOther Format = iota
	// FormatMD is `md`: Markdown DITA with implicit task sections enabled.
	FormatMD
	// FormatMarkdown is `markdown`: Markdown DITA.
	FormatMarkdown
	// FormatMdita is `mdita`: MDITA, extended profile unless $schema says core.
	FormatMdita
	// FormatMditamap is `mditamap`: an MDITA map.
	FormatMditamap
)

// FormatForPath returns the parser format a path's extension selects.
func FormatForPath(path string) Format {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")) {
	case "md":
		return FormatMD
	case "markdown":
		return FormatMarkdown
	case "mdita":
		return FormatMdita
	case "mditamap":
		return FormatMditamap
	default:
		return FormatOther
	}
}

// FormatForURI returns the parser format a URI's extension selects.
func FormatForURI(uri string) Format {
	path, err := URIToPath(uri)
	if err != nil {
		return FormatOther
	}
	return FormatForPath(path)
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
	return IsMarkdownFile(path, SourceExtensions)
}
