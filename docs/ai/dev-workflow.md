# Development Workflow

Common workflow guidance for AI agents working in the Optimism monorepo. Language-specific details are in [go-dev.md](go-dev.md) and [rust-dev.md](rust-dev.md). For running acceptance tests, see [acceptance-tests.md](acceptance-tests.md); for writing new ones, see [writing-acceptance-tests.md](writing-acceptance-tests.md).

## Tool Versions

All tool versions are pinned in `mise.toml` at the repo root. Always access tools through mise — never install or invoke system-global versions directly. Check `mise.toml` for current pinned versions when you need to know what's available.

If mise reports the repo isn't trusted, ask the user to run `mise trust` — never trust it automatically.

### Setup

Run `mise install` to install all pinned tools (just, gotestsum, forge, etc.). AI agent shells typically do not have mise activated, so prefix commands with `mise exec --` to ensure tools are on `PATH`:

```bash
mise exec -- just <target>
```

Then install the git hooks (once per clone — the setting is shared across all of a
clone's worktrees):

```bash
mise exec -- just install-git-hooks
```

This points `core.hooksPath` at `.githooks/`. The `pre-push` hook blocks pushing
unformatted Rust (mirroring CI's `rust-fmt` gate), so run it before you push any
Rust change.

## Build System

The repo uses [Just](https://github.com/casey/just) as its build system. Shared justfile infrastructure lives in `justfiles/`. Each component has its own justfile — run `just --list` in any directory to see available targets.

## Before Every PR

After running language-specific commit checks (lint, test):

1. **Run pre-push checks** — after committing and before pushing, run:
   ```bash
   ops/scripts/precommit-targets.sh --run
   ```
   This script selects a quick local sanity set based on the files changed on the branch. It is not a replacement for CI.

2. **Run affected tests broadly when needed** — don't just test the package/crate you changed when the change can affect dependents.

3. **Rebase on `develop`** — this is the default branch, not `main`:
   ```bash
   git fetch origin develop
   git rebase origin/develop
   ```

4. **Follow PR guidelines** — see `docs/handbook/pr-guidelines.md`. Keep the PR description brief — include only what isn't obvious from the diff.

## AI Agent Hooks

Claude and Codex have post-command hooks that remind the agent after `git commit` to run `ops/scripts/precommit-targets.sh --run` before pushing. The hook is a reminder, not a gate, so agents must still run the command and report the result.

Claude loads `.claude/settings.json` from the repo. Codex uses the workspace plugin in `plugins/optimism-ai-hooks`; if it is not installed, run:

```bash
codex plugin marketplace add .
codex plugin add optimism-ai-hooks --marketplace optimism-workspace
```

## CI

Some tests require CI-only environment variables and are skipped locally. Check the test code for environment variable guards if a test behaves differently than expected.
