# Admin Component Hierarchy

**Status: Proposal.** Organizing the admin frontend components into
a hierarchy that mirrors the frontend theme approach. The admin is
a second theme surface with its own component vocabulary, same
architecture.

---

## Why the Same Architecture

The frontend theme proved the pattern: templates receive typed data,
render HTML, the engine wires behavior. The admin is more complex
(more forms, more actions, more state) but the rendering model is
the same. A template receives data and produces HTML.

If we hardcode the admin HTML in the engine now, we will rewrite it
when we add the admin theme contract later. Starting with the contract
means the templates we write today are the ones we keep.

---

## Admin Shell

The shell wraps every authenticated admin page.

### AdminHeader

The site identity and navigation.

**Props:** SiteName, SiteURL, CurrentPage, MenuItems[], UserName,
ProfileURL, LogoutURL

### AdminNav

The horizontal menu. Items are filtered by the current user's access
level before reaching the template.

**Props:** Items[] (Label, URL, IsCurrent), SecondaryItems[]?

### AdminFooter

Credit line. Version number.

**Props:** Version

---

## Page Components

### WritePost

The new post form. The most important page in the admin.

**Props:** (empty form defaults)

**Fields:**

- Title (text input)
- Content (ProseMirror editor web component)
- Excerpt (textarea)
- Categories (multi-select or checkbox list)
- Post Status (radio: draft, publish, private)
- Comment Status (radio: open, closed)

**Actions:** save-draft, publish

**Regions:**

- `editor` — the ProseMirror web component mount point
- `categories` — category list, refreshable after adding a new one

### EditPost

Same as WritePost but pre-populated with existing post data.

**Props:** Post (ID, Title, Content, Excerpt, Status, CommentStatus,
Categories[], Date)

**Additional Actions:** save, delete
**Additional Fields:** Timestamp editor (if user level sufficient)

### ManagePosts

Post listing with filtering.

**Props:** Posts[], Filters (search, category, date, status),
Pagination (HasPrev, HasNext, PrevURL, NextURL, CurrentPage,
TotalPages)

**Columns:** Checkbox, Title, Author, Categories, Date, Status,
Comments (count), Actions (Edit, Delete)

**Actions:** bulk-delete, filter, search

**Regions:**

- `post-list` — the table body, swappable for filtering/pagination

### ManageComments

Comment listing.

**Props:** Comments[], PostFilter?, Pagination

**Columns:** Author, Comment (truncated), Post (linked), Date,
Status, Actions (Edit, Delete, Approve/Unapprove)

**Actions:** bulk-approve, bulk-delete

**Regions:**

- `comment-list` — swappable for filtering

### ModerateComments

The moderation queue. Unapproved comments with bulk actions.

**Props:** Comments[]

Each comment shows full detail (author, email, URL, IP, content)
with three radio options: Approve, Delete, Do Nothing.

**Actions:** moderate (bulk submit)

### CategoryManager

Combined list and create form on one page.

**Props:** Categories[] (Name, Slug, Description, Count, ParentName?),
EditingCategory?

**Actions:** create, edit, delete

**Regions:**

- `category-list` — refreshable after create/delete
- `category-form` — swaps between create and edit mode

### UserManager

User listing with create form.

**Props:** Authors[] (users with level > 0), Users[] (level = 0),
CurrentUserLevel

**Columns:** ID, Nickname, Name, Email, Level (with +/- controls),
Posts (count)

**Actions:** create, promote, demote, delete

**Regions:**

- `user-list` — refreshable after changes

### OptionsPage

Settings organized by group.

**Props:** Groups[] (Name, URL, Description), CurrentGroup?,
Options[] (Name, Value, Type, Label)

**Actions:** save (per group)

### ProfilePage

Current user's settings.

**Props:** User (FirstName, LastName, Nickname, Email, URL,
DisplayNameAs, Bio)

**Actions:** update

### LoginPage

Standalone page, no admin shell.

**Props:** Error?, RedirectTo

**Fields:** Login (text), Password (password)

**Actions:** login, lost-password

---

## Templates

The admin theme provides these required templates. Same contract
model as the frontend: the engine calls them by name via
`ExecuteTemplate`, each receives a typed data struct.

### Full Page Templates

| Template Name      | Handler                    | Description         |
| ------------------ | -------------------------- | ------------------- |
| `admin-login`      | `GET /wp-admin/login`      | Login form          |
| `admin-write`      | `GET /wp-admin/post/new`   | New post form       |
| `admin-edit`       | `GET /wp-admin/post/{id}`  | Edit post form      |
| `admin-posts`      | `GET /wp-admin/posts`      | Post listing        |
| `admin-comments`   | `GET /wp-admin/comments`   | Comment listing     |
| `admin-moderate`   | `GET /wp-admin/moderate`   | Moderation queue    |
| `admin-categories` | `GET /wp-admin/categories` | Category management |
| `admin-users`      | `GET /wp-admin/users`      | User management     |
| `admin-options`    | `GET /wp-admin/options`    | Site settings       |
| `admin-profile`    | `GET /wp-admin/profile`    | User profile        |

### Fragment Templates

| Template Name         | Handler                     | Description                      |
| --------------------- | --------------------------- | -------------------------------- |
| `admin-post-row`      | various                     | One row in the post list         |
| `admin-comment-row`   | various                     | One row in the comment list      |
| `admin-category-row`  | various                     | One row in the category list     |
| `admin-category-form` | `POST /wp-admin/categories` | Category form (create/edit swap) |

### Total: ~14 Required Templates

This will grow. The write/edit pages will likely need fragment
templates for the category picker, status controls, and the save
response.

---

## Compilation Order

Same as the frontend: templates compose components. The admin has
fewer levels of nesting because the layouts are simpler (forms and
tables, not posts-within-sidebars-within-shells).

Most admin components are flat: a page template that renders a form
or a table. The shell (header, nav, footer) wraps them. Fragment
templates handle htmx swaps for inline updates.

---

## The ProseMirror Editor

The write/edit page contains a ProseMirror editor. This is a web
component that the engine provides, not something the theme authors.
The admin theme places a `<the-editor>` element. The engine provides
the JavaScript, the initial content, the save endpoint, and the
WebSocket URL for collaboration.

The theme controls the element's position, size, and surrounding
chrome. The engine controls everything inside it.

---

## What Is Different From the Frontend

1. **Every page is authenticated.** The admin handler checks auth
   before rendering. Unauthenticated requests redirect to login.

2. **Forms are the primary interaction.** The frontend has one form
   (comment submission). The admin is almost entirely forms. The
   form action pattern (POST, process, redirect or htmx swap) is
   the core interaction model.

3. **No sidebar.** The admin has horizontal navigation, not a
   vertical sidebar with widgets. The layout is simpler.

4. **Tables for data.** Post lists, comment lists, user lists are
   all tables. The frontend uses flowing content. The admin uses
   structured tabular data.

5. **Bulk actions.** The frontend has no bulk operations. The admin
   has bulk delete, bulk moderate, bulk status change. These require
   checkbox selection and a submit-all pattern.

6. **The web component.** ProseMirror is the one piece of rich
   client-side state in the entire application. It lives exclusively
   on the write/edit page.

---

## Implementation Order

1. **Login** — everything else requires authentication
2. **Admin shell** — header, nav, footer (needed by every page)
3. **Write post** — the core loop, the reason Press exists
4. **Manage posts** — see what you have written
5. **Manage comments** — moderate the conversation
6. **Categories** — organize content
7. **Users** — manage who can write
8. **Options** — configure the blog
9. **Profile** — personal settings

Login and the shell are infrastructure. Write post is the product.
Everything after that is management tooling that supports the writing
experience.
