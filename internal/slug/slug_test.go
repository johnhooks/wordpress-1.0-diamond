package slug_test

import (
	"testing"

	"press/internal/slug"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Hello  World", "hello-world"},
		{"  Hello World  ", "hello-world"},
		{"Hello, World!", "hello-world"},
		{"Hello---World", "hello-world"},
		{"Héllo Wörld", "hello-world"},
		{"UPPERCASE", "uppercase"},
		{"already-a-slug", "already-a-slug"},
		{"foo & bar", "foo-bar"},
		{"What's New?", "whats-new"},
		{"100% Pure", "100-pure"},
		{"café résumé", "cafe-resume"},
		{"日本語", ""},
		{"", ""},
		{"---", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slug.Generate(tt.input)
			if got != tt.want {
				t.Errorf("Generate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnsureUnique(t *testing.T) {
	taken := map[string]bool{
		"hello-world":   true,
		"hello-world-2": true,
	}
	exists := func(s string) (bool, error) {
		return taken[s], nil
	}

	got, err := slug.EnsureUnique("hello-world", exists)
	if err != nil {
		t.Fatalf("EnsureUnique: %v", err)
	}
	if got != "hello-world-3" {
		t.Errorf("expected hello-world-3, got %q", got)
	}

	// Unique slug stays as-is
	got, err = slug.EnsureUnique("unique", exists)
	if err != nil {
		t.Fatalf("EnsureUnique unique: %v", err)
	}
	if got != "unique" {
		t.Errorf("expected unique, got %q", got)
	}
}
