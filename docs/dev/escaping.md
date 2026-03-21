# Escaping and Content Safety

How Press prevents user-supplied content from becoming a security
risk. This document covers what we escape, where we escape it, and
why.

---

## The Problem

A blog accepts content from users: post titles, post bodies, comment
text, comment author names, URLs. Any of these can contain characters
that, if inserted into HTML without care, change the meaning of the
page. The classic example is a comment author who enters
`<script>alert('xss')</script>` as their name. If that string is
inserted into the page verbatim, the browser executes the script.

The challenge is that some user content is legitimately HTML. Post
bodies written in ProseMirror produce HTML output. That HTML needs
to reach the page intact. But a comment author name must never
contain HTML tags.

---

## How WordPress Does It

WordPress uses a two-layer approach:

### Layer 1: Filter on input (save time)

When content is saved, WordPress runs it through `wp_kses` (a
tag-stripping filter) based on the user's role and the content
type. Comments get aggressive filtering that strips nearly all
HTML. Post content for trusted authors is stored with minimal
filtering. The idea is: clean it before it reaches the database.

Key functions:
- `wp_kses($content, $allowed_html)` strips tags not in the
  allowed list and validates attributes.
- `wp_kses_post($data)` allows the post-content tag set (most
  HTML formatting tags, links, images, tables).
- `wp_filter_kses($data)` uses a restrictive tag set for
  comments and titles.
- `sanitize_text_field($str)` strips all HTML and normalizes
  whitespace for plain text fields like author names.

### Layer 2: Escape on output (display time)

When rendering a page, WordPress escapes values for the specific
context they appear in:

- `esc_html($text)` for HTML text content. Escapes `&`, `<`,
  `>`, `"`, `'`.
- `esc_attr($text)` for HTML attribute values. Same characters
  as esc_html.
- `esc_url($url)` for URLs. Validates protocol (blocks
  `javascript:`), encodes special characters, normalizes entities.
- `esc_js($text)` for inline JavaScript strings. Escapes for
  safe inclusion in JS string literals.

The philosophy: filter aggressively on the way in, escape
appropriately on the way out. Trust that the input layer did its
job, but escape anyway because defense in depth matters.

### What WordPress gets wrong

WordPress's input filtering depends on user role. Users with the
`unfiltered_html` capability (administrators, editors) can store
arbitrary HTML. This means the output layer must still handle
potentially dangerous content from the database, but `the_content()`
outputs post bodies with minimal escaping. It relies on the input
filter having already cleaned the content. If a trusted user's
account is compromised, the attacker can inject arbitrary HTML and
JavaScript into posts.

Comments rely entirely on input-time filtering. `comment_text()`
does not re-filter on output. If the input filter missed something
(or was bypassed), the comment is served as-is.

---

## How Go's html/template Does It

Go's `html/template` package provides contextual auto-escaping.
The template author writes `{{.Title}}` and the engine
automatically detects whether that value appears in HTML text, an
attribute, a URL, CSS, or JavaScript, and applies the right
escaping for that context.

This is better than manual escaping because:
- The developer does not choose an escaping function. The engine
  chooses based on where the value appears.
- Every value is escaped by default. You must explicitly opt out.
- Context switches (text to attribute to URL) are handled
  automatically.

The opt-out mechanism is typed values: `template.HTML`,
`template.URL`, `template.JS`, etc. Wrapping a string in
`template.HTML(s)` tells the engine "this is already safe HTML,
do not escape it."

### What Go's html/template gets wrong

The typed bypass values have no provenance tracking. Nothing
prevents `template.HTML(userInput)`. Once wrapped, escaping is
skipped entirely. This is a foot-gun that relies on developer
discipline.

The package also does not handle JavaScript template literals
(backtick strings), does not differentiate resource URLs
(`<script src>`) from navigation URLs (`<a href>`), and has
permissive handling of unknown elements and attributes.

---

## How Press Does It

Press has a structural advantage: we control the entire pipeline
from content creation to rendering. ProseMirror constrains what
users can write. The template engine constrains what themes can
render. The vocabulary tag handlers control how data reaches the
page. There is no plugin system, no `unfiltered_html` capability,
no filter hooks that third parties can modify.

### Principle: The engine escapes, not the view, not the template

The view structs are dumb data carriers. They hold plain strings,
numbers, booleans. They do not contain `template.HTML` or any
safety annotations. They do not escape their own fields.

The template syntax has one interpolation form: `{expression}`.
It is always escaped. There is no `{html}` or raw insertion
syntax in the template language. The theme author cannot insert
unescaped content.

The vocabulary tag handlers are Go functions compiled into the
engine. They produce HTML fragments. When a handler needs to
include pre-rendered content (like a ProseMirror post body), it
writes the HTML directly in its own output. The handler knows
what is safe because it produced or validated the content. The
template never sees the raw HTML.

### Where escaping happens

| Content | Created by | Escaping point | Method |
|---------|-----------|----------------|--------|
| Post title | User via editor | Walker expression eval | HTML entity escaping |
| Post body | ProseMirror JSON | Post vocabulary handler | Handler writes raw; ProseMirror schema constrains output |
| Comment text | User via form | Comment vocabulary handler | Currently escaped on store; will become ProseMirror with schema constraints |
| Comment author | User via form | Walker expression eval | HTML entity escaping |
| URLs (permalink, author) | Engine-generated | Walker attribute resolution | HTML entity escaping in attributes |
| Blog name, description | Admin settings | Walker expression eval | HTML entity escaping |

### What gets escaped and how

**HTML text content** (inside elements):
Characters `<`, `>`, `&`, `"`, `'` are replaced with HTML
entities. This is what `html.EscapeString` does in Go and what
`esc_html` does in WordPress. This prevents tag injection.

**HTML attributes** (inside quotes):
Same character set. The walker resolves expressions inside
attribute values and escapes the result. Attribute values are
always double-quoted in the output.

**URLs**:
URLs are generated by the engine (permalinks, category URLs,
pagination) or come from admin settings. They are not
user-supplied in the template context. The walker escapes them
as attribute values. Protocol validation (blocking `javascript:`)
is not currently implemented but should be added for any URL
that originates from user input (comment author URL).

### What does NOT get escaped

**Vocabulary tag handler output**: Handlers return HTML fragments
that are inserted into the page without escaping. This is safe
because the handler is compiled Go code that we wrote. It is
responsible for escaping any user data it includes in its output.

This is the single point where raw HTML enters the page. It is
explicit, auditable, and contained within handler functions.

---

## ProseMirror as a Safety Layer

Post content does not go through the traditional
escape-on-output path because it is never free-form HTML. The
ProseMirror editor enforces a schema that defines exactly which
node types and marks are valid. The serializer converts the
document JSON to HTML using only the elements defined in the
schema.

A user cannot inject `<script>` tags because `script` is not a
node type in the schema. They cannot add arbitrary attributes
because the serializer only writes attributes defined per node
type. The safety comes from the schema constraint at the
structural level, combined with escaping at the text level. The
Go serializer calls `html.EscapeString` on text node content
and on attribute values (href, title, alt, src, etc.), so even
within the allowed schema, user-typed strings like `<b>` in a
paragraph render as literal text, not as HTML tags.

When comments move to ProseMirror (planned), the same principle
applies. The comment schema will be more restrictive than the
post schema (plain text, maybe links and basic formatting), and
the serializer enforces those constraints.

Until comments use ProseMirror, comment content is escaped with
`html.EscapeString` before storage. This is the stopgap. It
means comments are plain text only; any HTML is visible as
literal angle brackets.

---

## Contexts We Must Handle

### 1. HTML text content

Where `{post.Title}` appears between tags: `<h1>{post.Title}</h1>`.
The walker calls `html.EscapeString` on the evaluated string.

### 2. HTML attributes

Where expressions appear in attribute values:
`<a href="{post.Permalink}">`. The walker's `ResolveAttrExprs`
finds `{...}` patterns, evaluates them, and escapes the result
with `html.EscapeString`. Static text in attributes passes
through unmodified (it comes from the theme author, who is
trusted).

### 3. Pre-rendered HTML (vocabulary handler output)

Where a handler returns an HTML fragment containing a ProseMirror
post body or a form with CSRF tokens. The handler is responsible
for safety. The walker inserts the output without escaping.

### 4. URLs from user input

Comment author URLs are the main case. These should be validated
for protocol (allow `http`, `https`, `mailto`; block `javascript`,
`data`). This validation belongs in the comment handler or the
input validation layer, not the template engine.

---

## What We Do Not Need

**kses-style HTML filtering.** WordPress needs this because users
type HTML directly and it must be sanitized. Press users write in
ProseMirror, which enforces structure at the editor level. There is
no free-form HTML input to filter.

**Context-aware auto-escaping (Go html/template style).** The
template engine has one text context (HTML) and one attribute
context. There is no inline JavaScript or CSS in templates
(both are rejected at parse time). Two escaping paths (text and
attribute) cover all cases. The complexity of full contextual
escaping is not needed.

**template.HTML or equivalent typed bypass.** The template
language does not have a raw insertion mode. Pre-rendered HTML
enters the page through vocabulary tag handlers, not through
template expressions. No bypass type is needed in the view
contract.

---

## Rules for Contributors

1. **View structs hold plain values.** No `template.HTML`, no
   pre-escaped strings, no safety annotations. A `string` is a
   `string`.

2. **Template expressions are always escaped.** The walker
   escapes every `{expression}` in text and attributes. No
   exceptions, no opt-out.

3. **Vocabulary handlers own their output.** If a handler writes
   raw HTML, the handler is responsible for ensuring that HTML is
   safe. This responsibility is documented in the handler, not
   inferred from types.

4. **ProseMirror schemas are the input constraint.** Do not
   sanitize ProseMirror output after serialization. If the schema
   allows it, the serializer produces it, and it is safe by
   construction. If you need to restrict output, restrict the
   schema.

5. **Escape at the boundary, not in the middle.** Do not escape
   values as they flow through the system. Escape once, at the
   point of insertion into HTML. Double-escaping produces
   `&amp;amp;` and breaks the display.

6. **Validate URLs from user input.** Any URL that originates
   from user input (comment author URL, custom link fields) must
   be validated for protocol before reaching the page. Allow
   `http`, `https`, `mailto`. Block everything else.

7. **Sanitize admin input for display correctness.** Admin
   settings (blog name, description) are escaped on output by
   the walker, so they cannot cause XSS. But storing HTML in a
   blog name would display as literal angle brackets, which is
   ugly. The admin panel should strip HTML from plain text
   settings on input for cleanliness, not for security.
