# Permissions

WordPress had numeric user levels (1.0), then capabilities and roles (2.0+).
Both only answered "what can this user do globally?" Press replaces all of it
with **relationship tuples** — a lightweight version of the model behind
Google Zanzibar and OpenFGA, simple enough to run in SQLite and cache in
memory.

One system handles everything: who can edit posts, who can manage users,
who can view a specific draft, and share links for people without accounts.

## Tuples

The core data structure is a tuple: **subject → relation → object.**

```
user:alice    →  member   →  group:editors
group:editors →  editor   →  type:post
user:alice    →  editor   →  post:42
token:abc123  →  viewer   →  post:42
```

Everything is a tuple. Group membership, global permissions, per-resource
grants, share links. One table, one model.

```sql
CREATE TABLE wp_tuples (
    subject_type  TEXT NOT NULL,     -- 'user', 'group', or 'token'
    subject_id    TEXT NOT NULL,     -- user ID, group ID, or token string
    relation      TEXT NOT NULL,     -- 'member', 'viewer', 'editor', 'owner', etc.
    object_type   TEXT NOT NULL,     -- 'group', 'post', 'page', 'type', 'site'
    object_id     TEXT               -- specific ID, or null for "all of this type"
);
```

A small metadata table for groups and share tokens:

```sql
CREATE TABLE wp_groups (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE wp_share_tokens (
    token       TEXT PRIMARY KEY,
    created_by  INTEGER NOT NULL,
    expires_at  DATETIME            -- null = never expires
);
```

The groups and tokens tables hold display names and expiration. All
authorization logic lives in the tuples.

## Relations

| Relation    | Meaning                                         |
| ----------- | ----------------------------------------------- |
| `member`    | User belongs to a group                         |
| `viewer`    | Can view the object                             |
| `editor`    | Can edit the object                             |
| `owner`     | Full control — edit, delete, manage permissions |
| `commenter` | Can comment (even if comments are closed)       |

## Object Types

| Object type | What it means                                  |
| ----------- | ---------------------------------------------- |
| `group`     | Target of `member` relations                   |
| `post`      | A specific post (object_id = post ID)          |
| `page`      | A specific page (object_id = page ID)          |
| `type:post` | All posts (global permission on the post type) |
| `type:page` | All pages (global permission on the page type) |
| `site`      | Site-wide permissions (manage_options, etc.)   |

## How Checks Work

**"Can alice edit post 42?"**

1. Does alice have `editor` on `post:42`? (direct grant)
2. What groups is alice a `member` of?
3. Do any of those groups have `editor` on `post:42`? (group grant on resource)
4. Do any of those groups have `editor` on `type:post`? (group grant on type)
5. Is alice the `owner` of `post:42`?

First yes wins. This is two cache lookups, not a graph traversal. The
maximum depth is always 2: user → group → permission.

**"Can token abc123 view post 42?"**

1. Is the token valid and not expired? (check wp_share_tokens)
2. Does `token:abc123` have `viewer` on `post:42`?

One cache lookup after the token validation.

## Caching

The full tuple set for a blog is small — hundreds of rows, not millions.
Load all tuples into memory at startup. Every permission check is a map
lookup. No SQL for reads.

Invalidation: any tuple write (grant, revoke, group membership change)
rebuilds the cache. Writes are rare (someone shares a post, changes a role).
Reads are constant (every request checks permissions).

Same pattern as the options cache: load once, serve from memory, write-through
invalidation.

## Default Groups

Seed data creates groups that map to WordPress's classic roles:

| Group          | Tuples                                                           |
| -------------- | ---------------------------------------------------------------- |
| Administrators | `editor` on `type:post`, `type:page`; `owner` on `site`          |
| Editors        | `editor` on `type:post`, `type:page`; `commenter` on `type:post` |
| Authors        | `editor` on own posts only (checked via post.author)             |
| Subscribers    | (no tuples — can view public content only)                       |

The "Authors can edit own posts" case is the one place where a tuple
isn't enough — you also need to check `post.author == user.id`. This is
a simple AND condition in the check, not a new abstraction.

## Visibility

Every post has a visibility level stored on the post itself:

| Visibility | Who can see it                                   |
| ---------- | ------------------------------------------------ |
| `public`   | Anyone. In feeds and the post stream.            |
| `unlisted` | Anyone with the URL. Not in feeds or the stream. |
| `shared`   | Only subjects with a `viewer` or `editor` tuple. |
| `private`  | Only the author.                                 |

Visibility is orthogonal to status. A draft can be shared (collaborators
can see and edit it). A published post can be private (only the author
sees it on their blog).

## Share Links

A share link is a tuple with `token` as the subject type:

```
token:abc123  →  viewer    →  post:42
token:def456  →  commenter →  post:42
token:ghi789  →  editor    →  post:42
```

URL: `/s/abc123` — server validates the token, finds the tuple, serves
the post with the granted permission.

- **View tokens** — read only, no account required.
- **Commenter tokens** — can read and comment, no account required.
  The token is the identity. This is how you share a post to Twitter
  and let people discuss it without signing up.
- **Edit tokens** — can read, comment, and edit. Requires login so
  we know who's editing in collaborative sessions.
- **Expiration** is optional, checked against `wp_share_tokens.expires_at`.
- **Revocation** is deleting the tuple and the token row.

No account required for viewing or commenting. The token _is_ the
permission. This is the FGA-influenced idea applied to a blog: sharing
a link IS granting access, not just sending a URL and hoping the
recipient can figure out how to log in.

## Easy Private Posts

Making a post private is setting visibility, not managing roles:

1. Write a post
2. Set visibility to `private`
3. Done — only you can see it

Sharing a private post with someone is one action:

1. Click "Share"
2. Pick a user, a group, or generate a link
3. Choose the permission level (view, comment, edit)
4. A tuple is created. They can see it.

No role changes. No capability editing. No admin panel. The post author
controls access per-post with the same gesture as sharing a Google Doc.

For multi-author blogs, this creates natural workflows:

- Share a draft with a co-author for collaborative editing
- Share a private post with a "Friends" group for personal updates
- Share a published post to social media with a commenter token
  for controlled discussion
- Keep a post truly private — a personal notebook entry

All of this falls out of tuples + visibility. No special features needed.

## Real-Time Collaboration

When multiple subjects have `editor` on a post, they can edit simultaneously
via ProseMirror + WebSocket. The WebSocket upgrade handler checks the tuples
cache: does this subject have `editor` on this post? If yes, join the
editing session. If `viewer`, read-only connection (see changes live, can't
edit).

## Type-Scoped Permissions

"This group can edit posts but not pages" is just different tuples:

```
group:writers  →  editor  →  type:post
```

No `editor` tuple on `type:page` means writers can't edit pages. The check
for "can this user edit page 5?" walks the same path — it just doesn't find
a matching tuple for `type:page`.

## Open: Visibility Storage

The visibility model (`public`, `unlisted`, `shared`, `private`) is
described above as orthogonal to `post_status` — a draft can be shared,
a published post can be private. But the database schema has no
dedicated visibility field. `post_status` currently carries `publish`,
`draft`, `private`, conflating status and visibility.

If visibility is truly orthogonal, it needs its own column or a postmeta
key. The tuple system depends on visibility to gate access, but there's
no schema-level place to store it yet. This needs to be thought through
before the permission checks can be fully implemented.

## What This Is Not

- **Not full Zanzibar.** Max depth is 2 (user → group → resource). No
  recursive relations, no graph traversal, no external service.
- **Not RBAC with inheritance.** Groups don't inherit from other groups.
- **Not extensible.** The relation set is fixed. Extensions don't add new
  relation types.
- **Not multi-tenant.** One blog, one tuple set.

## What This Replaces

| WordPress concept         | Press equivalent                            |
| ------------------------- | ------------------------------------------- |
| `user_level` 0-10         | Group membership + tuples                   |
| `wp_capabilities` meta    | Tuples on `type:*` and `site`               |
| `post_status = 'private'` | `visibility = 'private'` on the post        |
| Post password protection  | Share link with a token (or keep passwords) |
| No per-post sharing       | Tuples on specific `post:<id>`              |
| No share links            | `token` subject type in tuples              |
