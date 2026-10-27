# Development Workflow

Common workflow guidance for AI agents working in the Optimism monorepo. Language-specific details are in [go-dev.md](go-dev.md) and [rust-dev.md](rust-dev.md). For running acceptance tests, see [acceptance-tests.md](acceptance-tests.md); for writing new ones, see [writing-acceptance-tests.md](writing-acceptance-tests.md).

## Code Simplicity and Design

Apply these principles to code you write or refactor, within the requested scope. Make the
normal flow, ownership and failure behavior easy to review. Fix bugs by removing the conditions
that permit them: prefer eliminating an invalid state or a synchronization obligation over adding
another flag, recovery transition or defensive check.

1. **Use the fewest concepts needed.** Prefer ordinary functions and concrete components. Every
   wrapper, builder, channel and task should justify its existence through a meaningful boundary or
   invariant. Forwarding calls alone usually does not.
2. **Name domain operations and values precisely.** Use established imported names consistently
   instead of introducing aliases or renamed wrappers. Distinguish unrelated errors instead of
   giving them the same vague name. See the language guides for package and module naming.
3. **Write control flow from top to bottom with minimal nesting.** Handle invalid inputs and
   exceptional cases early with guard clauses, early returns and the language's error-propagation
   constructs. Keep the successful path straightforward and the order of side effects clear.
4. **Choose dependencies and modes at construction.** Resolve configuration once and pass each
   component the actual values it requires. A handler that needs a dependency receives it explicitly.
   Register optional components only when enabled, with complete inputs. Components should not
   repeatedly rediscover their mode or check whether required dependencies were supplied.
5. **Make invalid states impossible and store less state.** Establish invariants through construction,
   ownership and operation boundaries. Remove redundant flags, duplicated values and independently
   mutable fields that must always agree. Represent necessary phases explicitly with their associated
   data, instead of arbitrary combinations of booleans and optional fields. Each retained transition
   should correspond to something the system needs to do.
6. **Give true failures a terminal outcome.** In a long-running service, an essential component that
   cannot continue correctly must propagate its error to the supervisor for service or node shutdown.
   Do not log the error and leave a component stalled, a caller waiting forever or the node apparently
   healthy. Libraries return errors to their callers. Expected conditions, including temporary
   dependency outages, stale work and rejected external input, need explicit retry or discard
   behavior. Preserve enough typed information to make that decision in one authoritative place.
   Intentional shutdown and abandoned reply receivers are normal lifecycle events.
7. **Keep one owner for each invariant and decision.** Keep correctness checks and recovery policy in
   the component that owns them. Avoid making several callers interpret the same low-level errors or
   coordinate duplicate recovery state. Prefer operation results that tell callers what happened over
   callbacks and extra messages that force them to reconstruct it.
8. **Push configuration and policy decisions upward.** Inject dependencies, durations, clocks and
   limits instead of creating them deep inside operations. Choose branches at the appropriate caller,
   and let an operation own its natural batch or loop, following matklad's
   [Push Ifs Up And Fors Down](https://matklad.github.io/2023/11/15/push-ifs-up-and-fors-down.html).
   Keep component invariants encapsulated while making environmental choices explicit and
   deterministic.
9. **Delete the scaffolding a refactor makes obsolete.** Remove unused fields, error variants,
   wrappers, dependencies and obsolete tests alongside their replacement, preserving relevant
   regression coverage. Change internal callers when that produces a cleaner interface. Preserve
   supported external behavior unless the task explicitly authorizes a compatibility change.

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

After the language-specific commit checks (lint, test), use
[pr-guidelines.md](../handbook/pr-guidelines.md): the review agents that apply to the diff,
tests, the rebase on `develop`, and the content of the PR description.

## CI

Some tests require CI-only environment variables and are skipped locally. Check the test code for environment variable guards if a test behaves differently than expected.
