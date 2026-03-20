# Frontend Components

What does a 2000s blog need to render? This document catalogs every UI
component required to support the feature set of WordPress 1.0 through
1.5 — the baseline for Press. Not how things are styled. What they are.

Derived from `wordpress-1.0-platinum` and `wordpress-develop` at 1.5.

---

## Page Types

A blog serves these distinct views. Each is a full HTML page composed
from shared components (header, sidebar, footer) plus page-specific
content.

| Page | WP 1.0 | WP 1.5 | Description |
|---|---|---|---|
| Home | index.php | index.php | Recent posts, paginated |
| Single Post | index.php (inline) | single.php | One post with comments |
| Static Page | — | page.php | Title + content, no metadata |
| Category Archive | index.php (filtered) | archive.php | Posts in a category |
| Date Archive | index.php (filtered) | archive.php | Posts by year/month/day |
| Author Archive | — | archive.php | Posts by author |
| Search Results | index.php (filtered) | search.php | Posts matching a query |
| 404 | — | 404.php | Not found |

Every page type shares the same shell: header, content area, sidebar,
footer. The content area changes; the shell does not.

---

## Page Shell

The shell wraps every page. Four components, always present, always in
the same order.

### Site Header

- Blog title (links to home)
- Blog description/tagline

WP 1.0 rendered these as an `<h1>` and nothing else. WP 1.5 added a
description line and a header image area. Both are minimal — no
navigation menu, no search bar in the header. The header is identity,
not navigation.

### Sidebar

The sidebar is the blog's navigation and discovery surface. It appears
on every page. Contents:

| Widget | WP 1.0 | WP 1.5 | Data Source |
|---|---|---|---|
| Search | text input + button | same | form → `?s=query` |
| Categories | flat list, linked | hierarchical list | `wp_terms` |
| Archives | monthly links with counts | same | `wp_posts` grouped by month |
| Calendar | table, days with posts linked | same | `wp_posts` by day |
| Links/Blogroll | categorized external links | same | `wp_links` |
| Meta | RSS, login, register, validator | same + logout | `wp_options`, auth state |
| Pages | — | list of static pages | `wp_posts` where type=page |

**Contextual sidebar** (WP 1.5 addition): On archive and search pages,
the sidebar shows a message like "You are currently browsing the
archives for the 'News' category" or "You searched for 'topic'." This
gives the user orientation.

### Site Footer

- "Powered by Press" credit line
- Optional: page generation time, feed links

WP 1.0's footer was a colored bar with a credit. WP 1.5 added feed
links. Both are one line of text.

---

## Post Components

A post is the core content unit. It appears in two contexts: as one of
many in a list (home, archives, search), or as a single focused view.

### Post in a List

When posts appear in a list (home, category, date, search), each shows:

| Element | Description | WP 1.0 | WP 1.5 |
|---|---|---|---|
| Date heading | Groups posts by day | `the_date()` as h2 | `the_time()` inline |
| Title | Post title, linked to permalink | h3 `.storytitle` | h2 with id |
| Meta line | Author, time, categories, edit link | below title | below title |
| Content | Full post body (home) or excerpt (archives) | `the_content()` | `the_excerpt()` on archives |
| Feedback | Comment count link | "Comments (N)" | "N Comments »" |

**the_content vs the_excerpt**: WP 1.0 always showed full content. WP
1.5 introduced `the_excerpt()` for archive views — showing a trimmed
version. The home page still shows full content. This is a reasonable
default: full on home, excerpt on archives/search.

**Date grouping**: WP 1.0 rendered date headings that only appeared once
per day — if three posts shared a date, one heading covered all three.
WP 1.5 moved the date inline into each post's metadata. Both approaches
work; date grouping is more distinctive.

### Single Post

When viewing one post, the display includes everything from the list
view plus:

| Element | Description |
|---|---|
| Full content | Always the complete post body, never excerpt |
| Page links | Navigation within multi-page posts (`<!--nextpage-->`) |
| Post navigation | Links to previous/next post |
| Trackback URI | URL for sending trackbacks (if open) |
| Comment feed | RSS link for this post's comments |
| Comments section | Full comment display + form (see below) |

### Post Metadata

The metadata line beneath each post title contains:

- **Categories**: Comma-separated, each linked to its category archive
- **Author**: Display name (WP 1.0 showed this; WP 1.5 commented it out in Kubrick but the tag existed)
- **Time**: Publication time
- **Edit link**: Visible only to logged-in users with permission

Format: `Filed under: Category1, Category2 — AuthorName @ 3:45 pm [Edit]`

---

## Comment Components

Comments appear on single post pages only.

### Comment List

| Element | Description |
|---|---|
| Heading | "N Responses to 'Post Title'" (WP 1.5) or "Comments" (WP 1.0) |
| Comment body | The comment text, with allowed HTML rendered |
| Comment author | Name, optionally linked to their URL |
| Comment date | Date and time of comment |
| Comment type | Comment, Trackback, or Pingback (WP 1.5 distinguished these) |
| Edit link | Visible to admins |

WP 1.0 rendered comments as an ordered list. WP 1.5 added alternating
background colors (`.graybox` on every other comment). Both used `<ol>`
with `<li id="comment-{ID}">`.

### Comment Form

Appears when comments are open. Fields:

| Field | Required | Populated From |
|---|---|---|
| Name | Yes | Cookie (if returning commenter) |
| Email | Yes | Cookie (not displayed publicly) |
| Website URL | No | Cookie |
| Comment text | Yes | — |
| Submit button | — | "Say it!" (WP 1.0) / "Submit Comment" (WP 1.5) |

Hidden fields: `comment_post_ID`, `redirect_to`.

### Comment States

- **Open**: Form shown, comments accepted
- **Closed**: "Comments are closed" message, no form
- **Password-protected**: "Enter your password to view comments"
- **Moderation**: "Your comment is awaiting moderation" notice

---

## Navigation Components

### Pagination (Post Lists)

Previous/next links for navigating through pages of posts on home,
archives, and search results.

- « Previous Entries / Next Entries »
- Appears at top and/or bottom of post list

### Post Navigation (Single Post)

Links to the chronologically adjacent posts.

- « Previous Post Title
- Next Post Title »

WP 1.0 didn't have this. WP 1.5 added `previous_post()` /
`next_post()` on single post pages.

### Multi-Page Post Navigation

For posts split with `<!--nextpage-->`:

- "Pages: 1 2 3" with current page unlinked

---

## Search

### Search Form

- Text input for query
- Submit button ("Search" or "Go!")
- GET form posting to home URL with `?s=query`
- Lives in the sidebar, also shown on search results and 404 pages

### Search Results Page

- Heading: "Search Results"
- Filtered post list using excerpt display
- Standard pagination
- Sidebar with context: "You have searched for 'query'"

---

## Archive Pages

### Category Archive

- Heading: "Archive for the 'Category Name' Category"
- Filtered post list (excerpts)
- Sidebar with context message
- Standard pagination

### Date Archive

Three granularities:

| Type | Heading Example |
|---|---|
| Year | "Archive for 2004" |
| Month | "Archive for January 2004" |
| Day | "Archive for January 3rd, 2004" |

- Filtered post list (excerpts)
- Sidebar with context message
- Standard pagination

### Author Archive

- Heading: "Author Archive"
- Filtered post list (excerpts)
- Standard pagination

---

## Static Pages

Pages are content without the blog metadata. No date, no categories, no
author line, no comments (by default). Just title and content.

- Full-width layout (WP 1.5 hid the sidebar on pages)
- Listed in sidebar under "Pages" widget
- Support for `<!--nextpage-->` breaks

---

## Feed Components

Not rendered as HTML but served as XML:

| Feed | Path | Content |
|---|---|---|
| RSS 2.0 | `/feed/rss2` | Recent posts |
| Atom | `/feed/atom` | Recent posts |
| Comments RSS | `/comments/feed` | Recent comments (all posts) |
| Post Comments RSS | `/post-slug/feed` | Comments on one post |

The sidebar links to these. The `<head>` includes `<link>` tags for
feed autodiscovery.

---

## Authentication-Dependent Elements

Some components change based on whether the user is logged in:

| Context | Logged Out | Logged In |
|---|---|---|
| Sidebar Meta | "Login", "Register" | "Logout", "Site Admin" |
| Post metadata | No edit link | "Edit This" link |
| Comment metadata | No edit link | "Edit" link on each comment |
| Comment form | Name/email/URL fields | Fields pre-populated or hidden |

---

## Summary: Component Inventory

### Shell (every page)
1. Site header (title + description)
2. Sidebar (search, categories, archives, calendar, links, pages, meta)
3. Footer (credit line)

### Content (varies by page type)
4. Post list (home, archives, search)
5. Single post (with full content)
6. Static page (title + content only)
7. Comment list
8. Comment form
9. 404 message

### Navigation
10. Pagination (previous/next entries)
11. Post navigation (previous/next post)
12. Multi-page post navigation
13. Category links (in metadata and sidebar)
14. Archive links (monthly, in sidebar)
15. Calendar (day links, in sidebar)

### Forms
16. Search form
17. Comment form

### Feeds
18. RSS 2.0 (posts)
19. Atom (posts)
20. Comments RSS
21. Per-post comments RSS

---

## What Press Has Today

| Component | Status |
|---|---|
| Site header | ✓ Title only, no description |
| Sidebar | ✓ Hardcoded placeholder content |
| Footer | ✓ Credit line |
| Post list (home) | ✓ With pagination |
| Single post | ✓ With comment display |
| Static page | ✗ |
| Comment form | ✗ |
| Category archive | ✗ Handler stub exists |
| Date archive | ✗ Handler stub exists |
| Author archive | ✗ |
| Search | ✗ |
| 404 page | ✗ |
| Post navigation | ✗ |
| Calendar | ✗ |
| Dynamic sidebar | ✗ Categories/archives/pages queries not wired |
| Feeds | ✗ |
| Auth-dependent UI | ✗ |
