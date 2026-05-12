package keyref

import "strings"

func fencedCodeLines(text string) map[int]bool {
	lines := strings.Split(text, "\n")
	fenced := make(map[int]bool)
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced[i] = true
			inFence = !inFence
			continue
		}
		if inFence {
			fenced[i] = true
		}
	}
	return fenced
}

func isInInlineCode(line string, col int) bool {
	inCode := false
	i := 0
	for i < len(line) {
		if line[i] == '`' {
			ticks := 0
			for i < len(line) && line[i] == '`' {
				ticks++
				i++
			}
			if inCode {
				inCode = false
			} else {
				inCode = true
			}
			continue
		}
		if i == col {
			return inCode
		}
		i++
	}
	return false
}
