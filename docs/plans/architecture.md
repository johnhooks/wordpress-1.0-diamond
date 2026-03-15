# Architecture

## The Premise

WordPress 1.0 was a blogging engine. Press is what happens when you take that engine and evolve it with modern tools instead of turning it into a page builder. Same soul, different language.

## Runtime: Go

Press compiles to a single binary. Run `press serve` and you have a blog. Static assets like CSS, images, and quicktags JS are embedded via `embed.FS`. SQLite is provided by `modernc.org/sqlite`, a pure Go implementation that requires no CGO.

## CLI

The CLI is built with Cobra. The binary is called `press`.

```
press serve              # Start the blog server
press migrate            # Run database migrations
press migrate status     # Show migration state
press user create        # Create a user
press user list          # List users
press secret generate    # Generate a secret key
press db seed            # Seed default data
press db reset           # Reset database (dev only)
```

## Configuration

The `.env` file holds all runtime configuration and is loaded on startup. Environment variables override `.env` values for containers and production deployments. Settings managed through the admin panel live in the `wp_options` table. The `--env` flag allows specifying a different `.env` path.

```env
# Application
APP_ENV=development
APP_HOST=localhost
APP_PORT=8080
APP_DEBUG=true

# Security
SECRET_KEY=change-me-in-production

# Database
DB_PATH=./storage/press.db

# Uploads
UPLOADS_DIR=./storage/uploads

# Theme
THEME=suspended
```

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
    ├── secret.go            # press secret generate
    └── db.go                # press db seed/reset

internal/
├── config/
│   └── config.go            # .env loading, env var override
├── db/
│   ├── db.go                # SQLite connection, query helpers
│   ├── schema.go            # Table creation, migrations
│   └── seed.go              # Default data (admin user, Hello World, options)
├── functions/
│   ├── functions.go         # Core utilities (wp-includes/functions.php)
│   ├── formatting.go        # wptexturize, wpautop, convert_smilies
│   ├── kses.go              # HTML sanitization
│   └── auth.go              # Password hashing, cookie auth
├── cache/
│   ├── cache.go             # In-process cache engine
│   ├── options.go           # Options multi-layer cache
│   ├── objects.go           # Post/comment/term object cache
│   └── query.go             # Query result cache
├── template/
│   ├── tags.go              # Template tags: the_title, the_content, etc.
│   ├── links.go             # Blogroll template functions
│   └── archives.go          # get_archives, get_calendar, list_cats
├── query/
│   └── query.go             # Query engine (wp-blog-header equivalent)
├── options/
│   └── options.go           # Options CRUD (backed by cache layer)
├── taxonomy/
│   └── taxonomy.go          # Category/link_category via terms tables
└── xmlrpc/
    └── xmlrpc.go            # XML-RPC (Blogger/metaWeblog API)

web/
├── handlers/
│   ├── blog.go              # Index, single post, archives, search
│   ├── feed.go              # RSS, RSS2, RDF, Atom
│   ├── comments.go          # Comment display and submission
│   ├── login.go             # Login, logout, registration
│   ├── trackback.go         # Trackback handler
│   └── mail.go              # Post-by-email
├── admin/
│   ├── dashboard.go         # Dashboard
│   ├── posts.go             # Write/manage posts
│   ├── comments.go          # Comment management, moderation
│   ├── categories.go        # Category management
│   ├── links.go             # Blogroll manager
│   ├── options.go           # Settings pages
│   ├── users.go             # User management, profile
│   ├── templates.go         # Theme selection
│   ├── upload.go            # File uploads
│   ├── install.go           # First-run wizard
│   └── import.go            # Import (MT, Blogger, etc.)
└── templates/
    ├── default/             # Default theme (Go html/template)
    │   ├── index.html
    │   ├── single.html
    │   ├── archive.html
    │   ├── search.html
    │   ├── comments.html
    │   └── popup.html
    ├── admin/               # Admin panel templates
    │   ├── header.html
    │   ├── footer.html
    │   ├── sidebar.html
    │   ├── dashboard.html
    │   ├── edit-posts.html
    │   ├── edit-form.html
    │   ├── edit-comments.html
    │   ├── categories.html
    │   ├── link-manager.html
    │   ├── options.html
    │   ├── profile.html
    │   ├── users.html
    │   ├── upload.html
    │   └── install.html
    └── feed/                # Feed templates
        ├── rss.xml
        ├── rss2.xml
        ├── rdf.xml
        └── atom.xml

static/
├── wp-layout.css            # Main stylesheet (faithful port)
├── print.css                # Print stylesheet
├── wp-images/               # Smilies, admin icons
└── js/
    ├── htmx.min.js          # htmx (embedded, no CDN)
    ├── quicktags.js          # Post editor tag insertion
    ├── slug-preview.js       # Permalink preview while typing
    ├── char-counter.js       # Excerpt character count
    ├── confirm.js            # Delete confirmation
    └── popup.js              # Comment popup window

local/                       # Example application (dev environment)
├── .env.example             # Example config
├── .gitignore               # Ignores .env, storage/
├── press.toml               # Project manifest
├── public/                  # Public static files
└── storage/                 # Database, uploads (gitignored)
```

## Request Flow

This mirrors `wp-blog-header.php`, the heart of WordPress 1.0.

```
HTTP Request
    -> Router (maps URLs to handlers)
    -> Query Engine (builds WHERE clause from params)
    -> Cache Check (return cached result if valid)
    -> Database Query (SELECT from wp_posts)
    -> Template Rendering (Go html/template with tag functions)
    -> HTML Response
```

### Query Parameters

These are unchanged from the original.

| Parameter | Purpose |
|-----------|---------|
| `m` | Month in YYYYMM or YYYYMMDD format |
| `p` | Post ID |
| `name` | Post slug |
| `cat` | Category ID |
| `author` | Author ID |
| `s` | Search term |
| `paged` | Pagination offset |
| `year`, `monthnum`, `day` | Date components |

## Cache System

This is where Go gives us something PHP never could. PHP dies after every request. Every page load rebuilds the world from scratch. WordPress bolted on object caching with external services like Memcached and Redis as a workaround. We do not need workarounds. We have a long-lived process with its own memory.

### Layers

```
┌─────────────────────────────────────────┐
│  Query Cache                            │  Full query results by WHERE clause
│  Key: hash(SQL + params)               │  Invalidated on writes to affected tables
├─────────────────────────────────────────┤
│  Object Cache                           │  Individual posts, comments, terms, users
│  Key: type:id (e.g. "post:42")         │  LRU eviction, invalidated on update/delete
├─────────────────────────────────────────┤
│  Options Cache                          │  Three-layer: autoload map, notoptions, DB
│  All autoloaded options in memory       │  Single query on startup, never touched again
├─────────────────────────────────────────┤
│  Template Cache                         │  Parsed Go templates, cached permanently
│  Parsed once, reused forever            │  Rebuilt only on theme change
├─────────────────────────────────────────┤
│  Term Cache                             │  Full taxonomy tree in memory
│  Categories + link categories           │  Rebuilt only on term writes
└─────────────────────────────────────────┘
```

### Why This Matters

Modern WordPress with Redis still cannot match this. Their object cache lives in another process, so every cache hit is a network round-trip. Our cache is a map lookup in the same process. Sub-microsecond reads versus sub-millisecond reads. For a blog that serves the same posts to every visitor, this is the difference between needing a CDN and not needing one.

### Invalidation Strategy

- **Write-through.** Every mutation (post save, comment approve, option update) immediately updates or evicts the relevant cache entries.
- **Table-level tracking.** The query cache knows which tables a query touched. A write to `wp_posts` invalidates all queries that read from `wp_posts`.
- **No TTL for most things.** Blog content does not expire. Cache entries live until explicitly invalidated by a write. Options, terms, and templates are effectively permanent.
- **LRU for objects.** Posts and comments use bounded LRU. A blog with 10,000 posts does not need all of them in memory, just the hot set.

## Authentication

Same model as WP 1.0, cleaned up.

- Cookie-based sessions.
- `user_level` (0-10) for permissions, using the same numeric levels as the original.
- bcrypt passwords instead of MD5.
- Login and logout at `/wp-login.php` equivalent routes.

## Client-Side: htmx + Vanilla JS

WordPress 1.0 was full page reloads for everything. That was the one thing that genuinely felt dated. htmx is the natural evolution of server-rendered HTML, letting the server stay in control while eliminating unnecessary page reloads.

### The Rule

The server renders all HTML. The client never assembles its own views. htmx handles server communication. Small, focused JS scripts handle the few interactions that cannot be a server round trip.

### htmx Handles

- **Admin forms.** Save a post, update options, approve a comment. htmx posts the form and swaps in the response without a full reload.
- **Comment moderation.** Click approve or delete, htmx swaps the row.
- **Post list filtering.** Change the category or date dropdown, htmx loads the filtered results into the table body.
- **Dashboard widgets.** Recent comments, recent posts, and stats each load independently on page load so the dashboard renders instantly.
- **Frontend comment submission.** Post a comment and it appears inline.
- **Pagination.** "Load more" or page links that swap the post list.
- **Archive navigation.** Click a month and htmx swaps the post list.

### Vanilla JS Handles

These are the interactions that need to happen entirely in the browser with no server round trip.

- `quicktags.js` inserts HTML tags around selected text in the post editor textarea. This was already JS in the original.
- `slug-preview.js` shows the permalink as the user types a title.
- `char-counter.js` provides a live character count for excerpts.
- `confirm.js` handles delete confirmation dialogs.
- `popup.js` opens the comment popup window.

### What We Do Not Use

No Alpine.js. No client-side templating. No reactive data binding in HTML attributes. No JS build step, no bundler, no transpiler, no node_modules. These are small scripts that do not need tooling. Each script is self-contained, embedded in the binary, and loaded only on the pages that need it.

### Embedding

htmx and all JS files are embedded in the binary via `embed.FS` alongside the CSS and images. Zero external dependencies at runtime.

## Template System

Go's `html/template` replaces PHP's inline tags. Template functions map one-to-one to WP 1.0 template tags.

| WP 1.0 PHP | Go Template |
|-------------|-------------|
| `<?php the_title(); ?>` | `{{ .TheTitle }}` |
| `<?php the_content(); ?>` | `{{ .TheContent }}` |
| `<?php the_date(); ?>` | `{{ .TheDate }}` |
| `<?php the_author(); ?>` | `{{ .TheAuthor }}` |
| `<?php the_category(); ?>` | `{{ .TheCategory }}` |
| `<?php bloginfo('name'); ?>` | `{{ .BlogInfo "name" }}` |
| `<?php get_archives(); ?>` | `{{ .GetArchives }}` |
| `<?php get_calendar(); ?>` | `{{ .GetCalendar }}` |
| `<?php get_links_list(); ?>` | `{{ .GetLinksList }}` |
| `<?php list_cats(); ?>` | `{{ .ListCats }}` |

## Content Filters

Same pipeline as the original, applied in order.

1. `wptexturize` converts straight quotes to smart quotes, double hyphens to en/em dashes, and triple dots to ellipses.
2. `wpautop` converts double line breaks to `<p>` tags.
3. `convert_bbcode` converts BBCode to HTML if enabled.
4. `convert_gmcode` converts GreyMatter markup if enabled.
5. `convert_smilies` converts text emoticons to smiley images if enabled.
6. `balanceTags` closes unclosed HTML tags if enabled.

## Feeds

Same formats as WP 1.0.

- RSS 0.92 at the `/wp-rss.php` equivalent route.
- RSS 2.0 at `/wp-rss2.php`.
- RDF 1.0 at `/wp-rdf.php`.
- Atom 0.3 at `/wp-atom.php`.
- Comments RSS 2.0 at `/wp-commentsrss2.php`.

## Admin Panel

Server-rendered HTML with the same pages and flow as WP 1.0.

- Dashboard.
- Write Post with quicktags toolbar.
- Manage Posts and Comments.
- Links (blogroll manager).
- Categories.
- Users and Profile.
- Options covering General, Writing, Reading, Discussion, and Permalinks.
- Upload.
- Templates for theme selection.
- Import for Blogger, MT, GreyMatter, Textpattern, and b2.

## Taxonomy System

The user sees "categories." The database stores terms and taxonomies. This is the one place where we clearly improve on the original because the modern taxonomy system is strictly better. But the surface is identical.

- `wp_terms` and `wp_term_taxonomy` replace `wp_categories` and `wp_linkcategories`.
- `wp_term_relationships` replaces `wp_post2cat`.
- Taxonomy types are `category` for post categories and `link_category` for blogroll categories.
- Category hierarchy is supported via the `parent` field on `wp_term_taxonomy`.
- Go code exposes category-specific functions that wrap taxonomy queries.

## Database

SQLite. Single file. See `database.md` for the full schema.

12 tables using the modern WordPress structure: posts, postmeta, comments, commentmeta, users, usermeta, terms, term_taxonomy, term_relationships, termmeta, links, and options.

## Project Phases

### Phase 1: Foundation
- Go project scaffold, Cobra CLI, and config loading.
- SQLite database layer with migrations and seed data.
- Core functions for formatting, date handling, and escaping.
- Query engine as the wp-blog-header equivalent.
- In-process cache system.
- Template tag system.
- Default theme.
- `press serve` starts the blog.

### Phase 2: Admin Panel
- Auth with login, sessions, and user levels.
- Dashboard.
- Post editor with quicktags.
- Post and comment management.
- Category management.
- Options pages.

### Phase 3: Interaction
- Comment submission and display.
- Comment popup.
- Comment moderation.
- Trackback send and receive.
- Pingback.

### Phase 4: Blogroll and Links
- Link manager.
- Link categories.
- OPML import and export.

### Phase 5: Feeds and APIs
- RSS, RDF, and Atom feeds.
- XML-RPC with Blogger API and metaWeblog API.
- Post-by-email.

### Phase 6: Polish
- File upload.
- User registration.
- Theme switching.
- Import tools for MT, Blogger, and others.
- Permalink structures.
- Search.
