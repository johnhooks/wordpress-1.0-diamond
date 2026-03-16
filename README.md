# Press

In January 2004, WordPress 1.0 "Platinum" shipped. 22,000 lines of PHP. 78 files. 10 database tables. It published words on the internet and it did it well.

Then WordPress kept going. It became a CMS, then a page builder, then a full site editor. That's a great story and we're happy for it.

But we liked the blogging part.

## What is Press?

Press is what happens when you fork WordPress 1.0 and keep evolving it as a blogging engine. Go instead of PHP. SQLite instead of MySQL. The modern WordPress database schema instead of the original's rough edges. But the same conviction: a blog is text.

```
press serve
```

You're blogging.

## Getting Started

```bash
# Generate your config
press config generate

# Set up the database
press migrate up
press db seed

# Start writing
press serve
```

## CLI

The binary is called `press`. WordPress forgot the press. We didn't.

```
press serve                        # Start the blog
press migrate up                   # Run database migrations
press config generate              # Generate .env with a fresh secret key
press db seed                      # Seed default data
press db reset                     # Drop everything and start over

press user create bob bob@b.com    # Create a user (generates password)
press user create bob bob@b.com --pass=hunter2
press user list
press user delete 2
press user delete 2 --reassign=1   # Reassign their posts first

press post list
press post delete 42

press option get blogname
press option set blogname "My Blog"
press option list

press secret generate              # Print a random secret key
```

## Design Principles

- **WordPress 1.0 is sacred ground.** Every feature maps back to the original. If WP 1.0 didn't do it, we probably don't either.
- **Fix the obviously broken stuff.** MD5 passwords and GeoURL are not worth preserving out of nostalgia.
- **Go idioms, not PHP port.** This is not a line-by-line translation. Single binary, `html/template`, goroutines, in-process caching.
- **SQLite. Single binary. Zero dependencies.** Run `press serve` and you're blogging.

## What Press Will Never Be

- A CMS
- A page builder
- A full site editor
- A marketplace
- A React app
- Gutenberg

## Tech Stack

| WordPress | Press |
|-----------|-------|
| PHP | Go |
| MySQL | SQLite |
| Apache/Nginx | Built-in |
| wp-config.php | .env |
| 1,200 files | One binary |

## Current Status

Press is under active development. We're rebuilding the WordPress 1.0 feature set from the ground up. The database schema is stable. The repositories are clean. The CLI works. The web server is next.

If you're the kind of person who thinks the best version of WordPress was the one that just published blog posts, you're in the right place.

## License

GPL-2.0, same as the original. Some things are worth keeping.
