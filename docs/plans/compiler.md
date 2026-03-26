# The Theme Compiler

The theme compiler transforms theme source templates into Go
templates. The source uses a slot-based syntax where the engine and
theme share ownership of the page. The engine owns behavior (forms,
loops, htmx wiring, data binding). The theme owns rendering (layout,
styling, visual structure). They meet at named slots.

This is the most ambitious piece of Press. It deserves careful design
before implementation.

---

## The Problem

Go's `html/template` is good at rendering data but bad at composition.
You cannot pass arguments to blocks. You cannot have named slots. You
cannot define a layout that a child fills with multiple distinct
regions. Every template call takes a single dot context.

This forces a choice: the theme owns everything (powerful but
fragile, no engine control over behavior) or the engine owns
everything (safe but creatively dead, no theme control over
rendering). WordPress went the first way. Block themes went the
second.

Press needs a third path.

---

## The Idea

Two kinds of slots:

**Engine slots**: the engine provides content. The theme decides
where to place it. Examples: post content, comment list, comment
form, pagination, category list, search form.

**Theme slots**: the theme provides content. The engine calls it at
the right point. Examples: how a form field looks, how a post row
renders, how a submit button appears.

The page is composed by nesting these. The theme provides a layout
with engine slot placeholders. The engine fills those slots with
structural HTML that contains theme slot placeholders. The compiler
resolves both directions and produces flat Go templates.

---

## Source Syntax

Theme source templates use HTML with custom elements for slots.
The syntax is valid HTML (custom elements are allowed) so editors
provide syntax highlighting and structure validation.

### Theme placing engine slots

```html
<!-- The theme's home page template -->
<Layout>
  <header />
  <main>
    <slot:post-list />
    <slot:pagination />
  </main>
  <Sidebar>
    <slot:search />
    <slot:categories />
    <slot:archives />
  </Sidebar>
  <footer />
</Layout>
```

### Engine slot containing theme slots

```html
<!-- The engine's comment form (theme author never writes this) -->
<form
  method="post"
  action="/comments"
  hx-post="/comments"
  hx-target="#comment-list"
  hx-swap="beforeend"
>
  <input type="hidden" name="_csrf" value="..." />
  <input type="hidden" name="comment_post_ID" value="..." />

  <slot:field name="author" label="Name" required />
  <slot:field name="email" label="Email" required />
  <slot:field name="url" label="Website" />
  <slot:field
    name="comment"
    label="Comment"
    type="editor"
    features="basic"
    required
  />

  <slot:submit label="Post Comment" />
</form>
```

### Theme defining how a field renders

```html
<!-- The theme's field molecule -->
<define:field>
  <p>
    <label for="{{.ID}}"><strong>{{.Label}}</strong></label><br>
    <input type="{{.Type}}" name="{{.Name}}" id="{{.ID}}"
           value="{{.Value}}" {{.Attributes}}>
    {{if .Error}}<br><small>{{.Error}}</small>{{end}}
  </p>
</define:field>
```

---

## Bidirectional Nesting

The composition flows both ways:

```
Theme layout
  └─ Engine slot: post list
       └─ Theme slot: post row (called per post)
            └─ Engine slot: post meta (date, author, categories)
                 └─ Theme slot: how metadata looks

Theme layout
  └─ Engine slot: comment form
       └─ Engine: <form> tag, CSRF, hidden fields, htmx attrs
            └─ Theme slot: field (called per field)
                 └─ Engine: field name, type, required, value
                      └─ Theme: label, input, error, styling
```

The compiler flattens this into sequential Go template calls. Each
level resolves its slots and produces the appropriate `{{template}}`
calls with the right data context.

---

## What the Compiler Produces

Standard `html/template` code. No custom runtime. The compiled
templates execute with Go's built-in template engine at full speed
with automatic escaping.

Example: the source slot `<slot:field name="author" label="Name" />`
in the comment form compiles to something like:

```go
{{template "field" (fieldProps "author" "Name" "text" .SavedAuthor true "")}}
```

Where `fieldProps` is a funcmap function that builds the field data
struct from the arguments. The theme's "field" template receives
the struct and renders it.

---

## Compilation Steps

1. **Parse** theme source templates (HTML with custom elements).
2. **Resolve** engine slots: replace `<slot:post-list />` with the
   engine's post list template code (loop + theme row call).
3. **Resolve** theme slots: replace `<slot:field .../>` with the
   theme's field molecule call, passing the engine-provided props.
4. **Wire** htmx: add `hx-*` attributes to forms, links, and
   swap targets based on the component contracts.
5. **Assign** IDs: generate stable IDs for swap targets and ARIA.
6. **Emit** Go `html/template` code with all slots resolved,
   all wiring in place, all data threading done.
7. **Validate**: check all required slots are filled, all field
   names are valid, all component contracts are satisfied.

---

## Closed Set Advantage

Press is not a framework. The set of components is known and fixed:

**Engine slots** (things the engine fills):

- Post (single post content)
- Post list (loop over posts)
- Comment list (loop over comments)
- Comment form (form with fields)
- Post form (admin, form with fields)
- Pagination (prev/next links)
- Categories (sidebar list)
- Archives (sidebar list)
- Pages (sidebar list)
- Search form
- Meta links (login/logout/feed)

**Theme slots** (things the theme fills):

- Field (text input rendering)
- Textarea (textarea rendering)
- Editor (ProseMirror wrapper rendering)
- Radio group
- Checkbox group
- Submit button
- Post row (how a post appears in a list)
- Comment (how a comment appears)
- Layout (page structure)
- Header, footer, sidebar

There are roughly 20 engine slots and 15 theme slots. This is not
a general-purpose system. Every slot is documented, typed, and
tested. The compiler knows the complete vocabulary.

---

## What This Enables

- **Theme authors do not write htmx.** They place slots. The
  compiler wires the interactivity.
- **Theme authors do not write form tags.** They define field
  visuals. The compiler builds the form structure.
- **Engine changes do not break themes.** If the comment endpoint
  changes, the compiler output changes. The theme source is
  unchanged.
- **Validation catches errors before deploy.** Missing slots,
  wrong field names, incomplete templates are compile-time errors.
- **Same field molecule works everywhere.** The comment form and
  the post editor use the same field template. Style once.

---

## Open Questions

- **How does the compiler handle conditional content?** A sidebar
  that only appears on some pages. A form that only appears when
  comments are open. The source syntax needs conditionals.

- **How does iteration work in the source syntax?** The post list
  is a loop. Does the theme write `<slot:post-list>` and the
  engine handles the loop internally? Or does the theme write
  `<for:post in posts>` and control the loop structure?

- **How are engine slots defined?** Do we write them as Go code,
  as special template files, or as another layer of the same
  source syntax?

- **How does the compiler handle fragments?** For htmx partial
  renders, the engine needs to call a subset of the compiled
  template. The compiler needs to produce independently callable
  functions for each swappable region.

- **Build step or runtime?** Does the compiler run at build time
  (producing Go files that are compiled into the binary) or at
  startup (parsing source templates on server start)? Build time
  is faster at runtime. Startup is faster for development.

- **Error messages.** When the compiled template fails, the error
  needs to reference the source template, not the compiled output.
  Source maps or equivalent.

---

## Relationship to Current Code

The current admin and site templates are hand-written Go templates.
They are what the compiler would produce. Building them by hand
teaches us what the compiler needs to generate. Every time we
hand-wire an htmx attribute, hardcode a form action, or thread
data through nested template calls, we are discovering a pattern
the compiler should automate.

The hand-written templates are scaffolding. They prove the
rendering model works. The compiler replaces them with generated
output from a cleaner source syntax.

---

## When to Build

After the admin is functional with hand-written templates. The
admin has more forms, more htmx interactions, and more slot
boundaries than the frontend. Building it by hand reveals every
pattern the compiler needs to handle. Premature compilation would
mean building a compiler for patterns we have not discovered yet.

Build the admin. Catalog the patterns. Then build the compiler.
