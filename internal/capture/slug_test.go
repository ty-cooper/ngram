package capture

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SQLi on Login Form (POST)", "sqli-on-login-form-post"},
		{"nmap -sV 10.10.10.8", "nmap-sv-10-10-10-8"},
		{"hello world", "hello-world"},
		{"  leading and trailing  ", "leading-and-trailing"},
		{"UPPER CASE", "upper-case"},
		{"special!@#chars$%^here", "special-chars-here"},
		{"", ""},
		{"a", "a"},
		{"a-b-c", "a-b-c"},
		// Truncation test — input longer than 50 chars
		{"this is a very long title that should definitely be truncated at a word boundary", "this-is-a-very-long-title-that-should-definitely"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if len(got) > 50 {
				t.Errorf("Slugify(%q) length %d > 50", tt.input, len(got))
			}
		})
	}
}

func TestAutoTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"found sqli on login param id on port 8080 extra words", "found sqli on login param id on port"},
		{"short", "short"},
		{"one two three", "one two three"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := AutoTitle(tt.input)
			if got != tt.want {
				t.Errorf("AutoTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
