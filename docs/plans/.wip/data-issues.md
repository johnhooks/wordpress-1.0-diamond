# Data Layer Issues

Divergences between Press repositories and WordPress's behavior that matter for a WP 1.0 blog engine. Identified by comparing `internal/repository/` against `wordpress-develop/src/wp-includes/` (post.php, comment.php, user.php, taxonomy.php, option.php, bookmark.php).

Excludes: hook/filter system (Go doesn't have it), REST API concerns, modern WP features (Gutenberg, full site editing, multisite), and anything that belongs in a service layer above the repositories (moderation keywords, flood checking, spam detection, auto-approval rules).

## Posts

### No slug generation from title
**Severity:** High
**WordPress behavior:** `wp_insert_post()` calls `sanitize_title()` which strips HTML, converts accents, lowercases, and replaces spaces/special chars with hyphens. Then `wp_unique_post_slug()` appends `-2`, `-3`, etc. for uniqueness. Drafts and pending posts can have empty slugs — slug is generated on publish.
**Press behavior:** `EnsureUniqueSlug()` handles the `-2` suffix logic, but there is no equivalent of `sanitize_title()`. The caller must provide a pre-sanitized slug or nothing happens.
**Fix:** Create a shared `internal/slug` package with a `Generate(title string) string` function that lowercases, strips non-alphanumeric, replaces spaces with hyphens, and collapses consecutive hyphens. Posts and terms both need this.

### No default category enforcement
**Severity:** Medium
**WordPress behavior:** `wp_insert_post()` enforces that every post with type `post` and status other than `auto-draft` has at least one category. If none provided, it assigns the `default_category` option value.
**Press behavior:** Posts can exist with no category. The term relationship is entirely caller-managed.
**Fix:** Could live in a service layer, but worth noting. On create/update of a published post, if no categories are assigned, assign the default category from `wp_options`.

### No future/scheduled post transitions
**Severity:** Low
**WordPress behavior:** If `post_date` is in the future and status is `publish`, WP auto-converts to `future` status. A cron job later transitions `future` → `publish` when the date arrives.
**Press behavior:** No status transitions based on date. Posts are whatever status the caller sets.
**Fix:** Service layer. When setting status to `publish`, check if `post_date > now` and set to `future` instead. A background goroutine checks for posts where `status = 'future' AND post_date <= now` and transitions them to `publish`.

### No content in post_content_filtered
**Severity:** Low
**WordPress behavior:** `post_content_filtered` stores a pre-processed version of content for search optimization.
**Press behavior:** Always empty string. Not used.
**Fix:** Not needed for WP 1.0 scope.

## Comments

### Update() doesn't sync comment_count
**Severity:** High
**WordPress behavior:** Any change to `comment_approved` triggers `wp_update_comment_count()` which recounts approved comments on the post.
**Press behavior:** Only the dedicated `Approve()` and `Unapprove()` methods sync the count. If a caller uses `Update()` directly and changes `comment_approved`, the post's `comment_count` becomes stale.
**Fix:** In `Update()`, fetch the old comment first. If `comment_approved` changed, update the post's `comment_count` accordingly (or recount).

### No child comment re-parenting on delete
**Severity:** Medium
**WordPress behavior:** `wp_delete_comment()` promotes child comments — sets their `comment_parent` to the deleted comment's `comment_parent`, preserving the thread structure.
**Press behavior:** Deletes the comment. Children still point to the deleted comment's ID via `comment_parent`, creating orphaned threads.
**Fix:** Before deleting, run `UPDATE wp_comments SET comment_parent = ? WHERE comment_parent = ?` with the deleted comment's parent ID.

### No commentmeta cleanup on delete
**Severity:** Low
**WordPress behavior:** Deletes all `wp_commentmeta` rows for the comment.
**Press behavior:** Doesn't clean up commentmeta. The table exists in the schema but isn't used yet.
**Fix:** Add `DELETE FROM wp_commentmeta WHERE comment_id = ?` to `Delete()`. Low priority since commentmeta isn't actively used.

## Users

### Delete doesn't reassign posts or links
**Severity:** High
**WordPress behavior:** `wp_delete_user()` takes a `$reassign` parameter. If provided, all the user's posts and links are reassigned to that user ID. If not provided, the user's posts and links are deleted.
**Press behavior:** `Delete()` only cleans up `wp_usermeta`. Posts and links authored by the deleted user become orphaned (reference a non-existent `post_author` / `link_owner`).
**Fix:** Add a `reassignTo` parameter to `Delete()`. If non-zero, `UPDATE wp_posts SET post_author = ? WHERE post_author = ?` and `UPDATE wp_links SET link_owner = ? WHERE link_owner = ?`. If zero, delete the user's posts (which cascades to their comments and term relationships) and links.

### No user_nicename auto-generation
**Severity:** Medium
**WordPress behavior:** `wp_insert_user()` generates `user_nicename` from `user_login` by running it through `sanitize_title()`, then ensures uniqueness by appending `-2`, `-3`, etc. (same pattern as post slugs). Nicename is used in author URLs like `/author/john-doe/`.
**Press behavior:** `user_nicename` is whatever the caller provides. The CLI `user create` command sets it to the login, but doesn't sanitize or ensure uniqueness.
**Fix:** In `Create()`, if `user_nicename` is empty, generate from `user_login` via the slug package. Ensure uniqueness with suffix logic (same as `EnsureUniqueSlug` but against `wp_users.user_nicename`).

### No display_name defaulting
**Severity:** Low
**WordPress behavior:** If `display_name` is empty, defaults to `user_login`.
**Press behavior:** Accepts empty `display_name`.
**Fix:** In `Create()`, if `display_name` is empty, set it to `user_login`.

## Terms

### No slug auto-generation from name
**Severity:** High
**WordPress behavior:** `wp_insert_term()` generates slug from term name via `sanitize_title()` if no slug provided. Then ensures uniqueness per taxonomy via `wp_unique_term_slug()`.
**Press behavior:** `Create()` accepts whatever slug is provided. No generation, no sanitization.
**Fix:** In `Create()`, if `term.Slug` is empty, generate from `term.Name` via the slug package. Ensure uniqueness against `wp_terms.slug` (slugs are globally unique in the schema due to the UNIQUE index).

### Default category not protected from deletion
**Severity:** High
**WordPress behavior:** `wp_delete_term()` checks if the term being deleted is the default category (stored in `wp_options` as `default_category`). If so, it refuses and returns a `WP_Error`.
**Press behavior:** `Delete()` will happily delete the default category, breaking post creation and display.
**Fix:** In `Delete()`, before deleting a term, check if it matches the `default_category` option. If so, return an error. This requires the repository to access options, or the check can happen in a service layer.

### No child term reassignment on delete
**Severity:** Medium
**WordPress behavior:** `wp_delete_term()` reassigns child terms (where `parent = deleted_term_taxonomy_id`) to the deleted term's own parent, preserving the hierarchy.
**Press behavior:** `Delete()` removes `wp_term_taxonomy` rows for the term, which means child terms still reference the deleted term as parent — creating orphaned hierarchy nodes.
**Fix:** Before deleting, `UPDATE wp_term_taxonomy SET parent = ? WHERE parent = ?` using the deleted term's parent ID.

### No object reassignment to default term on delete
**Severity:** Medium
**WordPress behavior:** When deleting a category, if a post's only category was the deleted one, WordPress reassigns it to the default category. Posts always have at least one category.
**Press behavior:** Deletes the term relationship. The post ends up with no categories.
**Fix:** After removing term relationships, find posts that now have zero categories and assign them to the default category. This ties into the "default category enforcement" issue above.

### SetPostTerms has no append mode
**Severity:** Low
**WordPress behavior:** `wp_set_object_terms()` supports an `$append` parameter. When true, new terms are added without removing existing ones.
**Press behavior:** `SetPostTerms()` always replaces all terms. `AddTermToPost()` exists for single additions but there's no batch append.
**Fix:** Add an `append bool` parameter to `SetPostTerms()`. When true, skip the DELETE and just INSERT new relationships.

### No hierarchy loop detection on update
**Severity:** Low
**WordPress behavior:** `wp_update_term()` validates that setting a new parent doesn't create a circular hierarchy (term A → parent B → parent A).
**Press behavior:** `UpdateTaxonomy()` accepts any parent value without validation.
**Fix:** Walk up the parent chain before setting. If the term being updated appears in its own ancestor chain, return an error.

## Options

### Get/GetValue has no default value parameter
**Severity:** Medium
**WordPress behavior:** `get_option($option, $default_value)` returns the default if the option doesn't exist, instead of an error. Many callers rely on this: `get_option('posts_per_page', 10)`.
**Press behavior:** `GetValue()` returns an error if the option doesn't exist. Every caller must handle the error case.
**Fix:** Add `GetValueDefault(name, defaultValue string) string` that returns the default on not-found instead of erroring. Or change `GetValue` to accept a default parameter.

### No old-value comparison on Set
**Severity:** Low
**WordPress behavior:** `update_option()` compares old and new values. If identical, it skips the DB write entirely and returns false.
**Press behavior:** Always writes to DB via `INSERT ON CONFLICT UPDATE`, even if the value hasn't changed.
**Fix:** Minor optimization. In `Set()`, check if option exists with same value and autoload before writing. Low priority.

## Links

### No major divergences
Links are simple in WP 1.0. The Press implementation covers CRUD, category filtering via term relationships, and visibility filtering. WordPress's `get_bookmarks()` has more sorting/filtering options but nothing critical is missing for the WP 1.0 feature set.

### Minor: link_updated not auto-set
**Severity:** Low
**WordPress behavior:** `wp_update_link()` updates the `link_updated` timestamp.
**Press behavior:** `link_updated` is whatever the caller provides.
**Fix:** In `Update()`, set `link.LinkUpdated = time.Now().UTC()` automatically.

## Shared Concerns

### No slug/sanitize_title package
Multiple repositories need the same slug generation logic (posts, terms, users). WordPress centralizes this in `sanitize_title()` and `sanitize_title_with_dashes()`. Press needs a shared package.

**Proposed: `internal/slug/slug.go`**
- `Generate(s string) string` — lowercase, strip HTML, convert accents, replace non-alphanumeric with hyphens, collapse consecutive hyphens, trim leading/trailing hyphens.
- `EnsureUnique(slug string, exists func(string) bool) string` — append `-2`, `-3` suffixes until `exists` returns false. Repositories pass their own existence check function.

### Error handling on side-effect queries
Several repositories fire "fire and forget" side-effect queries (comment count updates, meta cleanup, term relationship deletes) using bare `r.db.Exec()` without checking errors. These should at minimum log failures, even if they don't fail the parent operation.

## Priority Order

1. **Slug generation package** — unblocks posts, terms, users
2. **Comment count sync in Update()** — data integrity
3. **User delete with post reassignment** — data integrity
4. **Default category protection** — prevents broken state
5. **Child term reassignment on delete** — hierarchy integrity
6. **Child comment re-parenting on delete** — thread integrity
7. **Options GetValue with default** — API ergonomics
8. **Term slug auto-generation** — uses slug package from #1
9. **User nicename auto-generation** — uses slug package from #1
10. **Link updated timestamp** — trivial
