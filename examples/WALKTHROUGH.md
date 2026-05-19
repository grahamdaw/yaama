# yaama Walkthrough

Guided first-run setup for getting `yaama` working on your machine.

## 1) Prerequisites

From the repo root, make sure you can run:

```bash
go version
tmux -V
```

`yaama` also supports `mise` for matching the pinned Go toolchain.

## 2) Build and run once

```bash
make build
make run
```

This confirms the binary starts and your environment is healthy.

## 3) Install starter profiles

```bash
mkdir -p ~/.config/yaama/profiles
mkdir -p ~/.config/yaama/tmux
cp examples/profiles/default.toml ~/.config/yaama/profiles/default.toml
cp examples/profiles/dev.toml ~/.config/yaama/profiles/dev.toml
cp examples/tmux/dev-layout.tmux ~/.config/yaama/tmux/dev-layout.tmux
```

## 4) Edit profile values for your machine

Open copied profile files and update:

- `repo` to a local git repository path (required when `worktree = true`).
- The agent pane's `run` command (the pane with `agent = true`) so it points to an executable on your `PATH`.
- Toggle `worktree = true`/`false` depending on whether you want each session to get its own git worktree.
- Any sample `setup`/`teardown` script paths or pane `run` commands you do not want.

## 5) Understand the create flow

Profile-backed create is branch-first:

- Wizard flow: `profile -> task -> branch`
- Branch input is required
- When the profile sets `worktree = true`, the repo path must resolve to a git repository and `yaama` provisions a native git worktree at `<repo_parent>/.yaama-worktrees/<session-slug>`. With `worktree = false` (the default) the session uses `repo` directly.

No external worktree manager is required.

## 6) Create your first item

1. Start board: `make run`
2. Press `n`
3. Choose profile
4. Enter task ID/name
5. Enter branch name (example: `feat/my-task`)
6. Press `Enter` to create

On success, the agent row is created and tmux bootstrap runs in the resolved working directory.
Windows and panes are created in the order declared by `[[windows]]`; the pane marked `agent = true` is the agent pane and gets initial focus.

## 7) Daily operator keys

- `n` create
- `e` edit
- `Enter` attach to live session
- `r` recover dead session
- `s` status picker (`1..5` + `Enter`)
- `S` reverse status cycle
- `d` archive cleanup
- `D` hard prune cleanup
- `/` search by name/task/branch/session

## 8) Common troubleshooting

- **tmux unavailable**: install tmux or fix `PATH`.
- **not a git repository** on create: fix `repo` to a real git repo, or set `worktree = false`.
- **branch validation error**: use a git-safe branch name (for example `feat/my-task`).
- **dead session**: press `r`; if `working_dir` is invalid, edit with `e`.

## 9) Verify tests before changes

```bash
make test
```
