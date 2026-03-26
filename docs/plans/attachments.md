# Attachments

Files uploaded to the blog. Images in posts, PDFs linked from pages. WordPress
stored these as `post_type = 'attachment'` in `wp_posts` and it turns out
that's actually fine. The post table already has everything an attachment
needs: an author, a date, a title, a slug, a parent, and metadata via
`wp_postmeta`.

## Model

Attachments are rows in `wp_posts` with `post_type = 'attachment'`.

| Field          | Usage                                                   |
| -------------- | ------------------------------------------------------- |
| post_type      | `'attachment'`                                          |
| post_title     | Filename or user-provided title                         |
| post_name      | Slug (sanitized filename)                               |
| post_content   | Description (optional)                                  |
| post_excerpt   | Caption (optional)                                      |
| post_mime_type | MIME type — `image/jpeg`, `application/pdf`             |
| post_parent    | The post or page this was uploaded to (0 if unattached) |
| post_author    | Who uploaded it                                         |
| post_date      | Upload date                                             |
| post_status    | `'inherit'` (inherits parent's visibility)              |
| guid           | The file URL                                            |

Metadata in `wp_postmeta`:

| meta_key                  | Value                                     |
| ------------------------- | ----------------------------------------- |
| `_wp_attached_file`       | Relative path: `2004/01/photo.jpg`        |
| `_wp_attachment_metadata` | Dimensions, filesize (JSON or serialized) |

This means `post_type` now has three values: `post`, `page`, `attachment`.
The post table earns its keep. `post_mime_type` stays in the schema — it's
only meaningful for attachments but it costs nothing.

## File Storage

Uploaded files go to the uploads directory, organized by year/month:

```
storage/uploads/
└── 2004/
    └── 01/
        ├── photo.jpg
        └── document.pdf
```

The uploads directory is configured via `UPLOADS_DIR` in `.env`. The path
stored in `_wp_attached_file` is relative to this directory.

## Upload

### Admin upload

`GET /wp-admin/upload.php` — upload form and media library.

Upload form submits via htmx — server saves the file, creates the
`wp_posts` row and metadata, returns the new media item as an HTML
fragment prepended to the list.

### Inline upload (post editor)

When writing a post, uploads attach to that post (`post_parent` = the
post being edited). The editor supports drag-and-drop, paste, and a file
picker button. Server creates the attachment and returns the URL for the
editor to insert an image node or link.

## Serving Files

Uploaded files are served directly from the filesystem. The server registers
a route for the uploads path:

```
GET /uploads/2004/01/photo.jpg → static file serve from UPLOADS_DIR
```

No database lookup needed for serving. The URL is just a path to a file.

## Attachment Page

Each attachment can optionally have its own page at its permalink, showing
the file (image displayed, other types linked) with its title and
description. This is a WP 1.5 feature that's simple and occasionally
useful — someone links to an image and gets context instead of a raw file.

The Router needs an attachment rule, but this is low priority. The file
URL works regardless.

## What Attachments Are Not

- **Not a media library in the CMS sense.** No drag-and-drop galleries,
  no image editor, no focal point picker.
- **Not a CDN.** Files are local. If you want a CDN, put one in front.
- **Not versioned.** Upload a new version, it's a new file.
- **Not searchable.** Attachments don't appear in search results.

## Visibility

Attachments use `post_status = 'inherit'` — visibility inherits from
the parent post. But the permission system (see `docs/plans/permissions.md`)
defines visibility as a separate concept from status, with tuple-based
access control.

**Open:** When an attachment's parent post is `shared` or `private`,
the attachment file at `/uploads/2004/01/photo.jpg` is currently served
as a static file with no permission check. This means images in
shared/private posts are accessible to anyone who knows or guesses the
file path.

When an attachment is referenced inside a post, it should be viewable
because the author made the choice to include it. But direct access
to the file URL outside of the post context is a different question.
The rules here need to be figured out — possibly serving protected
attachments through a handler that checks permissions, or accepting the
leak as a reasonable tradeoff for simplicity.

## Open Questions

- **Image sizes:** WordPress generates thumbnails and multiple sizes on
  upload. Do we want this? A single size (original) is simpler. But
  serving a 4MB photo in a blog post is rough. Leaning toward generating
  one reasonable display size on upload.
- **Allowed types:** Whitelist of MIME types? WordPress has one. We should
  too — don't let someone upload a `.php` file.
- **Max file size:** Configurable via `.env`. Default something sane like
  10MB.
