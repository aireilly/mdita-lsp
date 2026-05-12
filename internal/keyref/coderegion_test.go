package keyref

import "testing"

func TestFencedCodeLines(t *testing.T) {
	text := "# Title\n\nSome text.\n\n```yaml\napiVersion: v1\nkind: Pod\n```\n\nMore text.\n"
	fenced := fencedCodeLines(text)
	// lines 4-7 (0-indexed) are inside the fenced block
	if !fenced[4] || !fenced[5] || !fenced[6] || !fenced[7] {
		t.Errorf("expected lines 4-7 to be fenced, got %v", fenced)
	}
	if fenced[0] || fenced[2] || fenced[8] || fenced[9] {
		t.Error("non-fenced lines should not be marked")
	}
}

func TestFencedCodeLinesMultiple(t *testing.T) {
	text := "Text\n\n```\ncode1\n```\n\nMiddle\n\n```\ncode2\n```\n"
	fenced := fencedCodeLines(text)
	if !fenced[2] || !fenced[3] || !fenced[4] {
		t.Error("first block should be fenced")
	}
	if fenced[5] || fenced[6] {
		t.Error("middle text should not be fenced")
	}
	if !fenced[8] || !fenced[9] || !fenced[10] {
		t.Error("second block should be fenced")
	}
}

func TestIsInInlineCode(t *testing.T) {
	tests := []struct {
		line   string
		col    int
		expect bool
	}{
		{"Use `{{key}}` here", 6, true},   // inside backticks
		{"Use `{{key}}` here", 0, false},  // before backticks
		{"Use `{{key}}` here", 14, false}, // after backticks
		{"No code here {{key}}", 14, false},
		{"Double ``{{key}}`` tick", 10, true},
	}
	for _, tt := range tests {
		got := isInInlineCode(tt.line, tt.col)
		if got != tt.expect {
			t.Errorf("isInInlineCode(%q, %d) = %v, want %v", tt.line, tt.col, got, tt.expect)
		}
	}
}
