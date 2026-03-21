# Admin Frontend Components

What does a 2004 blog admin need to render? This document catalogs
every UI component required to support the admin feature set of
WordPress 1.0 through 1.5. Not how things are styled. What they are.

Derived from `wordpress-1.0-platinum` wp-admin/ directory and
`wordpress-develop` at 1.5.

---

## Page Types

The admin serves these distinct screens. Every page shares the same
shell (header, navigation, content area, footer) with different content.

| Page | WP 1.0 | Description |
|---|---|---|
| Write Post | post.php / edit-form.php | New post form |
| Edit Post | edit-form-advanced.php | Edit existing post |
| Manage Posts | edit.php | Post list with filtering |
| Manage Comments | edit-comments.php | Comment list |
| Moderate Comments | moderation.php | Unapproved comments with bulk actions |
| Categories | categories.php | Category list + create/edit |
| Users | users.php | User list + create, level promotion |
| Options | options.php | Site settings by group |
| Profile | profile.php | Current user settings |
| Login | wp-login.php | Authentication form |

WP 1.0 had no dashboard. The admin index redirected straight to the
write post form. The write page was the home screen. That says
everything about priorities.

---

## Page Shell

The admin shell wraps every authenticated page. It is simpler than
the frontend shell. No sidebar. No footer widgets. Just navigation
and content.

### Admin Header

- WordPress wordmark (linked to wordpress.org, we link to home)
- Horizontal menu bar
- Page title (h2)

The header is identity and navigation in one strip. No description,
no tagline. The admin knows what site this is.

### Admin Navigation

A single horizontal menu. WP 1.0 loaded menu items from a
tab-delimited file (`menu.txt`) with a user level threshold per item.
Items only appeared if the current user's level met the threshold.

| Item | Level Required | Description |
|---|---|---|
| Write | 1 | New post form |
| Manage | 1 | Post/comment listing |
| Categories | 3 | Category management |
| Links | 5 | Blogroll management |
| Users | 3 | User management |
| Options | 4 | Site settings |
| Templates | 4 | Theme file editor |

Three permanent items appeared outside the dynamic menu:
- My Profile
- View Site (link to frontend)
- Logout

Some pages had a secondary navigation bar beneath the main one.
The Manage section had tabs: Latest Posts, Latest Comments, Comments
Awaiting Moderation.

### Admin Footer

One line. "WordPress" with a version number and a support link. Page
generation time. Nothing else.

---

## Write Post Page

The most important page in the admin. Where writing happens.

### New Post Form

The form posts to the same page with an action parameter. Fields
are organized in fieldsets, not a complex layout.

| Field | Name | Type | Description |
|---|---|---|---|
| Title | `post_title` | text input | Post title, auto-focused on page load |
| Categories | `post_category[]` | multi-select dropdown | Category assignment |
| Content | `content` | textarea | Post body, with quicktags toolbar above |
| Excerpt | `excerpt` | textarea (2 rows) | Optional summary |
| Post Status | `post_status` | radio buttons | publish, draft, private |
| Comment Status | `comment_status` | radio buttons | open, closed |
| Ping Status | `ping_status` | radio buttons | open, closed |
| Post Password | `post_password` | text input | Optional password protection |
| Trackback URLs | `trackback_url` | textarea | Newline-separated URLs |

Hidden fields: `user_ID`, `action` (post or editpost), `referredby`.

**Buttons:**
- Save as Draft
- Save as Private
- Publish (bold, the primary action)
- Advanced Editing (switches to edit-form-advanced.php)

### Edit Post Form

Same as the new post form plus:

- Post ID displayed in the heading ("Editing Post #42")
- Save and Continue Editing button
- Delete This Post link (for users with level > 4)
- Timestamp editor (year, month, day, hour, minute, second fields)
  visible to users with level > 4
- File upload button (if uploads enabled)

### Quicktags

A JavaScript toolbar above the content textarea. Buttons insert HTML
tags around selected text or at cursor position.

| Button | Tag |
|---|---|
| b | `<strong>` |
| i | `<em>` |
| del | `<del>` |
| a | `<a href="">` |
| img | `<img src="">` |
| blockquote | `<blockquote>` |
| code | `<code>` |
| ul | `<ul>` |
| ol | `<ol>` |
| li | `<li>` |

Press replaces quicktags with ProseMirror. The editor is a web
component that the admin theme places on the page. The engine
provides it.

---

## Manage Posts Page

Post listing with filtering and inline comment management.

### Filtering Controls

| Control | Description |
|---|---|
| Pagination | Previous/next N posts |
| Search | Text search across posts |
| Category filter | Dropdown of all categories |
| Date filter | Monthly/daily archive dropdown |
| Sort order | Newest first or oldest first |

### Post List Display

Each post renders as a block showing:

- Date and time (bold)
- Comment count (linked to inline comment view)
- Edit link
- Delete link (with confirmation dialog)
- Private indicator
- Title (linked to permalink)
- Author name and category

### Inline Comment View

Clicking the comment count on a post expands to show all comments
for that post. Each comment shows:

- Date and time
- Edit, Delete, Approve/Unapprove links
- Author name, email, URL, IP address
- Comment text

An inline comment form appears at the bottom for the admin to leave
a comment directly.

---

## Comment Moderation Page

Bulk moderation interface for unapproved comments. This is the
moderation queue.

### Comment Display

Each unapproved comment shows:
- Author name
- Email (linked)
- Website URL (linked)
- IP address (linked to ARIN lookup)
- Comment text
- Edit link
- Delete link

### Bulk Actions

Each comment has three radio buttons:
- Approve
- Delete
- Do nothing (default, checked)

A single "Moderate Comments" submit button processes all selected
actions. After processing, a summary shows how many were approved,
deleted, and ignored.

---

## Category Management Page

Combined list and create interface on one page.

### Category List

A table with columns:
- Name
- Description
- Post count
- Edit link
- Delete link (with confirmation)

The default category cannot be deleted. Deleting a category reassigns
its posts to the default category.

### Create Category Form

Two fields:
- Name (text input, required)
- Description (textarea, optional)

WP 1.0 had no parent category support. Subcategories came in 1.5.

### Edit Category Form

Same fields as create, pre-populated with existing values.

---

## User Management Page

Two sections: Authors (level > 0) and Users (level = 0).

### User List

A table with columns:
- ID
- Nickname
- Name (first + last)
- Email
- URI
- Level (with +/- promotion links)
- Post count

Level promotion/demotion rules:
- Can only change users below your own level
- Cannot promote above your own level minus one
- Level 0 users can be deleted (red X)

### Add User Form

| Field | Name | Type |
|---|---|---|
| Nickname | `user_login` | text |
| First Name | `firstname` | text |
| Last Name | `lastname` | text |
| Email | `email` | text |
| URI | `uri` | text |
| Password | `pass1` | text (not password type) |
| Password (again) | `pass2` | text |

Note: WP 1.0 displayed passwords in plain text inputs, not password
fields. We will not be doing that.

---

## Options/Settings

Two-level navigation. First level shows option groups with
descriptions. Selecting a group shows all options in that group.

### Option Groups (WP 1.0)

- General: Blog title, tagline, URL, email, membership
- Writing: Post settings, formatting
- Reading: Front page display, feed settings
- Discussion: Comment settings, moderation
- Permalinks: URL structure

Each option has an `option_admin_level` that controls which users can
see and change it. Options render different input types based on their
`option_type`.

---

## Profile Page

Current user's own settings.

| Field | Name | Description |
|---|---|---|
| First Name | `newuser_firstname` | |
| Last Name | `newuser_lastname` | |
| Nickname | `newuser_nickname` | |
| Email | `newuser_email` | Required |
| Website URI | `newuser_url` | |
| Password | `pass1` | Only if changing |
| Password (again) | `pass2` | Confirmation |
| Display name as | `newuser_idmode` | Dropdown: nickname, login, first, last, first last, last first |
| Bio | `user_description` | Textarea |

WP 1.0 also had ICQ, AIM, MSN, and Yahoo IM fields. We will not be
carrying those forward.

---

## Login Page

Standalone page, no admin shell.

### Login Form

| Field | Name | Type |
|---|---|---|
| Login | `log` | text |
| Password | `pwd` | password |

Hidden fields: `action` (login), `redirect_to` (where to go after
login).

Links: Register (if open), Lost Password, Back to Blog.

### Authentication Flow

1. User submits login and password
2. Server validates credentials against database
3. On success: set session/cookie, redirect to admin or redirect_to
4. On failure: re-display login form with error

WP 1.0 used cookies with md5-hashed passwords. Press uses bcrypt
and will use proper session management.

### Lost Password Flow

1. User enters login or email
2. Server generates new password and emails it
3. User logs in with new password

---

## Access Control Pattern

Every admin page checks user level before rendering. The pattern is
a level threshold at the top of each page handler. If the user is
below the threshold, they see an error message. There is no
capability system. There are no roles. There is a number from 0 to
10.

Press maps this to the FGA-lite permission system. The user level
numbers are preserved in usermeta for backward compatibility. The
actual access control uses group membership tuples. But the admin
surface presents it as levels, not as capabilities or roles. The user
sees "user level," not "subscriber role."

---

## Form Action Pattern

Every admin form follows the same flow:

1. Form posts to the current page with a hidden `action` field
2. Handler checks the action, validates the user level
3. Processes the form data (insert, update, or delete)
4. Redirects back to the same page or a related page
5. The redirected page displays the updated state

This is the standard POST-redirect-GET pattern. No AJAX. No partial
updates. Every action is a full page refresh.

Press replaces this with htmx where appropriate. The form posts,
the server returns a fragment, htmx swaps the updated content. The
redirect fallback still works for no-JS clients. The admin theme
does not need to know about htmx; the engine wires it.

---

## What Press Has Today

| Component | Status |
|---|---|
| Login | Not implemented |
| Write post | Not implemented |
| Edit post | Not implemented |
| Manage posts | Not implemented |
| Manage comments | Not implemented |
| Moderate comments | Not implemented |
| Categories | Not implemented |
| Users | Not implemented |
| Options | Not implemented |
| Profile | Not implemented |
| Admin shell | Not implemented |
| Admin navigation | Not implemented |
| Session/auth | Not implemented |

Everything is on the CLI. None of it is on the web.
