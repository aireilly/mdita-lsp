package paths

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		// Runs of separators are not collapsed: NO_DUPED_DASHES defaults to false.
		{"Hello  World", "hello--world"},
		{"Hello - World", "hello---world"},
		{"Hello's World!", "hellos-world"},
		// Outer whitespace is trimmed before the id is generated.
		{"  spaces  ", "spaces"},
		{"UPPER CASE", "upper-case"},
		{"already-slug", "already-slug"},
		{"", ""},
		{"a", "a"},
		{"Hello #1 World", "hello-1-world"},
		{"Héllo Wörld", "héllo-wörld"},
		{"foo---bar", "foo---bar"},
		{"!@#start", "start"},
		{"end!@#", "end"},
		// '_' is a dash character, which is how the plug-in turns
		// "My_var config" into "my-var-config".
		{"My_var config", "my-var-config"},
		{"a.b.c", "abc"},
		{"C++ and C#", "c-and-c"},
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

func TestAnchorIDsResolveDuplicates(t *testing.T) {
	got := AnchorIDs([]string{"Setup", "Install", "Setup", "Setup"})
	want := []string{"setup", "install", "setup-1", "setup-2"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AnchorIDs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
