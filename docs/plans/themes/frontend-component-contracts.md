# Frontend Component Contracts

**Status: Proposal.** This is a first draft of the component API for
the Press frontend theme system. Every prop, action, and region listed
here is a best guess based on what WordPress 1.0–1.5 needed to render
a blog. The actual contracts will change — probably significantly — as
we implement the skinning system and discover what's missing, what's
unnecessary, and what's shaped wrong.

This document covers the public blog only. Admin components are a
separate vocabulary and a separate problem.

---

## How to Read These Contracts

Each component lists:

- **Props** — data the theme can render. These are values the engine
  provides to the compiled template.
- **Actions** — interactive behaviors the engine wires up. The theme
  places an element with a `press:action` attribute; the compiler
  turns it into the appropriate htmx or form markup.
- **Regions** — areas the engine may need to target for content swaps.
  The theme places an element with a `press:region` attribute; the
  compiler assigns it an ID the engine knows how to target.
- **Context** — which page types this component appears on.

Props marked with `?` are optional — the data may not exist (e.g., a
post with no categories). The theme should handle their absence.

---

## Page Shell

These components appear on every page.

### SiteHeader

The blog's identity. Title and description.

| Prop              | Type   | Description  |
| ----------------- | ------ | ------------ |
| `SiteName`        | string | Blog title   |
| `SiteDescription` | string | Blog tagline |
| `SiteURL`         | string | Home URL     |
| `FeedURL`         | string | RSS feed URL |

**Actions:** None.
**Regions:** None.
**Context:** Every page.

### SiteFooter

Credit line and meta.

| Prop              | Type   | Description           |
| ----------------- | ------ | --------------------- |
| `SiteName`        | string | Blog title            |
| `FeedURL`         | string | RSS feed URL          |
| `CommentsFeedURL` | string | Comments RSS feed URL |

**Actions:** None.
**Regions:** None.
**Context:** Every page.

### Sidebar

Container for sidebar widgets. The sidebar itself isn't a single
component — it's a region where the theme places individual widget
components in whatever order it chooses.

**Regions:**

| Region    | Description                                           |
| --------- | ----------------------------------------------------- |
| `sidebar` | The sidebar area, targetable for full sidebar refresh |

**Context:** Every page (though a theme may choose to omit it).

---

## Sidebar Widgets

Each widget is an independent component. The theme places them in the
sidebar in any order, or anywhere else on the page.

### SearchForm

| Prop     | Type   | Description                                                   |
| -------- | ------ | ------------------------------------------------------------- |
| `Query?` | string | Current search query (pre-fills input on search results page) |

**Actions:**

| Action   | Trigger     | Description          |
| -------- | ----------- | -------------------- |
| `search` | form submit | Submits search query |

**Fields:**

| Field   | Name | Type       | Description      |
| ------- | ---- | ---------- | ---------------- |
| `query` | `s`  | text input | The search terms |

**Regions:** None.
**Context:** Every page (typically sidebar). Also embedded in search results and 404.

### CategoryList

| Prop               | Type       | Description                                      |
| ------------------ | ---------- | ------------------------------------------------ |
| `Categories`       | []Category | All categories                                   |
| `CurrentCategory?` | string     | Active category slug (on category archive pages) |

Where `Category` is:

| Field     | Type   | Description                          |
| --------- | ------ | ------------------------------------ |
| `Name`    | string | Display name                         |
| `Slug`    | string | URL slug                             |
| `URL`     | string | Category archive URL                 |
| `Count`   | int    | Number of posts                      |
| `Parent?` | string | Parent category slug (for hierarchy) |

**Actions:** None (plain links).
**Regions:** None.
**Context:** Every page (typically sidebar).

### ArchiveList

| Prop              | Type           | Description                                   |
| ----------------- | -------------- | --------------------------------------------- |
| `Archives`        | []ArchiveMonth | Monthly archives                              |
| `CurrentArchive?` | string         | Active archive period (on date archive pages) |

Where `ArchiveMonth` is:

| Field   | Type   | Description                    |
| ------- | ------ | ------------------------------ |
| `Label` | string | Display label ("January 2004") |
| `URL`   | string | Month archive URL              |
| `Count` | int    | Number of posts                |

**Actions:** None (plain links).
**Regions:** None.
**Context:** Every page (typically sidebar).

### PageList

| Prop    | Type       | Description     |
| ------- | ---------- | --------------- |
| `Pages` | []PageLink | Published pages |

Where `PageLink` is:

| Field   | Type   | Description |
| ------- | ------ | ----------- |
| `Title` | string | Page title  |
| `URL`   | string | Page URL    |

**Actions:** None (plain links).
**Regions:** None.
**Context:** Every page (typically sidebar).

### MetaLinks

| Prop              | Type   | Description                                                   |
| ----------------- | ------ | ------------------------------------------------------------- |
| `FeedURL`         | string | RSS feed URL                                                  |
| `CommentsFeedURL` | string | Comments RSS feed URL                                         |
| `IsLoggedIn`      | bool   | Whether current user is authenticated                         |
| `LoginURL`        | string | Login page URL                                                |
| `LogoutURL?`      | string | Logout URL (only when logged in)                              |
| `AdminURL?`       | string | Admin URL (only when logged in)                               |
| `RegisterURL?`    | string | Registration URL (only when logged out, if registration open) |

**Actions:** None (plain links).
**Regions:** None.
**Context:** Every page (typically sidebar).

### SidebarContext

The contextual orientation message for archive and search pages.

| Prop       | Type   | Description                                                 |
| ---------- | ------ | ----------------------------------------------------------- |
| `Message?` | string | Contextual message ("You are browsing the 'News' category") |

**Actions:** None.
**Regions:** None.
**Context:** Archive and search pages. Absent on home and single post.

---

## Content Components

### Post

A single post. Appears in two modes: as a list item (home, archives,
search) and as the full single post view. The same component, different
context.

| Prop           | Type           | Description                           |
| -------------- | -------------- | ------------------------------------- |
| `ID`           | int            | Post ID                               |
| `Title`        | string         | Post title                            |
| `Content`      | HTML           | Rendered post body                    |
| `Excerpt`      | string         | Plain text excerpt                    |
| `Permalink`    | string         | Canonical URL                         |
| `Date`         | string         | Formatted publication date            |
| `Time`         | string         | Formatted publication time            |
| `AuthorName`   | string         | Display name of author                |
| `AuthorURL?`   | string         | Author archive URL                    |
| `Categories`   | []CategoryLink | Post's categories as links            |
| `CommentCount` | int            | Number of approved comments           |
| `CommentsOpen` | bool           | Whether comments are accepted         |
| `EditURL?`     | string         | Edit link (only for authorized users) |

Where `CategoryLink` is:

| Field  | Type   | Description           |
| ------ | ------ | --------------------- |
| `Name` | string | Category display name |
| `URL`  | string | Category archive URL  |

**Actions:** None (post display is read-only).
**Regions:** None.

**Context:**

- Home page: list mode, may use Content or Excerpt
- Archives/search: list mode, typically Excerpt
- Single post page: full mode, always Content

### Page

A static page. Similar to Post but without blog metadata.

| Prop       | Type   | Description                           |
| ---------- | ------ | ------------------------------------- |
| `ID`       | int    | Page ID                               |
| `Title`    | string | Page title                            |
| `Content`  | HTML   | Rendered page body                    |
| `EditURL?` | string | Edit link (only for authorized users) |

**Actions:** None.
**Regions:** None.
**Context:** Static page view.

### CommentList

The list of comments on a post.

| Prop           | Type      | Description                                                    |
| -------------- | --------- | -------------------------------------------------------------- |
| `Comments`     | []Comment | Approved comments                                              |
| `PostTitle`    | string    | Title of the post (for "N responses to 'Title'")               |
| `CommentCount` | int       | Total approved comments                                        |
| `Order`        | string    | `"oldest"` or `"newest"` — engine needs this for swap strategy |

Where `Comment` is:

| Field        | Type   | Description                                 |
| ------------ | ------ | ------------------------------------------- |
| `ID`         | int    | Comment ID                                  |
| `Author`     | string | Commenter display name                      |
| `AuthorURL?` | string | Commenter website                           |
| `Date`       | string | Formatted comment date                      |
| `Content`    | HTML   | Rendered comment body                       |
| `Type`       | string | `"comment"`, `"trackback"`, or `"pingback"` |
| `EditURL?`   | string | Edit link (only for authorized users)       |

**Actions:** None (display only; the form is a separate component).

**Regions:**

| Region         | Description                                       |
| -------------- | ------------------------------------------------- |
| `comment-list` | The list container — swap target for new comments |

**Context:** Single post page.

### CommentForm

The form for submitting a comment.

| Prop               | Type   | Description                        |
| ------------------ | ------ | ---------------------------------- |
| `PostID`           | int    | ID of the post being commented on  |
| `CommentsOpen`     | bool   | Whether comments are accepted      |
| `RequireNameEmail` | bool   | Whether name/email are required    |
| `SavedAuthor?`     | string | Pre-filled author from cookie      |
| `SavedEmail?`      | string | Pre-filled email from cookie       |
| `SavedURL?`        | string | Pre-filled website URL from cookie |

**Actions:**

| Action   | Trigger     | Description         |
| -------- | ----------- | ------------------- |
| `submit` | form submit | Submits the comment |

**Fields:**

| Field     | Name      | Type       | Description       |
| --------- | --------- | ---------- | ----------------- |
| `author`  | `author`  | text input | Commenter name    |
| `email`   | `email`   | text input | Commenter email   |
| `url`     | `url`     | text input | Commenter website |
| `comment` | `comment` | textarea   | Comment body      |

**Regions:** None.
**Context:** Single post page, when CommentsOpen is true.

### NotFoundMessage

The 404 page content.

| Prop   | Type | Description |
| ------ | ---- | ----------- |
| (none) |      |             |

This component has no dynamic data. The theme provides the error
message and typically embeds a SearchForm.

**Actions:** None.
**Regions:** None.
**Context:** 404 page.

---

## Navigation Components

### Pagination

Previous/next navigation for paged post lists.

| Prop          | Type   | Description                  |
| ------------- | ------ | ---------------------------- |
| `HasPrev`     | bool   | Whether a newer page exists  |
| `HasNext`     | bool   | Whether an older page exists |
| `PrevURL`     | string | URL for newer entries        |
| `NextURL`     | string | URL for older entries        |
| `CurrentPage` | int    | Current page number          |
| `TotalPages`  | int    | Total number of pages        |

**Actions:** None (plain links; htmx enhancement is engine-controlled).
**Regions:** None.
**Context:** Home, archives, search results.

### PostNavigation

Previous/next navigation between individual posts on the single post
page.

| Prop        | Type     | Description           |
| ----------- | -------- | --------------------- |
| `PrevPost?` | PostLink | Previous (older) post |
| `NextPost?` | PostLink | Next (newer) post     |

Where `PostLink` is:

| Field   | Type   | Description    |
| ------- | ------ | -------------- |
| `Title` | string | Post title     |
| `URL`   | string | Post permalink |

**Actions:** None (plain links).
**Regions:** None.
**Context:** Single post page.

### PageNavigation

Navigation within a multi-page post (posts using `<!--nextpage-->`).

| Prop          | Type      | Description  |
| ------------- | --------- | ------------ |
| `Pages`       | []PageNum | Page numbers |
| `CurrentPage` | int       | Active page  |

Where `PageNum` is:

| Field       | Type   | Description                     |
| ----------- | ------ | ------------------------------- |
| `Number`    | int    | Page number                     |
| `URL`       | string | URL for this page               |
| `IsCurrent` | bool   | Whether this is the active page |

**Actions:** None (plain links).
**Regions:** None.
**Context:** Single post page, only when the post has multiple pages.

---

## Archive Headers

Archive pages need a heading that describes what the user is looking at.

### ArchiveHeader

| Prop           | Type   | Description                                       |
| -------------- | ------ | ------------------------------------------------- |
| `Title`        | string | The archive heading text                          |
| `Description?` | string | Optional description (e.g., category description) |

The engine provides a pre-formatted title based on the archive type:

- Category: "Archive for the 'News' Category"
- Month: "Archive for January 2004"
- Year: "Archive for 2004"
- Day: "Archive for January 3rd, 2004"
- Author: "Posts by AuthorName"
- Search: "Search Results for 'query'"

**Actions:** None.
**Regions:** None.
**Context:** All archive and search pages.

---

## Page Templates

Page templates compose components into full pages. These are not
components themselves — they're the theme's arrangement of components
for each page type.

| Template    | Required Components                                            | Optional Components                               |
| ----------- | -------------------------------------------------------------- | ------------------------------------------------- |
| Home        | SiteHeader, Post (loop), Pagination, SiteFooter                | Sidebar + widgets, ArchiveHeader                  |
| Single Post | SiteHeader, Post, CommentList, CommentForm, SiteFooter         | PostNavigation, PageNavigation, Sidebar + widgets |
| Static Page | SiteHeader, Page, SiteFooter                                   | Sidebar + widgets                                 |
| Archive     | SiteHeader, ArchiveHeader, Post (loop), Pagination, SiteFooter | Sidebar + widgets, SidebarContext                 |
| Search      | SiteHeader, ArchiveHeader, Post (loop), Pagination, SiteFooter | SearchForm, Sidebar + widgets, SidebarContext     |
| 404         | SiteHeader, NotFoundMessage, SiteFooter                        | SearchForm, Sidebar + widgets                     |

"Required" means the page doesn't make sense without it. "Optional"
means the theme can include it or not. The engine doesn't enforce this
— a theme can omit anything. The test bench would flag missing required
components as warnings.

---

## What's Not Here

Things deliberately excluded from this first pass:

- **Calendar widget.** WP 1.0 had it, but it's complex and rarely
  useful. May add later if we miss it.
- **Links/Blogroll widget.** WP 1.0 had a full link manager. This is
  a feature that needs its own data model first.
- **Trackback URI display.** Trackbacks are largely dead. If we
  support them, the prop gets added to Post.
- **Comment popup.** WP 1.0 had a popup window for comments. Not
  relevant.
- **Password-protected post handling.** Real requirement but adds
  complexity to Post and CommentList contracts. Deferred.
- **Multi-page post content.** The `<!--nextpage-->` feature requires
  content splitting in the engine. Deferred.
- **Date grouping.** WP 1.0 grouped posts under date headings. Nice
  feature, but it means posts aren't independent list items — the
  loop needs date-break awareness. Needs thought.

---

## Relationship to Current Code

The existing view structs in `views.go` (`PostView`, `CommentView`,
`HomeData`, `SingleData`) are a rough sketch of these contracts. They
cover the Post and Comment props partially and the Home/Single page
data. They'll evolve toward these contracts as the skinning system
takes shape.

What exists today that maps to these contracts:

| Contract        | Current Code                                   | Gap                                                                |
| --------------- | ---------------------------------------------- | ------------------------------------------------------------------ |
| Post props      | `PostView` struct                              | Missing: Time, AuthorURL, CommentsOpen, EditURL, CategoryLink URLs |
| Comment props   | `CommentView` struct                           | Missing: Type, EditURL                                             |
| CommentList     | Part of `SingleData`                           | Missing: Order, PostTitle for heading                              |
| CommentForm     | Not implemented                                | Everything                                                         |
| Pagination      | Part of `HomeData`                             | Missing: PrevURL, NextURL (has page numbers instead)               |
| PostNavigation  | Not implemented                                | Everything                                                         |
| SearchForm      | Not implemented                                | Everything                                                         |
| CategoryList    | Not implemented                                | Data exists in repository, not wired to templates                  |
| ArchiveList     | Not implemented                                | Query exists, not wired to templates                               |
| PageList        | Not implemented                                | Query exists, not wired to templates                               |
| MetaLinks       | Not implemented                                | Everything                                                         |
| SiteHeader      | Partial (BlogName, no description in template) | Missing: SiteDescription in template                               |
| SiteFooter      | Partial                                        | Missing: feed URLs                                                 |
| ArchiveHeader   | Not implemented                                | Everything                                                         |
| SidebarContext  | Not implemented                                | Everything                                                         |
| NotFoundMessage | Not implemented                                | Everything                                                         |
| Page            | Not implemented                                | Everything                                                         |

---

## Next Steps

These contracts are a starting point for conversation, not a
specification. The next step is to build the simplest possible theme
against them and discover what's wrong. Props will be missing. Types
will be awkward. The component boundaries will be in the wrong places.
That's the point — we find the right API by trying to use the wrong
one.
