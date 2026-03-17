# Writing Experience

The editor features that make writing smooth. These aren't the big
architectural pieces (ProseMirror, collab, revisions) — those are in
`docs/plans/editor.md`. These are the small things that add up to a writing
environment where you don't fight the tool.

## Slug Generation

Auto-generate the URL slug from the title as the author types. Live
preview of the full permalink below the title field.

- Generated on first save, not on every keystroke (avoid thrashing)
- Author can edit the slug manually
- Duplicate slugs get a `-2`, `-3` suffix automatically
- Once published, the slug is permanent — changing it would break URLs
- Slug preview updates via htmx or light JS as the title changes

## Autosave

The step log gives us crash recovery, but the author needs to see "saved."

- Periodic autosave creates a save point every N seconds while editing
- Visual indicator: "Saved" / "Saving..." / "Unsaved changes" in the
  editor chrome
- Close the tab, come back, you're where you left off
- Autosave points are visually distinct from manual saves in the
  revision timeline

## Image Upload

Drag an image into the editor, it uploads, an image node appears in the
document.

- ProseMirror drop handler catches the file
- Upload via fetch to the server, creates an attachment
  (`post_type = 'attachment'`, `post_parent` = current post)
- Server returns the attachment URL and metadata
- Editor inserts an image node at the drop position
- Loading indicator while uploading
- Also works via paste (screenshot paste is common)
- Also works via an "Add Image" button that opens a file picker

No "open the media library, browse, select, insert URL." The happy path
is drag and drop.

## Paste Handling

People paste from Google Docs, email, web pages, plain text. ProseMirror's
clipboard handling parses incoming HTML through the document schema.

- Strip styling garbage (`<span style="font-family: Arial">`)
- Keep structure: paragraphs, headings, lists, links, bold, italic
- Anything that doesn't fit the schema gets simplified, not mangled
- Paste plain text → paragraphs split on double newlines
- Paste a URL onto selected text → becomes a link
- Paste a URL on its own line → plain text link (no auto-embeds)

This is where WordPress's copy-paste from Docs breaks. Our strict schema
is an advantage — unknown markup simplifies cleanly instead of producing
invisible formatting landmines.

## Markdown Shortcuts

ProseMirror input rules for common markdown patterns:

| Type        | Trigger                        |
|-------------|--------------------------------|
| Heading 1   | `# ` at start of line          |
| Heading 2   | `## ` at start of line         |
| Heading 3   | `### ` at start of line        |
| Bullet list | `- ` or `* ` at start of line  |
| Ordered list| `1. ` at start of line         |
| Blockquote  | `> ` at start of line          |
| Code block  | ``` at start of line           |
| Horizontal rule | `---` on its own line      |
| Bold        | `**text**`                     |
| Italic      | `*text*`                       |
| Code inline | `` `text` ``                   |

These fire as you type — the markdown syntax is consumed and replaced
with the formatted node. Writers who think in markdown get their shortcuts.
Writers who don't never see markdown.

## Word Count

Live word count in the editor footer. Updates as you type.

- Total word count for the post
- Maybe: reading time estimate (words / 200 wpm)
- Maybe: character count for excerpt field

Simple but writers care about this deeply.

## Focus Mode

Dim everything except the current paragraph. The editor chrome, sidebar,
and surrounding paragraphs fade to low opacity. Just you and the sentence
you're writing.

- Toggle via keyboard shortcut or button
- ProseMirror decorations handle the dimming
- The cursor's block node gets full opacity, everything else fades
- Escape or toggle again to return to normal

WordPress added this in 4.1 ("distraction-free writing") and it was
genuinely one of the best features they shipped.

## Excerpt

The excerpt appears in feeds, archives, search results, and social share
previews. It's what people see before they click.

- Auto-generated from the first paragraph if not set manually
- Visible in the editor so the author knows what readers will see
- Editable in a field below the main content area
- Character count indicator (social platforms truncate at ~155 chars)

## Scheduling

Set a future publish date. The post is scheduled and goes live
automatically when the date arrives.

- Date/time picker in the publish panel
- "Schedule" button replaces "Publish" when a future date is set
- Scheduled posts visible in the post list with their publish date
- A background goroutine or on-request check flips status at the
  scheduled time

**Open:** The post needs a status or flag that distinguishes "scheduled
for future publication" from "draft — author isn't done." Without this,
the UI can't reliably show scheduled posts differently from drafts.
Options: a `future` or `scheduled` post_status value, or a separate
`scheduled_at` field alongside the existing `post_date`. WordPress used
a `future` status that auto-transitioned to `publish` via cron. We
should decide on the mechanism.

## Social & SEO Meta

Every WordPress site needs a plugin for Open Graph tags. That's absurd.
A blog should generate them automatically and let the author override
when needed.

### Auto-generated from post data

Every post automatically gets Open Graph, Twitter Card, standard meta
description, and canonical URL tags. The values are derived from post
data:

- **Title** — post title
- **Description** — excerpt, or first paragraph if no excerpt
- **Image** — first image in the post, or a site-wide default
- **URL** — canonical permalink
- **Article metadata** — published date, modified date, author name

Plus JSON-LD structured data (`BlogPosting` schema) for search engines.

All auto-generated. No plugin needed.

### Overridable in the editor

A collapsible "Sharing" panel in the editor sidebar for overriding any
auto-generated value:

- Custom social title (different from post title)
- Custom social description (different from excerpt)
- Custom social image (different from first post image)
- Stored in postmeta — if the meta exists, use it; if not, derive

The defaults should be good enough that most authors never open this panel.

### Site-level defaults

Options in `wp_options` for site-wide fallbacks:

- Default social image (used when a post has no images)
- Site social handle
- Site name for `og:site_name`
