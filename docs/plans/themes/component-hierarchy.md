# Component Hierarchy

**Status: Proposal.** Organizing the frontend components into an atomic
design hierarchy — atoms, molecules, organisms, templates. Each level
compiles into independent render functions that call down to the level
below. Any function can be called on its own for htmx fragment
rendering, or composed into a full page.

This hierarchy will change as we build against it. The boundaries
between atoms, molecules, and organisms are judgment calls that we'll
get wrong in places.

---

## Why This Matters

The hierarchy determines two things:

1. **Compilation order.** Atoms compile first (no dependencies),
   molecules compile next (call atom functions), and so on up. Each
   level's compiled functions are available to the level above.

2. **htmx granularity.** Any compiled function is a valid fragment
   endpoint. The engine can return a full page, an organism, a molecule,
   or even an atom. The theme defines the visual design at each level;
   the engine decides which level to render based on the request.

```
Request: GET /              → Template (full page)
Request: HX-GET /           → Template content area (organisms)
Request: HX-POST /comments  → Molecule (single comment)
```

Same compiled functions, different entry points.

---

## Atoms

Atoms are not compiled functions. They are props — pieces of data that
the theme renders inline inside its own markup. The theme author chooses
the HTML element, the class name, the position. The atom is just the
value.

```html
<!-- The theme renders the PostDate atom however it wants -->
<time class="post__date">{{.Date}}</time>

<!-- Another theme might do it differently -->
<span class="entry-date" style="color: gray;">{{.Date}}</span>
```

Atoms exist in the documentation so theme authors know what data is
available. They are not in the compiler.

| Atom             | Prop              | Description                  |
| ---------------- | ----------------- | ---------------------------- |
| Site name        | `SiteName`        | Blog title                   |
| Site description | `SiteDescription` | Blog tagline                 |
| Site URL         | `SiteURL`         | Home URL                     |
| Post title       | `Title`           | Post/page title              |
| Post date        | `Date`            | Formatted publication date   |
| Post time        | `Time`            | Formatted publication time   |
| Post content     | `Content`         | Rendered HTML body           |
| Post excerpt     | `Excerpt`         | Plain text excerpt           |
| Permalink        | `Permalink`       | Canonical post URL           |
| Author name      | `AuthorName`      | Display name                 |
| Author URL       | `AuthorURL?`      | Author archive link          |
| Comment count    | `CommentCount`    | Number of approved comments  |
| Edit URL         | `EditURL?`        | Admin edit link (auth-gated) |
| Category name    | `Category.Name`   | Display name                 |
| Category URL     | `Category.URL`    | Category archive link        |
| Feed URL         | `FeedURL`         | RSS/Atom feed link           |

---

## Molecules

The smallest compiled functions. A molecule is a self-contained piece
of UI built from atoms (inline props) inside theme-authored markup.
Molecules are the floor for htmx fragment rendering — the smallest
thing the engine would swap independently.

Each molecule compiles into a render function that can be called on
its own (for fragment responses) or called by an organism (for full
section renders).

### PostMeta

The metadata line beneath a post title. The theme renders date, time,
author, categories, and edit link in whatever arrangement it wants.

**Props:** Date, Time, AuthorName, AuthorURL?, Categories[], EditURL?

### PostFeedback

The comment count and interaction area at the bottom of a post.

**Props:** CommentCount, Permalink, CommentsOpen

### Comment

A single comment. The smallest unit in the comment list — when a new
comment is submitted, the engine renders one Comment molecule and
appends/prepends it.

**Props:** ID, Author, AuthorURL?, Date, Content (HTML), Type, EditURL?

**Regions:**

| Region         | Description                                    |
| -------------- | ---------------------------------------------- |
| `comment-{ID}` | Individual comment, targetable for edit/update |

### SearchForm

Search input and submit button.

**Props:** Query?
**Actions:** `search` (form submit)

### CommentFormFields

The input fields for submitting a comment. The organism (CommentForm)
handles the form element and action wiring; this molecule handles the
visible fields.

**Props:** RequireNameEmail, SavedAuthor?, SavedEmail?, SavedURL?

---

## Organisms

Distinct page sections. Organisms call molecule functions and render
atoms inline. This is the primary htmx swap level — most fragment
updates target an organism.

Each organism compiles into a render function that can be called
independently (htmx fragment) or by a template (full page).

### SiteHeader

Renders site identity. Atoms (SiteName, SiteDescription) are inline.

**Props:** SiteName, SiteDescription, SiteURL, FeedURL

### Post

A complete post. Calls `PostMeta` and `PostFeedback` molecule
functions. Title, content, and excerpt are atoms rendered inline.

**Props:** The full Post contract from the component contracts doc.

**Calls:** `PostMeta`, `PostFeedback`

**Regions:**

| Region      | Description                             |
| ----------- | --------------------------------------- |
| `post-{ID}` | The entire post, targetable for refresh |

### CommentList

The comment section. Calls the `Comment` molecule in a loop.

**Props:** Comments[], PostTitle, CommentCount, Order

**Calls:** `Comment` (per comment)

**Regions:**

| Region         | Description                                       |
| -------------- | ------------------------------------------------- |
| `comment-list` | The list container — swap target for new comments |

### CommentForm

The comment submission form. Calls `CommentFormFields` molecule.

**Props:** PostID, CommentsOpen, RequireNameEmail, SavedAuthor?,
SavedEmail?, SavedURL?

**Calls:** `CommentFormFields`

**Actions:** `submit` (engine wires endpoint and target to
`comment-list` region)

### Sidebar

Container for sidebar widgets. Calls widget organisms/molecules.

**Calls:** `SearchForm`, `CategoryList`, `ArchiveList`, `PageList`,
`MetaLinks`, `SidebarContext`

**Regions:**

| Region    | Description                          |
| --------- | ------------------------------------ |
| `sidebar` | Full sidebar, targetable for refresh |

### CategoryList

Loop over categories, rendering each as atoms inline.

**Props:** Categories[], CurrentCategory?

### ArchiveList

Loop over monthly archives, rendering each as atoms inline.

**Props:** Archives[], CurrentArchive?

### PageList

Loop over pages, rendering each as atoms inline.

**Props:** Pages[]

### MetaLinks

Feed links, login/logout, registration. All atoms rendered inline.

**Props:** FeedURL, CommentsFeedURL, IsLoggedIn, LoginURL, LogoutURL?,
AdminURL?, RegisterURL?

### SidebarContext

Contextual message for archive/search pages. Single atom inline.

**Props:** Message?

### Pagination

Previous/next page links. Atoms rendered inline.

**Props:** HasPrev, HasNext, PrevURL, NextURL, CurrentPage, TotalPages

### PostNavigation

Previous/next post links. Atoms rendered inline.

**Props:** PrevPost? (Title, URL), NextPost? (Title, URL)

### ArchiveHeader

Archive page heading and optional description. Atoms inline.

**Props:** Title, Description?

### Page

A static page. Like Post but without metadata, categories, or comments.
Title and content atoms rendered inline.

**Props:** ID, Title, Content (HTML), EditURL?

**Regions:**

| Region      | Description                             |
| ----------- | --------------------------------------- |
| `page-{ID}` | The entire page, targetable for refresh |

### SiteFooter

Credit line and feed links. Atoms inline.

**Props:** SiteName, FeedURL, CommentsFeedURL

---

## Templates

Templates compose organisms into full pages. They define the overall
page structure — which organisms appear, in what order, inside what
HTML wrapper.

A template is the top-level compiled function. On a full page request,
the engine calls the template. On an htmx request, the engine calls
the specific organism or molecule that needs updating.

### Home

```
SiteHeader
  Post (loop)
  Pagination
Sidebar
SiteFooter
```

### Single Post

```
SiteHeader
  Post
  CommentList
  CommentForm
Sidebar (optional — theme's choice)
SiteFooter
```

### Static Page

```
SiteHeader
  Page content
SiteFooter
```

### Archive (Category, Date, Author)

```
SiteHeader
  ArchiveHeader
  Post (loop — excerpts)
  Pagination
Sidebar
  SidebarContext
SiteFooter
```

### Search Results

```
SiteHeader
  ArchiveHeader ("Search Results for 'query'")
  Post (loop — excerpts)
  Pagination
Sidebar
  SidebarContext
  SearchForm (pre-filled)
SiteFooter
```

### 404

```
SiteHeader
  NotFoundMessage
  SearchForm
Sidebar
SiteFooter
```

---

## Compilation Order

The compiler processes bottom-up:

```
1. Molecules  — no function dependencies, compile first
                (atoms are just inline props, not function calls)
2. Organisms  — depend on molecule functions
3. Templates  — depend on organism functions
```

Each compiled function is registered by name. Higher levels call lower
levels by name. After compilation, every function is independently
callable.

The engine maintains a registry of compiled functions. For a full page
request, it calls a template function. For an htmx fragment request, it
looks up the specific function by name and calls it directly with the
appropriate props.

Rough function count for the frontend:

```
Molecules:  ~5  (PostMeta, PostFeedback, Comment, SearchForm, CommentFormFields)
Organisms:  ~15 (Post, Page, CommentList, CommentForm, Sidebar,
                 CategoryList, ArchiveList, PageList, MetaLinks,
                 SidebarContext, Pagination, PostNavigation,
                 ArchiveHeader, SiteHeader, SiteFooter)
Templates:  ~6  (Home, Single, Page, Archive, Search, 404)
Total:      ~26 compiled functions
```

---

## Fragment Rendering

This is where atomic hierarchy meets htmx.

| User Action     | Engine Response         | Function Called                  |
| --------------- | ----------------------- | -------------------------------- |
| Full page load  | Complete HTML document  | Template                         |
| htmx navigation | Content area swap       | Template content (organisms)     |
| Submit comment  | Single comment appended | `Comment` molecule               |
| Search          | Content area swap       | Post loop + Pagination organisms |
| Refresh sidebar | Sidebar HTML            | `Sidebar` organism               |

The theme author doesn't think about fragments. They write templates
that compose organisms that compose molecules that compose atoms. The
engine decides which level to render based on the request type. The
same visual design, the same markup, the same classes — just a
different entry point into the same function tree.

---

## Open Questions

- **Where exactly do the molecule/organism boundaries fall?** Is
  `CategoryList` an organism or a molecule? Is `PostMeta` a molecule
  or just inline markup within the Post organism? We'll discover the
  right boundaries by building real themes.

- **How does the theme author express this hierarchy?** Separate files
  per component? Naming conventions? Directory structure
  (`molecules/`, `organisms/`, `templates/`)? Or does the compiler
  infer the hierarchy from `press:component` nesting in the markup?

- **Are some organisms too small to be organisms?** `SidebarContext`
  is a one-prop component. `ArchiveHeader` is two props. Are these
  really worth being compiled functions, or are they just inline
  markup in the template? Building will tell us.

- **How granular should htmx targeting be?** Molecules are the floor.
  Can we swap a single Comment molecule inside a CommentList organism?
  That requires the molecule's region to be independently targetable
  even when it was rendered as part of a loop. Doable but adds
  complexity.
