# Contributor Guide

Welcome! This document outlines how you can help improve the OP Stack. There are lots of ways to contribute. No contribution is too small, and all contributions are valued.

Please note that we’re in the process of expanding community involvement in our core development. This work is ongoing, and we aren’t done yet. It may be difficult to make contributions to some areas of the codebase while we work out the kinks.

All contributions are in scope for RPGF funding.

# Code of Conduct

We expect all contributors to adhere to our [Code of Conduct](https://github.com/ethereum-optimism/.github/blob/master/CODE_OF_CONDUCT.md). Please follow it in all your interactions with the project. If you wish to report a Code of Conduct violation, please reach out to us on Discord.

# Getting in touch

To get in touch with us, you can:

- Join our [Discord](https://discord.optimism.io/)
- Open an issue on [GitHub](https://github.com/ethereum-optimism/optimism/issues/new/choose)
- Start a thread on our [Governance Forum](https://gov.optimism.io/)

# Ways to contribute

## Reporting bugs

Reporting bugs is one of the best ways to get started contributing to the OP Stack. To report a bug, first search for similar issues on our [Issue Tracker](https://github.com/ethereum-optimism/optimism/issues). Then, create a bug report issue and fill out the form. It’s OK if you can’t fill answer every question in the form, however the more information you provide the more likely it is that the core team will be able to reproduce and fix the bug. At minimum please try to provide a minimal, reproducible example as per [this guide](https://stackoverflow.com/help/minimal-reproducible-example).

The core team aims to provide a meaningful response to all bug reports within 3 business days.

### Security bugs

If you have a security bug, please ******do not****** report it as an issue on GitHub. Please refer to our [security policy](https://github.com/ethereum-optimism/.github/blob/master/SECURITY.md) instead.

## Requesting new features

If you’d like to see something added to the OP Stack, first search for similar requests on our [Issue Tracker](https://github.com/ethereum-optimism/optimism/issues). If you find a similar request, leave a comment on the issue voicing your support for it and describing your use case. Otherwise, create a feature request issue and fill out the form. When requesting a new feature, make sure to describe in detail what the feature does and what impact you expect it will have on OP Stack developers. This will help the core team prioritize the request against others.

The core team aims to provide a meaningful response to all feature requests within 5 business days. Please note that the core team may reject a feature request or schedule it for a later date. Features that are related to our current roadmap or Collective Missions are more likely to be accepted.

### Protocol changes

Some feature requests may change the way the protocol works or require coordination between multiple client developers. Changes of this kind require significant consideration and thoughtful design before being made, and are time consuming for the core team to triage. As a result, **the core team will generally reject requests for protocol changes unless they have been discussed in advance**. We expect to relax this constraint as we expand the number of core developers over the coming year.

## Opening a pull request

Opening a pull request (PR) is the best way to contribute code directly to the OP Stack. If you’re contributing a change to our docs, fixing a bug, improving test cases, or making other small changes that don’t affect the protocol or public APIs then you can submit a PR for your change right away. Otherwise, we recommend opening an issue first.

Like the above, we will generally reject PRs for protocol changes unless they have been discussed in advance.

### Pull request guidelines

When opening a PR, we ask that you follow the guidelines below:

- **Keep trunk stable** We follow a [trunk-based development](https://www.atlassian.com/continuous-delivery/continuous-integration/trunk-based-development) model, where everything merged into `develop` must be stable enough for production. Use [feature toggles](https://martinfowler.com/articles/feature-toggles.html) to merge in work-in-progress changes without impacting mainnet users.
- **Keep them small.** Large PRs are difficult to meaningfully review. Instead of making one large PR, split it into several smaller ones. You can even use tools like [Graphite](https://graphite.dev) to stack them on top of each other.
- **Limit them to a single concern.** PRs that combine multiple unrelated changes (or “concerns”) are difficult to review. For example, avoid PRs that fix multiple bugs at once or perform unrelated refactors. Instead, open PRs for these changes in isolation so that they can be reviewed independently.

In addition, prior opening your PR please make sure that you:

- Use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) for your PR title and inside your commit messages.
- Provide a detailed description that links back to any relevant issues.
- Ensure that your changes build, CI passes, and any relevant relevant linters report no issues.
- Ensure that your changes have appropriate test coverage.
- Ensure that you update the specs/meta directory as applicable.

Specific sub-projects in the monorepo may also have their own conventions and coding standards to follow. These conventions can be found in the `[CONTRIBUTING.md](http://CONTRIBUTING.md)` file at the root of each project.

### Abandoned or stale pull requests

Contributors are encouraged to take over pull requests that appear abandoned or stalled. Prior to doing so, please check with the original author to see if they intend to continue working on the abandoned PR. When taking over an abandoned PR, please give the original contributor credit for the work they started either by preserving their name and e-mail address in the commit log, or by using the `Author:` or `Co-authored-by:` metadata tag in the commits.

Maintainers may also make edits directly to contributor PRs, including updating the PR title/description or pushing additional commits.

## Your first pull request

If you haven’t contributed to the OP Stack before, check out our list of [Good First Issues](https://github.com/orgs/ethereum-optimism/projects/45/views/2). These issues are well-scoped bugs and features that we’ve curated to be a good place for new contributors to get started.

Please write a comment on any issues you plan to take on so that others don’t duplicate your work. If someone claims an issue but doesn’t open a PR or respond for over a week, you can take it over yourself. Please still leave a comment letting the community know that you’ll be taking over.

To set up your development environment, check out the Development Quickstart below.

## Your next N pull requests

You can be as involved as you want to with the OP Stack’s development. Once you’ve done a couple of Good First Issues, we recommend checking out our list of [Advanced Issues](https://github.com/orgs/ethereum-optimism/projects/45/views/1) for something more substantial to work on. These are issues that require more knowledge of the OP Stack or special skills in order to complete.

# Development Quickstart

### Dependencies

You'll need the following:

- [Git](https://git-scm.com/downloads)
- [NodeJS](https://nodejs.org/en/download/)
- [Node Version Manager](https://github.com/nvm-sh/nvm)
- [pnpm](https://pnpm.io/installation)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://go.dev/dl/)
- [Foundry](https://getfoundry.sh/)
- [go-ethereum](https://github.com/ethereum/go-ethereum)

### Setup

Clone the repository and open it:

```
git clone git@github.com:ethereum-optimism/optimism.git
cd optimism
```

### Install the Correct Version of NodeJS

Install the correct node version with [nvm](https://github.com/nvm-sh/nvm)

```
nvm use
```

### Install node modules with pnpm

```
pnpm i
```

### Building the TypeScript packages

[foundry](https://github.com/foundry-rs/foundry) is used for some smart contract development in the monorepo. It is required to build the TypeScript packages and compile the smart contracts. Install foundry [here](https://getfoundry.sh/).

To build all of the [TypeScript packages](https://github.com/ethereum-optimism/optimism/blob/develop/packages), run:

```
pnpm clean
pnpm build
```

Packages compiled when on one branch may not be compatible with packages on a different branch. **You should recompile all packages whenever you move from one branch to another.** Use the above commands to recompile the packages.

### Building the rest of the system

If you want to run an Optimism node OR **if you want to run the integration tests**, you'll need to build the rest of the system. Note that these environment variables significantly speed up build time.

```
cd ops-bedrock
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
docker-compose build
```

Source code changes can have an impact on more than one container. **If you're unsure about which containers to rebuild, just rebuild them all**:

```
cd ops-bedrock
docker-compose down
docker-compose build
docker-compose up
```

**If a node process exits with exit code: 137** you may need to increase the default memory limit of docker containers

Finally, **if you're running into weird problems and nothing seems to be working**, run:

```
cd optimism
pnpm clean
pnpm build
cd ops
docker-compose down -v
docker-compose build
docker-compose up
```

### Viewing docker container logs

By default, the `docker-compose up` command will show logs from all services, and that can be hard to filter through. In order to view the logs from a specific service, you can run:

```
docker-compose logs --follow <service name>
```

### Running tests

Before running tests: **follow the above instructions to get everything built.**

### Running unit tests

Run unit tests for all packages in parallel via:

```
pnpm test
```

To run unit tests for a specific package:

```
cd packages/package-to-test
pnpm test
```

### Running contract static analysis

We perform static analysis with `[slither](https://github.com/crytic/slither)`. You must have Python 3.x installed to run `slither`. To run `slither` locally, do:

```bash
cd packages/contracts
pip3 install slither-analyzer
pnpm test:slither
```

# Labels

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

## Filtering for Work

To find tickets available for external contribution, take a look at the https://github.com/ethereum-optimism/optimism/labels/M-community label.

You can filter by the https://github.com/ethereum-optimism/optimism/labels/D-good-first-issue
label to find issues that are intended to be easy to implement or fix.

Also, all labels can be seen by visiting the [labels page][labels]

[labels]: https://github.com/ethereum-optimism/optimism/labels

## Modifying Labels

When altering label names or deleting labels there are a few things you must be aware of.

- This may affect the mergify bot's use of labels. See the [mergify config](.github/mergify.yml).
- If the https://github.com/ethereum-optimism/labels/S-stale label is altered, the [close-stale](.github/workflows/close-stale.yml) workflow should be updated.
- If the https://github.com/ethereum-optimism/labels/M-dependabot label is altered, the [dependabot config](.github/dependabot.yml) file should be adjusted.
- Saved label filters for project boards will not automatically update. These should be updated if label names change.

# Acknowledgements

This guide was adapted from the [Reth contributing guide](https://github.com/paradigmxyz/reth/blob/main/CONTRIBUTING.md) and the React [How to Contribute](https://legacy.reactjs.org/docs/how-to-contribute.html) guide.