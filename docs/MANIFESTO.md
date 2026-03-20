# WordPress 1.6

WordPress 1.0 "Platinum" shipped in January 2004. WordPress 1.5
"Strayhorn" shipped in February 2005. Then the version number jumped
to 2.0 and WordPress walked one path. Press walks the other one.

What if the blog stayed a blog? What if someone kept working on 1.x?
Not out of disagreement with where WordPress went, but out of curiosity
about the road not taken.

Press is WordPress 1.6. The release that never happened. We're making
it happen to see what we find.

---

## The Idea

Take a 2004 blogging engine. Rebuild it with 2025 tools. Go instead of
PHP. SQLite instead of MySQL. htmx instead of React. But keep the soul
of the original. A tool for writing. A way to share words on the
internet.

Then follow the 1.x line forward. What does a blog-first platform look
like when you give it twenty years of hindsight but none of the CMS
ambitions? What if you build a writing tool instead of a website
builder?

We don't know. That's the point. This is an exploration disguised as
software.

---

## The Constraints

The interesting part of Press isn't the technology. It's the
constraints. We impose rules on ourselves that aren't normal in modern
development, because constraints produce interesting work.

### Server-Rendered HTML. As Far As It Goes.

The server sends HTML. The browser displays it. Every page, every
fragment, every admin panel. No client-side framework. htmx swaps HTML
fragments for interactivity.

We're not dogmatic about it. Some things need rich client-side
interaction. The post editor is ProseMirror, a JavaScript application
that talks to the server over a clean API. That's the right tool for
collaborative rich text editing, and pretending otherwise would be
stubborn, not principled.

The line is: server-rendered HTML is the default. When something
genuinely needs client-side state, we use web components or purpose-
built JavaScript. We don't reach for a framework. We reach for the
smallest tool that solves the specific problem.

### SQLite. Single Binary. Zero Dependencies.

`press serve` and you're blogging. One file for the binary, one file
for the database. No Docker. No database server. Your blog is a file
on your computer.

### Templates Use `TheTitle`, `TheContent`, `TheDate`

WordPress template tags were `the_title()`, `the_content()`,
`the_date()`. Press carries the same names: `{{.TheTitle}}`,
`{{.TheContent}}`, `{{.TheDate}}`.

This serves no technical purpose. It exists because this is WordPress
1.6 and the heritage matters, even when it makes Go developers twitch.
Sometimes you just have to make bad decisions for the joy of it.

### No Plugin System. Just Go Packages.

Press is a Go library. Want to extend it? Import a package, compile
your binary. The Go compiler catches incompatibilities. No runtime
surprises.

This is harder than dropping a plugin into a folder. It's also a
fundamentally different safety model. We're curious what falls out of
that trade-off.

### No Hooks. No Filters.

WordPress 1.5 introduced themes and hooks at the same time. Themes
controlled presentation. Hooks let anyone modify anything at runtime.
Press takes the themes and leaves the hooks. What happens when
extension is explicit and compiled instead of implicit and runtime?

### Themes Are Compiled Artifacts

No live editing on production. You build a theme, preview it, apply it.
The writing experience is the priority. Pick a theme, tweak the
colors, forget about it, write for years. The writer who just wants to
write is who we're building for.

### We Never Leave 1.x

There is no 2.0. There is no CMS pivot. The version number stays in
1.x forever. If we run out we'll use 1.10 and keep going. This
constraint is the whole experiment. What happens when you commit to
"this is a blog" and never waver?

---

## Where This Comes From

I've spent the last three years building a Laravel application that
manages WordPress sites remotely. I work around the edges of the
WordPress ecosystem every day. I have active explorations into React
Server Components for blocks and a Go-based WordPress alternative.
Those aren't done. This project exists alongside them, not instead of
them.

But I have a frustration I can't shake. Site editing in WordPress has
become extraordinary for site designers. The Full Site Editor is a
genuine page builder. But the priority has shifted toward building
pages and away from the writing and document lifecycle experience. The
writer, the person who just wants to open an editor and write something
and share it, is no longer the primary audience.

There's something that feels fundamentally off about the block editor
architecture. Blocks are written in React but can't be rendered on the
server or hydrated on the frontend. A block is a React component in
the editor and a PHP render function on the server, two implementations
of the same thing in two different languages. The current system is a
bit too weird for me to be willing to dive in fully, even though I
admire the ambition behind it. The problem is genuinely hard when your
server is PHP and your editor is React.

Press sidesteps the problem entirely by asking a different question.
What if we care less about blocks and more about the document? I can
express most of what I want to write in a markdown file. I usually do
it in a repository with a text editor. That works, but it's not a
sharing tool. It's not a publishing tool. It doesn't have permissions
or collaboration or a way to send someone a link to a draft.

Press is my attempt to build that missing tool. The problems I discover
along the way will teach me things I can't learn by staying on the
current path.

---

## What We're Actually Building

A written word document tool, one you own. The blog is the public face,
but underneath it's a document system. Private notes, shared drafts,
one-time links, published posts. All the same engine, all the same
permission system, all on your server.

Can a self-hosted blog be as easy to share as Google Docs? Can you send
someone a link to a draft as naturally as you'd share a document? Can
the permission model be that simple while the data stays on your
machine?

We think so. The technology has been there for years. We're going back
to the beginning to try.

---

## The Weird Stuff

Some of these decisions will be wrong. The `TheTitle` naming is almost
certainly wrong. Server-rendering the entire admin might hit a wall.
The theme compilation system is a gigantic experiment that might not
work.

We're not hiding this. Press is a curiosity project. It's a vehicle
for exploring ideas about the web, about writing tools, about what
software looks like when you choose constraints over flexibility. And
hopefully, eventually, it's a tool we actually use to write about the
journey.

A blog about blogging, built on a blogging engine that never existed,
in a version of WordPress from a timeline that didn't happen.

There's only one way to find out if any of this works.

---

*Press. WordPress 1.6. The release that never happened.*

*Let's see where it goes.*
