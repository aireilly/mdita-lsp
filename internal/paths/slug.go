package paths

import (
	"strconv"
	"strings"
	"unicode"
)

type Slug string

// Slugify reproduces flexmark's HeaderIdGenerator.generateId, which is what
// the plug-in uses for a heading's @id and therefore for the fragment a link
// has to match. The rule is narrow:
//
//   - a letter becomes its lowercase form
//   - a digit is kept
//   - a space, '-' or '_' becomes '-'
//   - everything else is dropped
//
// Runs are not collapsed, because the plug-in's defaults are
// HEADER_ID_GENERATOR_TO_DASH_CHARS=" -_" and NO_DUPED_DASHES=false. An
// earlier version collapsed runs of separators and dropped '_' entirely, so
// "My_var config" produced "myvar-config" where the build produces
// "my-var-config", and the link to it was reported broken.
func Slugify(s string) string {
	// HEADER_ID_REF_TEXT_TRIM_LEADING_SPACES and _TRAILING_SPACES both default
	// to true, so the generator never sees the heading's outer whitespace.
	s = strings.TrimSpace(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, ch := range s {
		switch {
		case unicode.IsLetter(ch):
			b.WriteRune(unicode.ToLower(ch))
		case unicode.IsDigit(ch):
			b.WriteRune(ch)
		case ch == ' ' || ch == '-' || ch == '_':
			b.WriteByte('-')
		}
	}
	return b.String()
}

func SlugOf(s string) Slug {
	return Slug(Slugify(s))
}

// AnchorIDs assigns each heading text the id the plug-in generates for it,
// resolving duplicates the way HEADER_ID_GENERATOR_RESOLVE_DUPES does: the
// first "Setup" keeps "setup", the second becomes "setup-1", the third
// "setup-2". Links to the suffixed form were reported broken because the
// server only ever knew the base id.
func AnchorIDs(texts []string) []string {
	seen := make(map[string]int, len(texts))
	ids := make([]string, len(texts))
	for i, text := range texts {
		base := Slugify(text)
		if base == "" {
			ids[i] = ""
			continue
		}
		if n, ok := seen[base]; ok {
			n++
			seen[base] = n
			ids[i] = base + "-" + strconv.Itoa(n)
		} else {
			seen[base] = 0
			ids[i] = base
		}
	}
	return ids
}

func (s Slug) String() string {
	return string(s)
}

func (s Slug) Contains(sub Slug) bool {
	return strings.Contains(string(s), string(sub))
}
