# Overview

This document contains guidelines best practices for PRs that we should enforce as much as possible. The motivations and goals behind these best practices are:

- **Ensure thorough reviews**: By the time the PR is merged, at least 1 other person (because there is always at least 1 reviewer) should understand the PR’s changes just as well as the PR author. This helps reduce improve security by reducing both bugs and single points of failure (i.e. we don’t want only one person to understand certain code).
- **Reduce PR churn**: PRs should be quickly reviewable and mergable without much churn (both in terms of code rewrites and comment cycles). This saves time by reducing the number of rebases due to conflicts. Similarly, too many review cycles is a burden for both PR authors and reviewers, and results in “review fatigue” where reviews become less careful and thorough, increasing the likelihood of bugs.
- **Traceability**: We should be able to look back at issues and PRs to understand why a certain decision was made or why a given approach was taken.

# PR Lifecycle Best Practices

This is organized by current state of PR, so it can be easily be referenced frequently to help internalize the guidelines.

## Before Starting PRs

- **~~Spec Out Major Work in Issues or Design Docs**: Before opening a PR for a major change, there should be an issue that defines what the PR will do. Exactly what to spec out will vary by issue, but this may include things like UX, behaviors, architecture decisions, or interfaces.
- **Keep PRs Focused**: Each PR should be a single, narrow, well-defined concern.

## Opening PRs

- **Review Your Own Code**: Reviewing the diff yourself *in a different context*, can be very useful for discovering issues, typos, and bugs before opening the PR. For example, write code in your IDE, then review it in the GitHub diff view. The perspective change forces you to slow down and helps reveal issues you may have missed.
- **Explain Decisions/Tradeoffs**: Explain rationale for any design/architecture decisions and implementation details in the PR description. If it closes an issue, remember to mention the issue it closes, e.g. `Closes <issueUrl>`. Otherwise, just link to the issue. If there is no issue, whatever details would have been in the issue should be in the PR description.
- **Guide PR reviewers:** Let them know about areas of concern, under-tested areas, or vague requirements that should be ironed out.

## Reviewing PRs

- **Verify Requirements are Met**: If the PR claims to fix or close an issue, check that all the requirements in the issue are actually met. Otherwise the issue may be in a good place to merge, but just shouldn’t close the issue.
- **Focus on Tests**: The tests are the spec and therefore should be the focus of reviews. If tests are thorough and passing, the rest is an implementation detail (to an extent—don’t skip source code reviews) that can be fixed in a future optimization/cleanup PR. Make sure edge case behaviors are defined and handled.
- **Think like an Auditor:** What edge cases were ignored? How can the code break? When might it behave incorrectly and unexpectedly? What code should have been changed that isn’t in the diff? What implicit assumptions are made that might be invalid?
- **Ensure Comment Significance is Clear**: Indicate which comments are nits/optionals that the PR author can resolve, compared to which you want to follow up on.
    - Prefix non-blocking comments with `[nit]` or `[non-blocking]`.
- **Consider Reviewing in Your IDE**: For example, GitHub has [this VSCode extension](https://marketplace.visualstudio.com/items?itemName=GitHub.vscode-pull-request-github) to review PRs. This provides more code context and enables review to benefit from your standard lints and IDE features, whereas GitHub’s diff shows none of that.

## Merging PRs

- **Resolve all Comments**: Comments can be resolved by (1) the PR author for nits/optionals, (2) the author or reviewer after discussions, or (3) extracting the comment into an issue to address in a future PR. For (3), ensure the new issue links to the specific comment thread. *Requiring no unresolved comments can be enabled as a merge requirement in GitHub settings.*
- **Other standard PR requirements**: Approved by the appropriate reviewers, CI passing, etc.

# References

- ScopeLift PR Guidelines (this was an internal doc, so no link)
- [Google Testing Toilet Collection](https://gerlacdt.github.io/posts/google-testing-toilet/) (lots of good articles here, thanks for the reference @Kevin Kz)
    - [Code Health: Too Many Comments on Your Code Reviews?](https://testing.googleblog.com/2017/06/code-health-too-many-comments-on-your.html)
    - [Code Health: Respectful Reviews == Useful Reviews](https://testing.googleblog.com/2019/11/code-health-respectful-reviews-useful.html)