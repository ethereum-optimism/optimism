# Git hooks

Version-controlled git hooks for this repo. Install them once per clone with:

```bash
just install-git-hooks
```

That points `core.hooksPath` at this directory, so every hook here runs for all
worktrees of the clone. Re-running the command is a no-op.

To add a hook, drop an executable script named after the git hook (e.g.
`pre-commit`, `commit-msg`) into this directory — no install change is needed.

## Hooks

- `pre-push` — blocks pushing unformatted Rust when the push contains `.rs`
  changes. Runs `just fmt-check-all`, the aggregate of the repo's Rust fmt checks
  (`fmt-check` + `fmt-check-sp1-guest`), so the hook doesn't hardcode that list.
