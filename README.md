# Press

Humans write words to share ideas, feelings, experiences, and stories.
They should own that content and share it how they want, whether that
is publicly, with a group, individually, or totally private just for
themselves.

Writing can be deeply collaborative. The tool should help people come
together to produce a final document they are prepared to share however
they see fit, when they want.

A self-hosted blog is accessible by the owner from anywhere in the
world without an app or payment beyond the server, and readable by
anyone, anywhere.

## What is Press?

Press is WordPress 1.6, the release that never happened.

WordPress 1.0 "Platinum" shipped in January 2004. WordPress 1.5
"Strayhorn" shipped in February 2005 and added themes, subcategories,
and static pages. Then the version number jumped to 2.0, the rich text
editor arrived, and WordPress began its long march toward becoming a
CMS, a page builder, and eventually a full site editor.

Press picks up where the 1.x line left off. It takes everything
WordPress built through 1.5 and evolves it forward as a blog. It uses
Go instead of PHP, SQLite instead of MySQL, and htmx instead of React.
The conviction is the same: a blog is text on the internet.

We never leave 1.x. There is no 2.0. There is no CMS pivot.

```
press serve
```

You're blogging.

## The Vision

Press is in its early days. We're figuring out what it means to build a
modern blogging tool that takes writing seriously. These are the ideas
we're exploring:

- **Collaborative editing.** Multiple authors can work on the same post
  in real time with full attribution. The ProseMirror editor keeps a step
  log that records every operation, including who wrote what and when.
- **Sharing as a first-class concept.** You can share a post with a link.
  The recipient can view, comment, or edit depending on the permissions
  you set. Reading does not require an account. The system is built on
  relationship tuples, not traditional roles.
- **AI writing assistance.** The AI is a collaborator, not a content
  generator. It reviews your writing, leaves comments, suggests edits,
  and has conversations about the text. It uses the same infrastructure
  as human collaboration. We have not figured out the right design yet,
  and we would rather ship nothing than ship the wrong thing.
- **Themes that are just HTML and CSS.** Themes use Go templates, not PHP
  runtimes. A theme cannot touch the filesystem or make network requests.
  The security model is structural.
- **htmx everywhere.** The server always renders HTML. It sends full pages
  on first load and fragments on navigation. The site feels like a SPA
  but every response is server-rendered. There is no client-side framework.

None of this is finished. Most of it is still in the planning stage.
The plans live in `docs/plans/`, and they change as we learn.

## Getting Started

```bash
press serve
```

Press is a single binary with a SQLite database, a built-in web server,
and zero dependencies.

## CLI

The binary is called `press`. WordPress forgot the press. We didn't.

```
press serve                        # Start the blog
press migrate up                   # Run database migrations
press db seed                      # Seed default data
press db reset                     # Drop everything and start over

press user create bob bob@b.com    # Create a user (generates password)
press user list
press user delete 2

press post list
press post delete 42

press option get blogname
press option set blogname "My Blog"
press option list

press secret generate              # Print a random secret key
```

## Tech Stack

| WordPress | Press |
|-----------|-------|
| PHP | Go |
| MySQL | SQLite |
| Apache/Nginx | Built-in |
| React (Gutenberg) | htmx |
| wp-config.php | .env |
| 1,200 files | One binary |

## Current Status

Press is under active development. We are in the early phases,
designing the systems, writing plans, and building the foundation.
The database schema is stable, the CLI works, and the permalink router
works. The editor, themes, collaboration, and sharing are ahead of us.

If you're the kind of person who thinks the best version of WordPress
was the one that just published blog posts, you might be in the right
place.

## License

GPL-2.0, same as the original. Some things are worth keeping.
