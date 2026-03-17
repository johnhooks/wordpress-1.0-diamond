# Press

## Why

Humans write words to share ideas, feelings, experiences, stories. They
should own that content and share it how they want — publicly, with a
group, individually, or totally private just for themselves. Writing can
be deeply collaborative; the tool should help people come together to
produce a final document they're prepared to share however they see fit,
when they want. A self-hosted blog is accessible by the owner from
anywhere in the world — no app, no payment beyond the server — and
readable by anyone, anywhere.

## What This Is

WordPress 1.6 — the release that never happened.

WordPress 1.0 "Platinum" shipped in January 2004. WordPress 1.5
"Strayhorn" shipped in February 2005 and added themes, subcategories,
and static pages. Then the version number jumped to 2.0, the rich text
editor arrived, and WordPress began its long march toward becoming a
CMS, a page builder, and eventually a full site editor.

Press picks up where the 1.x line left off. Everything WordPress built
through 1.5, evolved forward as a blog. Go instead of PHP. SQLite
instead of MySQL. htmx instead of React. The modern WordPress database
schema instead of the original's rough edges. But the same conviction:
a blog is text on the internet.

We never leave 1.x. There is no 2.0. There is no CMS pivot. Press is
WordPress in the timeline where the blog stayed a blog.

## Design Principles

- **WordPress through 1.5 is the north star.** The feature set of WP 1.0 through 1.5.1.2 defines the baseline. If it existed in that lineage, we probably want it. If it came in 2.0+, we probably don't.
- **Fix the obviously broken stuff.** The 4-table options system, MD5 passwords, and GeoURL are not worth preserving out of nostalgia. We preserve the spirit, not the mistakes.
- **Go idioms, not PHP port.** This is not a line-by-line translation. We use Go's strengths like `html/template`, goroutines, in-process caching, and single binary distribution to build the same experience better.
- **Modern schema, original surface.** The database uses WordPress's evolved table structure with taxonomies, meta tables, and simplified options. The user sees categories, not "taxonomies." The admin says "user level," not "capabilities."
- **htmx, not React.** Server-rendered HTML. The server responds with HTML for all page rendering — full pages on normal requests, fragments on htmx requests. JSON APIs exist for the editor (ProseMirror steps, collab protocol) and may be extended for external services, but they don't power a SPA. No client-side framework.
- **SQLite. Single binary. Zero dependencies.** Run `press serve` and you're blogging.

## The CLI

The binary is called `press`. WordPress forgot the press. We didn't.

```
press serve           # Start the blog
press migrate         # Run database migrations
press user create     # Create a user
press secret generate # Generate secret key
press db seed         # Seed default data
```

## Configuration

The `.env` file holds all runtime configuration and is loaded on startup. Environment variables override `.env` values for containers and production deployments. Admin-editable settings like blog name and posts per page live in the `wp_options` table and are managed through the admin panel. No config file format wars. `.env` is enough for a blog.

## Reference Codebases

### WordPress 1.0 Platinum
**Location:** `~/Projects/wordpress-1.0-platinum`

The starting point. The original blogging engine.

### WordPress Develop (Full History)
**Location:** `~/Projects/wordpress-develop`

The complete WordPress history from b2/cafelog (2003) through present. Use this to trace how features evolved through the 1.x line (up to 1.5.1.2) and for the modern schema at `src/wp-admin/includes/schema.php`. The 1.x line ended at `$wp_version = '1.5.1.2'` (May 2005) — after that, version bumps went to 1.6-ALPHA which became 2.0.

### Heidelberg
**Location:** `~/Projects/heidelberg`

Prior Go-based WordPress project. Reference for Go project structure, Cobra CLI patterns, config loading, database migrations, and test patterns.
