# Pages

Pages are blog furniture. An "About" page, a "Contact" page, maybe a "Now"
page. They are not posts — they have no date, no category, no place in the
chronological flow. They sit outside the stream and stay put.

## Model

Pages use the same `wp_posts` table with `post_type = 'page'`.

Relevant fields:

| Field        | Usage                                       |
| ------------ | ------------------------------------------- |
| post_type    | `'page'`                                    |
| post_name    | URL slug — `/about/`, `/contact/`           |
| post_parent  | Parent page ID. One level deep, not a tree. |
| menu_order   | Sort order for page listings                |
| post_status  | `'publish'` or `'draft'`                    |
| post_title   | Page title                                  |
| post_content | Page body                                   |

Fields that don't apply to pages: `post_excerpt`, categories, tags. A page is
a title and a body at a URL.

## URLs

Pages have simple slug-based URLs:

| Form       | Example            |
| ---------- | ------------------ |
| Pretty     | `/about/`          |
| Child page | `/about/colophon/` |

The Router needs a page rule. Since page slugs can collide with post slugs
(both match `[^/]+`), page resolution happens **after** post rules fail. The
server tries the post lookup first, then falls back to a page lookup. This
matches WordPress behavior — posts win slug collisions.

With no pretty permalinks, pages still use their slug: `/?page_id=5` as the
query-string fallback.

## Frontend

### Full page load

`GET /about/` — server renders the complete page: header, page content, footer.

### htmx navigation

Clicking a page link swaps just the content area — the server returns
the page content fragment on htmx requests, the full page on normal
requests. Same handler, same data, different template scope.

## Admin

### Page list

`GET /wp-admin/edit-pages.php` — table of all pages sorted by menu_order.

Each row has Edit and Delete actions via htmx.

### Page editor

`GET /wp-admin/page-new.php` — title field, content editor, parent
dropdown, menu order field, publish/draft toggle.

Save via htmx — server returns the updated editor fragment. No redirect,
no full reload.

## Not Supported

- Deep hierarchy. One level of nesting (parent/child). No grandchildren.
- Page templates. Every page renders the same way.
- Page builders. A page is a title and a body.
- Custom fields on pages. If you need metadata, it's not a page.
- Front page settings. The homepage is always the post stream.
