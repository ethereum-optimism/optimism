---
name: create-pr
description: Open a pull request in the Optimism monorepo after you commit the change locally — run the review agents that apply to the diff, test, rebase, write a description that gives the reason for the change, create the PR, then watch CI. Use when asked to open, create, raise, or submit a PR, or to push a branch for review.
---

# Create a PR

[docs/handbook/pr-guidelines.md](../../../docs/handbook/pr-guidelines.md) is the source of
truth for the steps — review agents, tests, rebase, title, description, and CI. Read it and
follow it in order. This file adds only what is specific to opening the PR under Claude Code,
and applies after you commit the change on a feature branch.

- **Run the review agents as parallel subagents** so their output stays out of your context.
- **Create the PR:**

  ```bash
  git push -u origin <branch>
  gh pr create --base develop --title '<title>' --body-file /tmp/pr-body.md [--draft]
  ```
