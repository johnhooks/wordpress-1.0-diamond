package parse

import (
	"fmt"
	"strings"
)

// readTemplateToken reads a template expression that started with '{'.
// The opening '{' has already been read by the tokenizer. It reads until
// the matching '}' respecting brace nesting and quoted strings.
//
// On success it returns TemplateToken with the expression content in
// z.data (not including the braces). On failure it returns ErrorToken
// and the '{' is treated as literal text by the caller.
func (z *Tokenizer) readTemplateToken() TokenType {
	// Position of the '{' we already read.
	openPos := z.raw.end - 1

	// If there is accumulated text before the '{', emit it first.
	// The template token will be returned on the next call to Next().
	if openPos > z.raw.start {
		z.raw.end = openPos
		z.data.end = openPos
		z.tt = TextToken
		return z.tt
	}

	depth := 1
	inDoubleQuote := false
	inBacktick := false

	for {
		c := z.readByte()
		if z.err != nil {
			z.err = fmt.Errorf("unclosed template expression starting at byte %d", openPos)
			return ErrorToken
		}

		if inDoubleQuote {
			if c == '\\' {
				// Skip escaped character.
				z.readByte()
				continue
			}
			if c == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inBacktick {
			if c == '`' {
				inBacktick = false
			}
			continue
		}

		switch c {
		case '"':
			inDoubleQuote = true
		case '`':
			inBacktick = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				// Data spans the content between the braces.
				z.data.start = openPos + 1
				z.data.end = z.raw.end - 1
				z.tt = TemplateToken
				return z.tt
			}
		}
	}
}

// TemplateTokenKind classifies what kind of template construct a
// template token represents, based on its first characters.
type TemplateTokenKind int

const (
	ExprToken         TemplateTokenKind = iota // {expression}
	IfOpenToken                                // {if condition}
	ElseToken                                  // {else}
	IfCloseToken                               // {/if}
	EachOpenToken                              // {each list as item}
	EachCloseToken                             // {/each}
	SnippetOpenToken                           // {snippet name(params)}
	SnippetCloseToken                          // {/snippet}
	ConstToken                                 // {const name = expression}
)

// ClassifyTemplate determines the kind of template token and returns
// the remaining content after the keyword.
//
// Keywords are plain English words: if, else, each, snippet, const.
// Closing tags use a / prefix: /if, /each, /snippet.
// Everything else is an expression.
func ClassifyTemplate(data string) (TemplateTokenKind, string) {
	s := strings.TrimSpace(data)

	switch {
	case strings.HasPrefix(s, "if ") || s == "if":
		return IfOpenToken, strings.TrimSpace(strings.TrimPrefix(s, "if"))
	case s == "else" || strings.HasPrefix(s, "else "):
		return ElseToken, strings.TrimSpace(strings.TrimPrefix(s, "else"))
	case s == "/if":
		return IfCloseToken, ""
	case strings.HasPrefix(s, "each ") || s == "each":
		return EachOpenToken, strings.TrimSpace(strings.TrimPrefix(s, "each"))
	case s == "/each":
		return EachCloseToken, ""
	case strings.HasPrefix(s, "snippet ") || s == "snippet":
		return SnippetOpenToken, strings.TrimSpace(strings.TrimPrefix(s, "snippet"))
	case s == "/snippet":
		return SnippetCloseToken, ""
	case strings.HasPrefix(s, "const ") || s == "const":
		return ConstToken, strings.TrimSpace(strings.TrimPrefix(s, "const"))
	default:
		return ExprToken, s
	}
}

// ParseEachBinding parses "list as item" or "list as item, index" from
// the content of an {each} token. Returns the list expression string,
// the item binding name, and the optional index binding name.
func ParseEachBinding(content string) (listExpr string, binding string, indexBinding string) {
	parts := strings.SplitN(content, " as ", 2)
	if len(parts) != 2 {
		return content, "", ""
	}
	listExpr = strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	if i := strings.Index(rest, ","); i >= 0 {
		binding = strings.TrimSpace(rest[:i])
		indexBinding = strings.TrimSpace(rest[i+1:])
	} else {
		binding = rest
	}
	return
}

// ParseSnippetDecl parses "name(params)" from the content of a
// {snippet} token. Returns the snippet name and parameter string.
func ParseSnippetDecl(content string) (name string, params string) {
	if i := strings.Index(content, "("); i >= 0 {
		name = strings.TrimSpace(content[:i])
		end := strings.LastIndex(content, ")")
		if end > i {
			params = strings.TrimSpace(content[i+1 : end])
		}
	} else {
		name = strings.TrimSpace(content)
	}
	return
}

// ParseConstDecl parses "name = expression" from the content of a
// {const} token. Returns the variable name and expression string.
func ParseConstDecl(content string) (name string, exprStr string) {
	parts := strings.SplitN(content, "=", 2)
	if len(parts) != 2 {
		return content, ""
	}
	name = strings.TrimSpace(parts[0])
	exprStr = strings.TrimSpace(parts[1])
	return
}
