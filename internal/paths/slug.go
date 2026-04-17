package paths

import (
	"strings"
	"unicode"
)

type Slug string

func Slugify(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	var sepSeen bool
	var chunkState int // 0=none, 1=in-progress, 2=finished

	for _, ch := range s {
		isPunct := unicode.IsPunct(ch) || unicode.IsSymbol(ch)
		isSep := unicode.IsSpace(ch) || ch == '-'
		isOut := !isPunct && !isSep

		if isSep {
			sepSeen = true
		}

		if isOut {
			if sepSeen && chunkState == 2 {
				b.WriteByte('-')
				sepSeen = false
			}
			chunkState = 1
			b.WriteRune(unicode.ToLower(ch))
		} else if chunkState == 1 {
			chunkState = 2
		}
	}
	return b.String()
}

func SlugOf(s string) Slug {
	return Slug(Slugify(s))
}

func (s Slug) String() string {
	return string(s)
}

func (s Slug) Contains(sub Slug) bool {
	return strings.Contains(string(s), string(sub))
}
