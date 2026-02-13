# CI/CD Operations

This document provides guidance for AI agents working with CI/CD operational tasks in the Optimism monorepo.

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
