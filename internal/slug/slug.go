// Package slug generates URL-friendly slugs from strings.
// Equivalent to WordPress's sanitize_title_with_dashes().
package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	reNonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]`)
	reMultipleHyphens = regexp.MustCompile(`-{2,}`)
)

// Generate creates a URL-friendly slug from a string.
// Lowercases, strips accents, replaces non-alphanumeric with hyphens,
// collapses consecutive hyphens, trims leading/trailing hyphens.
func Generate(s string) string {
	// Normalize unicode (NFD) so accented chars decompose into base + combining
	s = norm.NFD.String(s)

	// Strip combining marks (accents)
	var b strings.Builder
	for _, r := range s {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	s = b.String()

	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = reNonAlphanumeric.ReplaceAllString(s, "")
	s = reMultipleHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
}

// EnsureUnique appends -2, -3, etc. until exists returns false.
// Pass a function that checks whether the candidate slug is already taken.
func EnsureUnique(slug string, exists func(string) (bool, error)) (string, error) {
	candidate := slug
	suffix := 2
	for {
		taken, err := exists(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		candidate = slug + "-" + itoa(suffix)
		suffix++
	}
}

// itoa without importing strconv for a trivial case.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
