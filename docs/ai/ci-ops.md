# CI/CD Operations

This document provides guidance for AI agents working with CI/CD operational tasks in the Optimism monorepo.

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
