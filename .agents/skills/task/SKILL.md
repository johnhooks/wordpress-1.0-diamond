---
name: task
description: Use when the user wants to think through a task and produce a task file in `.plans/tasks/`.
---

Help the user think through a task and produce a task file in `.plans/tasks/`.

This is a conversation, not a one-shot generation. Work with the user to understand the problem before writing anything.

## Process

1. **Understand the problem.** If $ARGUMENTS gives context, use it as a starting point. Ask the user what's wrong or missing today and why it matters. Explore the codebase if needed to ground the discussion in what actually exists.

2. **Clarify the desired outcome.** Talk through what the solution should look like at a high level. Push back if the user is jumping to implementation details. The task should capture what and why, not step-by-step how.

3. **Draft the task.** Once the problem and outcome are clear, write the task file and show it to the user. Ask if anything needs to change.

4. **Report the file path** when the user is happy with it.

## Naming

Use `{group}-{NN}-{slug}.md` where:

- `group` names the area of related work (e.g. `editor`, `posts`, `admin`, `comments`, `auth`). Reuse an existing group if one fits; invent a new one if none does.
- `NN` is a zero-padded sequence number within the group. Treat it as a loose dependency or ordering hint, not a strict ranking.
- `slug` is a few lowercase hyphenated words summarizing the task.

## Writing style

Focus on the **what** and the **why**. The Problem section should make it clear what's wrong or missing today and why it matters. The Plan section should describe the desired outcome and constraints, not step-by-step implementation instructions. Only get into the how if there's a non-obvious technical decision that needs to be captured.

The Requirements section is the testable contract: short bullets, each a statement that is either true or false after implementation. Cover behaviors that must hold, regressions that must not appear, and scope boundaries. Aim for 5–10 bullets; if you need more, the task is probably too big and should split.

Write in plain, direct language. No em dashes. No colons as sentence interrupters.

## Metadata

The frontmatter uses:

- `status` (required): One of `draft`, `todo`, `in-progress`, `done`
- `pr` (optional): GitHub PR number when work is in progress or done, e.g. `"#145"`

## Template

```markdown
---
status: draft
---

# {Title}

## Problem

{What's wrong or missing today, and why it matters.}

## Plan

{The desired outcome and any constraints. Keep it focused on what should change, not how to implement it line by line.}

## Requirements

- {Testable bullet}
- {Testable bullet}
```
