# AI Writing Assistance

**Status: Early thinking. High risk of building the wrong thing.**

Press will have AI writing assistance as a first-class concept, not a
plugin. But the design is unresolved. This document captures constraints
and open questions, not a spec.

## The Philosophy

The AI is a writing tool FOR the human. Not a content generator. Not a
magic button that produces text. A collaborator that helps the author
create better content the way they want to create it.

The path: nail human collaboration first — the editor, the step log,
editorial comments, the collab protocol. Then bring the AI in through
those same primitives. The AI should feel like a talented editor sitting
next to you, not a machine you feed prompts into.

## How It Works (When We Get There)

The AI participates through the same primitives as human collaborators.
No special modes. No separate interface. The same tools a human editor
uses.

### Review Mode

The AI reads the post and leaves editorial comments — annotations
anchored to specific text, just like a human editor would:

- "This paragraph contradicts your intro"
- "This claim needs a source"
- "The tone shifts here — intentional?"
- "This sentence is doing too much work"

The author reads the comments and decides what to act on. Same as
working with a human editor.

### Suggested Edits

An editorial comment can become a suggested edit. The AI highlights a
passage and proposes a rewrite. The author sees the original and the
suggestion side by side. Accept, reject, or modify — same as track
changes, same as a human collaborator's edits in the step log.

The key: suggested edits are ProseMirror steps attributed to the AI's
user identity. They appear in the revision timeline as clearly as any
human edit. "Claude suggested rephrasing paragraph 3" is as visible as
"Alice edited paragraph 3."

### Conversation

A comment thread can become a conversation. The author replies to the
AI's comment — "I see what you mean but I want to keep the informal
tone" — and the AI adjusts its next suggestion. Back and forth, same
as talking to a human editor, until the author is happy.

A conversation can lead anywhere: a single word change, a paragraph
rewrite, a structural reorganization. The scope grows naturally from
the dialogue, not from a prompt template.

### The Whole Spectrum

```
comment  →  suggested edit  →  conversation  →  full rewrite
```

These aren't different features. They're points on a spectrum. A review
comment is the lightest touch. A suggested edit is more concrete. A
conversation explores what the author actually wants. A full rewrite is
what happens when the conversation reaches "yeah, just redo the whole
thing." The author moves along this spectrum naturally, and can stop
at any point.

## What Makes This Different

Every AI writing tool today is a prompt box. You type what you want,
the AI generates text, you paste it in. The AI never sees the document
in context. The AI never engages with what you've already written. The
AI doesn't remember what you told it last time.

Press's AI lives inside the document. It reads what you wrote. It
understands the structure. It sees the revision history — what you tried
and rejected. It leaves comments anchored to specific passages. Its
suggestions are diffs, not replacements. The conversation happens in
the context of the writing, not in a separate chat window.

This only works because the collaboration infrastructure supports it.
The step log, editorial comments, WebSocket collab, and attribution
system were built for humans. The AI is just another client.

## What We Know

### The AI participates through the collab protocol

- **ProseMirror steps** — AI edits are steps in the step log, attributed
  to the AI agent. The author can see exactly what changed, when, and
  undo any of it. No black box replacements.
- **Editorial comments** — the AI leaves annotations instead of making
  edits. The author decides what to act on.
- **WebSocket collab** — the AI connects as a client in the editing
  session. Same protocol as a human collaborator. Not a special mode.

The AI agent is not a user in the `wp_users` sense — it doesn't have an
account, a login, or a user level. It's a system-level participant that
appears as a collaborator in the editing session. Steps and editorial
comments from the AI are attributed to a well-known identifier (not a
user_id), and the UI presents them as coming from a named collaborator.
The AI never shows up in user lists, author archives, or admin panels.

This means: the editing, revision, and collaboration systems must not
assume all clients are browsers or all participants are users. Build the
collab protocol generically.

### The author stays in control

The AI suggests, the author decides. This is non-negotiable.

- Every AI edit is visible and reversible in the step log
- The AI never publishes or changes visibility
- The AI never edits without the author's session being active
- The author can disable AI assistance per-post or globally

### It should feel natural

The AI writing experience should feel like working with a good editor,
not like operating a machine. No prompt engineering. No special syntax.
No mode switches. You're writing, and the AI is there if you want it.
If you don't want it, it's invisible.

## What We Don't Know

### How much context does the AI have?

- **Just the current post?** Simple but limited. Can't say "write this
  in the same tone as my last post."
- **All published posts?** The AI knows your writing style, your topics,
  your voice. Powerful but raises questions about context window limits
  and privacy.
- **Editorial comments and conversation history?** The AI remembers
  past feedback. "You told me not to use passive voice" persists.
- **The step log?** The AI can see how the post evolved. Knows what
  was tried and rejected.

### What are saved prompts?

Authors will develop patterns: "check for passive voice", "suggest a
better headline", "is this too long?" Are these:

- **Per-post instructions?** "For this post, maintain a formal tone."
- **System-level preferences?** "Always flag passive voice in my posts."
- **Reusable templates?** "Write an introduction for a technical blog
  post about [topic]."

This is product design, not engineering — the right answer depends on
how authors actually use AI assistance, which we don't fully know yet.

### What's the right data model?

We don't know enough yet to pick. Building the wrong data model is
worse than having no AI features.

## What We Build Now

Nothing AI-specific. But we build the prerequisites:

1. **Generic collab protocol** — any WebSocket client can send steps,
   not just browsers.
2. **Step attribution** — every step has a user_id, AI gets a user_id.
3. **Editorial comments** — inline annotations that work for both human
   and AI feedback.
4. **The permission model** — the AI's access is a tuple like anyone
   else's.

These are all things we're building anyway for human collaboration.
The AI integration is a future client of infrastructure that already
exists for other reasons.

## What We Don't Build Yet

- AI-specific tables or post types
- Conversation UI
- Prompt management
- AI model integration
- Context assembly (which posts/history to send to the model)

These wait until we have the editor, collab, and editorial comments
working for humans first. Then we'll know what the right AI integration
looks like because we'll be using the system ourselves.

## Risk

The biggest risk is designing an AI writing system that nobody wants
to use because it doesn't match how people actually write with AI
assistance. Every AI writing tool on the market is struggling with this.
We should watch what works and what doesn't before committing to a
design.

The second risk is building AI features that make the non-AI experience
worse — adding complexity, UI clutter, or conceptual overhead for authors
who don't want AI help. The AI must be invisible when not in use.
