# Template

How to write Press theme templates. This document covers the template syntax,
expressions, control flow, escaping, and the rules that govern how values
behave.

Press templates are HTML files with single-brace expressions for dynamic
content. The syntax was inspired by Svelte but simplified for server rendering.
There is no client-side reactivity, no two-way binding, no component lifecycle.
Templates describe what the page looks like. The engine fills in the data.

---

## Expressions

Single braces interpolate a value into the page.

```html
<h1>{blogName}</h1>
<a href="{post.Permalink}">{post.TheTitle}</a>
<time>{post.TheDate}</time>
```

In text content, braces insert the value at that position. Inside attribute
values, braces appear within the quotes and are resolved at render time. Both
positions are auto-escaped for HTML safety. You cannot insert raw HTML through
an expression.

### Dot paths

Values come from the page data the engine provides. A bare name looks up a
top-level field. A dotted path walks into nested structures.

```
blogName                top-level field
post.TheTitle           field on a nested object
post.Author.Name        deeper nesting
```

What fields are available depends on which page template you are writing. A
single post page has `post`. An index page has `posts`. The engine documentation
lists the fields for each page type.

### Literals

Expressions can contain literal values. These are most useful in comparisons and
as fallback values.

```
"hello"                 string
42                      integer
3.14                    float
true / false            boolean
```

---

## Truthiness

Conditionals and logical operators evaluate values as truthy or falsy. The rules
are simple and consistent:

| Value              | Truthy? |
| ------------------ | ------- |
| `""` empty string  | no      |
| `0`                | no      |
| `false`            | no      |
| `nil` / missing    | no      |
| empty slice or map | no      |
| everything else    | yes     |

This means you can write `{if post.Comments}` to mean "if there are comments"
and `{if post.EditURL}` to mean "if there is an edit URL." No helper functions
needed. The template language trusts the data types that the engine provides.

---

## Operators

The expression language supports comparison, logical combination, and negation.
Nothing more. There is no arithmetic, no function calls, no ternary operator. If
a template needs complex logic, that logic belongs in the engine, exposed as a
value the template can check.

### Comparison

```html
{if post.CommentCount > 0}
{if post.Type == "post"}
{if post.CommentCount != 5}
{if post.CommentCount <= 100}
```

Operators: `==`, `!=`, `<`, `>`, `<=`, `>=`. Comparisons always return a
boolean. Numbers are compared numerically (an integer and a float with the same
value are equal). Strings are compared lexicographically. Comparing incompatible
types is an error.

### Negation

```html
{if not post.EditURL}
{if not commentsOpen}
```

`not` inverts the truthiness of a value. Always returns a boolean. The symbol
form `!` is also accepted (`{if !commentsOpen}`) but `not` is preferred in
templates for readability.

### Logical combination

```html
{if commentsOpen and post.CommentCount > 0}
{if post.EditURL or post.TrashURL}
{if loggedIn and commentsOpen and post.CommentCount > 0}
```

`and` and `or` combine conditions. The symbol forms `&&` and `||` are accepted
but the keyword forms are preferred.

Both operators short-circuit and return values, not booleans:

- **`and`**: evaluates the left side. If falsy, returns it without evaluating
  the right side. If truthy, evaluates and returns the right side.
- **`or`**: evaluates the left side. If truthy, returns it without evaluating
  the right side. If falsy, evaluates and returns the right side.

Short-circuiting means `{if obj and obj.Field}` is safe. When `obj` is nil, the
right side is never evaluated, so no error occurs.

The value-returning behavior means `{name or "Anonymous"}` outputs the name when
present, or the string "Anonymous" when the name is falsy. In practice this
matters less than the short-circuiting, but it means logical operators compose
naturally.

### Grouping

Parentheses override precedence when needed.

```html
{if (a or b) and c}
```

Without the parentheses, `and` binds tighter than `or`, so `a or b and c` means
`a or (b and c)`.

### Precedence

From lowest to highest:

| Priority | Operator                     | Behavior                     |
| -------- | ---------------------------- | ---------------------------- |
| 1        | `or` (also `\|\|`)           | short-circuit, returns value |
| 2        | `and` (also `&&`)            | short-circuit, returns value |
| 3        | `==` `!=` `<` `>` `<=` `>=` | always returns boolean       |
| 4        | `not` (also `!`)             | always returns boolean       |
| 5        | `.` member access            | returns the field value      |
| 6        | literals, `()`               | values and grouping          |

In `a or b and c`, the `and` binds first: `a or (b and c)`. In `not a == b`, the
`not` binds first: `(not a) == b`. If you want to negate a comparison, use
parentheses: `not (a == b)`.

Comparisons do not chain. `a < b < c` is a parse error. Write `a < b and b < c`
instead.

### Reserved words

`if`, `else`, `each`, `const`, `true`, `false`, `and`, `or`, `not` are reserved
and cannot be used as identifiers in expressions.

---

## Conditionals

`{if}` opens a conditional block. `{else}` provides the alternate branch.
`{/if}` closes it.

```html
{if commentsOpen}
<comment-form />
{else}
<p>Comments are closed.</p>
{/if}
```

The else branch is optional.

```html
{if post.EditURL}
<a href="{post.EditURL}">Edit</a>
{/if}
```

Conditions use the truthiness rules above. Any expression that produces a value
can be a condition: a bare identifier, a dot path, a comparison, a logical
combination.

Blocks nest freely.

```html
{if commentsOpen}
  {if post.Comments}
    <comment-list />
  {/if}
  <comment-form />
{/if}
```

---

## Iteration

`{each}` iterates over a list. The `as` clause binds each element to a name. An
optional second name provides the zero-based index.

```html
{each posts as post}
<h2><a href="{post.Permalink}">{post.TheTitle}</a></h2>
{/each}
```

```html
{each comments as comment, index}
<li>{comment.TheAuthor}: {comment.TheContent}</li>
{/each}
```

### Empty lists

`{else}` inside an `{each}` block renders when the list is empty.

```html
{each posts as post}
<h2>{post.TheTitle}</h2>
{else}
<p>No posts found.</p>
{/each}
```

The `{else}` branch is optional. Without it, an empty list produces no output.

### Scope

The loop binding (`post`, `comment`, `index`) is only visible inside the
`{each}` block. It does not leak to siblings or the parent scope. If a binding
shadows a name from the outer scope, the outer value is restored when the block
closes.

---

## Local constants

`{const}` binds a name to a computed value within the current block. The value
is evaluated once, when the `{const}` is reached.

```html
{const hasComments = post.CommentCount > 0}
{if hasComments}
  <p>{post.CommentCount} comments</p>
{/if}
```

Constants are scoped to their containing block. A `{const}` inside an `{each}`
does not leak to the outer scope.

---

## Escaping

All expression output is HTML-escaped. There are no exceptions and no opt-out
mechanism in the template language.

```html
<!-- If post.TheTitle is: A <bold> claim -->
<h1>{post.TheTitle}</h1>
<!-- Renders: <h1>A &lt;bold&gt; claim</h1> -->
```

Attribute values are also escaped.

```html
<!-- If post.TheTitle is: Say "hello" -->
<meta name="title" content="{post.TheTitle}" />
<!-- Renders: <meta name="title" content="Say &#34;hello&#34;"> -->
```

Pre-rendered HTML content (like post bodies produced by the editor) is handled
by the engine's vocabulary tags, not by template expressions. When a `<post />`
tag renders the post body, the engine inserts pre-sanitized HTML directly. The
theme author does not need to think about which content is safe and which is
not. The engine handles it.

### Literal braces

Single braces start a template expression. If you need a literal `{` or `}` in
your HTML output, use the HTML entities `&#123;` and `&#125;`. In practice this
almost never comes up. Blog content that contains braces is already HTML-escaped
by the editor before it reaches the template engine.

---

## Named tags

Theme templates use named tags to place engine-provided content. These tags look
like HTML custom elements but never reach the browser. The engine replaces them
during rendering.

```html
<site-header />
<post />
<comment-list />
<comment-form />
<sidebar />
<site-footer />
```

A `<comment-form />` becomes a `<form>` with CSRF tokens, hidden fields, and
htmx attributes. A `<post />` becomes an `<article>` with the rendered post
content. The theme author places the tag. The engine decides what it contains.

Tags can carry attributes that configure their behavior.

```html
<comment-form post-id="{postID}" />
```

The engine defines which tags exist, which attributes they accept, and what they
produce. Unrecognized tag names with a hyphen are errors. The engine
documentation lists every tag and its attributes.

---

## What templates cannot do

- **No raw HTML insertion.** There is no way to output unescaped content from a
  template expression. Pre-rendered HTML is handled by the engine, not the
  template.
- **No function calls.** You cannot call functions from expressions. If you need
  computed values, the engine provides them as fields.
- **No arithmetic.** No `+`, `-`, `*`, `/`. Compute in Go, expose the result.
- **No assignment.** `{const}` binds a name once. There are no mutable
  variables.
- **No array indexing.** No `items[0]` or `items[i]`. If a template needs a
  specific item, the engine provides it as a named field.
- **No string concatenation.** Concatenation happens naturally in attributes
  (`class="post {post.Type}"`) and in HTML text. There is no concatenation
  operator.
- **No inline styles or scripts.** `<style>` and `<script>` tags in template
  files are rejected by the parser. Styles belong in CSS files. Scripts are not
  the theme's concern.

These constraints are intentional. A theme template should describe structure
and make simple decisions. Anything that requires real logic belongs in Go code,
where it can be tested, typed, and reasoned about.
