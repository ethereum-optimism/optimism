# CI/CD Operations

This document provides guidance for AI agents working with CI/CD operational tasks in the Optimism monorepo.

For Docker image build failures — especially flaky `apt`/`apk`/`curl` downloads from package registries and CDNs — see [docker.md](docker.md).

## Diagnosing a CI failure on a feature branch

Before assuming a red check is caused by your change, rule out a failure the branch
**inherited** — especially a long-lived branch that has drifted from `develop`:

1. **Check the diff scope:** `git diff origin/develop...HEAD --name-only` (three dots).
   If the failing test exercises code your branch never touches, it almost certainly
   isn't your regression.
2. **Check `develop`:** the failure may be a known flake, or a real bug a *later*
   `develop` commit already fixed — your branch just predates the fix. Look for an open
   flake issue or a recent fix PR touching the failing area.
3. **Rebase onto latest `develop`** before deeper debugging — stale branches miss
   upstream fixes, and a rebase often clears failures that were never about your change.

Worked example (#21356): `go-tests-short` failed on `op-deployer` integration tests on a
branch that only added a new `op-core/types` package. The diff touched nothing in
`op-deployer`; the failures were a data-dependent flake already fixed on `develop` by
#21396. Rebasing made CI green with no change to the branch's own work.

## Missing `op-core/superchain` bundle (`superchain-configs.zip`)

A CI-only compile error like:

```
op-core/superchain/chain.go:NN: pattern superchain-configs.zip: no matching files found
```

means the job builds Go that transitively links `op-core/superchain` without the bundle
it `//go:embed`s. The zip is **gitignored** and built only by the `prep-superchain` job
(or `just build-superchain-go` locally), so **every** job that compiles an
`op-core/superchain` linker must provision it. The linker set is large and grows silently
— op-node and the other binaries, but also `packages/contracts-bedrock/scripts/go-ffi`,
`op-e2e`, `op-acceptance-tests`, `op-deployer`, and the `kona`/`op-reth` Go tests — so a
**Go-only** change can red-wash unrelated-looking jobs (contracts-bedrock, fuzz-golang,
rust-e2e) all at compile time. It passes locally and in review because the developer
already has the zip on disk (see [go-dev.md](go-dev.md)).

Provision the bundle in the failing job:

- **`main.yml` jobs** (the `prep-superchain` job lives in that workflow): add
  `prep-superchain` to the job's `requires` and attach its workspace
  (`uses_artifacts: true`) — the same pattern `go-tests` uses.
- **Other workflows** (`rust-ci.yml`, `rust-e2e.yml` — no `prep-superchain` job there):
  add an in-job `just build-superchain-go` step (verify mode: regenerates from the
  superchain-registry submodule and asserts the committed `.sha256`).
- **`just` recipes** that compile such Go (e.g. the contracts `build-go-ffi`): have the
  recipe run `just build-superchain-go` first.

When adding a new `op-core/superchain` consumer, audit every CI job that compiles it.

## TODO Checker Failures

The repo runs a scheduled CircleCI job every 4 hours that validates TODO comments don't reference closed GitHub issues. When this job fails, issues need to be reopened.

### Quick Instructions

1. Find the failed TODO checker job in CircleCI (scheduled workflow named `scheduled-todo-issues`)
2. Identify which issues were closed but still have active TODOs in the codebase
3. For each issue:
   - Determine who closed it (using GitHub timeline API)
   - Read the actual TODO comment from the code
   - Reopen with proper attribution and context
   - Include file location and CircleCI job link

### Detailed Workflow

For complete step-by-step instructions with all commands and error handling, see:
**[.claude/skills/fix-todo/SKILL.md](../../.claude/skills/fix-todo/SKILL.md)**

The skill includes:
- Detailed commands for querying CircleCI and GitHub APIs
- How to find who closed an issue
- Comment template for reopening
- Error handling for edge cases
- Output format and requirements
