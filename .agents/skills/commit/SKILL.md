---
name: commit
description: Use when the user wants to prepare a Press git commit, choose a commit type, or format a commit message using the repo's commit conventions.
---

# Commit

Create git commits for Press.

## Format

```text
type: brief description
```

## Types

- `feat`: New behavior or capability
- `fix`: Bug fix or regression fix
- `docs`: Documentation only
- `refactor`: Restructure without intended behavior change
- `test`: Tests only
- `chore`: Maintenance or repo housekeeping

## Rules

1. Use lowercase.
2. Follow the `type: description` pattern.
3. Keep the description short and specific.
4. Match the existing repo style.
5. Do not add punctuation at the end.

## Process

1. Run `git status` and inspect the diff.
2. Stage the intended files with `git add`.
3. Choose the narrowest correct type.
4. Commit with `git commit -m "type: description"`.

## Recent examples

Good examples from this repo:

```text
feat: add the editor
feat: theme template evaluator
fix: comment-form scope and double-submit
refactor: panic on template errors
docs: add reference projects
```

## Guidance

- Use `feat` for additive product or developer-facing capabilities.
- Use `fix` when the main point is correcting behavior.
- Use `refactor` when the change is structural and should preserve behavior.
- Use `docs` when only documentation changed.
- If a commit mixes multiple concerns, either split it or name the dominant one.
