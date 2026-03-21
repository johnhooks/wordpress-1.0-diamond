# The Theme Engine

The theme engine is the runtime that connects theme templates to the
Press server. It parses theme source files into an AST, resolves
named components by walking and modifying the tree, and serializes
the final tree to HTML. The engine owns behavior. The theme owns
presentation. They meet at named tags.

The template syntax is inspired by Svelte. Single-brace expressions,
block control flow, snippet definitions, scoped styles, and component
scripts. We take the patterns that serve server-rendered HTML and
leave the client-side reactivity behind.

---

## The Problem with Go Templates

Go's `html/template` passes a single value (the dot) into each
template call. Every piece of data a nested template needs must be
threaded through that one value. This works for simple pages but falls
apart when composition goes bidirectional: the theme places a form,
the engine fills it with fields, each field calls back to the theme
for its visual rendering, and the theme's rendering needs data the
engine computed.

Threading all of that through dot contexts means either the view
structs become enormous bags of everything every nested template might
need, or the Go code pre-renders fragments into `template.HTML` and
passes opaque blobs around. Both approaches fight the template system
instead of working with it.

Press needs its own template language. Not because Go templates are
bad, but because the ownership model requires two-way composition that
Go templates were not designed for.

---

## Named Tags

Theme templates are HTML files. Engine components and theme fragments
are represented as custom HTML element tags. The engine knows its
complete vocabulary. Any tag name the engine recognizes is a component
it will handle. Everything else passes through as regular HTML.

```html
<!-- Theme source: single.html -->
<site-header />

<post />
<post-navigation />
<comment-list />
<comment-form />

<sidebar />
<site-footer />
```

These tags never reach the browser. The engine replaces them during
rendering with the HTML they represent. A `<comment-form />` becomes
a `<form>` with all the right attributes, hidden fields, and CSRF
tokens. A `<post />` becomes an `<article>` with the post content
rendered inside.

The engine's closed vocabulary is the boundary: if the engine knows
the tag name, it handles it. If not, it is HTML. We own this system.
There is no need for prefixes or namespaces. `<editor>` is an engine
tag. `<div>` is HTML. The vocabulary is the namespace.

---

## Template Syntax

The template language uses single braces for all dynamic content.
This is Svelte's syntax adapted for server rendering. No reactivity,
no event bindings, no async. Just expressions, conditionals, loops,
snippets, and raw HTML.

### Expressions

```html
<h1>{blogName}</h1>
<a href={post.Permalink}>{post.TheTitle}</a>
<time>{post.TheDate}</time>
```

Single braces interpolate a value. Inside attributes, braces replace
the attribute value. In text content, braces insert the value at that
position. Values are auto-escaped for HTML safety.

### Conditionals

```html
{#if commentsOpen}
  <comment-form />
{:else}
  <p>Comments are closed.</p>
{/if}

{#if post.EditURL}
  <a href={post.EditURL}>Edit</a>
{/if}
```

`{#if}` opens a conditional block. `{:else}` provides the alternate
branch. `{/if}` closes it. Blocks can nest.

### Iteration

```html
{#each posts as post}
  <post-row title={post.TheTitle} permalink={post.Permalink} />
{:else}
  <p>No posts found.</p>
{/each}

{#each comments as comment, index}
  <comment author={comment.TheAuthor} index={index} />
{/each}
```

`{#each}` iterates over a list. The `as` clause binds each element.
An optional second binding provides the index. `{:else}` renders when
the list is empty.

### Raw HTML

```html
<div class="entry">{@html post.TheContent}</div>
```

`{@html}` inserts pre-rendered HTML without escaping. Used for post
content that has already been rendered from ProseMirror JSON to HTML
on the server.

### Local Constants

```html
{#each posts as post}
  {@const hasComments = post.CommentCount > 0}
  {#if hasComments}
    <span>{post.CommentCount} comments</span>
  {/if}
{/each}
```

`{@const}` declares a value scoped to its containing block.

---

## Attributes as Props

Tags carry attributes. Attributes are the props the engine or theme
passes to the component.

```html
<!-- Engine tag with attributes -->
<comment-form post-id={postID} />

<!-- Theme fragment with attributes -->
<field name="author" label="Name" type="text" required />
```

String literals use quotes. Dynamic values use braces. Boolean
attributes (like `required`) are true when present. The engine
defines which attributes each tag accepts, their types, and whether
they are required or optional. An unrecognized attribute is an error.
A missing required attribute is an error.

---

## Two Kinds of Tags

**Engine tags**: the engine provides the content. The theme decides
where to place them.

```
comment-form    The engine writes the <form>, CSRF, htmx, hidden fields.
comment-list    The engine writes the loop over comments.
post            The engine writes the article with rendered content.
post-list       The engine writes the loop over posts.
pagination      The engine writes prev/next links.
search-form     The engine writes the search <form>.
category-list   The engine writes the category links.
archive-list    The engine writes the archive links.
page-list       The engine writes the page links.
meta-links      The engine writes login/logout/feed links.
sidebar         The engine writes the sidebar container.
editor          The engine writes the ProseMirror web component.
```

**Theme fragments**: the theme provides the content. The engine calls
them at the right point during its own rendering.

```
field           How a form field looks (label, input, error).
submit          How a submit button looks.
comment         How a single comment renders.
post-row        How a post appears in a list.
post-meta       How post metadata renders (date, author, categories).
site-header     The page header (title, description, navigation).
site-footer     The page footer (credits, links).
```

The engine calls theme fragments when it needs visual rendering. The
theme places engine tags when it needs behavior. The engine always
initiates: it walks the theme's page template, encounters engine
tags, fills them in, and calls theme fragments from within its own
rendering.

Theme fragments are never placed in page templates directly by the
theme author. They are defined and then called by the engine. The
parser only sees engine tags in page templates. Fragment definitions
live in their own files.

---

## Bidirectional Flow

The composition nests in both directions:

```
Theme: single.html
  places <comment-form />
    Engine: writes <form>, CSRF, hidden fields, htmx
      renders theme snippet "field" with {name: "author", label: "Name", ...}
        Theme: writes <p><label>Name<br><input ...></label></p>
      renders theme snippet "field" with {name: "email", label: "Email", ...}
        Theme: writes <p><label>Email<br><input ...></label></p>
      renders theme snippet "field" with {name: "comment", label: "Comment", ...}
        Theme: writes <p><label>Comment<br><textarea ...></textarea></label></p>
      renders theme snippet "submit" with {label: "Say it!"}
        Theme: writes <p><button type="submit">Say it!</button></p>
    Engine: writes </form>
```

Each level writes its own HTML and delegates what it does not own.
The engine never writes a `<label>`. The theme never writes an
`hx-post`. Neither knows or cares about the other's implementation.

---

## Snippets

Snippets are the theme's fragment definitions. They are inspired by
Svelte 5's snippet syntax. A snippet is a named, reusable template
that receives props and renders HTML.

### Defining snippets

```html
<!-- molecules/field.html -->
{#snippet field(props)}
  <p>
    <label>{props.Label}<br>
      {#if props.Type == "textarea"}
        <textarea name={props.Name} rows="4" {props.Attributes}>{props.Value}</textarea>
      {:else}
        <input type={props.Type} name={props.Name} value={props.Value} {props.Attributes}>
      {/if}
    </label>
  </p>
{/snippet}
```

```html
<!-- molecules/submit.html -->
{#snippet submit(props)}
  <p><button type="submit">{props.Label}</button></p>
{/snippet}
```

```html
<!-- molecules/comment.html -->
{#snippet comment(props)}
  <li id="comment-{props.ID}">
    <p>{@html props.TheContent}</p>
    <p><small>
      {#if props.URL}
        <a href={props.URL}>{props.TheAuthor}</a>
      {:else}
        {props.TheAuthor}
      {/if}
      — {props.TheDate}
    </small></p>
  </li>
{/snippet}
```

### Rendering snippets

The engine calls snippets from Go code during tree construction. When
the engine handles `<comment-form />`, it builds the form structure
as AST nodes and inserts rendered snippet subtrees for each field.
The snippet receives typed props from the engine and returns a
subtree that gets grafted into the document.

Snippets are the theme's creative surface. How a field looks, how a
comment renders, how the header is structured. The engine never
dictates visual rendering. Two themes can render the same props as
completely different HTML.

---

## Styles

Each component file can include a `<style>` block. Styles are scoped
to the component by default, following Svelte's scoping model.

```html
<!-- molecules/field.html -->
{#snippet field(props)}
  <p class="field">
    <label>{props.Label}<br>
      <input type={props.Type} name={props.Name} value={props.Value}>
    </label>
  </p>
{/snippet}

<style>
  .field {
    margin-bottom: 1em;
  }

  .field label {
    font-weight: bold;
  }

  .field input {
    width: 100%;
    padding: 0.25em;
  }
</style>
```

The engine scopes styles by adding a generated attribute to the
component's elements and rewriting the selectors to match. The
theme's `.field` selector only applies to `.field` elements rendered
by this snippet, not to any other `.field` in the page.

The engine collects all component styles and serves them as a single
stylesheet for the surface. No per-request CSS generation. The
stylesheet is built at parse time from the component files and cached.

A theme can also include a top-level `style.css` for global styles
that are not scoped to any component. This is the theme's baseline
typography, colors, and layout.

---

## Scripts

Each component file can include a `<script>` block for client-side
behavior that goes beyond what htmx provides.

```html
<!-- organisms/site-header.html -->
{#snippet site-header(props)}
  <header>
    <h1><a href="/">{props.SiteName}</a></h1>
    <p>{props.SiteDescription}</p>
    <button class="nav-toggle">Menu</button>
    <nav class="main-nav">
      <!-- navigation content -->
    </nav>
  </header>
{/snippet}

<script>
  const toggle = document.querySelector('.nav-toggle');
  const nav = document.querySelector('.main-nav');
  toggle?.addEventListener('click', () => {
    nav.classList.toggle('open');
  });
</script>
```

Theme scripts are for progressive enhancement: toggling a mobile
menu, animating a search reveal, initializing a third-party widget.
They do not handle form submissions, data loading, or page
transitions. That is the engine's job through htmx.

The engine collects all component scripts and includes them in the
page. Scripts run once on page load. Scripts that need to respond to
htmx swaps should listen for `htmx:afterSwap` on the document.

Scripts are optional. Most components will not need one. The engine
provides all interactivity through htmx attributes on the elements
it renders. A theme that uses zero JavaScript is a valid theme.

---

## Page Templates

A page template is the top-level file the engine renders for a route.
It is HTML with engine tags placed where the theme wants them.

```html
<!-- templates/single.html -->
<site-header />

<div class="content">
  <main>
    <post />
    <post-navigation />
    {#if commentsOpen}
      <comment-list />
      <comment-form />
    {:else}
      <p>Comments are closed.</p>
    {/if}
  </main>
  <aside>
    <sidebar />
  </aside>
</div>

<site-footer />
```

The theme controls the page structure: `<div>`, `<main>`, `<aside>`,
class names, wrapper elements, and conditionals around engine tags.
The engine fills in the named tags. A different theme might put the
sidebar before the content, wrap everything in a grid, or omit the
sidebar entirely. The engine does not care about the surrounding
structure.

---

## The AST

The engine works on a tree, not strings. Theme source files are
parsed into an AST. Engine tag handlers insert, remove, and replace
nodes in the tree. The final tree is serialized to HTML only at the
end, when the response is written.

```
Parse theme source → AST
Engine walks AST, finds engine tags
  Engine replaces <comment-form /> node with:
    ElementNode("form", {method: "post", action: "/comments", hx-post: ...})
      ElementNode("input", {type: "hidden", name: "_csrf", value: ...})
      ElementNode("input", {type: "hidden", name: "comment_post_ID", ...})
      [snippet "field" subtree for author]
      [snippet "field" subtree for email]
      [snippet "field" subtree for comment]
      [snippet "submit" subtree]
Full page is now a complete AST with no engine tags remaining
Serialize AST → HTML → response
```

Tree manipulation instead of string manipulation means:

- **htmx partials**: serialize a subtree instead of the full page.
  Every engine tag is a valid entry point for fragment rendering.
- **OOB swaps**: serialize the primary subtree, then serialize
  additional subtrees with an `hx-swap-oob` attribute injected into
  their root node. Node manipulation, not string injection.
- **Compilation**: once the patterns are stable, the AST walk can be
  compiled into Go functions that build the same tree structure
  directly, without parsing or walking at render time.
- **Testing**: assert on tree structure, not string matching.
- **Transformation**: the engine can modify any part of the tree
  before serialization. Add attributes, wrap elements, inject IDs
  for ARIA and htmx targeting.

---

## Parser Implementation

The parser is a fork of Go's `golang.org/x/net/html` package. We
copy `token.go` (the tokenizer) and `parse.go` (the tree builder)
into our own package and modify them. The rest of the html package
(node types, attributes, entity decoding, escaping, rendering) is
imported directly.

### Why fork

The stdlib HTML5 parser has two behaviors we need to override:

1. **Self-closing tags on non-void elements are silently ignored.**
   `<comment-form />` parses as `<comment-form>` (opening tag), and
   the parser looks for a closing tag, swallowing sibling elements as
   children. The HTML5 spec defines a fixed list of void elements
   (`br`, `hr`, `img`, `input`, etc.) that allow self-closing. Our
   engine vocabulary tags need the same treatment.

2. **No template expression awareness.** The tokenizer does not
   recognize `{` as a delimiter. Template expressions in text and
   attribute values need to be tokenized as distinct units so the
   parser can build expression nodes in the AST.

### What we modify

**`token.go`** (~1300 lines): Add recognition of `{` as a template
delimiter. When the tokenizer encounters `{` in text context, it
reads until the matching `}` and emits a template token. Same for
`{` in attribute values. Expressions (`{value}`), blocks
(`{#if}`, `{:else}`, `{/if}`, `{#each}`, `{/each}`), directives
(`{@html}`, `{@const}`), and snippet definitions
(`{#snippet}`, `{/snippet}`) each produce typed tokens.

**`parse.go`** (~2500 lines): Respect the self-closing flag for
elements in our known vocabulary. When the tree builder processes a
start tag for a vocabulary element that has the self-closing flag,
treat it like a void element instead of looking for children. Handle
template block tokens as structural nodes in the tree (if/else
blocks become branching nodes, each blocks become loop nodes, snippet
definitions become named subtrees).

### What we import unchanged

- `node.go` — `Node`, `NodeType`, `Attribute`, tree manipulation
- `entity.go` — HTML entity decoding tables
- `escape.go` — HTML escaping
- `render.go` — tree-to-HTML serialization
- `const.go` — void element list, other HTML constants

We extend the node types in our package to include expression nodes,
block nodes, and snippet definition nodes. These compose with the
stdlib `Node` type through the same parent/child/sibling links.

---

## The Closed Vocabulary

Press is not a framework. The set of tags is fixed:

**Engine tags** (~15):
comment-form, comment-list, post, post-list, post-navigation,
pagination, search-form, category-list, archive-list, page-list,
meta-links, sidebar, archive-header, editor

**Theme fragments** (~10):
field, submit, comment, post-row, post-meta, post-feedback,
site-header, site-footer

Every tag is documented, typed, and tested. The engine knows the
complete set. An unrecognized tag that contains a hyphen is an error
at parse time, not a silent pass-through. Standard HTML elements pass
through. Unknown custom elements fail validation.

---

## Language Server

The closed vocabulary makes a language server straightforward. The
engine knows every valid tag, every attribute, and every attribute
type. An LSP implementation provides:

**Autocomplete**: typing `<` in a template file offers the full tag
vocabulary. Typing an attribute name within a tag offers valid
attributes for that tag. Typing inside a snippet definition offers
the props available in that context.

**Diagnostics**: unknown tag names, missing required attributes,
wrong attribute types, unclosed blocks, unclosed snippets, duplicate
snippet definitions. All reported in real time as the theme author
types.

**Hover**: hovering over a tag name shows its documentation and prop
contract. Hovering over an attribute shows its type and description.
Hovering over a prop reference inside a snippet shows where the data
comes from.

**Go to definition**: clicking a tag name jumps to either the engine
handler (for engine tags) or the snippet file (for theme fragments).

The language server reads the same vocabulary definition the engine
uses at runtime. There is one source of truth for what tags exist,
what attributes they accept, and what props they provide. The LSP is
not a separate system that must be kept in sync. It reads the same
data.

---

## Compilation (Future)

The initial implementation interprets at runtime: parse on startup,
walk and modify the AST on each request, serialize to HTML. This is
fast enough for a blog and good for development because template
changes take effect on restart.

When the patterns are stable, the engine can compile the AST walk
into Go functions that build the tree directly without parsing or
walking. Each page becomes a generated function where every tag is
resolved, every snippet is inlined, and every expression is a Go
expression. The compiled output produces the same AST, then
serializes it. Or, as a further optimization, the compiler emits
functions that write HTML directly to an `io.Writer`, skipping the
tree entirely.

The interpreter and compiler share the same parser, the same
vocabulary, the same validation. The compiler is an optimization of
the interpreter, not a replacement. The interpreter remains available
for development. The compiler produces the production binary.

---

## Relationship to Current Code

The current templates are hand-written Go templates that produce what
the engine would produce. The comment form template has `hx-post`,
CSRF inputs, and hidden fields written by the theme author. In the
engine model, the theme author writes `<comment-form />` and the
engine produces all of that.

Every time we hand-wire an htmx attribute, hardcode a form action,
or thread data through nested template calls, we are discovering a
pattern the engine should automate. The hand-written templates are
the specification. The engine replaces them.

---

## Implementation Order

1. Fork the parser: copy `token.go` and `parse.go` from
   `golang.org/x/net/html`. Add self-closing support for vocabulary
   tags. Add template expression tokenization.
2. Extend the AST: add expression nodes, block nodes (if/each), and
   snippet definition nodes to the tree.
3. Build the renderer: walk the AST, resolve engine tags by inserting
   subtrees, resolve snippets by grafting fragment subtrees,
   serialize the final tree to HTML.
4. Wire the comment form: first engine tag handler, first snippet
   call, first bidirectional composition.
5. Wire the remaining engine tags one at a time, starting with the
   simplest (site-header, site-footer) and ending with the most
   complex (post-list with pagination).
6. Collect and serve component styles as a scoped stylesheet.
7. Collect and serve component scripts.
8. Build the language server against the same vocabulary.
9. Build the compiler when the interpreter is stable.
