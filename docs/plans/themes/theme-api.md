# Theme API

**Status: Initial draft.** This defines the contract between Press and
a theme. What the theme must provide, what data it receives, and what
the engine does with the templates. This will expand as we build out
the system — more fragment endpoints, more view data, more required
templates.

---

## The Contract

A theme must provide a set of named templates. The engine calls them
by name via `ExecuteTemplate`. Each template receives a typed data
struct as its context. The template renders HTML using that data however
it wants — any structure, any classes, any CSS.

Everything beyond the required templates is the theme's internal
business. The theme can split markup into molecules, organisms, partials,
includes — whatever organizational scheme it prefers. The engine never
calls those internal templates. It only calls the required set.

---

## Required Templates

These are the templates the engine calls directly. A theme must provide
all of them.

### Full Page Templates

Called by route handlers for full page renders.

| Template Name | Handler | Description |
|---|---|---|
| `home` | `GET /` | Homepage — post list with pagination |
| `single` | `GET /{permalink}` | Single post with comments |
| `page` | `GET /{page-slug}` | Static page |
| `archive` | `GET /category/*`, `GET /2004/01/*`, etc. | Archive listing (category, date, author) |
| `search` | `GET /?s=query` | Search results |
| `404` | any unmatched route | Not found |

### Fragment Templates

Called by action handlers for htmx partial responses. These are also
used inside full page templates for composition — same template, two
entry points.

| Template Name | Handler | Description |
|---|---|---|
| `post` | `GET /fragments/posts?page=N` | One post — used in full pages and load-more |
| `comment` | `POST /comments` | One comment — used in full pages, submit, and load-more |
| `comment-form` | `POST /comments` (swap) | Comment form — used in full pages and post-submit refresh |

### Total: 9 Required Templates

This set will grow. Known future additions:

- Taxonomy item fragments (category, tag, series) for sidebar and
  filtering
- Sidebar fragment for full sidebar refresh
- Pagination fragment if pagination becomes htmx-driven
- Post navigation fragment
- Any new htmx endpoint adds a required template

---

## View Data

Each required template receives a data struct. These are the props
available to the template. The template uses `{{.FieldName}}` to
access them and `{{if .FieldName}}` for conditionals.

### home

| Prop | Type | Description |
|---|---|---|
| `BlogName` | string | Site title |
| `BlogDescription` | string | Site tagline |
| `PageTitle` | string | Page title for `<title>` tag (empty on home) |
| `Posts` | []PostView | Posts for the current page |
| `HasPrev` | bool | Whether a newer page exists |
| `HasNext` | bool | Whether an older page exists |
| `PrevURL` | string | URL for newer entries |
| `NextURL` | string | URL for older entries |
| `CurrentPage` | int | Current page number |
| `TotalPages` | int | Total pages |
| `Categories` | []CategoryView | All categories (sidebar) |
| `Archives` | []ArchiveView | Monthly archives (sidebar) |
| `Pages` | []PageLink | Published pages (sidebar) |
| `IsLoggedIn` | bool | Whether current user is authenticated |
| `LoginURL` | string | Login page URL |
| `LogoutURL` | string | Logout URL |
| `AdminURL` | string | Admin URL |
| `SearchQuery` | string | Current search query (empty on home) |

### single

| Prop | Type | Description |
|---|---|---|
| `BlogName` | string | Site title |
| `BlogDescription` | string | Site tagline |
| `PageTitle` | string | Post title |
| `Post` | PostView | The post |
| `Comments` | []CommentView | Approved comments |
| `CommentsOpen` | bool | Whether comments are accepted |
| `SavedAuthor` | string | Pre-filled commenter name (cookie) |
| `SavedEmail` | string | Pre-filled commenter email (cookie) |
| `SavedURL` | string | Pre-filled commenter URL (cookie) |
| `PrevPost` | *PostLink | Previous (older) post, nil if none |
| `NextPost` | *PostLink | Next (newer) post, nil if none |
| `Categories` | []CategoryView | All categories (sidebar) |
| `Archives` | []ArchiveView | Monthly archives (sidebar) |
| `Pages` | []PageLink | Published pages (sidebar) |
| `IsLoggedIn` | bool | Auth state |
| `LoginURL` | string | Login page URL |
| `LogoutURL` | string | Logout URL |
| `AdminURL` | string | Admin URL |

### page

| Prop | Type | Description |
|---|---|---|
| `BlogName` | string | Site title |
| `BlogDescription` | string | Site tagline |
| `PageTitle` | string | Page title |
| `Page` | PageView | The page |
| `IsLoggedIn` | bool | Auth state |
| `LoginURL` | string | Login page URL |
| `LogoutURL` | string | Logout URL |
| `AdminURL` | string | Admin URL |

### archive

| Prop | Type | Description |
|---|---|---|
| `BlogName` | string | Site title |
| `BlogDescription` | string | Site tagline |
| `PageTitle` | string | Archive description for `<title>` |
| `ArchiveTitle` | string | Display heading ("Archive for January 2004") |
| `ArchiveDescription` | string | Optional description (e.g., category description) |
| `Posts` | []PostView | Posts for the current page |
| `HasPrev` | bool | Newer page exists |
| `HasNext` | bool | Older page exists |
| `PrevURL` | string | URL for newer entries |
| `NextURL` | string | URL for older entries |
| `CurrentPage` | int | Current page number |
| `TotalPages` | int | Total pages |
| `Categories` | []CategoryView | All categories (sidebar) |
| `Archives` | []ArchiveView | Monthly archives (sidebar) |
| `Pages` | []PageLink | Published pages (sidebar) |
| `SidebarContext` | string | Contextual message for sidebar |
| `IsLoggedIn` | bool | Auth state |
| `LoginURL` | string | Login page URL |
| `LogoutURL` | string | Logout URL |
| `AdminURL` | string | Admin URL |

### search

Same as `archive`, plus:

| Prop | Type | Description |
|---|---|---|
| `SearchQuery` | string | The search terms |

### 404

| Prop | Type | Description |
|---|---|---|
| `BlogName` | string | Site title |
| `BlogDescription` | string | Site tagline |
| `PageTitle` | string | "Not Found" |
| `Categories` | []CategoryView | All categories (sidebar) |
| `Archives` | []ArchiveView | Monthly archives (sidebar) |
| `Pages` | []PageLink | Published pages (sidebar) |
| `IsLoggedIn` | bool | Auth state |
| `LoginURL` | string | Login page URL |
| `SearchQuery` | string | Empty string |

### post (fragment)

Receives a single PostView. Used inside `{{range .Posts}}` in full
page templates and called directly for load-more fragments.

| Prop | Type | Description |
|---|---|---|
| `ID` | int | Post ID |
| `TheTitle` | string | Post title |
| `TheContent` | HTML | Rendered post body |
| `TheExcerpt` | string | Plain text excerpt |
| `Permalink` | string | Canonical URL |
| `TheDate` | string | Formatted publication date |
| `TheTime` | string | Formatted publication time |
| `TheAuthor` | string | Display name |
| `AuthorURL` | string | Author archive URL |
| `TheCategories` | []CategoryLink | Post's categories as links |
| `CommentCount` | int | Number of approved comments |
| `CommentsOpen` | bool | Whether comments are accepted |
| `EditURL` | string | Edit link, empty if not authorized |

### comment (fragment)

Receives a single CommentView. Used inside `{{range .Comments}}` and
called directly after comment submission or load-more.

| Prop | Type | Description |
|---|---|---|
| `ID` | int | Comment ID |
| `TheAuthor` | string | Commenter name |
| `URL` | string | Commenter website |
| `TheDate` | string | Formatted date |
| `TheContent` | HTML | Rendered comment body |
| `Type` | string | "comment", "trackback", or "pingback" |
| `EditURL` | string | Edit link, empty if not authorized |

### comment-form (fragment)

Receives the comment form context. Used in single post pages and
swapped after submission.

| Prop | Type | Description |
|---|---|---|
| `Post` | PostView | The post (for post ID) |
| `CommentsOpen` | bool | Whether comments are accepted |
| `SavedAuthor` | string | Pre-filled name |
| `SavedEmail` | string | Pre-filled email |
| `SavedURL` | string | Pre-filled website |

---

## Shared Sub-Types

### PostView

See `post` fragment above.

### CommentView

See `comment` fragment above.

### CategoryView (sidebar)

| Field | Type | Description |
|---|---|---|
| `Name` | string | Category display name |
| `Slug` | string | URL slug |
| `URL` | string | Category archive URL |
| `Count` | int | Number of posts |

### CategoryLink (post metadata)

| Field | Type | Description |
|---|---|---|
| `Name` | string | Category display name |
| `URL` | string | Category archive URL |

### ArchiveView (sidebar)

| Field | Type | Description |
|---|---|---|
| `Label` | string | Display label ("January 2004") |
| `URL` | string | Month archive URL |
| `Count` | int | Number of posts |

### PageLink (sidebar + post navigation)

| Field | Type | Description |
|---|---|---|
| `Title` | string | Page or post title |
| `URL` | string | URL |

### PageView (static pages)

| Field | Type | Description |
|---|---|---|
| `ID` | int | Page ID |
| `TheTitle` | string | Page title |
| `TheContent` | HTML | Rendered page body |
| `EditURL` | string | Edit link, empty if not authorized |

---

## What the Theme Controls

Everything visual. Given the data above, the theme decides:

- What HTML elements to use
- What CSS classes to apply
- What order to render components in
- Whether to include optional sections (sidebar, pagination, etc.)
- How to split templates internally (molecules, organisms, flat — up
  to the theme)
- What CSS methodology to use (BEM, Tailwind, inline, none)

The freerange theme demonstrates one organization strategy (molecules/
organisms/templates directories). A different theme could put everything
in 9 flat files. The engine calls the same 9 template names either way.

---

## What the Engine Controls

Everything behavioral:

- htmx attributes (hx-post, hx-target, hx-swap) — the theme writes
  these in its templates for now; the future compiler will inject them
- IDs for swap targets — the theme assigns these for now
- Hidden form fields — the theme includes these for now
- Form field names — the theme writes these for now
- Permission resolution — the handler precomputes, the template
  receives results as props

"For now" means the freerange theme hand-writes what the future theme
compiler will automate. The templates are the compiler's target output.

---

## Template Functions

The engine registers a funcmap with utility functions available to all
templates.

| Function | Signature | Description |
|---|---|---|
| `categories` | `func([]string) string` | Joins category names with commas, returns "Uncategorized" if empty |
| `add` | `func(int, int) int` | Addition |
| `sub` | `func(int, int) int` | Subtraction |

This will grow. Known future additions:

- Date formatting helpers
- URL builders
- Pluralization ("1 Comment" vs "2 Comments")
- Truncation / excerpt generation
- HTML sanitization helpers
