# Getting Started

How to set up a local development environment for Press.

## Requirements

- **Go 1.23+** — Press is a Go project. Install from https://go.dev/dl/.
- **GCC** — Required by the `go-sqlite3` driver, which uses cgo. Most
  Linux distributions include it. On macOS, `xcode-select --install`
  provides it.
- **Make** — The Makefile is the primary interface for development tasks.
- **air** (optional) — File watcher for automatic rebuilds during
  development. Install with `go install github.com/air-verse/air@latest`.

There is no Docker, no database server, and no JavaScript build step
required for the core engine. Node is only needed if you are working on
ProseMirror editor packages.

## Project Structure

```
cmd/press/          CLI commands (Cobra)
internal/           Engine code (server, repositories, models, permissions)
themes/             Bundled themes (freerange is the default)
local/              Example Press instance (config, database, symlinks to themes)
data/fixtures/      JSON fixture data and loader for local development
docs/               Documentation and plans
```

Press is a Go library. The `cmd/press/` directory builds the CLI binary.
The `local/` directory is a working Press instance that you develop
against. It has its own `.env`, storage directory, and theme files.

## First Time Setup

Clone the repository and set up the local environment:

```bash
git clone <repo-url>
cd press

# Copy the example environment file
cp .env.example local/.env

# Build the binary, create the database, and load fixture data
make fresh
```

`make fresh` drops the database (if it exists), runs migrations, and
loads the fixture data from `data/fixtures/`. After it completes you
have 10 users, 20 posts, 30 comments, and 5 categories. All user
passwords are `password`.

## Running the Server

With file watching (recommended during development):

```bash
make dev
```

This uses air to watch Go source files and theme templates. When you
edit a `.go`, `.html`, or `.css` file, air rebuilds the binary and
restarts the server automatically.

Without file watching:

```bash
make serve
```

The blog is at http://localhost:3000.

## The CLI

The `make press` target runs CLI commands against the local instance:

```bash
make press CMD="user list"
make press CMD="post list --format=json"
make press CMD="user get admin"
make press CMD="category list"
```

Or build the binary once and run it directly:

```bash
make build
cd local && ../press user list
```

### Entity Commands

Every entity follows the same pattern:

```bash
# Users
press user create <login> <email> [--pass=] [--role=] [--display-name=]
press user get <id-or-login>
press user list [--role=] [--format=table|json|csv|ids]
press user delete <id-or-login>

# Posts
press post create [--title=] [--content=] [--status=] [--author=] [--category=]
press post get <id-or-slug>
press post list [--status=] [--author=] [--format=table|json|csv|ids]
press post delete <id-or-slug>

# Comments
press comment create [--post=] [--content=] [--author=] [--author-email=]
press comment get <id>
press comment list [--post=] [--format=table|json|csv|ids]
press comment delete <id>

# Categories
press category create <name> [--slug=] [--parent=] [--description=]
press category list [--format=table|json|csv|ids]
press category delete <id-or-slug>

# Options
press option get <name>
press option set <name> <value>
press option list
```

All create commands support `--porcelain` which outputs just the new
entity ID. This is useful for scripting:

```bash
USER_ID=$(press user create sam sam@test.com --porcelain)
press post create --title="Sam's Post" --author=$USER_ID --porcelain
```

## Fixture Data

The local development data lives in `data/fixtures/` as JSON files:

- `users.json` — 10 users across four roles
- `categories.json` — 5 categories
- `posts.json` — 20 posts with plain text content (wrapped in
  ProseMirror JSON by the loader)
- `comments.json` — 30 comments distributed across posts

To reload fixture data after changing the JSON files:

```bash
make fresh
```

The loader is a standalone Go program at `data/fixtures/load.go`. It
reads the JSON files, clears the database, and inserts everything in
a single transaction.

## Database

Press uses SQLite. The database file lives at `local/storage/press.db`.
There is no database server to install or manage.

```bash
make reset      # drop and re-migrate (empty database)
make fresh      # drop, migrate, and load fixtures
```

Migrations are in `internal/database/migrations/` and managed by goose.

## Themes

The default theme is Freerange, located at `themes/freerange/`.
It is organized into molecules, organisms, and templates following an
atomic design hierarchy. The engine calls 9 required template names;
everything else is the theme's internal organization.

When running with `make dev`, editing any `.html` or `.css` file in
the theme directory triggers a server restart so templates are
re-parsed.

## Tests

```bash
make test
```

This runs all Go tests across the project.
