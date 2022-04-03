<div align="center">
  <a href="https://community.optimism.io"><img alt="Optimism" src="https://user-images.githubusercontent.com/14298799/122151157-0b197500-ce2d-11eb-89d8-6240e3ebe130.png" width=280></a>
  <br />
  <h1> The Optimism Monorepo</h1>
</div>
<p align="center">
  <a href="https://github.com/ethereum-optimism/optimism/actions/workflows/ts-packages.yml?query=branch%3Amaster"><img src="https://github.com/ethereum-optimism/optimism/workflows/typescript%20/%20contracts/badge.svg" /></a>
  <a href="https://github.com/ethereum-optimism/optimism/actions/workflows/integration.yml?query=branch%3Amaster"><img src="https://github.com/ethereum-optimism/optimism/workflows/integration/badge.svg" /></a>
  <a href="https://github.com/ethereum-optimism/optimism/actions/workflows/geth.yml?query=branch%3Amaster"><img src="https://github.com/ethereum-optimism/optimism/workflows/geth%20unit%20tests/badge.svg" /></a>
</p>

## TL;DR

This is where [Optimism](https://optimism.io) gets built.

## Documentation

Extensive documentation is available [here](http://community.optimism.io/).

## Community

Come hang on our very active [discord](https://discord.optimism.io) 🔴✨

## Contributing

Read through [CONTRIBUTING.md](./CONTRIBUTING.md) for a general overview of our contribution process.
Then check out our list of [good first issues](https://github.com/ethereum-optimism/optimism/contribute) to find something fun to work on!

## Directory Structure

<pre>
root
├── <a href="./packages">packages</a>
│   ├── <a href="./packages/contracts">contracts</a>: L1 and L2 smart contracts for Optimism
│   ├── <a href="./packages/core-utils">core-utils</a>: Low-level utilities that make building Optimism easier
│   ├── <a href="./packages/common-ts">common-ts</a>: Common tools for building apps in TypeScript
│   ├── <a href="./packages/data-transport-layer">data-transport-layer</a>: Service for indexing Optimism-related L1 data
│   ├── <a href="./packages/message-relayer">message-relayer</a>: Tool for automatically relaying L1<>L2 messages in development
│   └── <a href="./packages/replica-healthcheck">replica-healthcheck</a>: Service for monitoring the health of a replica node
├── <a href="./go">go</a>
│   ├── <a href="./go/batch-submitter">batch-submitter</a>: Service for submitting batches of transactions and results to L1
│   ├── <a href="./go/bss-core">bss-core</a>: Core batch-submitter logic and utilities
│   ├── <a href="./go/gas-oracle">gas-oracle</a>: Service for updating L1 gas prices on L2
│   └── <a href="./go/proxyd">proxyd</a>: Configurable RPC request router and proxy
├── <a href="./l2geth">l2geth</a>: Optimism client software, a fork of <a href="https://github.com/ethereum/go-ethereum/tree/v1.9.10">geth v1.9.10</a>
├── <a href="./integration-tests">integration-tests</a>: Various integration tests for the Optimism network
└── <a href="./ops">ops</a>: Tools for running Optimism nodes and networks
</pre>

## Branching Model and Releases

<!-- TODO: explain about changesets + how we do npm publishing + docker publishing -->

### Active Branches

| Branch          | Status                                                                           |
| --------------- | -------------------------------------------------------------------------------- |
| [master](https://github.com/ethereum-optimism/optimism/tree/master/)                   | Accepts PRs from `develop` when we intend to deploy to mainnet.                                      |
| [develop](https://github.com/ethereum-optimism/optimism/tree/develop/)                 | Accepts PRs that are compatible with `master` OR from `release/X.X.X` branches.                    |
| release/X.X.X                                                                          | Accepts PRs for all changes, particularly those not backwards compatible with `develop` and `master`. |

### Overview

We generally follow [this Git branching model](https://nvie.com/posts/a-successful-git-branching-model/).
Please read the linked post if you're planning to make frequent PRs into this repository (e.g., people working at/with Optimism).

### The `master` branch

The `master` branch contains the code for our latest "stable" releases.
Updates from `master` always come from the `develop` branch.
We only ever update the `master` branch when we intend to deploy code within the `develop` to the Optimism mainnet.
Our update process takes the form of a PR merging the `develop` branch into the `master` branch.

### The `develop` branch

Our primary development branch is [`develop`](https://github.com/ethereum-optimism/optimism/tree/develop/).
`develop` contains the most up-to-date software that remains backwards compatible with our latest experimental [network deployments](https://community.optimism.io/docs/useful-tools/networks/).
If you're making a backwards compatible change, please direct your pull request towards `develop`.

**Changes to contracts within `packages/contracts/contracts` are usually NOT considered backwards compatible and SHOULD be made against a release candidate branch**.
Some exceptions to this rule exist for cases in which we absolutely must deploy some new contract after a release candidate branch has already been fully deployed.
If you're changing or adding a contract and you're unsure about which branch to make a PR into, default to using the latest release candidate branch.
See below for info about release candidate branches.

### Release new versions

Developers can release new versions of the software by adding changesets to their pull requests using `yarn changeset`. Changesets will persist over time on the `develop` branch without triggering new version bumps to be proposed by the Changesets bot. Once changesets are merged into `master`, the bot will create a new pull request called "Version Packages" which bumps the versions of packages. The correct flow for triggering releases is to update the base branch of these pull requests onto `develop` and merge them, and then create a new pull request to merge `develop` into `master`. Then, the `release` workflow will trigger the actual publishing to `npm` and Docker hub.

Be sure to not merge other pull requests into `develop` if partially through the release process. This can cause problems with Changesets doing releases and will require manual intervention to fix it.

### Release candidate branches

Branches marked `release/X.X.X` are **release candidate branches**.
Changes that are not backwards compatible and all changes to contracts within `packages/contracts/contracts` MUST be directed towards a release candidate branch.
Release candidates are merged into `develop` and then into `master` once they've been fully deployed.
We may sometimes have more than one active `release/X.X.X` branch if we're in the middle of a deployment.
See table in the **Active Branches** section above to find the right branch to target.

### Releasing new versions

Developers can release new versions of the software by adding changesets to their pull requests using `yarn changeset`. Changesets will persist over time on the `develop` branch without triggering new version bumps to be proposed by the Changesets bot. Once changesets are merged into `master`, the bot will create a new pull request called "Version Packages" which bumps the versions of packages. The correct flow for triggering releases is to re-base these pull requests onto `develop` and merge them, and then create a new pull request to merge `develop` onto `master`. Then, the `release` workflow will trigger the actual publishing to `npm` and Docker hub.

## License

Code forked from [`go-ethereum`](https://github.com/ethereum/go-ethereum) under the name [`l2geth`](https://github.com/ethereum-optimism/optimism/tree/master/l2geth) is licensed under the [GNU GPLv3](https://gist.github.com/kn9ts/cbe95340d29fc1aaeaa5dd5c059d2e60) in accordance with the [original license](https://github.com/ethereum/go-ethereum/blob/master/COPYING).

All other files within this repository are licensed under the [MIT License](https://github.com/ethereum-optimism/optimism/blob/master/LICENSE) unless stated otherwise.
