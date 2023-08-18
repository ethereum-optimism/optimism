## Workflow for Pull Requests

> **Warning**
>
> Before making any non-trivial change, please first open an issue describing the
> change to solicit feedback and guidance.
> This will increase the likelihood of the PR getting merged.

In general, the smaller the diff the easier it will be for us to review quickly.

If you are writing a new feature, please ensure you add appropriate test cases.

To set up your local development environment, visit the [Development Quick Start][quickstart].

[quickstart]: ./quickstart.md

### Basic Rules

We recommend using the [Conventional Commits][convention] format on commit messages.

Unless your PR is ready for immediate review and merging, please mark it as `draft`
(or simply do not open a PR yet).

**Bonus:** Add comments to the diff under the "Files Changed" tab on the PR page to
clarify any sections where you think we might have questions about the approach taken.

[convention]: https://www.conventionalcommits.org/en/v1.0.0/

### Branching

By default, you should work off the `develop` branch in your forked repository.

To work off a release candidate, they follow the a `release/X.X.X` branch naming
convention. See [details about our branching model][branching-details] for more
detailed branching info.

[branching-details]: https://github.com/ethereum-optimism/optimism/blob/develop/README.md#branching-model-and-releases

### Rebasing

We use the `git rebase` command to keep our commit history tidy.

Rebasing is an easy way to make sure that each PR includes a series of clean commits
with descriptive commit messages. See git's [rebase tutorial][rebase] for a detailed
explanation of `git rebase` and how you should use it to maintain a clean commit history.

[rebase]: https://docs.gitlab.com/ee/topics/git/git_rebase.html

### Response Time

We aim to provide a meaningful response to all PRs and issues from external
contributors. Please be give us time to respond, but feel free to nudge or bump
a response request.

### Changesets

_Note: changesets only apply to javascript packages in the `packages/` directory._

We use [changesets](https://github.com/atlassian/changesets) to manage releases of
various packages. You *must* include a `changeset` file in your PR when making a
change that would require a new package release.

To add a `changeset` file:

1. Navigate to the root of the monorepo.
2. Run `pnpm changeset`. You'll be prompted to select packages to include in the
changeset. Use the arrow keys to move the cursor up and down, hit the `spacebar`
to select a package, and hit `enter` to confirm your selection. Select *all* packages
that require a new release as a result of your PR.
3. Once you hit `enter` you'll be prompted to decide whether your selected packages
need a `major`, `minor`, or `patch` release. We follow the [Semantic Versioning][semver]
scheme. Please avoid using `major` releases for any packages that are still in version
`0.y.z`.
4. Commit your changeset and push it to your PR. The changeset bot will notice your
changeset file and leave a little comment to this effect on GitHub.

[semver]: https://semver.org/
