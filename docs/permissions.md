# Permissions

Press uses a Zanzibar-inspired tuple system for authorization. Every
permission is a tuple: `subject → relation → object`. Subjects are
users, groups, or tokens. Objects are content (posts, pages,
attachments), the site itself, or groups.

## Relations

Relations are hierarchical — a higher relation implies all lower ones
in its branch.

```
owner
├── editor          (edit/delete/publish anyone's content of a type)
│   └── writer      (create, edit/delete/publish own content)
│       └── commenter
│           └── viewer
├── moderator       (moderate comments, manage categories/links)
└── member          (group membership — separate branch, no hierarchy)
```

## Actions

Each action maps to a minimum required relation:

| Action | Required relation |
|--------|-------------------|
| view | viewer |
| comment | commenter |
| create | writer |
| edit | writer |
| delete | writer |
| publish | writer |
| upload | writer |
| moderate | moderator |
| manage_categories | moderator |
| manage_links | moderator |
| manage_options | owner |
| manage_users | owner |

## Object scopes

| Object | Example | Meaning |
|--------|---------|---------|
| `site:""` | `owner → site:""` | Site administration only (manage_options, manage_users, moderate) |
| `type:post` | `editor → type:post` | All objects of that post type |
| `type:page` | `viewer → type:page` | All pages |
| `type:attachment` | `writer → type:attachment` | All attachments |
| `post:"42"` | `writer → post:"42"` | A specific post |
| `page:"17"` | `writer → page:"17"` | A specific page |
| `attachment:"5"` | `writer → attachment:"5"` | A specific attachment |
| `group:"editors"` | `member → group:"editors"` | Group membership |

Site-level grants only cover site-level checks. They do not cascade to
content. An `owner → site:""` grant lets you manage options and users,
but to edit posts you also need `editor → type:post`.

## Ownership-sensitive actions

Edit, delete, and publish are ownership-sensitive. At the **type level**,
only `editor` or higher covers these actions. A `writer → type:post`
grant lets you create posts and view/comment/upload, but not edit or
delete anyone's posts (including your own) — that requires a direct
grant on the specific post.

Post authorship is a direct grant: `user:N → writer → post:ID`, created
when the post is written. This is how an author can edit their own post.
An editor can edit anyone's post because `editor → type:post` covers
ownership-sensitive actions.

## Default groups

| Group | Purpose |
|-------|---------|
| public | Grants that apply to everyone, including anonymous users. Not a membership — always included in every check. |
| administrators | Site owners. Full control. |
| editors | Can edit/publish anyone's content. Can moderate. |
| authors | Can create and manage their own posts. |
| subscribers | No special grants. Public group grants only. |

## Seed tuples

### Site administration

| Subject | Relation | Object | Covers |
|---------|----------|--------|--------|
| `group:administrators` | `owner` | `site:""` | manage_options, manage_users, moderate, manage_categories, manage_links |
| `group:editors` | `moderator` | `site:""` | moderate, manage_categories, manage_links |

### Content — posts

| Subject | Relation | Object | Covers |
|---------|----------|--------|--------|
| `group:public` | `viewer` | `type:post` | anyone can view posts |
| `group:public` | `commenter` | `type:post` | anyone can comment on posts |
| `group:administrators` | `editor` | `type:post` | edit/delete/publish any post |
| `group:editors` | `editor` | `type:post` | edit/delete/publish any post |
| `group:authors` | `writer` | `type:post` | create posts, edit/delete/publish own |

### Content — pages

| Subject | Relation | Object | Covers |
|---------|----------|--------|--------|
| `group:public` | `viewer` | `type:page` | anyone can view pages |
| `group:administrators` | `editor` | `type:page` | edit/delete/publish any page |
| `group:editors` | `editor` | `type:page` | edit/delete/publish any page |

### Content — attachments

| Subject | Relation | Object | Covers |
|---------|----------|--------|--------|
| `group:administrators` | `editor` | `type:attachment` | manage any attachment |
| `group:editors` | `writer` | `type:attachment` | upload, manage own |
| `group:authors` | `writer` | `type:attachment` | upload, manage own |

### Memberships

| Subject | Relation | Object |
|---------|----------|--------|
| `user:1` | `member` | `group:administrators` |

### Dynamic tuples (created by handlers)

| Subject | Relation | Object | When |
|---------|----------|--------|------|
| `user:N` | `writer` | `post:ID` | post creation (authorship) |
| `user:N` | `writer` | `page:ID` | page creation |
| `user:N` | `writer` | `attachment:ID` | file upload |

## Permissions vs content status

The permission system answers "does this user have the capability to do
this?" It does not answer "is this specific post visible right now?"
Those are separate concerns.

Post status (draft, publish, private, scheduled) is a content concern.
The type-level viewer grant (`group:public → viewer → type:post`) means
"this class of user can read published content" — not "every post is
visible." Handlers compose both:

- **Published posts:** anyone who passes the `viewer → type:post` check
  can see them.
- **Draft/private posts:** only the author (who has `writer → post:ID`),
  editors (who have `editor → type:post`), or someone with an explicit
  direct grant or share token on that specific post.
- **Scheduled posts:** same as drafts until the publish time arrives.

This separation keeps permission checks simple — they don't need to
know about post status. Per-post tuples are for authorship and sharing,
not for publication status.

## Validation

`CreateTuple` validates all fields before inserting:

| Field | Valid values |
|-------|-------------|
| Subject type | `user`, `group`, `token` |
| Object type | `site`, `type`, `post`, `page`, `attachment`, `group` |
| Relation | `member`, `viewer`, `commenter`, `writer`, `editor`, `moderator`, `owner` |

Invalid values are rejected with an error. This prevents typos
(`"eidtor"`) and invented relations (`"superadmin"`) from silently
creating dead tuples.

These sets define what the system supports today. Extending them (e.g.,
adding a new content type) means adding to the valid sets in code.

## Cleanup on deletion

When a group is deleted, all tuples where that group is subject or
object are deleted in the same transaction. No orphaned grants.

When a post, page, or attachment is deleted, all tuples referencing
that object should be deleted (`DeleteTuplesForObject`). This removes
authorship grants, share token grants, and any group-level per-object
grants.

When a user is deleted, all tuples where that user is the subject
should be deleted (`DeleteTuplesForSubject`). This removes group
memberships, authorship grants, and any direct grants.

When a share token is deleted, both the token row and its tuples are
removed in one transaction.

## Check algorithm

```
Can(userID, action, object) → bool

1. Map action to required relation
2. Group-level check (DB query):
   a. Get user's groups
   b. Always include the public group
   c. Query group grants relevant to this object:
      - Specific object match (grant on post:42 for post:42)
      - Type-level match (grant on type:post for post:42)
        → writer only covers non-ownership actions
        → editor covers all actions
      - Site-level match (grant on site:"" for site:"")
   d. Return true on first match
3. Direct grant check (DB query, specific objects only):
   a. Query tuples where subject = user and object = this object
   b. Check if any tuple's relation satisfies the required relation
4. Deny
```

Token checks (`CanToken`) validate the token exists and isn't expired,
then check the token's grant tuples on the specific object.

## Share tokens

A share token is a URL-safe string that grants a specific relation on a
specific object. Creating a token inserts both a `wp_share_tokens` row
(metadata, expiration) and a `wp_tuples` row (`token:abc → viewer →
post:42`). The token table handles validation; the tuple handles
authorization.
