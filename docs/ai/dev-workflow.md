# Development Workflow

Common workflow guidance for AI agents working in the Optimism monorepo. Language-specific details are in [go-dev.md](go-dev.md) and [rust-dev.md](rust-dev.md).

## Tool Versions

All tool versions are pinned in `mise.toml` at the repo root. Always access tools through mise — never install or invoke system-global versions directly. Check `mise.toml` for current pinned versions when you need to know what's available.

If mise reports the repo isn't trusted, ask the user to run `mise trust` — never trust it automatically.

## Build System

The repo uses [Just](https://github.com/casey/just) as its build system. Shared justfile infrastructure lives in `justfiles/`. Each component has its own justfile — run `just --list` in any directory to see available targets.

## Before Every PR

After running language-specific commit checks (lint, test):

1. **Run affected tests broadly** — don't just test the package/crate you changed. Test packages that depend on it too.

2. **Rebase on `develop`** — this is the default branch, not `main`:
   ```bash
   git fetch origin develop
   git rebase origin/develop
   ```

3. **Follow PR guidelines** — see `docs/handbook/pr-guidelines.md`.

## CI

Some tests require CI-only environment variables and are skipped locally. Check the test code for environment variable guards if a test behaves differently than expected.
