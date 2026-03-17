# Permalinks & Routing

The permalink package contains the URL router for Press. The name is a WordPress
holdover — in practice it's a full routing system that resolves every public URL
the blog serves.

## How It Works

At startup, the `permalink_structure` option is read from `wp_options`. This
single string (e.g. `/%year%/%monthnum%/%day%/%postname%/`) drives both URL
generation (`Linker`) and URL resolution (`Router`).

The Router compiles the structure into an ordered list of regex rules — one per
content type. On each request, `Resolve(path)` walks the rules top-to-bottom.
First match wins.

When the structure is empty, no rewrite rules exist. All routing falls back to
query-string parameters on `/` — the WP 1.0 default.

## Permalink Structure Tags

These tags can appear in the permalink structure string:

| Tag          | Matches             | Linker Output |
|--------------|---------------------|---------------|
| `%year%`     | 4-digit year        | Zero-padded   |
| `%monthnum%` | 1-2 digit month     | Zero-padded   |
| `%day%`      | 1-2 digit day       | Zero-padded   |
| `%hour%`     | 1-2 digit hour      | Zero-padded   |
| `%minute%`   | 1-2 digit minute    | Zero-padded   |
| `%second%`   | 1-2 digit second    | Zero-padded   |
| `%postname%` | Post slug           | As-is         |
| `%post_id%`  | Numeric post ID     | As-is         |

Common structures:

```
/%year%/%monthnum%/%day%/%postname%/    → /2004/01/03/hello-world/
/%year%/%monthnum%/%postname%/          → /2004/01/hello-world/
/%postname%/                            → /hello-world/
/archives/%post_id%/                    → /archives/42/
```

## Route Types

The Router resolves URLs into these route types. Each has a pretty-permalink
form (rewrite rules) and a query-string fallback.

### Single Post

The core route. Displays one published post with its comments.

| Form       | Example                          |
|------------|----------------------------------|
| Pretty     | `/2004/01/03/hello-world/`       |
| Query      | `/?p=42`                         |

When pretty permalinks are on, `?p=42` redirects (301) to the canonical URL.

Lookup: by `%postname%` slug or `%post_id%`. Date components in the URL are
validated against the post's actual date — a stale URL with a reused slug 404s.

### Category Archive

Posts filtered by category, paginated.

| Form       | Example                          |
|------------|----------------------------------|
| Pretty     | `/category/tech/`                |
| Query      | `/?cat=5`                        |

The pretty URL uses the "front" of the permalink structure (everything before
the first `%tag%`) plus `category/<slug>/`. So `/blog/%postname%/` produces
`/blog/category/tech/`.

### Date Archives

Posts filtered by date, paginated. The Router derives these by truncating the
permalink structure at each date tag.

**Day archive:**

| Form       | Example                          |
|------------|----------------------------------|
| Pretty     | `/2004/01/03/`                   |
| Query      | `/?m=20040103`                   |

**Month archive:**

| Form       | Example                          |
|------------|----------------------------------|
| Pretty     | `/2004/01/`                      |
| Query      | `/?m=200401`                     |

**Year archive:**

| Form       | Example                          |
|------------|----------------------------------|
| Pretty     | `/2004/`                         |
| Query      | `/?m=2004`                       |

Only generated when the structure contains the relevant date tags. A
`/%postname%/` structure produces no date archive rules.

The `?m=` parameter uses WordPress's variable-length format: `YYYY`, `YYYYMM`,
or `YYYYMMDD`.

### Search

Posts matching a search query, paginated.

| Form       | Example                          |
|------------|----------------------------------|
| Query      | `/?s=hello`                      |

No pretty URL. Search is always query-string based.

### Author Archive

Posts by a single author, paginated.

| Form       | Example                          |
|------------|----------------------------------|
| Pretty     | `/author/<nicename>/`            |
| Query      | `/?author=2`                     |

### Feed

RSS/Atom feeds for the blog and for individual post comments.

| Form       | Example                          |
|------------|----------------------------------|
| Blog feed  | `/feed/`                         |
| Post feed  | `/2004/01/03/hello-world/feed/`  |

WP 1.0 used separate PHP files (`wp-rss2.php`, `wp-atom.php`). We use `/feed/`
with content negotiation or a format parameter.

## Fixed Routes

These don't derive from the permalink structure. They're static paths registered
directly on the HTTP mux.

| Path                     | Purpose                   |
|--------------------------|---------------------------|
| `/static/`               | CSS, JS, images           |
| `/wp-login.php`          | Login                     |
| `/wp-comments-post.php`  | Comment submission (POST) |

## Pagination

Pagination uses the `?paged=N` query parameter on any list route (home,
archives, categories, search, author). This applies regardless of whether
pretty permalinks are on.

## Resolution Order

Pretty permalink resolution (rewrite rules) follows this priority:

1. **Fixed routes** — matched by the HTTP mux before the catch-all
2. **Category** — `/category/<slug>/` (literal prefix makes it unambiguous)
3. **Single post** — the full permalink structure
4. **Day archive** — structure truncated after `%day%`
5. **Month archive** — structure truncated after `%monthnum%`
6. **Year archive** — structure truncated after `%year%`

Query-string resolution (no pretty permalinks or root path with params):

1. `?p=` — single post
2. `?cat=` — category archive
3. `?m=` — date archive (length determines year/month/day)
4. `?s=` — search
5. `?author=` — author archive
6. None of the above — homepage

## Slug Collision with Route Prefixes

Page slugs can collide with fixed route prefixes like `/category/`,
`/author/`, `/feed/`, `/series/`. A page named "feed" would be
unreachable at `/feed/` because fixed routes match first.

The resolution order needs to be configurable — the site owner defines
the priority. WordPress had the same problem and solved it with
reserved slug lists. We should at minimum warn on collision, and
potentially let the admin configure the category/author/series base
prefixes to avoid conflicts.

## Not Supported

Things WordPress 1.0 had that we don't:

- **Trackbacks** — dead protocol.
- **XML-RPC** — we'll have a proper API if we need one.
- **`?w=` week archives** — nobody used these.
- **`?c=` legacy category param** — b2 holdover, WordPress itself dropped it.
- **Multi-page posts** (`?page=2`, `<!--nextpage-->`) — a post is one page.
- **`?preview=`** — preview will work differently (draft URLs, not a flag).
