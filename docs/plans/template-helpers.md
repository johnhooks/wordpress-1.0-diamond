# Template Helpers

Some values in a template cannot be pre-computed in the view struct
because the theme controls the argument. A date format, a
translation string, a pluralization rule. These are decisions the
theme author makes, not the engine. The view does not know how the
theme wants to display a date or what language it speaks.

Template helpers are a small set of functions available in template
expressions. They are not a general-purpose extension mechanism.
The set is fixed by the engine. Theme authors cannot register their
own.

---

## The Need

Five cases have come up where a view field is not enough:

**Date formatting.** The view provides a time value. The theme
decides how to display it. `{date post.TheDate "Jan 2, 2006"}`
produces `"Mar 22, 2026"`. The format string is Go's reference
time format. The view cannot know the theme's preferred format.

**Relative time.** `{timeago post.TheDate}` produces "3 days ago"
or "just now." The computation depends on the current time and
the display rules are a presentation concern.

**Translation.** `{__ "Leave a comment"}` returns the translated
string for the current locale, or the input string if no
translation exists. The theme author writes the strings. The
engine resolves them.

**Pluralization.** `{_n post.CommentCount "comment" "comments"}`
returns the correct plural form based on the count and locale
rules. English has two forms. Other languages have more.

**String interpolation.** `{sprintf "%d comments" post.CommentCount}`
formats a string with values. Most useful inside translated strings
where word order varies by language.

---

## Design Constraints

**Helpers never fail.** A bad date format returns the raw value. A
missing translation key returns the key itself. An invalid sprintf
pattern returns the pattern. The blog always renders something.
The theme author sees the unformatted value and knows something is
wrong. The blog reader still sees a page.

**No error returns.** The function signature is `func(args) string`.
No error, no panic, no recovery needed. The template language has
no error handling and should not need any.

**Fixed set.** The engine defines which helpers exist. There is no
registration API, no plugin hooks, no user-defined functions. If a
new helper is needed, it is added to the engine and documented.

**Expression syntax.** Helpers will require function call syntax in
the expression parser, which does not exist yet. The parser needs
to learn `name(arg1, arg2)` as a primary expression. The evaluator
needs a function registry that the walker provides.

---

## Not Now

This is future work. The current template engine handles everything
through view struct fields, and that is correct for now. When we
build the first theme that needs date formatting or translation,
that is when we add helpers. The parser change is small. The
function registry is small. The five functions listed above are
the likely starting set.

The important decision is already made: helpers are pure functions
that always return a value. They do not fail. They do not panic.
They do not need error handling in the template language.
