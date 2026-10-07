package document

import "strings"

// MaskFencedCode blanks out the contents of fenced code blocks, keeping every
// byte offset and line break in place so a regex scan over the result reports
// positions that are still valid for the original text.
//
// The plug-in turns a fenced block into a literal <codeblock>, so "[^1]" or
// "[in-code]" inside one is text, not a footnote or a key reference. Scanning
// the raw source reported both as errors in sample code.
func MaskFencedCode(text string) string {
	lines := strings.Split(text, "\n")
	fence := ""
	changed := false

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)

		if fence == "" {
			// An opening fence is indented by at most three spaces.
			if indent <= 3 {
				if marker := fenceMarker(trimmed); marker != "" {
					fence = marker
				}
			}
			continue
		}

		// A closing fence is the same character, at least as long, with
		// nothing after it.
		if indent <= 3 && isClosingFence(trimmed, fence) {
			fence = ""
			continue
		}

		if line != "" {
			lines[i] = strings.Repeat(" ", len(line))
			changed = true
		}
	}

	if !changed {
		return text
	}
	return strings.Join(lines, "\n")
}

// fenceMarker returns the run of backticks or tildes that opens a fence, or
// "" when the line does not open one.
func fenceMarker(trimmed string) string {
	for _, ch := range []byte{'`', '~'} {
		n := 0
		for n < len(trimmed) && trimmed[n] == ch {
			n++
		}
		if n >= 3 {
			// A backtick fence's info string may not contain a backtick.
			if ch == '`' && strings.ContainsRune(trimmed[n:], '`') {
				return ""
			}
			return strings.Repeat(string(ch), n)
		}
	}
	return ""
}

func isClosingFence(trimmed, fence string) bool {
	if !strings.HasPrefix(trimmed, fence) {
		return false
	}
	return strings.Trim(trimmed, string(fence[0])) == "" ||
		strings.TrimSpace(strings.TrimLeft(trimmed, string(fence[0]))) == ""
}
