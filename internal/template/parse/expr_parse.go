package parse

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxExprDepth is the maximum nesting depth for expressions.
// Prevents stack overflow from pathologically nested input like (((((...))))).
const maxExprDepth = 32

// ParseExpr parses an expression string into an Expr AST node.
// The input is the raw content between { and }, with no braces.
func ParseExpr(input string) (Expr, error) {
	p := &exprParser{input: input, pos: 0}
	p.skipWhitespace()
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected character %q at position %d", p.peekRune(), p.pos)
	}
	return expr, nil
}

type exprParser struct {
	input string
	pos   int
	depth int
}

func (p *exprParser) enter() error {
	p.depth++
	if p.depth > maxExprDepth {
		return fmt.Errorf("expression nesting too deep (max %d)", maxExprDepth)
	}
	return nil
}

func (p *exprParser) leave() {
	p.depth--
}

func (p *exprParser) skipWhitespace() {
	for p.pos < len(p.input) {
		r, size := utf8.DecodeRuneInString(p.input[p.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		p.pos += size
	}
}

func (p *exprParser) peekRune() rune {
	if p.pos >= len(p.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return r
}

func (p *exprParser) peekString(n int) string {
	end := p.pos + n
	if end > len(p.input) {
		end = len(p.input)
	}
	return p.input[p.pos:end]
}

func (p *exprParser) advanceRune() rune {
	r, size := utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += size
	return r
}

// matchKeyword checks if the input at the current position matches a keyword
// followed by a non-identifier character (whitespace, operator, EOF).
// If matched, advances past the keyword and returns true.
func (p *exprParser) matchKeyword(kw string) bool {
	if p.pos+len(kw) > len(p.input) {
		return false
	}
	if p.input[p.pos:p.pos+len(kw)] != kw {
		return false
	}
	// Ensure the keyword is not a prefix of a longer identifier.
	after := p.pos + len(kw)
	if after < len(p.input) {
		r, _ := utf8.DecodeRuneInString(p.input[after:])
		if isIdentPart(r) {
			return false
		}
	}
	p.pos += len(kw)
	return true
}

// parseOr handles: expr || expr, expr or expr
func (p *exprParser) parseOr() (Expr, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipWhitespace()
		if p.peekString(2) == "||" {
			p.pos += 2
			p.skipWhitespace()
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{Left: left, Op: "or", Right: right}
		} else if p.matchKeyword("or") {
			p.skipWhitespace()
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{Left: left, Op: "or", Right: right}
		} else {
			return left, nil
		}
	}
}

// parseAnd handles: expr && expr, expr and expr
func (p *exprParser) parseAnd() (Expr, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for {
		p.skipWhitespace()
		if p.peekString(2) == "&&" {
			p.pos += 2
			p.skipWhitespace()
			right, err := p.parseComparison()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{Left: left, Op: "and", Right: right}
		} else if p.matchKeyword("and") {
			p.skipWhitespace()
			right, err := p.parseComparison()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{Left: left, Op: "and", Right: right}
		} else {
			return left, nil
		}
	}
}

// parseComparison handles: expr ==|!=|<|>|<=|>= expr
func (p *exprParser) parseComparison() (Expr, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()

	var op string
	switch s := p.peekString(2); {
	case s == "==":
		op = "=="
	case s == "!=":
		op = "!="
	case s == "<=":
		op = "<="
	case s == ">=":
		op = ">="
	case len(s) > 0 && s[0] == '<':
		op = "<"
	case len(s) > 0 && s[0] == '>':
		op = ">"
	default:
		return left, nil
	}

	p.pos += len(op)
	p.skipWhitespace()
	right, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return &BinaryExpr{Left: left, Op: op, Right: right}, nil
}

// parseUnary handles: !expr, not expr
func (p *exprParser) parseUnary() (Expr, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	p.skipWhitespace()
	if p.peekRune() == '!' && p.peekString(2) != "!=" {
		p.advanceRune()
		p.skipWhitespace()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "not", Operand: operand}, nil
	}
	if p.matchKeyword("not") {
		p.skipWhitespace()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "not", Operand: operand}, nil
	}
	return p.parsePrimary()
}

// parsePrimary handles: identifier, member expression, string, number, bool, parenthesized.
func (p *exprParser) parsePrimary() (Expr, error) {
	p.skipWhitespace()
	c := p.peekRune()

	switch {
	case c == '"':
		return p.parseString()
	case c == '(':
		return p.parseGrouped()
	case isDigit(c):
		return p.parseNumber()
	case isIdentStart(c):
		return p.parseIdentOrMember()
	case c == 0:
		return nil, fmt.Errorf("unexpected end of expression")
	default:
		return nil, fmt.Errorf("unexpected character %q at position %d", c, p.pos)
	}
}

func (p *exprParser) parseString() (Expr, error) {
	p.advanceRune() // consume opening "
	var sb strings.Builder
	for p.pos < len(p.input) {
		r := p.advanceRune()
		if r == '\\' && p.pos < len(p.input) {
			next := p.advanceRune()
			switch next {
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(next)
			}
			continue
		}
		if r == '"' {
			return &StringLit{Value: sb.String()}, nil
		}
		sb.WriteRune(r)
	}
	return nil, fmt.Errorf("unterminated string literal")
}

func (p *exprParser) parseGrouped() (Expr, error) {
	p.advanceRune() // consume (
	p.skipWhitespace()
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.peekRune() != ')' {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}
	p.advanceRune()
	return expr, nil
}

func (p *exprParser) parseNumber() (Expr, error) {
	start := p.pos
	seenDot := false
	for p.pos < len(p.input) {
		r, size := utf8.DecodeRuneInString(p.input[p.pos:])
		if r == '.' {
			if seenDot {
				break // second dot ends the number
			}
			seenDot = true
			p.pos += size
			continue
		}
		if !isDigit(r) {
			break
		}
		p.pos += size
	}
	val, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q: %w", p.input[start:p.pos], err)
	}
	return &NumberLit{Value: val}, nil
}

func (p *exprParser) parseIdentOrMember() (Expr, error) {
	name := p.readIdent()

	// Check for boolean literals.
	if name == "true" {
		return &BoolLit{Value: true}, nil
	}
	if name == "false" {
		return &BoolLit{Value: false}, nil
	}

	var expr Expr = &Ident{Name: name}

	// Read dot-separated member accesses.
	for p.peekRune() == '.' {
		p.advanceRune() // consume .
		if !isIdentStart(p.peekRune()) {
			return nil, fmt.Errorf("expected field name after '.' at position %d", p.pos)
		}
		field := p.readIdent()
		expr = &MemberExpr{Object: expr, Field: field}
	}
	return expr, nil
}

func (p *exprParser) readIdent() string {
	start := p.pos
	for p.pos < len(p.input) {
		r, size := utf8.DecodeRuneInString(p.input[p.pos:])
		if !isIdentPart(r) {
			break
		}
		p.pos += size
	}
	return p.input[start:p.pos]
}

func isIdentStart(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentPart(c rune) bool {
	return isIdentStart(c) || isDigit(c)
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}
