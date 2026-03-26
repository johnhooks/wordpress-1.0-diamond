// Package parse is a fork of golang.org/x/net/html.
//
// Upstream: golang.org/x/net/html
// Version:  v0.52.0
// Copied:   2026-03-21
//
// Modifications (search for "PRESS PATCH"):
//
// token.go (4 sites):
//   - Added TemplateToken to TokenType constants and String().
//   - Added Parts field to Attribute struct for pre-compiled
//     attribute expressions. Changed Attribute literal in Token()
//     to use named fields.
//   - In Next() main loop: when '{' is encountered, delegate to
//     readTemplateToken() in template_token.go. Propagate error
//     on unclosed brace.
//   - In Token(): extract data for TemplateToken.
//   - In Text(): return raw content for TemplateToken.
//
// parse.go (2 sites):
//   - In parseCurrentToken(): classify TemplateToken and create the
//     appropriate node type (IfNode, EachNode, SnippetNode, etc.)
//     with parsed TemplateData.
//   - In parseCurrentToken(): respect self-closing on vocabulary tags.
//
// render.go (1 site):
//   - In render1 switch: handle ExpressionNode via RenderExpression()
//     and block node types via RenderTemplateBlock().
//
// node.go (1 site):
//   - Added TemplateData field to Node struct for parsed template
//     metadata. Also copied in clone().
//
// All custom logic lives in separate files:
//   - template_token.go:  readTemplateToken(), ClassifyTemplate(), binding parsers
//   - template_nodes.go:  template NodeType constants and TemplateData struct
//   - template_render.go: RenderExpression(), RenderTemplateBlock()
//   - template_parse.go:  ParseTemplate() wrapper, block restructuring
//   - expr.go:            expression AST types
//   - expr_parse.go:      recursive descent expression parser
//   - selfclose.go:       vocabulary tag lookup
//
// To update from upstream:
//  1. Copy new source files into this directory.
//  2. Change package name from html to parse.
//  3. Reapply patches marked with "PRESS PATCH".
//  4. Run tests.
package parse
