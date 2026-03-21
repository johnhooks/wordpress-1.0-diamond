# Admin Legacy Design Inspiration

The design language of WordPress 1.0's admin interface. A separate
world from the frontend. Where the blog is warm and editorial, the
admin is cool and utilitarian. Two different designs for two different
activities: reading and writing.

---

## The Admin Aesthetic

The WP 1.0 admin is a tool interface. It borrows from the Windows XP
era of application design. Gray backgrounds, navy blue links, Georgia
serif text. No personality. No branding beyond the WordPress wordmark.
The admin exists to get out of the way so you can write.

### Color Palette

| Color | Hex | Usage |
|---|---|---|
| White | `#fff` | Page background, input backgrounds |
| Light gray | `#eee` | Alternating table rows (.alternate) |
| Medium gray | `#ccc` | Borders, dividers |
| Dark gray | `#666` | Secondary text |
| Black | `#000` | Primary text |
| Navy blue | `#00019b` | Links — institutional, not playful |
| Red | `#c00` | Delete hover state — danger signal |
| White on red | `#fff` on `#c00` | Delete button hover text |

No accent color. No brand color. The admin is deliberately colorless.
The only color moments are navy for links and red for destructive
actions. This red-for-delete convention persisted through every
WordPress version that followed.

### Typography

**Body text:** Georgia serif throughout. 10pt base size. This is more
formal than the frontend's Lucida Grande sans-serif. The admin reads
like a document, not a website.

**Headings:** Georgia bold at slightly larger sizes. No font contrast
between heading and body. The hierarchy comes from size and weight
alone.

**Form labels:** Same Georgia, no special treatment. Labels and values
use the same typeface.

The overall typographic feel is institutional. A government form. A
bank statement. Deliberately boring so the content you are writing is
the interesting part.

### Layout

No fixed width. The admin stretches edge to edge. Content is wrapped
in a `.wrap` div with borders, margin, and padding that creates a
card-like container within the full-width page.

Navigation is a horizontal bar of text links at the top. No icons.
No dropdown menus. Flat list of page names separated by space. The
current page is bold.

Below navigation, a page title (h2). Below that, the content area.
Tables, forms, and text. No columns. No sidebar. One column of
content.

### Tables

Tables are the primary data display pattern. Post lists, comment
lists, user lists, option lists. All tables.

Alternating row colors (`.alternate` class) use `#eee` for every
other row. No hover states on rows. Cell padding of 3px. Headers in
bold. Links in cells use the navy blue.

### Forms

Forms use a simple pattern: label above or beside input, full-width
or fixed-width inputs. No placeholder text. No inline validation. No
floating labels. A label, an input, another label, another input.

Textareas for content are wide (99% or cols=50). Text inputs are
sized by context (30 for titles, 18 for passwords). Submit buttons
have no special styling beyond font-weight: bold for the primary
action.

### Interactive Elements

Links change color on hover. Delete links gain a red background with
white text on hover. This is the single moment of visual urgency in
the entire interface.

No animations. No transitions. No loading states. Click a link, the
page refreshes, the new state appears. The interaction model is
entirely synchronous.

### The Write Page

The write page is the admin's best work. Title input at the top,
auto-focused. Content textarea below, wide and tall. Category
selector floated to the right. Status controls at the bottom.
Three buttons: Save as Draft, Save as Private, Publish (bold).

Everything serves the writing. The title is the first thing you
type. The content area is the largest element on the page. The
publish button is the last thing you click. The flow is top to
bottom: name it, write it, publish it.

The quicktags toolbar sits above the content area. Small text
buttons: b, i, a, img, blockquote, code. No icons. No dropdowns.
Text labels that insert HTML tags. It is minimal to the point of
being nearly useless, which is why Press replaces it with ProseMirror.

### The Manage Page

The post list is a stream of blocks. Each block shows the date (bold),
action links (edit, delete, comment count), the title (linked), the
author and category (in a cite element), and the post content. This
is not a table. It is a vertical list of post summaries.

This is unusual. Most admin interfaces use tables for data lists.
WordPress 1.0 used a narrative format. Each post reads like a card.
Later versions moved to tables. The card format has charm but scales
poorly.

### The Moderation Page

The moderation queue is the most interactive page in the admin. Each
comment shows full metadata (author, email, URL, IP) and the comment
text. Below each comment, three radio buttons: Approve, Delete, Do
Nothing. One submit button at the bottom processes everything.

This is a batch operation interface. Review many items, make decisions
for each, submit once. No AJAX. No inline actions. Review, decide,
submit, see results. The WP 1.0 admin was patient software.

---

## What Carries Forward

**Content first.** The write page puts the title and content area
front and center. Everything else is secondary. Press should do the
same.

**Minimal chrome.** The admin has no decorative elements. No icons,
no illustrations, no color blocks. Just text, inputs, and links. The
admin is a tool. It should look like one.

**Horizontal navigation.** One row of links. No hamburger menus, no
collapsible sidebars, no icon rails. If it does not fit in one row,
there are too many items.

**Tables for data.** Post lists, comment lists, user lists. Tables
work. They are accessible, scannable, and sortable. Do not reinvent
them.

**Red for delete.** The only color signal in the admin. Destructive
actions turn red on interaction. Everything else is neutral.

**The action pattern.** Fill out a form, click submit, see the
result. Press replaces the full-page refresh with htmx swaps, but
the mental model is the same: act, then see.

## What Does Not Carry Forward

**Edge-to-edge layout.** The admin should have a maximum width. Wide
monitors make 100% width forms unusable.

**Tiny type.** 10pt Georgia was already small in 2004. Use a readable
base size.

**Plain text password fields.** WP 1.0 used `type="text"` for
password inputs in the user creation form. No.

**No focus states.** Inputs had no visual feedback on focus. Every
input should have a clear focus indicator.

**No responsive design.** The admin was designed for 800x600. It
should work on phones. Not as a priority, but it should not break.

**Manual SQL in templates.** WP 1.0 ran database queries inside admin
page files. The handler provides the data. The template renders it.
The template never touches the database.
