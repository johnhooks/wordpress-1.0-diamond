# Editor & Revisions

WordPress 1.0 had a textarea with quicktags. WordPress 2.0 brought TinyMCE.
WordPress 5.0 brought Gutenberg. Each step moved further from writing and
closer to page building.

Press uses ProseMirror — a structured editor that's about writing, not
layout. And the revision system isn't WordPress's brute-force "save a full
copy every time." It's step-based diffs, like git for your posts.

## ProseMirror

The post editor is a ProseMirror instance. ProseMirror operates on a
structured document tree (JSON), not an HTML string. The document has a
schema — paragraphs, headings, lists, links, images, code blocks. No
freeform HTML, no arbitrary divs, no layout primitives.

### Storage

Post content is stored as a ProseMirror JSON document, not HTML. HTML is
an output format rendered from the document tree at display time.

```
post_content column: ProseMirror JSON document
display: JSON → HTML via ProseMirror schema's toDOM
editing: JSON → ProseMirror editor state
```

This means the same content renders identically everywhere — the editor,
the published post, the RSS feed. No "the editor shows it differently
than the frontend" bugs.

### Quicktags Fallback

For simple posts, a plain textarea with quicktags (the WP 1.0 experience)
should still work. The server accepts both raw text/HTML and ProseMirror
JSON. Raw input gets parsed into a ProseMirror document on save. This
keeps the simple path simple — not every post needs a rich editor.

## Revisions

WordPress stored revisions as full copies of the post in `wp_posts` with
`post_type = 'revision'` and `post_parent` pointing to the original. Every
save duplicated the entire content. Ten saves, ten full copies.

Press uses ProseMirror steps — atomic operations that describe what changed.
Insert text here, delete text there, change this heading level. Steps are
small, precise, and composable.

### Two Levels

**Step log:**

Every change is a ProseMirror step — an atomic operation like "insert text",
"delete selection", "change heading level", or "add link." ProseMirror
batches rapid typing into steps, so it's not one step per keystroke but
one step per operation. The step log is the complete, ordered, attributed
history of every edit ever made to a post. Git blame for blog posts.

```
wp_steps
    id          INTEGER PRIMARY KEY
    post_id     INTEGER
    version     INTEGER         -- incrementing version number
    step        TEXT            -- ProseMirror step JSON
    user_id     INTEGER         -- who made this edit
    created_at  DATETIME
```

Steps are kept permanently. They're tiny — a few hundred bytes each. A
post might accumulate a few thousand steps over its lifetime. That's
kilobytes. Not a storage concern.

If a post eventually has tens of thousands of steps, old ones can be
trimmed from the tail. The only consequence is you can only walk back
so far in history — not a big deal.

The step log powers:
- **Undo/redo** — walk backwards and forwards through operations
- **Real-time collaboration** — steps sent over WebSocket, server rebases
  conflicts, broadcasts to all connected clients
- **Attribution** — every step has a user_id, so "who wrote this sentence?"
  is answerable
- **Time-travel** — replay the construction of a post from first keystroke

**Save points:**

An explicit save is just a marker in the step log — "the author considered
the document done at this version." The save records the version number
of the current step, creating a named point in the timeline.

```
wp_save_points
    id          INTEGER PRIMARY KEY
    post_id     INTEGER
    version     INTEGER         -- step version at time of save
    user_id     INTEGER         -- who saved
    created_at  DATETIME
```

No diffs, no full copies. The steps already describe every transformation.
A save point just says "this version was intentional." Reverting to a save
point means unapplying steps back to that version number.

### Reverting

To revert to a previous version: unapply steps backwards from the current
version to the target save point. ProseMirror steps are invertible — every
step has a corresponding inverse step that undoes it.

### Diff UI

The revision screen shows the difference between any two save points.
Since we have every step between them, we can:

- Show a visual diff — green/red like git diff, rendered as formatted text
- Show who made each change (steps carry user_id)
- Browse the save point timeline
- Revert to any save point with one click
- Potentially replay the editing session like a video

The diffs are structural — ProseMirror steps know the difference between
"changed a heading level" and "rewrote a paragraph" because the operations
are semantic, not textual. This is aspirational polish, not day-one.

## Collaborative Editing

When multiple users have `editor` permission on a post (via tuples), they
can edit simultaneously.

### Architecture

```
Client A (ProseMirror) ←→ WebSocket ←→ Server (Go) ←→ WebSocket ←→ Client B (ProseMirror)
                                          ↓
                                     Step log (wp_steps)
                                     Authority document
```

- Each editor holds a ProseMirror editor state.
- Changes produce steps, sent to the server via WebSocket.
- The server is the authority. It applies steps to the canonical document,
  rebases any conflicts (two people typed at the same position), and
  broadcasts the result to all connected clients.
- The step log records every operation with the user who made it.
- Presence (cursor positions, selections) is broadcast to all clients
  so you can see where others are editing.

### Permission Gate

The WebSocket upgrade handler checks the tuples cache:
- `editor` tuple on this post → read-write connection
- `viewer` tuple → read-only connection (live preview, no editing)
- No tuple → rejected

### Goroutines

Each WebSocket connection is a goroutine. Each post being edited has a
coordination goroutine that manages the step log, rebases conflicts, and
broadcasts to connected clients. Go's concurrency model makes this natural
— no event loop, no Node.js sidecar, no external pubsub. It's goroutines
and channels.

## Forks

Steps, save points, and collaboration all operate on a single timeline —
edits flow into the post as they happen. That's fine for drafts and for
small fixes to published work. But sometimes you need to substantially
rework something that's already live. WordPress has no answer for this.
Once a post is published, every edit is immediately visible to readers.
Your only option is to edit live and hope nobody's reading mid-rewrite.

A fork is a working copy of a published post. The published version stays
live and untouched. The fork goes through its own full lifecycle — draft,
edit, collaborate, review — completely invisible to readers. When it's
ready, it replaces the published version atomically.

### Why this matters

Writing is rewriting. A published essay might need a structural overhaul
months later — new sections, reordered arguments, updated conclusions.
That's not a typo fix. That's days of work, possibly with collaborators
and editorial review. The author shouldn't have to choose between editing
live (readers see half-finished work) and drafting in a separate tool
(loses all the editor tooling, revision history, and collaboration).

A fork lets the published version stand while the next version takes
shape behind it. The same editor, the same collaboration, the same step
history — just on a branch that readers can't see yet.

### How it works

A fork is a new post row in `wp_posts` with `post_parent` pointing to
the original and `post_status = 'draft'`. Its content starts as a
snapshot of the published post's document at the time of forking. From
there it has its own step log, its own save points, its own
collaborators.

The published post keeps its visibility tuples (`group:public → viewer →
post:42`). The fork gets its own tuples — the author's writer grant,
maybe an editor's grant for review. Readers see the published version.
The fork is invisible to anyone without a tuple on it.

When the fork is ready, publishing it replaces the original: the fork's
content becomes the published post's content, the public visibility
tuple stays where it is, and the fork row is kept as history. The
original content is still in the step log if you ever need it back.

### What a fork is not

A fork is not a new post. It doesn't get its own URL, its own slug, its
own place in the chronological stream. It's a working copy of an
existing post. When it lands, the post at its original URL has new
content. Readers who bookmarked it, linked to it, or subscribed to it
see the updated version at the same address.

A fork is also not required for every edit. Fixing a typo on a published
post is a normal edit — step into the editor, change the word, done. The
fork exists for the case where edits are substantial enough that you
don't want them visible until they're finished.

## Editorial Comments

The editor supports inline annotations — comments anchored to a specific
text range in the document. "This paragraph needs a source." "Can we
rephrase this?" These are editorial comments (`comment_type = 'editorial'`),
visible only to users with `editor` permission. See `docs/comments.md`.

ProseMirror marks pin to document positions and track through edits. An
editor leaves a comment on a sentence, someone inserts a paragraph above,
the comment stays with its sentence.

Because we have WebSockets, editorial comments are live. Leave a note,
the co-author sees it appear. Respond, resolve, the mark disappears. All
on the same WebSocket connection used for collaborative editing — editorial
comments are just another message type.

This is the Google Docs review experience, built into the blog editor.

## What This Is Not

- **Not a block editor.** No layout primitives, no columns, no widgets.
  The schema is writing: paragraphs, headings, lists, quotes, code, images.
- **Not Notion.** No databases, no toggles, no embeds. Write text.
- **Not optional.** The ProseMirror document model is the storage format.
  There's no "classic editor" toggle. The quicktags fallback parses input
  into the same document model on save.
