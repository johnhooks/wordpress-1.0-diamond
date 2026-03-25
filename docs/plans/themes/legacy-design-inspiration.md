# Legacy Design Inspiration

The design language of WordPress 1.0 Platinum (January 2004) and
WordPress 1.5 Strayhorn (February 2005). What a blog looked like when
a blog was still a blog.

---

## WordPress 1.0 Platinum — "Dave Shea's Canvas"

The default stylesheet (`wp-layout.css`) was written by Dave Shea of CSS
Zen Garden fame, modified by Matt Mullenweg. The author's note: "This is
just a basic layout, with only the bare minimum defined. Please tweak
this and make it your own. :)"

That note tells you everything about the philosophy. The design is a
starting point, not a destination. It provides enough visual identity to
be pleasant and enough restraint to invite personalization.

### Visual Personality

Calm, intellectual, inviting. The palette is warm and earthy — sage
greens, olive browns, muted grays. No bright colors anywhere. No
gradients, no shadows, no rounded corners. It feels like a personal
notebook that happens to be on the internet. The design says: "This is
a place for writing. We'll get out of the way."

### Color Palette

The entire design is built on a warm, muted palette with sage green as
the single accent color.

| Color        | Hex       | Usage                                    |
| ------------ | --------- | ---------------------------------------- |
| Sage green   | `#90a090` | Header bar, footer bar — the brand color |
| Lighter sage | `#aba`    | Border accents on header/footer          |
| Dark olive   | `#565`    | Body border, header accent               |
| Brown        | `#675`    | Links — warm, not aggressive             |
| Dark brown   | `#342`    | Visited links — nearly black             |
| Light sage   | `#9a8`    | Hover links — gentle feedback            |
| Gray         | `#808080` | Metadata text — de-emphasized            |
| Light gray   | `#ccc`    | Borders, dividers — subtle structure     |
| White        | `#fff`    | Background                               |
| Black        | `#000`    | Body text                                |

No blues, no reds, no corporate colors. The palette comes from nature —
muted forest tones. Links are brown, not the web-default blue. This is
a deliberate rejection of the institutional web.

### Typography

Two font families create a clear hierarchy through contrast, not size.

**Body text**: `Lucida Grande, Lucida Sans Unicode, Verdana` — humanist
sans-serifs. Not Helvetica (too corporate), not Arial (too generic).
Lucida Grande was the Mac OS X system font; this choice signaled the
blog's audience. Set at 90% with 175% line-height — generous vertical
rhythm that prioritizes comfortable reading. Negative letter-spacing
(-1px) tightens the sans-serif slightly.

**Headings**: `Times New Roman, Times, serif` — editorial, journalistic.
The serif/sans-serif contrast between headings and body creates visual
hierarchy without relying on size or weight alone. Date headings are set
at 80% (smaller than body text) with wide letter-spacing (0.2em) — they
gain presence through spacing, not scale. Post titles use the serif face
with a light dotted underline.

**Metadata**: 75% size, gray (#808080), zero letter-spacing. Deliberately
small and quiet — present but not competing with the content.

The overall typographic feel is editorial. Serifs for structure,
sans-serifs for reading. Like a newspaper where the dateline is set
differently from the body copy.

### Layout

Two columns. Content on the left, sidebar on the right. The sidebar
uses `position: absolute` — a 2004 technique that pins it to the
top-right corner of the page, independent of content flow.

Content gets a 3em left margin and a 13em right margin (to clear the
sidebar), plus 60px of right padding. The sidebar is 11em wide with a
dotted left border. No wrapper constrains the maximum width — the body
stretches edge to edge, bounded only by its dark olive border.

The header is a full-width sage green bar with the blog title in large
italic serif. The footer mirrors it — same sage green, same width, with
a double top border. The content lives between these two bars.

### Borders and Depth

The body itself has an asymmetric border: 3px top, 2px sides, 1px
bottom, all in dark olive (#565). This creates a subtle sense of
physical weight — heavier at the top, lighter at the bottom.

The header has four different borders: top and left in lighter sage
(#9a9), bottom in a double line (#aba), right in dark olive (#565).
This layering creates a faux-3D bevel effect — the bar appears to
have depth despite being a flat green rectangle.

Dotted borders separate content. Date headings get a dotted bottom in
light gray. Post titles get a dotted bottom in near-white (#eee) — so
faint you barely notice it. The sidebar's left edge is dotted. Dotted
lines suggest structure without weight.

### Interactive Elements

Links change color on hover but never change background. No underlines
by default; underlines appear only on hover in specific contexts. The
interaction design is minimal — color shifts only, no animations, no
visual effects.

The sidebar's nested links are black (not the default brown) and gain a
solid bottom border on hover — a single green pixel line that appears
under the word. This is the most visually distinctive hover state in the
entire design.

Form inputs are white with a 1px dark border. No rounded corners, no
shadows, no focus states. The comment submit button says "Say it!" — the
only moment of personality in the UI.

### The Sidebar

The sidebar uses italic serif headings for section titles ("categories",
"archives", "meta") at 110% with letter-spacing. These are lowercase,
bold, with an italic serif face — they look like labels in a notebook
margin. Below each heading, nested links use the body sans-serif at 70%
— dramatically smaller, switching from the decorative to the functional.

This size contrast (large italic serif heading → small sans-serif links)
gives the sidebar a distinctive rhythm. Each section feels like a titled
list in a reference work.

### The Credit Bar

The footer is a narrow sage green bar with an 11px white credit line
("Powered by WordPress"). It mirrors the header in color but not in
scale — it's minimal, almost an afterthought. A double top border in
lighter sage (#aba) separates it from the content above.

---

## WordPress 1.5 Strayhorn — "Kubrick"

When WordPress 1.5 shipped, it introduced themes. The default theme was
Kubrick, designed by Michael Heilemann. The old WP 1.0 design was
preserved as "WordPress Classic" — the same stylesheet in a theme
directory.

Kubrick was a leap forward in visual sophistication. Where WP 1.0 was a
canvas, Kubrick was a finished design.

### Visual Personality

Polished, professional, centered. Kubrick introduced the idea that a
blog could look like a designed artifact, not a text document with
margins. The dusty blue header, the centered layout, the generous
whitespace — this was Web 2.0 minimalism before the term existed.

### Color Palette

Kubrick is far more restrained than WP 1.0 — it's essentially
monochrome with a single accent color.

| Color           | Hex       | Usage                                    |
| --------------- | --------- | ---------------------------------------- |
| Dusty blue      | `#73a0c5` | Header background — the brand color      |
| Near-black      | `#333`    | Body text, headings                      |
| Bright blue     | `#06c`    | Links — standard web blue                |
| Dark blue       | `#147`    | Link hover                               |
| Dusty rose      | `#b85b5a` | Visited links — warm contrast            |
| Medium gray     | `#777`    | Secondary text (sidebar, metadata)       |
| Light gray      | `#eee`    | Footer background                        |
| Off-white       | `#f8f8f8` | Alternating comment rows                 |
| Border gray     | `#959596` | Page border                              |
| Background gray | `#d5d6d7` | Body background (visible as page margin) |
| White           | `#fff`    | Page background                          |

Where WP 1.0 used warm earth tones, Kubrick uses cool blue-grays. The
only warmth is the visited link color (#b85b5a, dusty rose) — everything
else is clinical and clean.

### Typography

All sans-serif. The serif/sans-serif contrast from WP 1.0 is gone.

**Blog title**: `Trebuchet MS` at 4em (40px) — massive, centered, white
on blue. This is the most dramatic typographic element in either design.

**Headings**: `Trebuchet MS, Lucida Grande, Verdana` — the same stack but
with Trebuchet first. Trebuchet was the "designer's sans-serif" of the
era — sharper than Verdana, less corporate than Helvetica.

**Body text**: `Lucida Grande, Verdana, Arial` — same as WP 1.0 but with
a 62.5% base size reset (setting 1em = 10px for easier math). Content
renders at 1.2em (12px) with 1.4em line-height — tighter than WP 1.0's
175%.

**Sidebar headings**: `Lucida Grande, Verdana` — smaller (1.2em), no
Trebuchet. The sidebar is explicitly less prominent than the content.

The typographic personality is more uniform than WP 1.0. Everything is
sans-serif, everything is Trebuchet or Lucida Grande. The hierarchy
comes from size and weight, not font contrast.

### Layout

Fixed-width centered box. 760px wide (the safe width for 800×600
monitors with scrollbars), centered on a gray background with auto
margins. The page is a white rectangle floating on gray — a card.

**Two layouts depending on context:**

Posts with sidebar (home, archives): 450px content column, 190px
sidebar. The content floats left, the sidebar uses a left margin of
545px to position itself.

Posts without sidebar (single post, pages): 450px wide column with a
150px left margin. The sidebar disappears entirely, giving the content
more visual breathing room.

The choice of which layout to use is determined by images — the header
PHP switches between `kubrickbg.jpg` (with sidebar column shading) and
`kubrickbgwide.jpg` (without). The layout difference is literally baked
into the background image.

### The Header

200 pixels tall. Blue (#73a0c5) background with a decorative image
(`kubrickheader.jpg`) positioned at the bottom. Inside that, a
`#headerimg` div allows a custom image (`personalheader.jpg`, minimum
760×200) to replace the default.

The blog title is centered at 4em white text with 70px of top padding,
pushing it into the upper portion of the header area. Below it, the blog
description in 1.2em centered white text.

This header is the defining visual element of Kubrick. It turns the blog
title into a banner — something WP 1.0 never attempted.

### Image-Driven Design

Kubrick uses background images extensively:

- `kubrickbgcolor.jpg` — body background (tile color for the gray surround)
- `kubrickbg.jpg` — page background with sidebar column (repeats vertically)
- `kubrickbgwide.jpg` — page background without sidebar (repeats vertically)
- `kubrickheader.jpg` — header decoration
- `kubrickfooter.jpg` — footer decoration
- `personalheader.jpg` — user-customizable header photo

This was a significant design technique of the era. Before CSS3
gradients and border-radius, visual effects were achieved through tiled
images. The column differentiation (content vs sidebar) is done entirely
through the background image — there's no CSS border between them.

### Comments

Kubrick introduced alternating comment styling. Every other comment gets
the `.graybox` class — a barely-off-white background (#f8f8f8) with thin
top/bottom borders (#ddd). This zebra-striping makes long comment
threads scannable.

The comment count heading uses contextual language: "No Responses to
'Post Title'", "One Response to 'Post Title'", "X Responses to 'Post
Title'" — grammatically correct pluralization.

### The Footer

Light gray (#eee) background with the blog name, a "proudly powered by
WordPress" credit, and links to the RSS and comments RSS feeds. Much
more informational than WP 1.0's minimal credit bar.

---

## Comparing the Two Eras

### What Changed

| Aspect         | WP 1.0 Platinum                     | WP 1.5 Kubrick                    |
| -------------- | ----------------------------------- | --------------------------------- |
| Layout         | Fluid width, positioned sidebar     | Fixed 760px centered card         |
| Color identity | Sage green, earth tones             | Dusty blue, cool grays            |
| Typography     | Serif headings, sans body           | All sans-serif                    |
| Header         | One-line green bar                  | 200px banner with image           |
| Sidebar        | Absolute positioned, always present | Float-based, conditionally hidden |
| Background     | White, edge-to-edge                 | Gray surround, white card         |
| Design assets  | Pure CSS                            | Background images for effects     |
| Personality    | Personal notebook                   | Professional publication          |
| Borders        | Dotted lines, 3D bevels             | Thin solid lines, flat            |
| Title size     | 230% italic serif                   | 400% bold sans-serif              |
| Customization  | "Tweak this and make it your own"   | Drop in personalheader.jpg        |

### What Stayed the Same

Both designs share core principles that define the 2000s blog:

**Content first.** The post content is the largest visual element on the
page. Everything else — sidebar, header, footer, metadata — is smaller,
quieter, and less prominent.

**Two-column layout.** Content on the left, navigation on the right.
This is the blog layout. It wasn't invented by WordPress but WordPress
made it the default expectation.

**The sidebar is a reference panel.** Categories, archives, search,
meta links. It's a table of contents for the blog, not a feature area.
Both designs make the sidebar visually subordinate to the content.

**Metadata is small.** Author, date, categories — this information
appears but never competes with the title or content. It's set smaller,
grayer, less prominent.

**Links are the primary interactive element.** No buttons (except form
submits), no cards, no hover effects on containers. You read text and
you click links. The blog is a document, not an application.

**The header establishes identity.** Blog name, always linked to home.
The header is the only branded element — everything else is neutral.

**The footer is minimal.** A credit line. Maybe feed links. Nothing more.

**No navigation menu.** Neither design has a horizontal nav bar. The
sidebar IS the navigation. This is a deliberate choice — a blog is not
a website with pages. It's a chronological stream with a reference panel.

**Server-rendered HTML.** No JavaScript framework, no client-side
rendering, no loading states. The page arrives complete. Every link is a
full URL. Every page works without JavaScript.

---

## The Blog Aesthetic

Stripped to its essence, the 2000s blog design language is:

**A column of text with a margin of reference.** The text is the blog.
The margin tells you where you are and what else exists. Everything
serves the text.

**Muted, warm, readable.** Neither design uses bright colors for content.
The accent color (green or blue) appears only in the header/footer
frame. Inside that frame, it's black text on white with gray for
secondary information.

**Typographically opinionated.** Both designs make deliberate font
choices — not system defaults, not web-safe fallbacks chosen for
compatibility, but specific faces chosen for personality. Lucida Grande
for body copy (the Mac font, signaling the creative class). Serif or
Trebuchet for headings (signaling editorial authority or design
awareness).

**Horizontally bounded, vertically infinite.** The page has a defined
width but no defined height. Content flows downward forever. Pagination
exists but the metaphor is a scroll, not a book.

**Personally owned.** The blog carries its owner's name, not a
platform's brand. The "powered by" credit is a footer afterthought.
The header says who writes here.

---

## The Admin

WordPress 1.0's admin interface uses a completely separate design
language from the frontend. Where the blog is warm and editorial, the
admin is cool and utilitarian.

**Colors**: Navy blue (#00019b) links, gray borders, white backgrounds.
No sage green, no earth tones. The admin borrows from the Windows XP–era
application aesthetic.

**Typography**: Georgia serif throughout — more formal than the frontend's
Lucida Grande. 10pt base size, tighter spacing.

**Interaction**: Edit links have gray hover backgrounds. Delete links
turn red on hover (#c00 background, white text) — the only moment of
color urgency in the entire design. This red-for-delete convention
persists across all future WordPress versions.

**Layout**: Horizontal tab bar at top, content below. Form-driven pages
with labeled inputs. Categories and options in a right-side panel. The
admin is a tool interface, not a reading experience.

The frontend and admin share no visual DNA. They are two different
designs for two different activities: reading and writing.
