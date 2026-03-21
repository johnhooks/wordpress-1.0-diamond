package parse

// Template node types extend the upstream NodeType constants.
// Values start at 100 to avoid collision with upstream additions.
const (
	// ExpressionNode represents a value interpolation: {expression}.
	// Data holds the raw expression string. Parsed holds the parsed Expr.
	ExpressionNode NodeType = 100 + iota

	// IfNode is the root of an {if}...{else}...{/if} block.
	// Data holds the raw condition. Parsed holds the parsed Expr.
	// FirstChild is the "then" branch. The else branch is stored
	// as a sibling ElseNode.
	IfNode

	// ElseNode marks the {else} branch of an if block.
	// It is a child of the IfNode, following the then-branch children.
	ElseNode

	// EachNode is the root of an {each list as item}...{/each} block.
	// Data holds the raw expression. Binding and IndexBinding hold
	// the "as" variable names.
	EachNode

	// SnippetNode is a snippet definition: {snippet name(props)}...{/snippet}.
	// Data holds the snippet name.
	SnippetNode

	// ConstNode is a local constant: {const name = expression}.
	ConstNode

	// Close markers are temporary nodes inserted during parsing.
	// They are consumed by restructureBlocks and never appear in
	// the final tree.
	ifCloseNode
	eachCloseNode
	snippetCloseNode
)

// AttrPart is one segment of a pre-compiled attribute value.
// An attribute like href="/search?q={Query}" becomes three parts:
// {Text: "/search?q="}, {Expr: <compiled>}, {Text: ""}.
// Static-only attributes have no Parts (the walker uses Val directly).
type AttrPart struct {
	Text string // static text segment (used when Expr is nil)
	Expr Expr   // compiled expression (used when non-nil)
}

// TemplateData holds parsed template data attached to a Node
// via the TemplateData field on the Node struct.
type TemplateData struct {
	// Expr is the parsed expression for ExpressionNode, IfNode,
	// EachNode, and ConstNode.
	Expr Expr

	// Binding is the "as" variable name for EachNode (e.g., "post").
	Binding string

	// IndexBinding is the optional index variable for EachNode (e.g., "i").
	IndexBinding string

	// SnippetParams is the parameter list for SnippetNode (e.g., "props").
	SnippetParams string

	// ConstName is the variable name for ConstNode.
	ConstName string
}
