# Contributing

Thanks for contributing to `yaama`. The project is in early bootstrap, so
small, focused changes and clear context are especially helpful.

## Quick contribution checklist

Before opening a PR:

- [ ] Your branch is based on the latest `main`
- [ ] Your branch name is descriptive and follows the branch format below
- [ ] Your commit messages follow Conventional Commits
- [ ] You updated docs/specs when behavior or workflow changed
- [ ] You filled out `.github/PULL_REQUEST_TEMPLATE.md`
- [ ] You included a practical test/verification plan in the PR

## Getting started

1. Clone the repository and read `README.md`.
2. Read the relevant work item in `.plans/work/`.
3. Implement one logical slice at a time.

## Finding work

Work is currently tracked in `.plans/work/`:

- Start from `00-repository-init.md` and continue in numeric order unless a
  maintainer asks for re-prioritization.
- Keep scope tight to one work item or one clear sub-slice per PR.

## Development workflow

### Branches

Create branches from `main` using:

`<type>/<optional-issue>-<short-kebab-name>`

Examples:

- `feat/board-shell`
- `fix/tmux-attach-error`
- `docs/repository-init`

Recommended type prefixes:

- `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

### Commits

Use Conventional Commits:

```text
feat: add board shell startup model
fix: handle missing tmux binary gracefully
docs: clarify recovery flow in spec iteration
```

### Pull requests

1. Rebase or merge latest `main` into your branch.
2. Open a PR to `main`.
3. Complete `.github/PULL_REQUEST_TEMPLATE.md` fully.
4. Keep PRs reviewable (single concern, clear test plan).

## Code standards

- Follow the architecture and behavior defined in `.plans/` unless explicitly
  superseded.
- Keep changes minimal and consistent with existing naming and folder layout.
- Prefer clear, maintainable code over clever shortcuts.

## Testing and verification

While scaffolding is being built, include explicit verification evidence in PRs:

- What you ran locally
- What output/result confirmed success
- Any known gaps or TODOs

When Make targets and CI are in place, run the relevant checks before opening
or updating a PR.

## Documentation expectations

Update docs when you change:

- contributor workflow
- agent instructions
- command surface
- architecture or behavior described in `.plans/`

At minimum, keep `README.md`, `CONTRIBUTING.md`, and `AGENTS.md` aligned.

## Review process

- A maintainer reviews for correctness, scope, and clarity.
- Feedback should be addressed in follow-up commits on the same branch.
- PRs are merged once review concerns are resolved and verification is clear.
