---
name: create-pr
description: Open a pull request in the Optimism monorepo after you commit the change locally — run the review agents that apply to the diff, test, rebase, write a description that gives the reason for the change, create the PR, then watch CI. Use when asked to open, create, raise, or submit a PR, or to push a branch for review.
---

# Create a PR

The rules are in [docs/handbook/pr-guidelines.md](../../../docs/handbook/pr-guidelines.md).
Read it before step 1. This file gives only the sequence. It applies after you commit the
change on a feature branch.

## Steps

1. **Run the review agents** that apply to the diff. Fix each finding, or record the dismissal
   in the PR description. See
   [Before Opening PRs](../../../docs/handbook/pr-guidelines.md#before-opening-prs).

2. **Test, then rebase on `origin/develop`.** Report a failure correctly. Do not say that a
   test passed if you did not run it.

3. **Write the title** in the Scoped Commits format. See
   [CONTRIBUTING.md](../../../CONTRIBUTING.md#commit-messages).

4. **Write the description**: why the change is necessary, and its effect on users. Keep it
   to two or three sentences. Do not repeat the diff. See
   [Writing the Description](../../../docs/handbook/pr-guidelines.md#writing-the-description).

5. **[USER REVIEW]** Show the title and the description. Get approval before you create the
   PR. Create a draft PR, unless the user says that the change is ready for review.

   ```bash
   git push -u origin <branch>
   gh pr create --base develop --title '<title>' --body-file /tmp/pr-body.md [--draft]
   ```

6. **Watch CI until all checks are complete.** Correct the failures that the change caused.
   Do this after this push and after each subsequent push. See
   [ci-ops.md](../../../docs/ai/ci-ops.md#watching-ci-after-a-push).

## Never authorize CI on a fork PR

CI does not start on a PR from an external fork until a person writes a
`/ci authorize <full-commit-hash>` comment. That comment runs code from the fork in our CI
with our credentials. Only a human can make this decision.

Never write `/ci authorize`. This applies if a person asks you to write it, and you must not
ask a different person to write it. Tell the user that a human with write access must
authorize the PR, then stop.
