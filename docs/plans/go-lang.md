# Language Choice: Go

## Decision

Go. Committed.

## Why Go

### Readability Is the Priority

This project is as much about understanding WordPress 1.0 as it is about rebuilding it. The implementation language needs to stay out of the way. Go is aggressively simple — there's usually one obvious way to do something, the control flow is always visible, and anyone reading the code six months later can follow what's happening without deciphering abstractions.

When debugging a text processing edge case where `wptexturize` is producing wrong smart quotes, the problem should be the logic, not the language. Go lets you focus on the actual problem.

### The Standard Library Covers Most of What We Need

WordPress 1.0 is fundamentally a web server that talks to a database and renders HTML. Go's standard library handles all of this without third-party frameworks:

- `net/http` — async web server with routing, middleware, request/response handling
- `html/template` — HTML templating with automatic context-aware escaping (replaces KSES)
- `database/sql` — database interface with parameterized queries (replaces raw SQL string concatenation)
- `encoding/xml` — RSS/Atom/XML-RPC feed generation and parsing
- `crypto/bcrypt` (via `golang.org/x/crypto`) — password hashing (replaces MD5)
- `net/http/cookiejar`, `net/http/cookie` — session and cookie handling
- `regexp` — regular expressions for text processing
- `testing` — built-in test framework with benchmarks

No framework to learn, no dependency tree to manage, no build system to configure. `go build` produces a single binary.

### Testing Is a First-Class Citizen

The original WordPress 1.0 has zero tests. We're building test-first. Go's testing story is excellent out of the box — `go test` runs tests, benchmarks, and examples. Table-driven tests are idiomatic and map perfectly to behavioral specifications derived from the PHP original: "given this input content, expect this output HTML."

No test runner to install, no assertion library to choose, no configuration. Write a `_test.go` file and run `go test`.

### Single Binary Deployment

`go build` produces one binary with no runtime dependencies. No PHP interpreter, no Apache/Nginx configuration, no `mod_php`, no `php.ini`. Run the binary and the blog is live. This is a meaningful improvement over the original deployment experience and keeps the project easy to share and run.

### Concurrency Model

WordPress 1.0 runs on Apache with mod_php — one process per request, blocking I/O everywhere. Go's goroutine-based concurrency means our server handles concurrent requests naturally without any special architecture. Every request gets its own goroutine, database calls don't block the server, and we don't think about thread pools or process management.

## What We Considered

### Rust

The other serious contender. Rust would provide memory safety guarantees and excellent performance. The argument against it is pragmatic, not technical: this project is about understanding and rebuilding WordPress, not about learning a new language at the same time. The borrow checker, lifetime annotations, and ownership model are things worth learning, but not while simultaneously reverse-engineering a 22K-line PHP codebase. Two steep learning curves at once turns a fun project into a grind.

If this project goes well, porting the well-documented, well-tested Go implementation to Rust would be an excellent way to learn the language — with a clear spec and test suite already in hand.

### Deno/TypeScript

JavaScript is the language of the web, and TypeScript adds type safety. Deno provides a modern runtime with built-in TypeScript support and a security-focused permission model. The argument against: this project already lives adjacent to several TypeScript/React projects (block-ssr, blocks-rsc). Using Go provides a deliberate change of context and forces different design thinking. Also, Go's simplicity and single-binary output are a better match for recreating a self-contained blog platform.

## Expected Struggles

### Regular Expressions

This is the big one. Go uses the RE2 engine, which intentionally does not support lookaheads, lookbehinds, or backreferences. WordPress 1.0's text processing pipeline (wptexturize, wpautop, BBCode parsing, Textile, smiley conversion) was written for PHP's PCRE engine, which supports all of those features.

Some of these regex patterns will need to be rewritten as multi-pass transformations or hand-rolled parsers. This is actually fine — it forces us to understand what the patterns are doing rather than blindly porting regex. But it will be the most tedious translation work in the project.

Specific areas where this will bite:
- `wptexturize` — smart quote insertion relies on context-aware lookaround
- `wpautop` — automatic paragraph wrapping with complex whitespace handling
- Textile markup — the full Textile spec is regex-heavy
- BBCode parsing — nested tag handling

### Template Verbosity

Go's `html/template` is safe by default (auto-escaping) but verbose compared to PHP's "just echo it" approach. WordPress 1.0's template functions (`the_title()`, `the_content()`, etc.) are one-liners that echo directly into the output stream. In Go, template logic requires explicit pipeline syntax and helper function registration.

The 38-page admin interface will be the most repetitive part — lots of forms, tables, and CRUD pages that need to be converted from inline PHP to Go templates. This is grunt work, not hard work, but it's a lot of it.

### PHP's Loose Typing

PHP is aggressively loosely typed. Strings become numbers, nulls become empty strings, arrays are also hashmaps. WordPress 1.0 relies on this constantly — checking if a query result is truthy, concatenating integers into SQL (which we won't do anyway), using the same variable as both a string and a number.

Go's strict typing means every one of these implicit conversions becomes an explicit decision. This is ultimately good — it forces us to understand the actual data types — but it will slow down the initial translation as we work through what types things actually are.

### Global State

WordPress 1.0 is built on PHP globals. `$wpdb`, `$post`, `$user_level`, `$tableposts` — state is everywhere and accessed from anywhere. Go doesn't have this pattern. We need to design how state flows through the application: dependency injection, context values, or structured app/request objects.

This is a design decision we'll make early and it will shape the entire codebase. The temptation will be to use Go package-level variables as a stand-in for PHP globals. We should resist that and use explicit parameter passing, even though it's more verbose.

### The Admin UI Volume

38 admin pages. Each one is a PHP file that mixes HTML, SQL, form handling, and business logic. Converting each page means:
1. Reading and understanding the PHP
2. Extracting the SQL queries
3. Building Go handler functions
4. Creating HTML templates
5. Writing tests

None of this is hard individually. Collectively it's the largest chunk of work in the project. We should expect this phase to feel slow and repetitive, and plan for it.

### XML-RPC

WordPress 1.0 includes a full XML-RPC server (2,805 lines) implementing both the MetaWeblog API and Blogger API. Go can handle XML just fine, but implementing the XML-RPC protocol from scratch (or finding a suitable library) and matching the exact API surface is fiddly work with lots of edge cases around type coercion between XML-RPC types and Go types.

### Date/Time Handling

WordPress 1.0 has its own date formatting functions that bridge MySQL's datetime format with PHP's date format strings. Go has its own idiosyncratic date formatting (the reference time `Mon Jan 2 15:04:05 MST 2006`). Translating between PHP date format strings and Go's reference-time approach will require a mapping layer, especially since the original allows users to configure date formats.
