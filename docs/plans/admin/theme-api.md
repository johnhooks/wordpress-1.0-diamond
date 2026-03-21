# Admin Theme API

**Status: Initial draft.** This defines the contract between Press
and an admin theme. What the theme must provide, what data it
receives, and what the engine does with the templates.

The admin is the second theme surface. Same rendering architecture
as the frontend. Different component vocabulary.

---

## The Contract

An admin theme must provide a set of named templates. The engine
calls them by name via `ExecuteTemplate`. Each template receives a
typed data struct as its context. The template renders HTML using
that data however it wants.

The admin theme controls the admin experience the same way the
frontend theme controls the reading experience. Different theme,
different layout, different workflow. Same engine underneath.

---

## Required Templates

### Full Page Templates

| Template Name | Handler | Description |
|---|---|---|
| `admin-login` | `GET /wp-admin/login` | Login form |
| `admin-write` | `GET /wp-admin/post/new` | New post form |
| `admin-edit` | `GET /wp-admin/post/{id}/edit` | Edit existing post |
| `admin-posts` | `GET /wp-admin/posts` | Post listing |
| `admin-comments` | `GET /wp-admin/comments` | Comment listing |
| `admin-moderate` | `GET /wp-admin/moderate` | Comment moderation queue |
| `admin-categories` | `GET /wp-admin/categories` | Category management |
| `admin-users` | `GET /wp-admin/users` | User management |
| `admin-options` | `GET /wp-admin/options` | Site settings |
| `admin-profile` | `GET /wp-admin/profile` | Current user profile |

### Fragment Templates

| Template Name | Handler | Description |
|---|---|---|
| `admin-post-row` | various | Single row in post table |
| `admin-comment-row` | various | Single row in comment table |
| `admin-category-row` | `POST /wp-admin/categories` | Single row in category table |
| `admin-category-form` | various | Category create/edit form |

### Total: 14 Required Templates

---

## View Data

### AdminShell

Shared context embedded in every admin page template.

| Prop | Type | Description |
|---|---|---|
| `SiteName` | string | Blog title |
| `SiteURL` | string | Frontend URL |
| `CurrentUser` | AdminUser | Logged-in user |
| `CurrentPage` | string | Active page identifier |
| `MenuItems` | []MenuItem | Navigation items (filtered by access) |
| `Version` | string | Press version |

Where `AdminUser` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | User ID |
| `Login` | string | Username |
| `DisplayName` | string | Display name |
| `Level` | int | User level (0-10) |
| `ProfileURL` | string | Profile page URL |
| `LogoutURL` | string | Logout URL |

Where `MenuItem` is:

| Field | Type | Description |
|---|---|---|
| `Label` | string | Display text |
| `URL` | string | Page URL |
| `IsCurrent` | bool | Whether this is the active page |

### admin-login

| Prop | Type | Description |
|---|---|---|
| `SiteName` | string | Blog title |
| `SiteURL` | string | Frontend URL |
| `Error` | string | Login error message (empty if none) |
| `RedirectTo` | string | Where to go after login |
| `CanRegister` | bool | Whether registration is open |

### admin-write

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Categories` | []CategoryOption | All categories for the picker |

Where `CategoryOption` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | Term taxonomy ID |
| `Name` | string | Category name |
| `Slug` | string | Category slug |
| `Selected` | bool | Whether pre-selected |

### admin-edit

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Post` | AdminPostView | The post being edited |
| `Categories` | []CategoryOption | All categories (with selections) |
| `CanDelete` | bool | Whether user can delete this post |
| `CanChangeDate` | bool | Whether user can edit the timestamp |

Where `AdminPostView` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | Post ID |
| `Title` | string | Post title |
| `Content` | string | Raw content (ProseMirror JSON) |
| `Excerpt` | string | Post excerpt |
| `Status` | string | draft, publish, private |
| `CommentStatus` | string | open, closed |
| `Slug` | string | Post slug |
| `Date` | time | Publication date |
| `Permalink` | string | Public URL |

### admin-posts

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Posts` | []AdminPostRow | Posts for current page |
| `SearchQuery` | string | Current search query |
| `CategoryFilter` | int | Current category filter |
| `StatusFilter` | string | Current status filter |
| `Categories` | []CategoryOption | For the filter dropdown |
| `HasPrev` | bool | Newer page exists |
| `HasNext` | bool | Older page exists |
| `PrevURL` | string | URL for newer entries |
| `NextURL` | string | URL for older entries |
| `CurrentPage` | int | Current page number |
| `TotalPages` | int | Total pages |

Where `AdminPostRow` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | Post ID |
| `Title` | string | Post title |
| `AuthorName` | string | Author display name |
| `Categories` | string | Comma-separated category names |
| `Date` | string | Formatted date |
| `Status` | string | Post status |
| `CommentCount` | int | Number of comments |
| `EditURL` | string | Edit page URL |
| `DeleteURL` | string | Delete action URL |
| `ViewURL` | string | Frontend permalink |

### admin-comments

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Comments` | []AdminCommentRow | Comments for current page |
| `PostFilter` | int | Filter by post ID (0 = all) |
| `HasPrev` | bool | Newer page exists |
| `HasNext` | bool | Older page exists |
| `PrevURL` | string | URL for newer entries |
| `NextURL` | string | URL for older entries |

Where `AdminCommentRow` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | Comment ID |
| `Author` | string | Commenter name |
| `Email` | string | Commenter email |
| `Content` | string | Comment text (truncated for list) |
| `PostTitle` | string | Parent post title |
| `PostEditURL` | string | Link to parent post edit page |
| `Date` | string | Formatted date |
| `Status` | string | Approval status |
| `EditURL` | string | Edit URL |
| `DeleteURL` | string | Delete URL |
| `ApproveURL` | string | Approve action URL (empty if approved) |
| `UnapproveURL` | string | Unapprove action URL (empty if not) |

### admin-moderate

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Comments` | []ModerationComment | Unapproved comments |
| `ApprovedCount` | int | Count from last moderation (0 if fresh) |
| `DeletedCount` | int | Count from last moderation |

Where `ModerationComment` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | Comment ID |
| `Author` | string | Commenter name |
| `Email` | string | Commenter email |
| `URL` | string | Commenter website |
| `IP` | string | Commenter IP |
| `Content` | HTML | Full comment body |
| `PostTitle` | string | Parent post title |
| `Date` | string | Formatted date |
| `EditURL` | string | Edit URL |
| `DeleteURL` | string | Delete URL |

### admin-categories

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Categories` | []AdminCategoryRow | All categories |
| `Editing` | *AdminCategoryRow | Category being edited (nil if creating) |

Where `AdminCategoryRow` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | Term ID |
| `Name` | string | Category name |
| `Slug` | string | URL slug |
| `Description` | string | Category description |
| `Count` | int | Post count |
| `EditURL` | string | Edit URL |
| `DeleteURL` | string | Delete URL |
| `IsDefault` | bool | Whether this is the default category |

### admin-users

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Authors` | []AdminUserRow | Users with level > 0 |
| `Subscribers` | []AdminUserRow | Users with level = 0 |
| `CurrentUserLevel` | int | For showing/hiding promote controls |

Where `AdminUserRow` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | User ID |
| `Login` | string | Username |
| `DisplayName` | string | Display name |
| `Email` | string | Email address |
| `URL` | string | Website |
| `Level` | int | User level |
| `PostCount` | int | Number of posts |
| `CanPromote` | bool | Can current user promote this user |
| `CanDemote` | bool | Can current user demote this user |
| `CanDelete` | bool | Can current user delete this user |
| `PromoteURL` | string | Promote action URL |
| `DemoteURL` | string | Demote action URL |
| `DeleteURL` | string | Delete action URL |

### admin-options

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `Groups` | []OptionGroup | Option groups |
| `CurrentGroup` | string | Selected group slug |
| `Options` | []AdminOption | Options in current group |
| `Saved` | bool | Whether options were just saved |

Where `AdminOption` is:

| Field | Type | Description |
|---|---|---|
| `Name` | string | Option name |
| `Value` | string | Current value |
| `Type` | string | Input type (text, textarea, radio, select) |
| `Label` | string | Display label |
| `Choices` | []string | For radio/select types |

### admin-profile

| Prop | Type | Description |
|---|---|---|
| `AdminShell` | | Embedded shell data |
| `User` | ProfileView | Current user data |
| `Updated` | bool | Whether profile was just updated |

Where `ProfileView` is:

| Field | Type | Description |
|---|---|---|
| `ID` | int | User ID |
| `Login` | string | Username (read-only) |
| `FirstName` | string | First name |
| `LastName` | string | Last name |
| `Nickname` | string | Nickname |
| `Email` | string | Email (required) |
| `URL` | string | Website |
| `DisplayNameAs` | string | Current display mode |
| `DisplayNameOptions` | []string | Available display modes |
| `Bio` | string | User description |

---

## What the Admin Theme Controls

Everything visual. The admin theme decides:

- The layout of every page
- The HTML structure of forms, tables, and navigation
- The CSS classes, methodology, and visual design
- The order and grouping of form fields
- Whether the sidebar exists, and what it contains
- How the menu is rendered (horizontal bar, vertical list, whatever)

One admin theme can look like WP 1.0. Another can look like a modern
dashboard. Another can be a single-column mobile-first writing tool.
The engine provides the same data and wires the same behavior.

## What the Engine Controls

Everything behavioral:

- Authentication and session management
- Form field names and hidden fields
- htmx attributes for interactive updates
- Form endpoints and redirect targets
- Permission checks (what data reaches the template)
- The ProseMirror editor (web component, fully engine-owned)

---

## Admin URLs

| URL | Method | Description |
|---|---|---|
| `/wp-admin/login` | GET | Login form |
| `/wp-admin/login` | POST | Login submission |
| `/wp-admin/logout` | GET | Logout |
| `/wp-admin/` | GET | Dashboard (redirect to write) |
| `/wp-admin/post/new` | GET | Write new post |
| `/wp-admin/post/new` | POST | Save new post |
| `/wp-admin/post/{id}/edit` | GET | Edit post |
| `/wp-admin/post/{id}/edit` | POST | Update post |
| `/wp-admin/post/{id}/delete` | POST | Delete post |
| `/wp-admin/posts` | GET | Post listing |
| `/wp-admin/comments` | GET | Comment listing |
| `/wp-admin/comment/{id}/edit` | GET | Edit comment |
| `/wp-admin/comment/{id}/edit` | POST | Update comment |
| `/wp-admin/comment/{id}/delete` | POST | Delete comment |
| `/wp-admin/comment/{id}/approve` | POST | Approve comment |
| `/wp-admin/moderate` | GET | Moderation queue |
| `/wp-admin/moderate` | POST | Process moderation |
| `/wp-admin/categories` | GET | Category management |
| `/wp-admin/categories` | POST | Create/update category |
| `/wp-admin/category/{id}/delete` | POST | Delete category |
| `/wp-admin/users` | GET | User management |
| `/wp-admin/users` | POST | Create user |
| `/wp-admin/user/{id}/promote` | POST | Promote user |
| `/wp-admin/user/{id}/demote` | POST | Demote user |
| `/wp-admin/user/{id}/delete` | POST | Delete user |
| `/wp-admin/options` | GET | Options page |
| `/wp-admin/options` | POST | Save options |
| `/wp-admin/profile` | GET | Profile page |
| `/wp-admin/profile` | POST | Update profile |
