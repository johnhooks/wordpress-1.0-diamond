# Admin Component Contracts

The admin theme uses the same architecture as the frontend theme:
templates receive typed data, render HTML, the engine wires behavior.
The admin has fewer component types but more repetition. Form fields,
radio groups, and list rows appear on almost every page.

This document defines the component hierarchy and contracts for the
admin theme. The freerange admin theme is the reference implementation.

---

## Molecules

The smallest reusable pieces. Each is a named template that can be
called by organisms and page templates.

### TextField

A labeled text input with an optional error message.

**Template name:** `admin-text-field`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `ID` | string | Input id and label for attribute |
| `Name` | string | Form field name |
| `Label` | string | Display label |
| `Value` | string | Current value (empty for new) |
| `Error` | string | Validation error (empty if none) |
| `Autofocus` | bool | Whether to autofocus this field |

### TextareaField

A labeled textarea with configurable rows and an optional error.

**Template name:** `admin-textarea-field`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `ID` | string | Textarea id and label for attribute |
| `Name` | string | Form field name |
| `Label` | string | Display label |
| `Value` | string | Current value |
| `Rows` | int | Number of visible rows |
| `Error` | string | Validation error (empty if none) |
| `HelpText` | string | Optional hint text below the field |

### RadioGroup

A fieldset of radio buttons.

**Template name:** `admin-radio-group`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `Name` | string | Form field name |
| `Legend` | string | Fieldset legend text |
| `Options` | []RadioOption | Available choices |

Where `RadioOption` is:

| Field | Type | Description |
|---|---|---|
| `Value` | string | Option value |
| `Label` | string | Display label |
| `Selected` | bool | Whether this option is selected |

### CheckboxGroup

A fieldset of checkboxes from dynamic data.

**Template name:** `admin-checkbox-group`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `Name` | string | Form field name |
| `Legend` | string | Fieldset legend text |
| `Options` | []CheckboxOption | Available choices |

Where `CheckboxOption` is:

| Field | Type | Description |
|---|---|---|
| `Value` | string | Option value (usually an ID) |
| `Label` | string | Display label |
| `Checked` | bool | Whether this option is checked |

### Pagination

Previous/next navigation for paged lists.

**Template name:** `admin-pagination`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `HasPrev` | bool | Whether a newer page exists |
| `HasNext` | bool | Whether an older page exists |
| `PrevURL` | string | URL for newer entries |
| `NextURL` | string | URL for older entries |

### ActionLinks

A set of action links for a list row (edit, view, delete, etc.).
The template renders whatever links are non-empty.

**Template name:** `admin-action-links`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `EditURL` | string | Edit link (empty to hide) |
| `ViewURL` | string | View link (empty to hide) |
| `DeleteURL` | string | Delete link (empty to hide) |
| `ApproveURL` | string | Approve link (empty to hide) |

### FormErrors

Form-level error messages displayed at the top of a form.

**Template name:** `admin-form-errors`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `Errors` | []string | General error messages |

---

## Organisms

Page sections composed from molecules.

### PostForm

The write/edit post form. Used by both the write and edit page
templates. The same organism handles both new and existing posts.
An empty post (zero ID, empty fields) means new. A populated post
means edit.

**Template name:** `admin-post-form`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `Action` | string | Form action URL |
| `CSRFAction` | string | CSRF action name |
| `CSRF` | CSRFHelper | CSRF token generator |
| `Post` | AdminPostEdit | Post data (empty for new) |
| `Categories` | []CheckboxOption | Category checkboxes |
| `StatusOptions` | []RadioOption | Post status radios |
| `CommentOptions` | []RadioOption | Comment status radios |
| `Errors` | FormErrors | Validation errors |
| `IsEdit` | bool | Whether this is an edit (shows save vs publish) |
| `CanDelete` | bool | Whether to show delete link |
| `DeleteURL` | string | Delete action URL |
| `Permalink` | string | Public URL (edit only) |

### PostRow

A single post in the manage posts list.

**Template name:** `admin-post-row`

**Props:**

| Prop | Type | Description |
|---|---|---|
| `ID` | int | Post ID |
| `Title` | string | Post title |
| `Status` | string | Post status |
| `Date` | string | Formatted date |
| `AuthorName` | string | Author display name |
| `Categories` | string | Comma-separated category names |
| `CommentCount` | int | Number of comments |
| `EditURL` | string | Edit page URL |
| `ViewURL` | string | Frontend permalink |
| `DeleteURL` | string | Delete action URL |

### AdminHeader

The admin page header with navigation.

**Template name:** `admin-header`

**Props:** Accessed through `.Shell` on the page data.

### AdminFooter

The admin page footer.

**Template name:** `admin-footer`

**Props:** Accessed through `.Shell` on the page data.

---

## Templates

Full pages the engine calls by name. Each template includes
admin-header and admin-footer and composes organisms and molecules
for its content area.

### Required Templates

| Template | Engine calls | Description |
|---|---|---|
| `admin-login` | `GET/POST /wp-admin/login` | Login form (no shell) |
| `admin-write` | `GET /wp-admin/post/new` | New post form |
| `admin-edit` | `GET /wp-admin/post/{id}/edit` | Edit post form |
| `admin-posts` | `GET /wp-admin/posts` | Post listing |
| `admin-comments` | `GET /wp-admin/comments` | Comment listing |
| `admin-moderate` | `GET /wp-admin/moderate` | Moderation queue |
| `admin-categories` | `GET /wp-admin/categories` | Category management |
| `admin-users` | `GET /wp-admin/users` | User management |
| `admin-options` | `GET /wp-admin/options` | Site settings |
| `admin-profile` | `GET /wp-admin/profile` | User profile |

### Template Composition

Each template follows the same pattern:

```
admin-header
  [page-specific organisms and molecules]
admin-footer
```

The write and edit pages both use the `admin-post-form` organism.
The difference is the data: write passes an empty post, edit passes
a populated one. The organism handles both cases.

```
admin-write:
  admin-header
  admin-post-form (empty post, action=/wp-admin/post/new)
  admin-footer

admin-edit:
  admin-header
  admin-post-form (populated post, action=/wp-admin/post/{id}/edit)
  admin-footer

admin-posts:
  admin-header
  admin-post-row (loop)
  admin-pagination
  admin-footer
```

---

## Form Validation Flow

Validation errors flow through the same templates. The handler
validates the submission, and if it fails, re-renders the form with
errors populated in the props.

### Field-Level Errors

Each form field molecule accepts an `Error` string. When non-empty,
the template renders the error message adjacent to the field.

### Form-Level Errors

The `admin-form-errors` molecule renders general errors at the top
of the form (CSRF failure, internal errors, multi-field validation).

### The Handler Pattern

```go
// On POST:
errors := validate(r)
if len(errors) > 0 {
    // Re-render the form with errors and the submitted values
    s.renderAdmin(w, "admin-write", WritePostData{
        Post:   postFromForm(r),  // preserve what the user typed
        Errors: errors,
    })
    return
}
```

The form preserves the user's input on validation failure. The user
sees their text with error messages, not a blank form.

### Client Validation (Future)

Client-side validation will use the same field error slots. The
engine will add validation attributes (required, pattern, maxlength)
to form fields. JavaScript validates before submit and populates the
error spans. The server always validates again.

htmx can also validate inline: submit a single field to a validation
endpoint, get back an error span, swap it into the field's error
slot. This gives real-time feedback without a full form submission.
Both approaches use the same error slot in the molecule.

---

## What Exists Today vs What Is Needed

| Component | Status |
|---|---|
| admin-header | Implemented (inline in shell.html) |
| admin-footer | Implemented (inline in shell.html) |
| admin-login | Implemented |
| admin-write | Implemented (monolithic, needs decomposition) |
| admin-edit | Implemented (monolithic, needs decomposition) |
| admin-posts | Implemented (monolithic, needs decomposition) |
| admin-post-form | Not extracted yet |
| admin-post-row | Not extracted yet |
| admin-text-field | Not extracted yet |
| admin-textarea-field | Not extracted yet |
| admin-radio-group | Not extracted yet |
| admin-checkbox-group | Not extracted yet |
| admin-pagination | Not extracted yet |
| admin-action-links | Not extracted yet |
| admin-form-errors | Not extracted yet |
| admin-comments | Not implemented |
| admin-moderate | Not implemented |
| admin-categories | Not implemented |
| admin-users | Not implemented |
| admin-options | Not implemented |
| admin-profile | Not implemented |
