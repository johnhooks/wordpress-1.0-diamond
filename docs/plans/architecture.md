# Architecture

## The Premise

WordPress 1.6 — the release that never happened. Press picks up where the
1.x line left off and evolves it as a blog. Go instead of PHP. SQLite
instead of MySQL. htmx instead of React. Same soul, modern tools.

## Runtime

Press compiles to a single binary. Run `press serve` and you have a blog.
Static assets are embedded via `embed.FS`. SQLite is provided by
`modernc.org/sqlite`, a pure Go implementation — no CGO required.

## CLI

The CLI is built with Cobra. The binary is called `press`.

```
press serve              # Start the blog server
press migrate            # Run database migrations
press user create        # Create a user
press db seed            # Seed default data
press db reset           # Reset database (dev only)
press secret generate    # Generate a secret key
press option get/set     # Read and write options
press post list/delete   # Manage posts
```

## Configuration

The `.env` file holds all runtime configuration. Environment variables
override `.env` values for containers and production. Admin-editable
settings (blog name, posts per page) live in `wp_options`. The `--env`
flag allows specifying a different `.env` path.

No TOML, no YAML, no config directory. A blog does not need that.

## Project Structure

```
cmd/
└── press/
    ├── main.go              # Entry point
    ├── root.go              # Cobra root command
    ├── serve.go             # press serve
    ├── migrate.go           # press migrate
    ├── user.go              # press user create/list
    ├── post.go              # press post list/delete
    ├── option.go            # press option get/set
    ├── secret.go            # press secret generate
    └── db.go                # press db seed/reset

internal/
├── config/                  # .env loading, env var override
├── database/                # SQLite connection, schema, migrations, seed
├── model/                   # Domain types: Post, Comment, User, Term, etc.
├── repository/              # Database queries per model
├── permalink/               # URL generation (Linker) and resolution (Router)
├── server/                  # HTTP handlers, routing, template rendering
├── cache/                   # In-process caching (options, objects, queries, tuples)
└── functions/               # Core utilities (formatting, auth, sanitization)

themes/
└── suspended/               # Default theme (Go html/template files + CSS)

local/                       # Example application / dev environment
├── .env.example
├── public/
└── storage/                 # Database file, uploads (gitignored)
```

Press is a library at the top level — `internal/` is the engine. The
`cmd/press/` binary is one way to run it. The `local/` directory is
the example application for development.

## Request Flow

Every request follows the same path:

1. **Router** — resolves the URL to a route type (post, archive, category,
   page, search). See `docs/plans/permalinks.md`.
2. **Permission check** — validates the subject's access via tuple cache.
   See `docs/plans/permissions.md`.
3. **Query** — fetches the data from the database (or cache).
4. **Render** — executes the theme template. Full page on normal requests,
   fragment on htmx requests. See `docs/plans/themes.md`.
5. **Response** — HTML. Always HTML.

### Query Parameters

These are the query-string fallbacks when pretty permalinks are off, and
they work as overrides on any route:

| Parameter | Purpose |
|-----------|---------|
| `p` | Post ID |
| `page_id` | Page ID |
| `name` | Post slug |
| `cat` | Category ID |
| `author` | Author ID |
| `s` | Search term |
| `m` | Date in YYYYMM or YYYYMMDD format |
| `year`, `monthnum`, `day` | Date components |
| `paged` | Pagination offset |

## Cache System

This is where Go gives us something PHP never could. PHP dies after every
request — every page load rebuilds the world from scratch. WordPress bolted
on object caching with Memcached and Redis as a workaround. We have a
long-lived process with its own memory. No workarounds needed.

### Layers

| Layer | Contents | Invalidation |
|-------|----------|--------------|
| Options cache | All autoloaded options in memory | Write-through on update |
| Notoptions cache | Known non-existent option keys | Cleared on relevant write |
| Tuple cache | Full permission set in memory | Rebuild on any tuple change |
| Term cache | Full taxonomy tree in memory | Rebuild on term writes |
| Object cache | Individual posts, comments, users (LRU) | Evict on update/delete |
| Query cache | Full query results by WHERE clause | Invalidate on table writes |
| Template cache | Parsed Go templates | Rebuild on theme change |

Every cache is in-process. Map lookups, not network calls. Sub-microsecond
reads versus sub-millisecond with Redis. For a blog serving the same posts
to every visitor, this is the difference between needing a CDN and not.

### Invalidation Strategy

- **Write-through.** Every mutation immediately updates or evicts the
  relevant cache entries.
- **Table-level tracking.** The query cache knows which tables a query
  touched. A write to `wp_posts` invalidates queries that read `wp_posts`.
- **No TTL for most things.** Blog content doesn't expire. Cache entries
  live until explicitly invalidated.
- **LRU for objects.** Posts and comments use bounded LRU. A blog with
  10,000 posts doesn't need all of them in memory.

## Authentication

- Cookie-based sessions
- bcrypt passwords (not MD5)
- Permission checks via tuple cache — not `user_level` or capabilities meta
- Login/logout at dedicated routes

See `docs/plans/permissions.md` for the full tuple-based permission system.

## Client-Side: htmx + Vanilla JS

The server renders all HTML. The client never assembles its own views.
htmx handles server communication. Small, focused vanilla JS scripts handle
the few interactions that need client-side logic.

### htmx handles

- Admin form submissions (save post, update options, approve comment)
- Comment moderation (approve/delete swaps the row)
- Post list filtering (category/date dropdowns reload the table body)
- Frontend comment submission (comment appears inline)
- Pagination (post list swaps without full reload)
- All navigation (content area swaps, header/sidebar/footer stay)

### Vanilla JS handles

- `quicktags.js` — tag insertion in the post editor textarea
- `slug-preview.js` — live permalink preview while typing a title
- `char-counter.js` — character count for excerpts
- `confirm.js` — delete confirmation dialogs
- `popup.js` — comment popup window

No Alpine.js, no client-side templating, no reactive data binding, no JS
build step, no bundler, no node_modules. htmx and all JS files are
embedded in the binary via `embed.FS`.

JSON APIs exist for the ProseMirror editor (steps, collab protocol,
editorial comments) and may be extended for external services. But JSON
APIs don't power the blog itself — the blog is server-rendered HTML.
The JSON surface is for the editor and for external integrations, not
for building a SPA.

See `docs/plans/themes.md` for the full htmx fragment architecture.

## Content Model

Post content is stored as ProseMirror JSON, not HTML. HTML is rendered at
display time from the document schema. No content filter pipeline — the
structured document model produces well-formed HTML by construction.

See `docs/plans/editor.md` for the editor and revision system.

## Template System

Go's `html/template` with a curated funcmap. Template functions map to
WordPress template tags — `TheTitle`, `TheContent`, `TheDate`. The funcmap
is the security boundary: theme authors write HTML and template calls, they
cannot execute arbitrary code.

See `docs/plans/themes.md` for the full template system.

## Taxonomy System

The user sees "categories." The database stores terms and taxonomies. This
is the one place where we clearly improve on the original — the modern
taxonomy system is strictly better, but the surface is identical.

- `wp_terms` and `wp_term_taxonomy` replace `wp_categories` and
  `wp_linkcategories`
- `wp_term_relationships` replaces `wp_post2cat`
- Taxonomy types: `category`, `link_category`, `series`
- Category hierarchy via the `parent` field on `wp_term_taxonomy`

See `docs/plans/series.md` for the series taxonomy.

## Database

SQLite. Single file. See `docs/plans/database.md` for the full schema.

WordPress tables (modernized): posts, postmeta, comments, commentmeta,
users, usermeta, terms, term_taxonomy, term_relationships, termmeta,
links, options.

Press tables (new): steps, save_points, groups, tuples, share_tokens.

## Feeds

RSS 2.0 and Atom at `/feed/`. Per-post comment feeds. Auto-generated from
post data, rendered via Go templates.

## Admin Panel

Server-rendered HTML with htmx interactions. The same page structure as
WordPress through 1.5:

- **Dashboard** — overview, recent posts, recent comments
- **Write** — post editor (ProseMirror), page editor
- **Manage** — post list, comment list, comment moderation
- **Links** — blogroll manager, link categories
- **Categories** — category management
- **Users** — user list, profile editing
- **Options** — general, writing, reading, discussion, permalinks
- **Upload** — file upload and media list
- **Themes** — theme selection and preview

All admin interactions use htmx — form saves, moderation actions, filtering,
and inline editing happen without full page reloads.

## Not Supported

- Trackbacks and pingbacks — dead protocols
- XML-RPC — if we need an API it'll be a proper one
- Post-by-email — niche feature, not worth the complexity
- Importers — defer until the core is solid
- BBCode / GreyMatter markup — legacy formats from b2/cafelog
- Content filter pipeline — ProseMirror handles structure
