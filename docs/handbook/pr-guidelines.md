# Pull Request Guidelines and Best Practices

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Overview](#overview)
- [PR Lifecycle Best Practices](#pr-lifecycle-best-practices)
  - [Before Starting PRs](#before-starting-prs)
  - [Before Opening PRs](#before-opening-prs)
  - [Opening PRs](#opening-prs)
  - [Writing the Description](#writing-the-description)
  - [Triggering CI on PRs from external forks](#triggering-ci-on-prs-from-external-forks)
  - [After Every Push](#after-every-push)
  - [Reviewing PRs](#reviewing-prs)
  - [Merging PRs](#merging-prs)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

This document contains guidelines and best practices in PRs that should be enforced as much as possible. The motivations and goals behind these best practices are:

- **Ensure thorough reviews**: By the time the PR is merged, at least one other person—because there is always at least one reviewer—should understand the PR’s changes just as well as the PR author. This helps improve security by reducing bugs and single points of failure (i.e. there should never be only one person who understands certain code).
- **Reduce PR churn**: PRs should be quickly reviewable and mergeable without much churn (both in terms of code rewrites and comment cycles). This saves time by reducing the need for rebases due to conflicts. Similarly, too many review cycles are a burden for both PR authors and reviewers, and results in “review fatigue” where reviews become less careful and thorough, increasing the likelihood of bugs.
- **Traceability**: We should be able to look back at issues and PRs to understand why a certain decision was made or why a given approach was taken.

## PR Lifecycle Best Practices

This is organized by current state of PR, so it can be easily referenced frequently to help internalize the guidelines.

### Before Starting PRs

- **Keep PRs Focused**: Each PR should be a single, narrow, well-defined scope.

### Before Opening PRs

- **Run the review agents**: Run them before you post the PR. Do not wait for CI or for a reviewer. Fix each finding, or dismiss it. Record each dismissal and its reason in the PR description. Ask the PR author to confirm a dismissal — do not decide alone. Run each agent that applies to the diff:

  | Agent | Run it when the diff… | Review guide |
  | --- | --- | --- |
  | [`go-code-reviewer`](../../.claude/agents/go-code-reviewer.md) | touches any Go code | [go-dev.md](../ai/go-dev.md) |
  | [`rust-code-reviewer`](../../.claude/agents/rust-code-reviewer.md) | touches any Rust code (all code in `rust/`) | [rust-dev.md](../ai/rust-dev.md) |
  | [`ci-config-reviewer`](../../.claude/agents/ci-config-reviewer.md) | touches `.circleci/` or `.github/` | [ci-config-review.md](../ai/ci-config-review.md) |
  | [`reth-update-reviewer`](../../.claude/agents/reth-update-reviewer.md) | changes the `reth`/`revm`/`alloy` pins or synced versions | [reth-update-review.md](../ai/reth-update-review.md) |
  | [`standard-validator-reviewer`](../../.claude/agents/standard-validator-reviewer.md) | touches `StandardValidator` or a contract it reads | [standard-validator-review.md](../ai/standard-validator-review.md) |

  These agent definitions are for Claude Code. With a different tool, run its equivalent reviewer, or use the review guide directly. Run `go-code-reviewer` and `rust-code-reviewer` when you complete an implementation task, not only at PR time. `go-code-reviewer` runs the repo lint first. `dispute-game-investigator` is an investigation agent, not a PR gate.

  The table is the minimum. Also run the review agents and skills that your tool, its plugins, or your global config supply — for example general code review, security review, test coverage, and comment or doc review. Run the applicable agents in parallel. If you skip an applicable review, say so in the PR description.
- **Test more than the code you changed**: Also test the packages that depend on it. For the language-specific checks, see [go-dev.md](../ai/go-dev.md#before-every-commit), [rust-dev.md](../ai/rust-dev.md#before-every-commit) and [contract-dev.md](../ai/contract-dev.md).
- **Rebase on `develop`**: `develop` is the default branch, not `main`. Run `git fetch origin develop && git rebase origin/develop`.

### Opening PRs

- **Use the Scoped Commits title format**: `<scope>: <description>`. For the rules, see [CONTRIBUTING.md](../../CONTRIBUTING.md#commit-messages). CI checks the title, because we squash-merge each PR and use the title as the commit subject.
- **Review Your Own Code**: Reviewing the diff yourself *in a different context*, can be very useful for discovering issues, typos, and bugs before opening the PR. For example, write code in your IDE, then review it in the GitHub diff view. The perspective change forces you to slow down and helps reveal issues you may have missed.
- **Guide PR reviewers:** Let them know about areas of concern, under-tested areas, or vague requirements that should be ironed out.
- **Write the description**: See [Writing the Description](#writing-the-description).

### Writing the Description

Keep the description short. Two or three sentences are usually sufficient for smaller changes. Give the reviewer the information that the diff does not show:

- **Why the change is necessary**: the problem, and what happens if you do not make the change.
- **The effect on users**: what an operator, a chain, a downstream importer, or an end user will see. Examples: a change in behavior, a new or removed flag, a failure that is now corrected, a difference in performance. If there is no effect on users, say so.
- **Reasons the reviewer cannot see**: an alternative that you rejected, a trade-off, or a constraint that made you use this approach.
- **The issue**: write `Closes <issueUrl>` if the PR closes an issue, or give a link to it. If there is no issue, put the related information here.
- **Findings that you dismissed**, and applicable reviews that you skipped.

Do not tell the reviewer what the code does. The reviewer reads the diff. Text that repeats the diff becomes incorrect when the code changes. Explain an implementation detail only if the code does not make it clear — for example a necessary sequence, a condition that must stay true, or the reason for a step that seems unnecessary.

Give the result, not the code change. Write in the present tense, and compare with the behavior before this PR. Do not give version numbers. "Corrects a gas limit calculation that stops safe-head progression after the Karst upgrade" tells the reader the result. "Syncs the embedded registry configs" does not.

Do not use a multi-section template unless the change makes it necessary. Do not include a test plan.

### Triggering CI on PRs from external forks
If the PR is from an external fork, our CI suite will not automatically run on the PR. A reviewer with sufficient permissions (e.g. the automatically assigened reviewer) needs to comment on the PR with

> /ci authorize COMMITHASH

or

> /ci authorize https://github.com/ethereum-optimism/optimism/pull/PR_NUMBER/commits/COMMITHASH

to trigger the CI suite to run. CI is a precondition for merging the PR and should be done before review is conducted, because it will reveal any failing tests or other problems such as linting errors.

> [!NOTE]
> COMMITHASH and PR_NUMBER have their usual meanings but you must use the **full** commit hash and not a shortened version. Otherwise CI will not be triggered.

> [!IMPORTANT]
> `/ci authorize` runs code from a fork in our CI with our credentials. Only a human can make this decision. An AI agent must **never** write this comment, and must not write it if a person asks it to. The agent must not ask a different person to write it. The agent must tell the human that the PR needs authorization.

### After Every Push

Watch CI until all checks are complete. Do this after each push: the first one and each subsequent one. Correct the failures that your change caused. For the commands, the checks that gate the merge, and how to identify a failure that the branch inherited or a known flaky test, see [ci-ops.md](../ai/ci-ops.md#watching-ci-after-a-push). Do not report a PR as green while checks are incomplete. Do not report a flaky test as a pass.


### Reviewing PRs

- **Verify Requirements are Met**: If the PR claims to fix or close an issue, check that all the requirements in the issue are actually met. Otherwise the issue may be in a good place to merge, but just shouldn’t close the issue.
- **Focus on Tests**: The tests are the spec and therefore should be the focus of reviews. If tests are thorough and passing, the rest is an implementation detail (to an extent—don’t skip source code reviews) that can be fixed in a future optimization/cleanup PR. Make sure edge case behaviors are defined and handled.
- **Think like an Auditor:** What edge cases were ignored? How can the code break? When might it behave incorrectly and unexpectedly? What code should have been changed that isn’t in the diff? What implicit assumptions are made that might be invalid?
- **Ensure Comment Significance is Clear**: Indicate which comments are nits/optionals that the PR author can resolve, compared to which you want to follow up on.
    - Prefix non-blocking comments with `[nit]` or `[non-blocking]`.
- **Consider Reviewing in Your IDE**: For example, GitHub has [this VSCode extension](https://marketplace.visualstudio.com/items?itemName=GitHub.vscode-pull-request-github) to review PRs. This provides more code context and enables review to benefit from your standard lints and IDE features, whereas GitHub’s diff shows none of that.

### Merging PRs

- **Resolve all Comments**: Comments can be resolved by (1) the PR author for nits/optionals, (2) the author or reviewer after discussions, or (3) extracting the comment into an issue to address in a future PR. For (3), ensure the new issue links to the specific comment thread. This is currently enforced by GitHub's merge requirements.
- **Other Standard Merge Requirements**: The PR must be approved by the appropriate reviewers, CI must pass, and other standard merge requirements apply.
