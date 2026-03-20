# View Permissions

How permissions flow through the rendering system. The short version:
the server precomputes every permission check before rendering begins.
Templates receive booleans and optional URLs. They conditionally show
or hide HTML. They never grant access to anything.

---

## The Security Boundary Is the Handler

The template is not a security boundary. It never was, and it never
will be. The handler is.

When a user clicks an edit link, the `POST /edit` handler checks the
actual permission tuples and accepts or rejects the request. It doesn't
matter what the template showed. The template could render an edit
button for every visitor on earth — the handler would still reject
unauthorized requests.

This means template bugs are visual bugs, not security bugs:

- **Template shows something it shouldn't** — user sees an edit button,
  clicks it, gets a 403. Annoying, not dangerous.
- **Template hides something it should show** — user doesn't see the
  edit button even though they have permission. Missing feature, not
  a vulnerability.
- **Template error** — a permission prop is nil or wrong type, the
  template produces an empty section or an error. Visible, not
  exploitable.

The browser never sees permission data. It sees HTML with an edit link
or without one. There is nothing to tamper with.

---

## Precomputed Permission Props

The server resolves all permissions before rendering begins. The
template receives the results as props — booleans, optional URLs,
optional data. By the time `ExecuteTemplate` runs, every permission
question is answered.

The template uses `{{if}}` on these props. It has no idea a permission
system exists. It doesn't know about tuples, relations, or Zanzibar.
It sees a struct field and renders or doesn't.

```html
<!-- Template just checks a prop -->
{{if .EditURL}}<a href="{{.EditURL}}">Edit</a>{{end}}

<!-- Template doesn't know or care why EditURL is empty -->
```

The permission resolution happens in the handler:

```go
// Handler resolves permissions once, up front
perms := s.permissions.ForPostPage(ctx, userID, postID, commentIDs)

// Results go into the view struct as simple props
view := PostView{
    Title:   post.Title,
    Content: rendered,
    EditURL: perms.EditURL,   // set or empty
    // ...
}
```

---

## The Permission Set Is Closed

Press has no plugins. There is no functionality outside what we provide.
This means the set of permission checks needed to render any page is
known at compile time. We don't need a generic "can user do X"
function in templates. We know exactly what X is for every component.

For a single post page, the permission set is:

| Check | Result | Used By |
|---|---|---|
| Can user edit this post? | `EditURL` on PostView | post-meta molecule |
| Can user delete this post? | `DeleteURL` on PostView | post organism |
| Can user share this post? | `ShareURL` on PostView | post organism |
| Can user comment on this post? | `CommentsOpen` on SingleData | comment-form organism |
| Can user edit each comment? | `EditURL` on each CommentView | comment molecule |
| Can user delete each comment? | `DeleteURL` on each CommentView | comment molecule |
| Is user logged in? | `IsLoggedIn` on page data | meta-links organism |
| Can user access admin? | `AdminURL` on page data | meta-links organism |

That's the full set. No surprises. No plugin adding a "Can user
moderate?" check at runtime. The handler knows exactly what to resolve.

---

## Batch Resolution

Because the permission set is closed and known, we can batch-resolve
all checks in one pass. Instead of N+1 tuple lookups (one per comment),
the handler asks: "give me all permissions user:42 has on post:99 and
comments [1, 2, 3, 7, 15]." One query to the permission system.

The Zanzibar tuple model supports this naturally. Checking a set of
tuples is not fundamentally different from checking one — it's a
batch lookup against the relation store.

```go
// One call, all permissions for this page
perms := s.permissions.ForPostPage(ctx, userID, postID, commentIDs)

// perms contains:
//   .CanEditPost    bool
//   .CanDeletePost  bool
//   .CanSharePost   bool
//   .CanComment     bool
//   .CommentPerms   map[int64]CommentPerms  (per-comment)
```

Each page type has its own permission batch function. The home page
needs fewer checks (just edit links on each post in the list). The
single post page needs more (post permissions + per-comment
permissions). The 404 page needs almost none (just auth state for the
sidebar).

---

## Per-Page Permission Sets

| Page Type | Permission Checks |
|---|---|
| Home | IsLoggedIn, CanEdit per post in list |
| Single Post | IsLoggedIn, CanEdit/CanDelete/CanShare post, CanComment, CanEdit/CanDelete per comment |
| Static Page | IsLoggedIn, CanEdit page |
| Archive | IsLoggedIn, CanEdit per post in list |
| Search | IsLoggedIn, CanEdit per post in list |
| 404 | IsLoggedIn |

The handler calls the appropriate batch function for its page type.
No over-fetching — the 404 page doesn't resolve post permissions it
will never use.

---

## Fragment Permissions

When htmx requests a fragment, the handler resolves permissions for
that fragment only.

| Fragment | Permission Checks |
|---|---|
| Single comment (after submission) | CanEdit/CanDelete this comment |
| Comment list refresh | CanEdit/CanDelete per comment |
| Sidebar refresh | IsLoggedIn |
| Post refresh | CanEdit/CanDelete/CanShare this post |

Same principle: resolve before render, pass as props, template checks
`{{if}}`. The fragment handler is smaller, the permission set is
smaller, but the pattern is identical.

---

## What Templates See

Templates never see:

- User IDs
- Permission tuples
- Relation types
- The word "permission"

Templates only see:

- `EditURL` — a URL string or empty
- `DeleteURL` — a URL string or empty
- `ShareURL` — a URL string or empty
- `CommentsOpen` — a boolean
- `IsLoggedIn` — a boolean
- `AdminURL` — a URL string or empty
- `LoginURL` / `LogoutURL` — URL strings

These are the atoms. The template renders them or doesn't based on
`{{if}}`. The permission system is invisible to the rendering layer.

---

## Why This Works

1. **No over-fetching.** Each page type resolves exactly the permissions
   it needs. The set is known because the component set is known.

2. **No N+1.** Permissions batch-resolve in one call per page. Comment
   permissions resolve as a set, not one at a time.

3. **No template security bugs.** The template can't grant access. The
   handler validates every action independently.

4. **No permission logic in templates.** Templates check props. They
   don't evaluate policies. The permission system can change its
   internals without touching any template.

5. **Fragments work identically.** Same pattern, smaller scope. The
   permission model doesn't change based on whether you're rendering
   a full page or an htmx fragment.
