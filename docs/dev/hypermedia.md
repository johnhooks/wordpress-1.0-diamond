# Hypermedia Architecture

Press is a hypermedia system. The server renders HTML. htmx handles
interactivity. There is no client-side application, no JSON API
powering the frontend, no state management library. When the user
does something, the server renders new HTML and the browser swaps
it in. This is not a simplification. It is the architecture, and
the template engine is purpose-built to make it work.

---

## Why This Is Complex

A naive server-rendered blog does not need a custom template engine.
Go's `html/template` would suffice. The complexity exists because
Press has three goals that conflict under a naive approach:

1. **Themes are user-supplied.** A non-technical writer picks a
   theme. The theme controls all visual presentation. The engine
   must not leak implementation details into the theme, and the
   theme must not be able to break the engine's behavioral
   guarantees.

2. **The server must render correct, targetable HTML.** Every
   interactive element needs a unique, predictable identity so
   htmx can swap it precisely. Comment 10 on post 42 must have
   an id the server can target in an out-of-band (OOB) swap without
   the theme author coordinating anything.

3. **Full pages and partials are the same thing.** When htmx
   submits a comment form, the server responds with a fresh form
   fragment and an OOB fragment containing the new comment. These
   fragments must be identical to what the full page would have
   rendered. If they diverge, the page drifts out of sync.

These three goals are the reason the template engine exists. The
engine provides a closed vocabulary of named tags that themes can
place but cannot modify the behavior of. The engine controls
identity, accessibility, and interactivity attributes. The theme
controls structure, content, and style. Neither can interfere with
the other.

---

## The Rendering Pipeline

Every render, full page or partial, follows the same pipeline:

```
HTTP Request
  → Request Handler (loads data from database)
    → View Struct (typed, complete data for the page or fragment)
      → Template Engine (evaluates template against scope)
        → Tag Handlers (reshape scope, inject engine attributes)
          → Molecule Templates (pure HTML views of their props)
            → HTML AST
              → Serialized HTML Response
```

The pipeline has two intermediate representations. The template
engine parses theme HTML into an AST, evaluates it with data, and
produces a new AST representing the final HTML. Only then is the
AST serialized to a string. This is not an optimization. It is what
allows the engine to inject attributes, validate structure, and
guarantee correctness before any bytes reach the wire.

### Three layers, three concerns

**Request handlers** are the only code that touches the database.
They load posts, comments, categories, whatever the page needs.
They build a view struct and hand it to the template engine. They
do not know about HTML, templates, or htmx. Their job is to answer
the question: what data does this page need?

**Tag handlers** are the bridge between the page's data and each
component's contract. When the engine encounters `<comment-form />`
in a template, it calls the comment-form handler. The handler looks
up the post ID from the current scope, generates a CSRF token from
the request context, and packages these into a flat struct that
matches exactly what the comment-form molecule expects. It also
tells the engine what attributes to inject on the wrapper element
(the id, the data attributes, the ARIA roles). The molecule never
sees the parent scope. It only sees what the handler gave it.

**Molecule templates** are pure views. They receive props and
render HTML. They do not load data, walk scope chains, compute
values, or set their own identity. If a molecule needs something,
the tag handler must provide it. This purity is what makes the
system safe for user-supplied themes: a theme author cannot
accidentally break swap targeting because they never touch the
attributes the engine owns.

---

## Same Pipeline, Full Page or Partial

The distinction between a full page and an htmx partial is where
you enter the pipeline, not how the pipeline works.

A full page render starts at a page template like `single.html`.
The engine evaluates it top-down, calling tag handlers as it
encounters vocabulary tags. The result is wrapped in the document
shell (`<html>`, `<head>`, `<body>`) and sent as a complete page.

An htmx partial enters at a molecule or organism template directly.
The request handler builds the same view struct (or a subset of it),
and `RenderFragment` evaluates the template without the document
shell. The output is an HTML fragment that htmx swaps into the
existing page.

Because both paths use the same templates, same handlers, and same
scope resolution, a molecule rendered as a partial is byte-identical
to that molecule rendered inside a full page. This is the property
that makes htmx work reliably. If the server renders a comment
differently in a partial than in a full page, the page drifts out of
sync after a swap. Press prevents this by construction: there is
only one rendering path.

---

## Engine-Owned Attributes

The engine owns certain HTML attributes. Theme authors place
vocabulary tags; the engine decides what attributes appear on the
output. This split is the core contract.

### Identity

The engine owns `id` on every vocabulary tag wrapper. Theme authors
must not set `id` on vocabulary tags or their molecule wrappers; the
engine panics if they try. This is enforced, not advisory.

The engine generates ids that are unique and predictable:
`post-{id}`, `comment-{id}`, `post-{post_id}-comments`,
`sidebar`. These ids serve three purposes:

- **Swap targeting.** htmx OOB swaps use these ids to place new
  content precisely. After a comment submission, the new comment
  is appended to `#post-42-comments .comment-list`. No coordination
  with the theme required.
- **Fragment anchoring.** Permalink fragments like `#comment-10`
  link directly to a specific comment.
- **Collision prevention.** When the same component appears multiple
  times on a page (comments on different posts in a feed view),
  engine-generated ids guarantee uniqueness.

### State

Data attributes mark engine-managed state that CSS and htmx can
react to without JavaScript. `data-empty` on the empty comment
placeholder lets the theme hide it with `[data-empty]:has(~ *)`
when sibling comments appear. The engine injects the attribute.
The theme writes the CSS. Neither needs to know how the other works.

### Accessibility

The engine injects ARIA attributes where it has the semantic
knowledge to do so. `role="complementary"` on the sidebar is an
engine concern because the engine knows the sidebar is a
complementary landmark. The theme author should not need to know
ARIA roles to build a correct page.

---

## The AST Is the Product

React builds a virtual DOM to diff and patch the browser's real DOM.
Press builds an HTML AST to transform and serialize on the server.
The motivation is different but the shape is similar: when you need
fine-grained control over the output, you need a tree you can
manipulate before it becomes a string.

A traditional template engine interpolates values into a string and
sends it. That works until you need to inject attributes the theme
author did not write, validate structure the theme author cannot
see, and guarantee that a fragment rendered in isolation matches
the same fragment rendered inside a full page. At that point, string
interpolation is not enough. You need a representation you can
inspect and transform.

Press accepts the cost of two intermediate representations (the
template AST from parsing, the HTML AST from evaluation) because
the goals demand it: a themeable, dynamic hypermedia system where
the engine controls identity and behavior while the theme controls
presentation. There are likely opportunities to optimize this
pipeline, but right now the priority is getting the architecture
right and understanding what the system needs to do. The
performance work comes after the design is proven.

The two-AST approach gives the engine three capabilities that
string-based rendering cannot provide:

1. **Engine attributes are injected on nodes, not spliced into
   strings.** When the engine adds `id="comment-10"` to a comment
   wrapper, it sets an attribute on an AST node. There is no regex
   replacement, no string insertion at the first `>` character, no
   risk of malformed output.

2. **The output is testable as structure.** Golden tests verify the
   AST as s-expressions, not as HTML strings. A test can assert
   that the comment wrapper has `(id "comment-10")` without parsing
   HTML or depending on attribute order. This is how the engine
   validates that tag handlers inject the right attributes.

3. **Validation happens before serialization.** The engine can
   reject invalid structure (an `id` on a molecule wrapper, a
   missing required attribute) before any bytes reach the response.
   Errors are caught at render time, not after the browser
   receives broken HTML.

The golden tests operate at two levels. Template AST tests verify
that theme templates parse and evaluate correctly with fixture data,
without engine attributes. Rendered HTML AST tests verify the final
output after tag handlers inject engine attributes. The first layer
catches theme regressions. The second catches engine regressions.

---

## Where This Is Going

The engine currently owns `id`, `data-*`, and `role` on vocabulary
tag wrappers. The vocabulary will expand until the engine owns
every `id` in the rendered output:

- **Input molecules** (`<text-field />`, `<email-field />`, etc.)
  will replace raw form elements. The engine will generate unique
  ids, wire `label[for]` to `input[id]` automatically, and provide
  injection points for htmx validation and autosave attributes.

- **Page layout tags** (`<site-header />`, `<site-content />`) will
  replace the repeated `<div id="wrapper">` boilerplate in every
  page template. The engine will own the structural ids.

- **Form organisms** will own their children's ids, scoped to the
  form context, so the same form appearing twice on a page gets
  unique field ids without theme author intervention.

The goal is a system where the theme author writes semantic
structure and the engine guarantees that every interactive element
is uniquely identified, correctly wired, and ready for htmx. The
complexity of hypermedia integration disappears into the engine.
The theme author writes `<comment-form />` and it works.
