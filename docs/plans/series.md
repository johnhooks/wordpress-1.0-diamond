# Post Series

## The Problem

Writers organize posts into series. A multi-part tutorial, a serialized
essay, a travelogue, a build log. "This is part 3 of Building a Blog
Engine." The posts belong together, in a specific order, and the reader
should be able to navigate between them.

WordPress never built this. The closest it offered was categories — but
categories are unordered collections, not sequences. Plugin solutions
exist but they're old, fragile, and bolted onto a system that wasn't
designed for them. Every plugin invented its own data model.

A series is a fundamental writing concept. It should be in the platform.

## What a Series Is

A series is an ordered group of posts on a shared topic. It has:

- **A name** — "Building a Blog Engine"
- **A description** — what the series is about
- **A slug** — `building-a-blog-engine`
- **An ordered list of posts** — Part 1, Part 2, Part 3
- **A status** — ongoing or complete

A series is not a category. Categories are unordered buckets for loose
grouping. A series is a sequence with a reading order. A post about Go
might be in the "Go" category and also Part 4 of "Building a Blog Engine."
These are different relationships.

## What the Reader Sees

### On a post that's in a series

A banner or box showing:

```
Part 3 of 7 in "Building a Blog Engine"
← Part 2: The Database    Part 4: The Router →
```

Previous/next navigation within the series. The series name links to the
series landing page. If the series is ongoing, "Part 3 of 7" becomes
"Part 3 in Building a Blog Engine" — no total count.

### Series landing page

`/series/building-a-blog-engine/` — the series name, description, and a
table of contents showing all posts in order. Posts display in series
order, not reverse-chronological. The reader starts at Part 1, not the
latest.

### In the sidebar

A list of active/recent series, similar to the category list.

## What the Writer Sees

### Creating a series

A series is created in the admin — name, slug, description. Or created
inline when editing a post ("add to new series").

### Adding a post to a series

On the post editor, a series picker (similar to categories). Select a
series, and the post is appended to the end. The writer can reorder
posts within the series from the series admin page.

### Series management

An admin page listing all series with post count, status (ongoing/complete),
and a drag-to-reorder interface for the posts within each series.

## How to Implement

The taxonomy system handles grouping — a series is a taxonomy type like
`category` or `link_category`. The ordering within a series is the open
question.

### Grouping: Taxonomy

A series is `taxonomy = 'series'` in the existing taxonomy tables:

- `wp_terms` — series name, slug
- `wp_term_taxonomy` — taxonomy type, description, post count
- `wp_term_relationships` — links posts to their series
- `wp_termmeta` — series metadata (complete/ongoing status)

This gives us naming, slugs, descriptions, and the post-to-series
relationship for free. No new tables for the grouping itself.

### Ordering: Open Question

The taxonomy system groups posts into a series but doesn't order them
within it. `term_order` on `wp_term_relationships` is the wrong direction
— it orders terms on a post, not posts within a term.

**Option A — Chronological order:**
Posts display in the order they were published. No explicit ordering
field needed. A series is usually written in sequence, so dates give
the natural reading order. Simple. But you can't reorder after the
fact, and backdating a post to insert it earlier in the series is a
hack.

**Option B — Postmeta:**
A `series_position` meta key on each post. `postmeta('series_position')
= 3` means "this is part 3." Queryable, simple to update. But the
ordering lives on the post, not on the relationship — and if a post is
in two series (unlikely but possible), it can only have one position.

**Option C — Termmeta on the series:**
Store the ordered list of post IDs as a JSON array in `wp_termmeta`:
`series_order = [12, 45, 23, 67]`. One value to read, easy to reorder
(just rearrange the array). But it's denormalized and could drift from
reality if a post is deleted without updating the array.

**Option D — Add `object_order` to `wp_term_relationships`:**
A new column on the junction table: the position of this object within
this term. Structurally correct — the ordering is on the relationship
itself. But it changes the taxonomy schema for one use case. We own the
schema so we can do this, but it's a bigger decision.

**Option E — Dedicated ordering table:**
A `wp_series_order` table with `series_id`, `post_id`, `position`. Clean
separation. The taxonomy handles identity and grouping, the ordering
table handles sequence. More schema, but explicit.

No decision yet. Each has tradeoffs. The right answer probably depends
on how the rest of the system evolves.

## Theme Integration

```
{{ .Post.TheSeries }}         — series name, linked
{{ .Post.SeriesPosition }}    — "Part 3"
{{ .Post.SeriesTotal }}       — "of 7" (if complete)
{{ .Post.SeriesPrev }}        — previous post in series
{{ .Post.SeriesNext }}        — next post in series
{{ .Sidebar.Series }}         — list of series
```

The series landing page uses the archive template pattern — same as
category archives but ordered by position, not date.

## What We Build Now

Nothing series-specific. But the taxonomy system supports it — adding
`taxonomy = 'series'` is the same mechanism as categories. The ordering
question is the only piece that needs new thinking.

## What We Don't Build Yet

- Series UI in the admin
- Series navigation in themes
- Series landing pages
- Ordering mechanism (whichever option we pick)
- Series RSS feeds
