# Press

## What This Is

A continuation of WordPress 1.0.

In January 2004, WordPress 1.0 "Platinum" shipped with 22,000 lines of PHP, 78 files, and 10 database tables. A blogging engine. It published words on the internet and it did it well.

Then WordPress kept going. It became a CMS, then a page builder, then a full site editor. Twenty years later, the best blogging platform in history no longer thinks of itself as a blogging platform.

We think WordPress 1.0 is sacred ground. We have planted our flag and we are not leaving. Press is what happens when you fork WordPress 1.0 and keep evolving it as a blogging engine. Go instead of PHP. SQLite instead of MySQL. The modern WordPress database schema instead of the original's rough edges. But the same conviction: a blog is text.

Once the WP 1.0 feature set is running, we keep going. We evolve Press as if it is WordPress in an alternate timeline where the blog never became a CMS. New features, new ideas, but always in service of writing and publishing text on the internet.

## Design Principles

- **WordPress 1.0 is sacred ground.** Every feature maps back to the original. If WP 1.0 didn't do it, we probably don't either.
- **Fix the obviously broken stuff.** The 4-table options system, MD5 passwords, and GeoURL are not worth preserving out of nostalgia. We preserve the spirit, not the mistakes.
- **Go idioms, not PHP port.** This is not a line-by-line translation. We use Go's strengths like `html/template`, goroutines, in-process caching, and single binary distribution to build the same experience better.
- **Modern schema, original surface.** The database uses WordPress's evolved table structure with taxonomies, meta tables, and simplified options. The user sees categories, not "taxonomies." The admin says "user level," not "capabilities."
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

The primary reference. The sacred ground. Every feature we implement should trace back to something in this codebase.

### WordPress Develop (Modern)
**Location:** `~/Projects/wordpress-develop`

Schema reference at `src/wp-admin/includes/schema.php`. We use the modern table structure but only expose WP 1.0 features through it.

### Heidelberg
**Location:** `~/Projects/heidelberg`

Prior Go-based WordPress project. Reference for Go project structure, Cobra CLI patterns, config loading, database migrations, and test patterns.
