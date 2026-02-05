# fix-todo

Resolve TODO checker CI failures by reopening GitHub issues that still have active TODOs in the codebase.

## When to Use

Use this skill when the scheduled TODO checker CI job fails. The TODO checker validates that TODO comments in the codebase don't reference closed GitHub issues.

### Trigger Phrases

- "Fix the latest TODO checker failure"
- "Resolve the TODO checker CI failure"
- "Handle the TODO checker issue"
- "Reopen issues from TODO checker"

## What This Skill Does

This skill automates the process of:
1. Finding the latest failed TODO checker job in CircleCI
2. Extracting which issues were closed but still have active TODOs
3. Determining who closed each issue
4. Reading the actual TODO comment from the code
5. Reopening the issue with proper context and attribution

## Workflow Documentation

The complete step-by-step workflow is documented in:
**[docs/ai/ci-ops.md](../../docs/ai/ci-ops.md#todo-checker-failures)**

Follow the "TODO Checker Failures" section which includes:
- Prerequisites (gh CLI authentication)
- Detailed commands for each step
- Error handling for edge cases
- Output format and requirements

## Key Requirements

- Always tag the person who closed the issue (found via GitHub timeline API)
- Include the exact file location where the TODO exists
- Include the actual TODO comment line from the code
- Provide context about what was completed vs what remains
- Link to the CircleCI job for traceability

## Background

The repository runs a scheduled CircleCI workflow (`scheduled-todo-issues`) every 4 hours that validates TODO comments. TODO comments can reference issues in formats like:
- `TODO(#1234)` - references ethereum-optimism/optimism
- `TODO(repo#1234)` - references ethereum-optimism/repo
- `TODO(org/repo#1234)` - full reference

When an issue is closed but TODOs still reference it, the job fails and issues need to be reopened to track the remaining work.
