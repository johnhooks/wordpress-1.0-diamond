package parse

import "fmt"

// Expr is a node in the expression AST. Expressions appear inside
// template delimiters: {post.Title}, {if count > 0}, {each posts as post}.
type Expr interface {
	exprNode()
	String() string
}

// Ident is a bare identifier: name, post, commentsOpen.
type Ident struct {
	Name string
}

func (*Ident) exprNode() {}
func (e *Ident) String() string { return e.Name }

// MemberExpr is a dot-access path: post.Title, post.Author.Name.
// Object is the left side, Field is the rightmost field name.
// post.Author.Name is MemberExpr{Object: MemberExpr{Object: Ident{post}, Field: Author}, Field: Name}.
type MemberExpr struct {
	Object Expr
	Field  string
}

func (*MemberExpr) exprNode() {}
func (e *MemberExpr) String() string { return fmt.Sprintf("%s.%s", e.Object, e.Field) }

// StringLit is a quoted string: "hello", "post".
type StringLit struct {
	Value string
}

func (*StringLit) exprNode() {}
func (e *StringLit) String() string { return fmt.Sprintf("%q", e.Value) }

// NumberLit is a numeric literal: 0, 42, 3.14.
type NumberLit struct {
	Value float64
}

func (*NumberLit) exprNode() {}
func (e *NumberLit) String() string { return fmt.Sprintf("%g", e.Value) }

// BoolLit is true or false.
type BoolLit struct {
	Value bool
}

func (*BoolLit) exprNode() {}
func (e *BoolLit) String() string {
	if e.Value {
		return "true"
	}
	return "false"
}

// BinaryExpr is a comparison or logical operation: a == b, count > 0, a and b.
type BinaryExpr struct {
	Left  Expr
	Op    string // ==, !=, <, >, <=, >=, and, or
	Right Expr
}

func (*BinaryExpr) exprNode() {}
func (e *BinaryExpr) String() string { return fmt.Sprintf("%s %s %s", e.Left, e.Op, e.Right) }

// UnaryExpr is a prefix operation: not commentsOpen.
type UnaryExpr struct {
	Op      string // not
	Operand Expr
}

func (*UnaryExpr) exprNode() {}
func (e *UnaryExpr) String() string { return fmt.Sprintf("%s %s", e.Op, e.Operand) }
