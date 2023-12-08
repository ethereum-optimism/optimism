# Optimism monorepo contributing guide

🎈 Thanks for your help improving the project! We are so happy to have you!

**No contribution is too small and all contributions are valued.**

There are plenty of ways to contribute, in particular we appreciate support in the following areas:

- Reporting issues. For security issues see [Security policy](https://github.com/ethereum-optimism/.github/blob/master/SECURITY.md).
- Fixing and responding to existing issues. You can start off with those tagged ["good first issue"](https://github.com/ethereum-optimism/optimism/contribute) which are meant as introductory issues for external contributors.
- Improving the [community site](https://community.optimism.io/), [documentation](https://github.com/ethereum-optimism/community-hub) and [tutorials](https://github.com/ethereum-optimism/optimism-tutorial).
- Become an "Optimizer" and answer questions in the [Optimism Discord](https://discord.optimism.io).
- Get involved in the protocol design process by proposing changes or new features or write parts of the spec yourself in the [specs subdirectory](./specs/).

Note that we have a [Code of Conduct](https://github.com/ethereum-optimism/.github/blob/master/CODE_OF_CONDUCT.md), please follow it in all your interactions with the project.

## Workflow for Pull Requests

🚨 Before making any non-trivial change, please first open an issue describing the change to solicit feedback and guidance. This will increase the likelihood of the PR getting merged.

In general, the smaller the diff the easier it will be for us to review quickly.

In order to contribute, fork the appropriate branch, for non-breaking changes to production that is `develop` and for the next release that is normally `release/X.X.X` branch, see [details about our branching model](https://github.com/ethereum-optimism/optimism/blob/develop/README.md#branching-model-and-releases).

Additionally, if you are writing a new feature, please ensure you add appropriate test cases.

Follow the [Development Quick Start](#development-quick-start) to set up your local development environment.

We recommend using the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format on commit messages.

Unless your PR is ready for immediate review and merging, please mark it as 'draft' (or simply do not open a PR yet).

Once ready for review, make sure to include a thorough PR description to help reviewers. You can read more about the guidelines for opening PRs in the [PR Guidelines](docs/handbook/pr-guidelines.md) file.

**Bonus:** Add comments to the diff under the "Files Changed" tab on the PR page to clarify any sections where you think we might have questions about the approach taken.

### Response time:
We aim to provide a meaningful response to all PRs and issues from external contributors within 2 business days.

### Changesets

We use [changesets](https://github.com/atlassian/changesets) to manage releases of our various packages.
You *must* include a `changeset` file in your PR when making a change that would require a new package release.

Adding a `changeset` file is easy:

1. Navigate to the root of the monorepo.
2. Run `pnpm changeset`. You'll be prompted to select packages to include in the changeset. Use the arrow keys to move the cursor up and down, hit the `spacebar` to select a package, and hit `enter` to confirm your selection. Select *all* packages that require a new release as a result of your PR.
3. Once you hit `enter` you'll be prompted to decide whether your selected packages need a `major`, `minor`, or `patch` release. We follow the [Semantic Versioning](https://semver.org/) scheme. Please avoid using `major` releases for any packages that are still in version `0.y.z`.
4. Commit your changeset and push it into your PR. The changeset bot will notice your changeset file and leave a little comment to this effect on GitHub.
5. Voilà, c'est fini!

### Rebasing

We use the `git rebase` command to keep our commit history tidy.
Rebasing is an easy way to make sure that each PR includes a series of clean commits with descriptive commit messages
See [this tutorial](https://docs.gitlab.com/ee/topics/git/git_rebase.html) for a detailed explanation of `git rebase` and how you should use it to maintain a clean commit history.

## Development Quick Start

### Dependencies

You'll need the following:

* [Git](https://git-scm.com/downloads)
* [NodeJS](https://nodejs.org/en/download/)
* [Node Version Manager](https://github.com/nvm-sh/nvm)
* [pnpm](https://pnpm.io/installation)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/install/)
* [Go](https://go.dev/dl/)
* [Foundry](https://getfoundry.sh)
* [go-ethereum](https://github.com/ethereum/go-ethereum)

### Setup

Clone the repository and open it:

```bash
git clone git@github.com:ethereum-optimism/optimism.git
cd optimism
```

### Install the Correct Version of NodeJS

Install the correct node version with [nvm](https://github.com/nvm-sh/nvm)

```bash
nvm use
```

### Install node modules with pnpm

```bash
pnpm i
```

### Building the TypeScript packages

[foundry](https://github.com/foundry-rs/foundry) is used for some smart contract
development in the monorepo. It is required to build the TypeScript packages
and compile the smart contracts. Install foundry [here](https://getfoundry.sh/).

To build all of the [TypeScript packages](./packages), run:

```bash
pnpm clean
pnpm build
```

Packages compiled when on one branch may not be compatible with packages on a different branch.
**You should recompile all packages whenever you move from one branch to another.**
Use the above commands to recompile the packages.

### Building the rest of the system

If you want to run an Optimism node OR **if you want to run the integration tests**, you'll need to build the rest of the system.
Note that these environment variables significantly speed up build time.

```bash
cd ops-bedrock
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
docker compose build
```

Source code changes can have an impact on more than one container.
**If you're unsure about which containers to rebuild, just rebuild them all**:

```bash
cd ops-bedrock
docker compose down
docker compose build
docker compose up
```

**If a node process exits with exit code: 137** you may need to increase the default memory limit of docker containers

Finally, **if you're running into weird problems and nothing seems to be working**, run:

```bash
cd optimism
pnpm clean
pnpm build
cd ops
docker compose down -v
docker compose build
docker compose up
```

#### Viewing docker container logs

By default, the `docker compose up` command will show logs from all services, and that
can be hard to filter through. In order to view the logs from a specific service, you can run:

```bash
docker compose logs --follow <service name>
```

### Running tests

Before running tests: **follow the above instructions to get everything built.**

#### Running unit tests

Run unit tests for all packages in parallel via:

```bash
pnpm test
```

To run unit tests for a specific package:

```bash
cd packages/package-to-test
pnpm test
```

#### Running contract static analysis

We perform static analysis with [`slither`](https://github.com/crytic/slither).
You must have Python 3.x installed to run `slither`.
To run `slither` locally, do:

```bash
cd packages/contracts-bedrock
pip3 install slither-analyzer
pnpm slither
```

## Labels

Labels are divided into categories with their descriptions annotated as `<Category Name>: <description>`.

The following are a comprehensive list of label categories.

- **Area labels** ([`A-`][area]): Denote the general area for the related issue or PR changes.
- **Category labels** ([`C-`][category]): Contextualize the type of issue or change.
- **Meta labels** ([`M-`][meta]): These add context to the issues or prs themselves primarily relating to process.
- **Difficulty labels** ([`D-`][difficulty]): Describe the associated implementation's difficulty level.
- **Status labels** ([`S-`][status]): Specify the status of an issue or pr.

Labels also provide a versatile filter for finding tickets that need help or are open for assignment.
This makes them a great tool for contributors!

[area]: https://github.com/ethereum-optimism/optimism/labels?q=a-
[category]: https://github.com/ethereum-optimism/optimism/labels?q=c-
[meta]: https://github.com/ethereum-optimism/optimism/labels?q=m-
[difficulty]: https://github.com/ethereum-optimism/optimism/labels?q=d-
[status]: https://github.com/ethereum-optimism/optimism/labels?q=s-

#### Filtering for Work

To find tickets available for external contribution, take a look at the https://github.com/ethereum-optimism/optimism/labels/M-community label.

You can filter by the https://github.com/ethereum-optimism/optimism/labels/D-good-first-issue
label to find issues that are intended to be easy to implement or fix.

Also, all labels can be seen by visiting the [labels page][labels]

[labels]: https://github.com/ethereum-optimism/optimism/labels

#### Modifying Labels

When altering label names or deleting labels there are a few things you must be aware of.

- This may affect the mergify bot's use of labels. See the [mergify config](.github/mergify.yml).
- If the https://github.com/ethereum-optimism/labels/S-stale label is altered, the [close-stale](.github/workflows/close-stale.yml) workflow should be updated.
- If the https://github.com/ethereum-optimism/labels/M-dependabot label is altered, the [dependabot config](.github/dependabot.yml) file should be adjusted.
- Saved label filters for project boards will not automatically update. These should be updated if label names change.
