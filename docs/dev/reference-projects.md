# Reference Projects

External codebases we consult while building Press.

## `~/Projects/wordpress-1.0-platinum`

The original WordPress 1.0 blogging engine. The starting point for Press. Reach for it when you need to see how a specific surface looked at the start of the 1.x line.

## `~/Projects/wordpress-develop`

The full WordPress history from b2/cafelog through the present. Trace feature evolution through the 1.x line, which ended at `$wp_version = '1.5.1.2'` in May 2005. The modern database schema lives at `src/wp-admin/includes/schema.php`.

## `~/Projects/prosemirror`

The ProseMirror monorepo. `schema-basic/` and `schema-list/` define the node and mark types our post editor uses. `model/` is where ProseMirror's own schema validation lives, a useful reference when implementing our Go-side validator. `markdown/` documents the `code_block` params attribute we carry forward.
