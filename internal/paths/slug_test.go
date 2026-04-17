package paths

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Hello  World", "hello-world"},
		{"Hello - World", "hello-world"},
		{"Hello's World!", "hellos-world"},
		{"  spaces  ", "spaces"},
		{"UPPER CASE", "upper-case"},
		{"already-slug", "already-slug"},
		{"", ""},
		{"a", "a"},
		{"Hello #1 World", "hello-1-world"},
		{"Héllo Wörld", "héllo-wörld"},
		{"foo---bar", "foo-bar"},
		{"!@#start", "start"},
		{"end!@#", "end"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlugIsSubstring(t *testing.T) {
	tests := []struct {
		haystack string
		needle   string
		want     bool
	}{
		{"hello-world", "hello", true},
		{"hello-world", "world", true},
		{"hello-world", "hello-world", true},
		{"hello-world", "xyz", false},
		{"hello-world", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.haystack+"_"+tt.needle, func(t *testing.T) {
			h := Slug(tt.haystack)
			n := Slug(tt.needle)
			if got := h.Contains(n); got != tt.want {
				t.Errorf("Slug(%q).Contains(%q) = %v, want %v", tt.haystack, tt.needle, got, tt.want)
			}
		})
	}
}
