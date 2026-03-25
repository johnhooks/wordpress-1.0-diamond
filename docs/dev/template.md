# Template Engine Internals

How the template engine works, for developers working on the engine itself. If
you are writing a theme, read `docs/template.md` instead.

---

## Architecture

The template engine is three layers:

```
parse (internal/template/parse/)
  Tokenizer → Parser → AST → Block restructuring → Attribute compilation → Validation

eval (internal/template/)
  Expression evaluator, truthiness, scope chain

walk (internal/template/)
  Tree walker that renders the AST to HTML
```

A template file goes through `ParseTemplate`, which runs the full pipeline:
HTML5 tokenization, tree construction, block restructuring, attribute expression
compilation, and validation. The result is an AST ready for the walker.

The walker traverses the AST, evaluates expressions against a scope chain, calls
vocabulary tag handlers, and writes HTML to an `io.Writer`.

---

## The Parser Fork

The parser is a fork of `golang.org/x/net/html` v0.52.0, copied into
`internal/template/parse/`. We fork the entire package because the upstream code
uses unexported symbols across files. Partial import is not possible.

All modifications to upstream files are marked with `// PRESS PATCH:` comments.
The fork is documented in `doc.go` with every patch site listed and an upgrade
procedure.

### Why fork

Two behaviors needed to change:

1. **Self-closing tags on non-void elements.** The HTML5 spec defines a fixed
   list of void elements (`br`, `img`, `input`, etc.) that allow self-closing.
   `<comment-form />` would parse as an opening tag with no close, swallowing
   siblings. Our vocabulary tags need self-closing support. The patch checks
   `selfclose.go` for known vocabulary tags.

2. **Template expression tokenization.** The upstream tokenizer does not
   recognize `{` as a delimiter. The patch intercepts `{` in text context and
   delegates to `readTemplateToken()` in `template_token.go`, which reads
   balanced braces (respecting quotes and nesting) and emits a `TemplateToken`.

### Patch sites

**token.go** (4 sites): `TemplateToken` constant, `Attribute.Parts` field, `{`
interception in `Next()`, `TemplateToken` cases in `Token()` and `Text()`.

**parse.go** (2 sites): `TemplateToken` handling in `parseCurrentToken()`,
self-closing for vocabulary tags.

**render.go** (1 site): template node rendering delegation.

**node.go** (1 site): `TemplateData` field on `Node`, copied in `clone()`.

### Custom files

Everything else lives in separate files that do not touch upstream code:

| File                 | Purpose                                       |
| -------------------- | --------------------------------------------- |
| `template_token.go`  | `readTemplateToken()`, `ClassifyTemplate()`   |
| `template_nodes.go`  | Template `NodeType` constants, `TemplateData` |
| `template_parse.go`  | `ParseTemplate()`, block restructuring        |
| `template_render.go` | `RenderExpression()`, block rendering         |
| `template_print.go`  | Debug printing (`Sprint`, etc.)               |
| `expr.go`            | Expression AST types                          |
| `expr_parse.go`      | Recursive descent expression parser           |
| `selfclose.go`       | Vocabulary tag set                            |

---

## Template Token Lifecycle

A template expression like `{post.TheTitle}` goes through several stages:

1. **Tokenization.** The tokenizer reads `{`, delegates to
   `readTemplateToken()`, which scans to the matching `}` and emits a
   `TemplateToken` with `Data = "post.TheTitle"`.

2. **Classification.** `parseCurrentToken()` calls `ClassifyTemplate(data)`
   which returns the token kind (`ExprToken`, `IfOpenToken`, `EachOpenToken`,
   etc.) and the remaining content after the keyword.

3. **Expression parsing.** For tokens that contain expressions (`ExprToken`,
   `IfOpenToken`, `EachOpenToken`, `ConstToken`), the content is passed to
   `ParseExpr()` which returns an `Expr` AST node.

4. **Tree insertion.** The parser creates a `Node` with the appropriate type
   (`ExpressionNode`, `IfNode`, etc.) and attaches the parsed expression in the
   `TemplateData` field.

5. **Block restructuring.** `restructureBlocks()` converts the flat sequence of
   block markers into a nested tree. Before:
   `[IfNode, <p>, ElseNode, <p>, ifCloseNode]`. After:
   `[IfNode → [<p>, ElseNode → [<p>]]]`. Close markers are consumed and removed.

6. **Attribute compilation.** `compileAttrExprs()` walks the tree and
   pre-compiles `{expr}` patterns inside attribute values into `AttrPart` slices
   on the `Attribute.Parts` field.

7. **Validation.** `validatePass()` rejects `<script>` and `<style>` elements.

---

## Expression Language

The expression parser is a recursive descent parser in `expr_parse.go`. The
grammar:

```
expr       = or_expr
or_expr    = and_expr ( ("or" | "||") and_expr )*
and_expr   = cmp_expr ( ("and" | "&&") cmp_expr )*
cmp_expr   = unary_expr ( ("==" | "!=" | "<" | ">" | "<=" | ">=") unary_expr )?
unary_expr = ("not" | "!") unary_expr | primary
primary    = ident ("." ident)* | string_lit | number_lit | bool_lit | "(" expr ")"
```

Precedence from lowest to highest:

| Level | Operators                   | Associativity |
| ----- | --------------------------- | ------------- |
| 1     | `or` (also `\|\|`)          | left          |
| 2     | `and` (also `&&`)           | left          |
| 3     | `==` `!=` `<` `>` `<=` `>=` | none          |
| 4     | `not` (also `!`)            | right (unary) |
| 5     | `.` member access           | left          |
| 6     | `(` `)` grouping            | n/a           |

Parentheses override precedence. `{if (a or b) and c}` evaluates the `or` first.
The parser enforces a maximum nesting depth of 32 to prevent stack overflow.

Comparisons do not chain. `a < b < c` is a parse error because after parsing
`a < b` as a `BinaryExpr`, the parser returns to `parseAnd` which does not
recognize `<`.

Keywords (`and`, `or`, `not`) are matched with boundary checking via
`matchKeyword()` so that `android`, `order`, `nothing` parse as identifiers.

### AST node types

| Type         | Fields                     | Example         |
| ------------ | -------------------------- | --------------- |
| `Ident`      | `Name`                     | `blogName`      |
| `MemberExpr` | `Object`, `Field`          | `post.Title`    |
| `StringLit`  | `Value`                    | `"hello"`       |
| `NumberLit`  | `Value` (int64 or float64) | `42`, `3.14`    |
| `BoolLit`    | `Value`                    | `true`, `false` |
| `BinaryExpr` | `Left`, `Op`, `Right`      | `a == b`        |
| `UnaryExpr`  | `Op`, `Operand`            | `not x`         |

All operator symbol forms (`&&`, `||`, `!`) normalize to keyword form (`and`,
`or`, `not`) in the AST.

### Depth limiting

`maxExprDepth = 32` prevents stack overflow from pathological nesting. Every
recursive descent function calls `enter()` which increments a depth counter and
returns an error if exceeded.

---

## Evaluation

The evaluator in `eval.go` walks an `Expr` tree against a data context and
produces a Go value.

### Value resolution

`Eval()` dispatches on the expression type:

- **Ident**: calls `lookupIdent()`, which checks the scope chain first, then
  falls through to the page data struct.
- **MemberExpr**: evaluates the object, then calls `lookupField()` on the
  result.
- **Literals**: return the Go value directly.
- **BinaryExpr**: calls `evalBinary()`.
- **UnaryExpr**: calls `evalUnary()`.

### Field lookup

`lookupField()` uses reflection. For a struct, it resolves exported fields via
`FieldByName`. For a map with string keys, it does a map index. Nil pointers
return `nil` without error. Methods are not callable from templates. If the
engine needs a computed value, it belongs in the view struct as a field. This
keeps templates from encountering errors at render time.

### Short-circuit evaluation

`and` and `or` short-circuit. The right side is only evaluated when the left
side does not determine the result.

```go
// and: return left if falsy, otherwise evaluate right
left, err := Eval(e.Left, ctx)
if !IsTruthy(left) { return left, nil }
return Eval(e.Right, ctx)

// or: return left if truthy, otherwise evaluate right
left, err := Eval(e.Left, ctx)
if IsTruthy(left) { return left, nil }
return Eval(e.Right, ctx)
```

Both return the deciding value, not a boolean. This makes
`{if obj and obj.Field}` safe (nil short-circuits) and `{name or "Anonymous"}`
return a string.

### Truthiness

`IsTruthy()` defines what is falsy:

- `nil`
- empty string
- `0` (any integer or float zero)
- `false`
- empty slice or map
- nil pointer or interface

Everything else is truthy. Structs, non-empty strings, non-zero numbers, non-nil
pointers are all truthy.

### Comparison

Integer literals (`0`, `42`) are `int64`. Decimal literals (`3.14`) are
`float64`. Integers and floats do not mix: comparing an int to a float is a type
error, not a silent coercion. This prevents precision loss for large int64 values
(post IDs, timestamps) that cannot be represented exactly as float64.

`==` and `!=` try integer comparison first (`toInt`), then float comparison
(`toFloat`), then fall back to `reflect.DeepEqual`. Ordered comparisons (`<`,
`>`, `<=`, `>=`) follow the same order: integers, then floats, then strings.
Comparing incompatible types is an error.

---

## Scope Chain

The scope chain in `scope.go` is a linked list of variable maps. Each `{each}`
and `{const}` binding pushes a child scope. Lookup walks from innermost to
outermost, then falls through to the page data.

```
Root scope (page data: {BlogName: "Press", Post: {...}})
  └─ Each scope (post = <current item>)
       └─ Const scope (hasComments = true)
```

`NewScope(data)` creates the root. `scope.Push()` creates a child.
`scope.Set(name, val)` binds a variable. `scope.Lookup(name)` walks the chain.

The evaluator receives a `*Scope` as its context. `lookupIdent()` detects this
type and checks scope bindings before falling through to `lookupField()` on the
root data.

---

## Walker

The walker in `walker.go` traverses the AST and writes HTML. It holds a
reference to the writer, the scope chain, vocabulary tag handlers, and snippet
definitions.

### Node dispatch

`walkNode()` switches on `NodeType`:

| Node type        | Behavior                                                                                                       |
| ---------------- | -------------------------------------------------------------------------------------------------------------- |
| `DocumentNode`   | Walk children                                                                                                  |
| `TextNode`       | Write `html.EscapeString(n.Data)`                                                                              |
| `ElementNode`    | Write open tag, resolve attributes, walk children, write close tag. Vocabulary tags delegate to their handler. |
| `ExpressionNode` | Evaluate, format, escape, write                                                                                |
| `IfNode`         | Evaluate condition. Walk then-branch or else-branch.                                                           |
| `EachNode`       | Evaluate list. For each item: push scope, walk body. Empty list: walk else branch if present.                  |
| `ConstNode`      | Evaluate expression, bind in current scope                                                                     |
| `SnippetNode`    | Skip (extracted at compile time)                                                                               |
| `CommentNode`    | Write `<!--data-->`                                                                                            |
| `DoctypeNode`    | Write `<!DOCTYPE data>`                                                                                        |

### Vocabulary tags

When the walker encounters an `ElementNode` whose tag name matches a registered
handler, it resolves the element's attributes (evaluating any expression parts),
builds a `RenderContext`, and calls the handler. The handler returns an HTML
string that is written directly to the output without escaping. Handlers are
trusted Go code.

### Attribute resolution

Static attributes pass through as-is. Attributes with expression parts
(pre-compiled by `compileAttrExprs`) are evaluated via `EvalAttrParts()`, which
formats each expression value and HTML-escapes it before assembling the final
string.

### Escaping

Escaping happens at three points:

1. **Text nodes**: `html.EscapeString(n.Data)`.
2. **Expression output**: `html.EscapeString(formattedValue)`.
3. **Attribute expressions**: `html.EscapeString(formattedValue)` inside
   `EvalAttrParts`.

Vocabulary tag handler output is not escaped. Handlers produce pre-sanitized
HTML. This is the only path for inserting raw HTML into the output, and it is
only available to compiled Go code, not to template authors.

See `docs/dev/escaping.md` for the full escaping design.

---

## Template Node Types

Template node types start at 100 to avoid collision with upstream `NodeType`
values. Defined in `template_nodes.go`:

| Constant           | Value | Purpose                           |
| ------------------ | ----- | --------------------------------- |
| `ExpressionNode`   | 100   | Value interpolation: `{expr}`     |
| `IfNode`           | 101   | Conditional block root            |
| `ElseNode`         | 102   | Else branch (child of IfNode)     |
| `EachNode`         | 103   | Iteration block root              |
| `SnippetNode`      | 104   | Snippet definition                |
| `ConstNode`        | 105   | Local constant binding            |
| `ifCloseNode`      | 106   | Temporary close marker (consumed) |
| `eachCloseNode`    | 107   | Temporary close marker (consumed) |
| `snippetCloseNode` | 108   | Temporary close marker (consumed) |

Close markers are temporary. They exist in the flat tree after parsing and are
consumed by `restructureBlocks()`. They never appear in the final AST.

### TemplateData

Each template node carries a `TemplateData` struct with parsed metadata:

```go
type TemplateData struct {
    Expr          Expr   // parsed expression (ExpressionNode, IfNode, EachNode, ConstNode)
    Binding       string // "as" variable name (EachNode)
    IndexBinding  string // index variable name (EachNode)
    SnippetParams string // parameter list (SnippetNode)
    ConstName     string // variable name (ConstNode)
}
```

---

## Block Restructuring

The HTML5 parser produces a flat tree. Template blocks (`{if}...{/if}`,
`{each}...{/each}`) need to become nested parent-child structures.
`restructureBlocks()` in `template_parse.go` handles this.

The algorithm uses a stack of open block nodes. For each child of a parent node:

- **Block open** (`IfNode`, `EachNode`, `SnippetNode`): attach to the current
  target, push onto stack.
- **Block middle** (`ElseNode`): attach to the block on top of stack, push so
  subsequent nodes go into it.
- **Block close** (`ifCloseNode`, etc.): pop the stack back to the matching
  open. The close marker is consumed and discarded.
- **Everything else**: attach to the current target (top of stack, or the parent
  if stack is empty).

Error cases: unclosed blocks, mismatched closes, orphaned else nodes.

---

## Missing: Template Helpers

The expression language has no function calls. Some values cannot be
pre-computed in view structs because the theme controls the argument: date
formats, translation strings, pluralization. These need a small set of built-in
helper functions (`date`, `timeago`, `__`, `_n`, `sprintf`) that are pure, never
fail, and always return a string. See `docs/plans/template-helpers.md` for the
full design.

---

## Known Issues

See `docs/plans/.wip/issues.md` for the current issue list, including open items
in the template engine.
