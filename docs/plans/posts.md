# Posts

Posts are the point. A person writes something, publishes it on a date, and it
enters the chronological stream. Everything else in Press exists to support
this.

## Model

Posts live in `wp_posts` with `post_type = 'post'`.

| Field          | Usage                                |
| -------------- | ------------------------------------ |
| post_type      | `'post'`                             |
| post_title     | The title                            |
| post_content   | The body — ProseMirror JSON document |
| post_excerpt   | Optional summary                     |
| post_name      | URL slug — `hello-world`             |
| post_status    | `'publish'`, `'draft'`, `'private'`  |
| post_author    | User ID of the author                |
| post_date      | Publication date                     |
| post_modified  | Last edit date                       |
| post_password  | Optional password protection         |
| comment_status | `'open'` or `'closed'`               |
| comment_count  | Denormalized comment count           |
| guid           | Permanent identifier, used in feeds  |

Posts have categories via `wp_term_relationships`. A post belongs to at least
one category (defaults to "Uncategorized"). Custom fields via `wp_postmeta`.

## URLs

Post URLs are defined by the permalink structure. See `docs/permalinks.md`.

| Form   | Example                    |
| ------ | -------------------------- |
| Pretty | `/2004/01/03/hello-world/` |
| Query  | `/?p=42`                   |

With pretty permalinks, `?p=42` redirects (301) to the canonical URL.

## Post Lifecycle

### Draft → Publish

A post starts as a draft. The author writes, previews, and publishes. On
publish, `post_status` flips to `'publish'`, `post_date` is set (or
confirmed if pre-dated), and the post enters the stream.

### Slug Generation

The slug is auto-generated from the title on first save. The author can
edit it. Duplicate slugs get a `-2`, `-3` suffix. The slug is permanent
once published — changing it would break URLs.

## Frontend

### Single post (full page load)

`GET /2004/01/03/hello-world/` — server renders the complete page: header,
post content, comments, comment form, sidebar, footer.

### Single post (htmx navigation)

Clicking a post link from the homepage or an archive swaps just the
content area — header, sidebar, and footer stay put. The link carries
both a regular `href` (full page for crawlers and JS-off) and `hx-get`
(fragment for htmx).

### Homepage (post list)

The homepage shows the most recent posts, paginated. Each post shows title,
date, author, excerpt or full content (based on settings), category list,
and comment count. Pagination via `?paged=N`, swapping just the post list
on htmx requests.

## Admin

### Post list

`GET /wp-admin/edit.php` — table of posts with title, author, categories,
date, status. Filterable by category and date. Sortable.

Each row has Edit and Delete actions via htmx — delete removes the row
without a page reload.

### Post editor

`GET /wp-admin/post-new.php` — title field, ProseMirror editor, excerpt
field, category checkboxes, slug field, status toggle.

The editor is a ProseMirror instance. Content is stored as a JSON document,
not HTML. HTML is an output format rendered at display time. A quicktags
textarea exists as a fallback for simple posts — raw input gets parsed into
a ProseMirror document on save. See `docs/plans/editor.md`.

Save via htmx — server returns the updated editor fragment with slug
preview and status indicator. No redirect, no full reload.

### Quick editing

The post list supports inline editing of title, categories, and status
without leaving the list. Clicking "Quick Edit" transforms the row into
an inline form. Save swaps it back to a normal row with updated values.

## Content Rendering

Post content is stored as ProseMirror JSON and rendered to HTML at display
time via the document schema's `toDOM` rules. No content filter pipeline.
ProseMirror's document model produces well-structured HTML by construction —
paragraph structure, heading levels, and tag balance are all in the JSON,
not inferred from raw text conventions like `wpautop` or `balanceTags`.

## Not Supported

- Block editor (Gutenberg). The editor is ProseMirror — writing, not layout.
- Post formats (aside, gallery, link, etc.). A post is a post.
- Sticky posts. Chronological order. No pinning.
- Custom post types beyond `post`, `page`, `attachment`.
