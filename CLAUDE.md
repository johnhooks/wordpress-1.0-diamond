# Press

Press is WordPress 1.6, the release that never happened. WordPress 1.5 shipped themes and static pages; then 2.0 arrived with a rich text editor and the project began its long march toward CMS, page builder, and full site editor. Press picks up the 1.x line instead and stays there. No 2.0, no CMS pivot, no page builder. When a choice points at "flexible site builder" versus "better writing experience," pick writing. The full vision is in `docs/MANIFESTO.md`.

Press is written in Go with SQLite and htmx. The server renders HTML. Reach for JavaScript only when a problem genuinely needs client state, the way the ProseMirror editor does. There is no plugin system and no hooks. Extension is a Go import compiled into the user's binary, so incompatibilities surface at compile time rather than runtime. Constraint over flexibility is a recurring theme, and the answer to "can we make this configurable?" or "should we add a hook here?" is usually no.
