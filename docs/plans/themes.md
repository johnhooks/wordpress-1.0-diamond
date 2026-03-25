# Themes (Superseded)

> **This document has been superseded.** The theme architecture evolved
> significantly during design. The current vision lives in
> `docs/plans/themes/skinning.md` and its companion docs. This file is
> kept for historical reference — it represents an earlier iteration
> where Press controlled the visual system through CSS custom properties
> and themes were variations within it. The current model inverts this:
> themes control all rendering, Press compiles and serves them.

WordPress's theme system was a PHP runtime with no sandbox. `functions.php`
could call `exec()`. Every theme was a potential attack surface. When
Gutenberg tried to fix the editor it made things worse — server-rendered PHP
blocks that React also had to understand, meaning every block became a
full-stack application.

Press separates three things WordPress conflated:

| Concern             | Owner                 | Language           |
| ------------------- | --------------------- | ------------------ |
| Document content    | ProseMirror + server  | JSON → HTML        |
| Page composition    | Theme templates       | Go `html/template` |
| Visual presentation | CSS custom properties | CSS                |

These three layers never reach into each other. The theme never touches the
document model. The document model never touches the CSS. The server renders
everything; the client decorates it.

---

## Layer 1 — Composition: Go Templates

A theme is a directory of `.html` files using Go's `html/template`. A
WordPress author squints at it and recognises the Loop, `the_title()`,
`the_content()`, `get_template_part()` — but it is Go's template language,
which means auto-escaping is structural and there is no filesystem access,
no network calls, no arbitrary code execution. The only functions available
are what Press explicitly registers in the funcmap.

```
themes/
└── suspended/
    ├── theme.toml          # Name, author, description
    ├── style.css           # Main stylesheet
    ├── header.html         # Site header
    ├── index.html          # Homepage / the loop
    ├── single.html         # Single post
    ├── page.html           # Static page
    ├── archive.html        # Date/category archives
    ├── search.html         # Search results
    ├── comments.html       # Comment list + form
    ├── sidebar.html        # Sidebar
    ├── footer.html         # Site footer
    └── static/             # Theme-specific assets
        └── images/
```

When you squint, you see WordPress. That's the point.

### Template Hierarchy

Press resolves the most specific template that exists on disk:

```
single post  →  single.html  →  index.html
search       →  search.html  →  archive.html  →  index.html
archive      →  archive.html  →  index.html
home         →  home.html    →  index.html
```

### Composition

Templates compose using Go's native `block` and `define`. The layout is
HTML that includes parts — header, sidebar, footer — just like WordPress:

```html
<!-- layout.html -->
<body class="{{ .BodyClass }}">
  <div id="wrap">
    {{ template "header" . }}
    <div id="content">{{ block "main" . }}{{ end }}</div>
    {{ template "sidebar" . }} {{ template "footer" . }}
  </div>
</body>
```

A page template overrides the `main` block:

```html
<!-- single.html -->
{{ define "main" }}
<article class="post" id="post-{{ .Post.ID }}">
  <h2>{{ .Post.TheTitle }}</h2>
  <p class="post-meta">
    <time>{{ .Post.TheDate }}</time> — {{ .Post.TheAuthor }} — {{
    .Post.TheCategory }}
  </p>
  <div class="entry">{{ .Post.TheContent }}</div>
  {{ template "comments" . }}
</article>
{{ end }}
```

### Template Tags

The funcmap is the template tag library. Press's curated, safe API surface.
WP naming — a Go developer squints and feels uncomfortable, a WordPress
developer squints and feels at home. That's the point.

```
{{ .Post.TheTitle }}          the_title()
{{ .Post.TheContent }}        the_content()
{{ .Post.TheDate }}           the_date()
{{ .Post.TheAuthor }}         the_author()
{{ .Post.TheCategory }}       the_category()
{{ .Post.CommentsLink }}      comments_popup_link()
{{ .Blog.Info "name" }}       bloginfo('name')
{{ .Sidebar.Categories }}     list_cats()
{{ .Sidebar.Archives }}       get_archives()
{{ .Sidebar.Links }}          get_links_list()
```

---

## Full Pages and Fragments

This is where themes meet htmx. Every page the blog serves is a composition
of templates. A full page load assembles all of them into one HTML document.
An htmx request returns just the piece that changed.

The same templates build both. The `render()` method on the server checks
for the `HX-Request` header. Full request: render layout + header + main +
sidebar + footer. htmx request: render just the `main` block.

```go
func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data any) {
    if r.Header.Get("HX-Request") != "" {
        // Render only the content block — no layout shell
        s.tmpl.ExecuteTemplate(w, name, data)
        return
    }
    // Full page — layout wraps the content block
    s.tmpl.ExecuteTemplate(w, "layout", data)
}
```

This is the core of the htmx architecture. Every navigation, every form
submission, every action goes through this pattern. The server always
renders HTML. The only question is how much HTML.

### Navigation

All internal links are htmx-enabled:

```html
<a
  href="/2004/01/03/hello-world/"
  hx-get="/2004/01/03/hello-world/"
  hx-target="#content"
  hx-push-url="true"
  >Hello World</a
>
```

- `href` is the real URL (works without JS, works for crawlers)
- `hx-get` makes it an htmx request
- `hx-target="#content"` swaps just the content area
- `hx-push-url="true"` updates the browser URL bar

Clicking any link swaps the content area without reloading header, sidebar,
or footer. The site feels like a SPA but every response is server-rendered
HTML. Back button works because htmx manages the history stack.

First load is always a full page (no `HX-Request` header). Subsequent
navigation is fragments. Refresh always works because the full layout
renders a complete page.

### Inline htmx

Theme authors declare interactivity directly in template markup — no
JavaScript files to write:

```html
<!-- Comment form in comments.html -->
<form hx-post="/comments" hx-target="#comment-list" hx-swap="beforeend">
  <input type="text" name="author" placeholder="Name" />
  <textarea name="comment"></textarea>
  <button type="submit">Post Comment</button>
</form>
```

---

## Layer 2 — Interactivity: Scripts

Press ships a small set of focused vanilla JS scripts for interactions that
require client-side logic. Themes opt in via `theme.toml`. No theme author
writes JavaScript to get standard blog interactivity.

| Script            | Purpose                      | WP equivalent          |
| ----------------- | ---------------------------- | ---------------------- |
| `quicktags.js`    | Tag insertion in post editor | `quicktags.js`         |
| `slug-preview.js` | Live permalink preview       | custom                 |
| `char-counter.js` | Excerpt character count      | custom                 |
| `confirm.js`      | Delete confirmation dialogs  | inline onclick         |
| `popup.js`        | Comment popup window         | `wp-comments-popup.js` |

ProseMirror lives in the admin write-post page as a first-class JS
application. It talks to the server via a clean JSON API. It knows nothing
about themes. The theme receives rendered HTML into `{{ .Post.TheContent }}`
and is done.

---

## Layer 3 — Presentation: CSS Custom Properties

Every visual value in a theme is a CSS custom property. The stylesheet is
`var(--press-*)` references. Hardcoded values are the exception.

```css
/* style.css */
/*
Theme Name: Suspended
Description: A quiet theme for quiet writing
Version: 1.0
*/

body {
  font-family: var(--press-body-font);
  color: var(--press-text-color);
  background: var(--press-bg-color);
}

.post h2 a {
  font-family: var(--press-heading-font);
  color: var(--press-heading-color);
}

.entry {
  max-width: var(--press-content-width);
  line-height: var(--press-line-height);
}
```

`tokens.json` declares the custom property values — their defaults, types,
and admin display names:

```json
{
  "--press-body-font": {
    "default": "Georgia, serif",
    "type": "font",
    "label": "Body Font"
  },
  "--press-heading-font": {
    "default": "Arial, sans-serif",
    "type": "font",
    "label": "Heading Font"
  },
  "--press-text-color": {
    "default": "#333333",
    "type": "color",
    "label": "Text Color"
  },
  "--press-bg-color": {
    "default": "#ffffff",
    "type": "color",
    "label": "Background"
  },
  "--press-content-width": {
    "default": "680px",
    "type": "length",
    "label": "Content Width"
  },
  "--press-line-height": {
    "default": "1.7",
    "type": "number",
    "label": "Line Height"
  }
}
```

### Customizer (Future)

The admin customizer reads `tokens.json`, generates controls (color swatch,
font picker, length slider), and writes
`element.style.setProperty('--press-heading-font', value)`. The preview is
the actual rendering engine — same CSS file, same custom properties, no
separate preview mode. What you see is genuinely what you get.

This is not a first priority. First: render the theme. The theme IS the
customization — you edit the files. The customizer UI comes later, once
themes work and we know what controls actually matter.

---

## Theme Manifest

`theme.toml` declares the theme's identity and capabilities. A WordPress
author squints and sees `style.css` header comments and
`add_theme_support()` — but it is declarative data, not executable code.

```toml
name        = "Suspended"
version     = "1.0"
description = "A quiet theme for quiet writing"
author      = "John Hooks"
author_uri  = "https://example.com"

[features]
featured-image  = false
custom-header   = false
wide-layout     = false
comment-avatars = true

[scripts]
frontend = []
admin    = ["quicktags"]
```

---

## Theme Selection

The active theme is set via the `THEME` environment variable or the
`template` option in `wp_options`. The admin panel has a theme picker that
shows installed themes with their name and description from `theme.toml`.

Switching themes swaps the template directory. The server reloads templates
from the new directory.

## Default Theme

Press ships with a default theme embedded in the binary via `embed.FS`.
This is the fallback when no theme directory is configured. It's minimal
and functional — the "Classic" to someone else's "Kubrick."

Custom themes loaded from the filesystem override the embedded default.

---

## Security Model

The security boundary is the funcmap. Theme authors write HTML and template
calls. They cannot write Go. The only code that executes during template
rendering is code that Press has explicitly registered — field accessors
and safe formatting functions. There is no equivalent of `functions.php`.
There is no hook system that themes can reach into. There is no way for
an uploaded theme to touch the filesystem, make a network request, or
execute a system command.

A theme is: `.html` files, `style.css`, `tokens.json`, `theme.toml`,
and static assets. That is the entire attack surface.

---

## What Themes Do Not Do

- **No PHP.** Themes are Go templates. No arbitrary code execution.
- **No theme functions file.** No `functions.php` equivalent. Themes are
  presentation only.
- **No child themes.** Swap the whole directory.
- **No theme marketplace.** Themes are directories you put on the server.
- **No build step.** Raw files. A blog theme doesn't need webpack.
