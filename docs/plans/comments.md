# Comments

Comments are the other half of a blog. Someone writes, someone responds. WP 1.0
had comments from day one. We keep them, gate them with permissions, and split
them into two types: post comments (public discussion) and editorial comments
(annotations on the document itself).

## Comment Types

### Post Comments

The traditional blog comment. Discussion below a published post. But unlike
WordPress, commenting is a **permissioned action**, not an open door.

WordPress comments were "anyone with a name and email can say anything."
That's why they're spam-ridden and worthless. In Press, commenting requires
either:

- A user account on the blog
- A share token that grants `commenter` permission

When you share a post to Twitter, the share link carries a token. That token
grants `commenter` on that post. People who follow the link can read and
comment. People who find the direct URL without a token can read but not
comment (unless they have an account).

The post author controls who can speak. Sharing is curating your audience.

### Editorial Comments

Annotations on the document itself — not on the published post, but on the
draft in the editor. "This paragraph needs a source." "Love this section."
"Can we rephrase this?"

Editorial comments are anchored to a specific text range in the ProseMirror
document. They're visible only to users with `editor` permission on the post.
They're part of the editing workflow, not the published output.

ProseMirror supports this natively. Marks and decorations pin to document
positions and track their anchor through edits — if someone inserts a paragraph
above, the annotation moves with the text it's attached to.

With WebSockets, editorial comments are live. An editor leaves a note on
paragraph three, the author sees it appear in real-time. The author responds,
resolves it, the mark disappears. Google Docs review workflow, in a blog
editor.

WordPress is shipping something like this in 7.0, but without WebSockets
they're stuck with polling. We have goroutines holding live connections
already for collaborative editing — editorial comments are just another
message type on the same channel.

## Model

Both types live in `wp_comments`.

| Field            | Post Comment               | Editorial Comment                     |
| ---------------- | -------------------------- | ------------------------------------- |
| comment_post_ID  | Which post                 | Which post                            |
| comment_type     | `'comment'`                | `'editorial'`                         |
| comment_content  | The comment text           | The annotation text                   |
| comment_parent   | Parent comment (threading) | Parent comment (thread on annotation) |
| comment_approved | `'0'`, `'1'`, `'spam'`     | Always `'1'` (no moderation)          |
| comment_author   | Display name               | Display name                          |
| user_id          | User ID (if logged in)     | User ID (required)                    |
| comment_date     | When posted                | When posted                           |

Editorial comments need anchor data — where in the document they're
attached. Stored in `wp_commentmeta`:

| meta_key      | Value                          |
| ------------- | ------------------------------ |
| `anchor_from` | Start position in document     |
| `anchor_to`   | End position in document       |
| `resolved`    | Whether the thread is resolved |
| `resolved_by` | User who resolved it           |

## Permission Model

Commenting is gated by tuples. No tuple, no comment.

**Post comments:**

- `commenter` tuple on the specific post — from a share token or direct grant
- `editor` tuple on the post — editors can always comment
- `commenter` tuple on `type:post` — global comment permission (logged-in users)

**Editorial comments:**

- `editor` tuple on the post — only editors can see and create annotations
- Viewers cannot see editorial comments

**Share tokens and commenting:**

When you create a share link, you choose what it grants:

| Token permission | Can view | Can comment | Can edit |
| ---------------- | -------- | ----------- | -------- |
| `viewer`         | yes      | no          | no       |
| `commenter`      | yes      | yes         | no       |
| `editor`         | yes      | yes         | yes      |

This means sharing a post to social media with a `commenter` token creates
a controlled discussion space. The people you invited can talk. Everyone
else can read.

## Threading

Comments nest one level via `comment_parent`. A reply to a comment sets
`comment_parent` to the parent's ID. Replies to replies are flat — no
deeply nested threads.

Editorial comments also thread — a note on a paragraph can have replies,
creating a focused discussion about that specific piece of text.

Display order: top-level comments chronological, replies grouped under their
parent chronologically.

## Private Posts as Conversations

A private or shared post with `commenter` tuples for specific users is
effectively a thread. The post is the prompt, the comments are the
conversation. The permission system makes this natural:

1. Author writes a post, sets visibility to `shared`
2. Shares it with a group or specific users with `commenter` permission
3. Those people can read and discuss in comments
4. Nobody else can see it

This is threads/group chat energy but built on blog infrastructure.
Multi-author blogs get this for free — authors can share drafts with
each other for feedback via editorial comments, or create private posts
for internal discussion via post comments.

## Frontend

### Viewing comments

Comments render below the post. On a full page load, they're part of the
single post template. The comment list is a discrete fragment: `_comments.html`.

The comment form only appears if the current user/token has `commenter`
permission. No permission, no form — just the post content.

### Posting a comment

The comment form submits via htmx. Server validates the permission (token
or user has `commenter` on this post), saves the comment, and returns the
rendered comment HTML fragment appended to the list. No page reload.

If the user is logged in, author name/email come from their account. If
they're using a share token, they provide a display name. No email required
for token-based commenting — the token is the identity.

### Replying to a comment

Each comment has a Reply link. Server returns a reply form fragment with
`comment_parent` set. Same htmx flow.

### Error handling

Validation errors return a fragment with the error message, swapped into
the form area. No redirect to an error page.

## Admin

### Moderation queue

`GET /wp-admin/moderation.php` — list of pending comments.

Each comment has Approve, Spam, Delete actions via htmx row swaps.
Instant feedback, no full page reload.

### Comment management

`GET /wp-admin/edit-comments.php` — all comments with filtering and bulk
actions. Filterable by post, comment type, status.

## Spam

Token-based commenting dramatically reduces spam — you can't comment
without a valid token or account. But for public posts where the author
grants broad commenting permission, keyword filtering still helps:

- `moderation_keys` option: comments containing these words go to moderation.
- `blacklist_keys` option: comments containing these words are marked spam.

No Akismet, no external service.

**Open:** Commenter tokens shared on social media are effectively public
— anyone who sees the URL can comment. A commenter token shared on
Twitter is as open as an anonymous comment form. The only advantage is
revocability (delete the token, stop the flood), but by then damage may
be done. The spam/abuse model for widely-shared commenter tokens needs
more thought.

## Not Supported

- Trackbacks and pingbacks. Dead protocols.
- Open commenting without permission. No "anyone can comment" mode.
- Deeply nested threads. One level of replies.
- Gravatar. Maybe later.

## Open Questions

- **Editorial comment lifecycle:** Editorial comments survive the publish
  process — they stay in `wp_comments` with `comment_type = 'editorial'`.
  Resolved threads accumulate over time. We may want a way to clear them
  later, but they're not automatically cleaned up on publish.
- **Attachment visibility in comments:** If a commenter embeds an image
  or a post contains images, the attachment visibility rules need to
  align with the post's permission model. See `docs/plans/attachments.md`.
