# WordPress 1.0 Diamond

## What This Is

A ground-up reimplementation of WordPress 1.0 "Platinum" (January 2004) in a modern language and runtime. The goal is to produce something that is **functionally equivalent** to the original and **visually identical** on the surface, but built with modern security practices, better architecture, and contemporary tooling underneath.

This is not a port. We are not translating PHP line-by-line. We are building a modern async web server that serves pages indistinguishable from the original WordPress 1.0, using whatever advancements the chosen language and ecosystem provide. If there is a better way to accomplish something, we do that. The constraint is the surface — it looks and behaves like WordPress 1.0 Platinum. The implementation underneath is 2026.

## Why This Exists

WordPress 1.0 is small enough to fully understand (~22K lines of PHP, 78 files, 10 database tables) but complete enough to be interesting — posts, comments, categories, users, feeds, an admin interface, an XML-RPC API, a blogroll system, text processing pipelines, and import tools. It's a real application with real features, not a toy.

Rebuilding it from scratch is an exercise in understanding what WordPress actually does at its core, making deliberate architectural decisions about each subsystem, and producing something that works identically but is more secure, more testable, and better thought through.

## Reference Codebases

### WordPress 1.0 Platinum (Primary Reference)
**Location:** `~/Projects/wordpress-1.0-platinum`

This is the source material. Every feature we implement should trace back to behavior observable in the original. Read the PHP, understand what it does, then build the modern equivalent.

### WordPress Develop (Modern Reference)
**Location:** `~/Projects/wordpress-develop`

Before implementing any backend component, consult the modern WordPress codebase to understand how the same problem evolved over 20+ years. This is not about copying modern WordPress — it's about being informed. If WordPress eventually solved a problem in a smarter way, we should know about it before we design our version. Compare, learn, then make our own decision.

## Design Principles

### Visually Identical, Architecturally Modern
The HTML output, CSS, and JavaScript should produce pages that look exactly like the original WordPress 1.0. Static assets (CSS, JS, images) should be lifted directly from the original to start. We may rethink the frontend later, but not until basic implementation is complete and functional.

### Functionally Equivalent, Not a Copy
We match the original's behavior, not its implementation. If the original builds SQL with string concatenation, we use parameterized queries. If the original stores passwords as MD5, we use bcrypt. If the original has an XSS vulnerability in comment rendering, we don't faithfully reproduce the XSS. Same behavior, better implementation.

### Drop What Doesn't Make Sense
Some components of a 2004 blog platform are irrelevant in 2026. If a feature doesn't make sense (blog-by-email via POP3 polling? trackback/pingback to a dead ecosystem?), we can drop it. Document what was dropped and why, but don't build dead functionality for the sake of completeness.

### Test-Driven From the Start
The original has zero tests. We write tests first. Use the PHP codebase as the specification — parse this content, expect this output. Submit this form, expect this database state. Request this feed, expect this XML. The Go (or whatever we choose) implementation is built against these behavioral tests.

### Security By Default
- Parameterized SQL queries, never string concatenation
- bcrypt for password hashing (not MD5)
- Template auto-escaping for HTML output
- CSRF protection on all form submissions
- Proper session management (not raw cookie values)
- SQLite as a default option alongside MySQL

## Language: Go

See `docs/plans/go-lang.md` for the full rationale and expected struggles.

## Project Phases

### Phase 0: Research and Documentation
Generate thorough documentation about every subsystem in WordPress 1.0. Understand the original before building anything. For each subsystem, document:
- What the original does (behavior specification)
- How the original does it (PHP implementation notes)
- How modern WordPress evolved it (consult wordpress-develop)
- How we plan to implement it (language-agnostic design, then language-specific)

### Phase 1: Core Read Path
The minimum viable blog — serve posts on the frontend. This proves out the full stack: database, routing, templates, and the text processing pipeline.

### Phase 2: Admin Interface
Post creation and editing, basic settings. The 38-page admin UI is the bulk of the work.

### Phase 3: Comments and Interaction
Comment submission, moderation, display. The feedback loop that makes it a blog platform.

### Phase 4: Feeds and API
RSS 2.0, Atom, XML-RPC. The syndication layer.

### Phase 5: Everything Else
Users, categories, blogroll, imports, file uploads. The features that round out the platform.

## What We're Not Doing

- Not building a modern CMS or competing with current WordPress
- Not adding features that didn't exist in 1.0 (no REST API, no blocks, no plugins beyond what 1.0 had)
- Not optimizing for production deployment (this is educational/exploratory)
- Not faithfully reproducing security vulnerabilities from 2004

## Repository Structure (Planned)

```
wordpress-1.0-diamond/
├── CLAUDE.md              # This file
├── docs/                  # Research and design documentation
│   ├── original/          # Documentation of WordPress 1.0 subsystems
│   ├── modern/            # Notes from wordpress-develop comparison
│   └── design/            # Our implementation designs
├── static/                # CSS, JS, images lifted from original
├── templates/             # HTML templates (converted from PHP)
├── ...                    # Implementation (structure TBD with language choice)
└── tests/                 # Behavioral tests derived from original
```
