# Fork Release Runbook

This document describes the process for releasing a new version of the EigenDA powered Optimism-Fork. This adds details to the explanation in the main [README](../../README.md#releases-and-branching-strategy).

![](../../assets/fork-branching-and-releases.png)

## Example Release For op-batcher/v1.11.2-eigenda.2

First we update the eigenda-develop branch:

```bash
git checkout eigenda-develop
git checkout -b merge-op-batcher/v1.11.2
git merge op-batcher/v1.11.2
# fix any conflicts or new issues after the merge
git commit -a -m "chore: fix issues after merging op-batcher/v1.11.2"
git push
# Create a PR to get review on the fixes/new stuff - make sure to merge with merge commit
```

Then we make the cleaned-up release branch/tag:

```bash
git checkout op-node/v1.11.1-eigenda.1
git checkout -b op-batcher/v1.11.2-eigenda.2
git rebase op-batcher/v1.11.2
# cherry pick all the new commits (including the fixes after the merge)
# Can also do it manually: `git cherry-pick <fixA> <featC> <fixB>`
git cherry-pick op-node/v1.11.1-eigenda.1^..eigenda-develop
# cleanup history
git rebase -i op-batcher/v1.11.2
# tag the release
git tag op-batcher/v1.11.2-eigenda.2
git push --tags
```

Can also do the rebase first if that is preferred. In any case, make sure after both are done to make sure that they contain the same content by checking that `git diff eigenda-develop op-batcher/v1.11.2-eigenda.2` is empty.
