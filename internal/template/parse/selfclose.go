package parse

// vocabularyTags is the set of engine tag names that should be treated
// as self-closing (void-like) elements by the parser. These tags never
// have children in theme source files. The parser respects the />
// self-closing syntax on these tags instead of ignoring it per the
// HTML5 spec.
//
// This set will grow as the engine vocabulary expands. It should match
// the vocabulary defined in the vocab package once that exists.
var vocabularyTags = map[string]bool{
	// Engine tags — the theme places these, the engine fills them.
	"comment":         true,
	"comment-form":    true,
	"comment-list":    true,
	"post":            true,
	"post-list":       true,
	"post-navigation": true,
	"pagination":      true,
	"search-form":     true,
	"category-list":   true,
	"archive-list":    true,
	"page-list":       true,
	"meta-links":      true,
	"sidebar":         true,
	"archive-header":  true,
	"editor":          true,
	"site-header":     true,
	"site-footer":     true,
}

// isVocabularyTag returns true if the tag name is a known engine
// vocabulary tag that should support self-closing syntax.
func isVocabularyTag(name string) bool {
	return vocabularyTags[name]
}
