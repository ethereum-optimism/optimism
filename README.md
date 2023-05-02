
<div align="center">
  <br />
  <br />
  <a href="https://optimism.io"><img alt="Optimism" src="https://raw.githubusercontent.com/ethereum-optimism/brand-kit/main/assets/svg/OPTIMISM-R.svg" width=600></a>
  <br />
  <h3><a href="https://optimism.io">Optimism</a> is Ethereum, scaled.</h3>
  <br />
</div>

## What is Optimism?

[Optimism](https://www.optimism.io/) is a project dedicated to scaling Ethereum's technology and expanding its ability to coordinate people from across the world to build effective decentralized economies and governance systems. The [Optimism Collective](https://app.optimism.io/announcement) builds open-source software for running L2 blockchains and aims to address key governance and economic challenges in the wider cryptocurrency ecosystem. Optimism operates on the principle of **impact=profit**, the idea that individuals who positively impact the Collective should be proportionally rewarded with profit. **Change the incentives and you change the world.**

In this repository, you'll find numerous core components of the OP Stack, the decentralized software stack maintained by the Optimism Collective that powers Optimism and forms the backbone of blockchains like [OP Mainnet](https://explorer.optimism.io/) and [Base](https://www.coinbase.com/blog/introducing-base). Designed to be "aggressively open source," the OP Stack encourages you to explore, modify, extend, and test the code as needed. Although not all elements of the OP Stack are contained here, many of its essential components can be found within this repository. By collaborating on free, open software and shared standards, the Optimism Collective aims to prevent siloed software development and rapidly accelerate the development of the Ethereum ecosystem. Come contribute, build the future, and redefine power, together.

## Documentation

- If you want to build on top of OP Mainnet, refer to the [Optimism Community Hub](https://community.optimism.io)
- If you want to build your own OP Stack based blockchain, refer to the [OP Stack docs](https://stack.optimism.io)
- If you want to contribute to the OP Stack, check out the [Protocol Specs](./specs)

## Community

General discussion happens most frequently on the [Optimism discord](https://discord.gg/optimism).
Governance discussion can also be found on the [Optimism Governance Forum](https://gov.optimism.io/).

## Contributing

Read through [CONTRIBUTING.md](./CONTRIBUTING.md) for a general overview of the contributing process for this repository.
Use the [Developer Quick Start](./CONTRIBUTING.md#development-quick-start) to get your development environment set up to start working on the Optimism Monorepo.
Then check out the list of [Good First Issues](https://github.com/ethereum-optimism/optimism/contribute) to find something fun to work on!

## Security Policy and Vulnerability Reporting

Please refer to the canonical [Security Policy](https://github.com/ethereum-optimism/.github/blob/master/SECURITY.md) document for detailed information about how to report vulnerabilities in this codebase.
Bounty hunters are encouraged to check out [the Optimism Immunefi bug bounty program](https://immunefi.com/bounty/optimism/).
The Optimism Immunefi program offers up to $2,000,042 for in-scope critical vulnerabilities.

## The Bedrock Upgrade

OP Mainnet is currently preparing for [its next major upgrade, Bedrock](https://dev.optimism.io/introducing-optimism-bedrock/).
You can find detailed specifications for the Bedrock upgrade within the [specs folder](./specs) in this repository.

Please note that a significant number of packages and folders within this repository are part of the Bedrock upgrade and are NOT currently running in production.
Refer to the Directory Structure section below to understand which packages are currently running in production and which are intended for use as part of the Bedrock upgrade.

## Directory Structure

<pre>
~~ Production ~~
├── <a href="./packages">packages</a>
│   ├── <a href="./packages/common-ts">common-ts</a>: Common tools for building apps in TypeScript
│   ├── <a href="./packages/contracts">contracts</a>: L1 and L2 smart contracts for Optimism
│   ├── <a href="./packages/contracts-periphery">contracts-periphery</a>: Peripheral contracts for Optimism
│   ├── <a href="./packages/core-utils">core-utils</a>: Low-level utilities that make building Optimism easier
│   ├── <a href="./packages/data-transport-layer">data-transport-layer</a>: Service for indexing Optimism-related L1 data
│   ├── <a href="./packages/chain-mon">chain-mon</a>: Chain monitoring services
│   ├── <a href="./packages/fault-detector">fault-detector</a>: Service for detecting Sequencer faults
│   ├── <a href="./packages/message-relayer">message-relayer</a>: Tool for automatically relaying L1<>L2 messages in development
│   ├── <a href="./packages/replica-healthcheck">replica-healthcheck</a>: Service for monitoring the health of a replica node
│   └── <a href="./packages/sdk">sdk</a>: provides a set of tools for interacting with Optimism
├── <a href="./batch-submitter">batch-submitter</a>: Service for submitting batches of transactions and results to L1
├── <a href="./bss-core">bss-core</a>: Core batch-submitter logic and utilities
├── <a href="./gas-oracle">gas-oracle</a>: Service for updating L1 gas prices on L2
├── <a href="./indexer">indexer</a>: indexes and syncs transactions
├── <a href="./infra/op-replica">infra/op-replica</a>: Deployment examples and resources for running an Optimism replica
├── <a href="./integration-tests">integration-tests</a>: Various integration tests for the Optimism network
├── <a href="./l2geth">l2geth</a>: Optimism client software, a fork of <a href="https://github.com/ethereum/go-ethereum/tree/v1.9.10">geth v1.9.10</a>
├── <a href="./l2geth-exporter">l2geth-exporter</a>: A prometheus exporter to collect/serve metrics from an L2 geth node
├── <a href="./op-exporter">op-exporter</a>: A prometheus exporter to collect/serve metrics from an Optimism node
├── <a href="./proxyd">proxyd</a>: Configurable RPC request router and proxy
├── <a href="./technical-documents">technical-documents</a>: audits and post-mortem documents

~~ BEDROCK upgrade - Not production-ready yet, part of next major upgrade ~~
├── <a href="./packages">packages</a>
│   └── <a href="./packages/contracts-bedrock">contracts-bedrock</a>: Bedrock smart contracts. To be merged with ./packages/contracts.
├── <a href="./op-bindings">op-bindings</a>: Go bindings for Bedrock smart contracts.
├── <a href="./op-batcher">op-batcher</a>: L2-Batch Submitter, submits bundles of batches to L1
├── <a href="./op-e2e">op-e2e</a>: End-to-End testing of all bedrock components in Go
├── <a href="./op-node">op-node</a>: rollup consensus-layer client.
├── <a href="./op-proposer">op-proposer</a>: L2-Output Submitter, submits proposals to L1
├── <a href="./ops-bedrock">ops-bedrock</a>: Bedrock devnet work
└── <a href="./specs">specs</a>: Specs of the rollup starting at the Bedrock upgrade
</pre>

## Branching Model

### Active Branches

| Branch          | Status                                                                           |
| --------------- | -------------------------------------------------------------------------------- |
| [master](https://github.com/ethereum-optimism/optimism/tree/master/)                   | Accepts PRs from `develop` when intending to deploy to production.                  |
| [develop](https://github.com/ethereum-optimism/optimism/tree/develop/)                 | Accepts PRs that are compatible with `master` OR from `release/X.X.X` branches.                    |
| release/X.X.X                                                                          | Accepts PRs for all changes, particularly those not backwards compatible with `develop` and `master`. |

### Overview

This repository generally follows [this Git branching model](https://nvie.com/posts/a-successful-git-branching-model/).
Please read the linked post if you're planning to make frequent PRs into this repository.

### Production branch

The production branch is `master`.
The `master` branch contains the code for latest "stable" releases.
Updates from `master` **always** come from the `develop` branch.

### Development branch

The primary development branch is [`develop`](https://github.com/ethereum-optimism/optimism/tree/develop/).
`develop` contains the most up-to-date software that remains backwards compatible with the latest experimental [network deployments](https://community.optimism.io/docs/useful-tools/networks/).
If you're making a backwards compatible change, please direct your pull request towards `develop`.

**Changes to contracts within `packages/contracts/contracts` are usually NOT considered backwards compatible and SHOULD be made against a release candidate branch**.
Some exceptions to this rule exist for cases in which we absolutely must deploy some new contract after a release candidate branch has already been fully deployed.
If you're changing or adding a contract and you're unsure about which branch to make a PR into, default to using the latest release candidate branch.
See below for info about release candidate branches.

### Release candidate branches

Branches marked `release/X.X.X` are **release candidate branches**.
Changes that are not backwards compatible and all changes to contracts within `packages/contracts/contracts` MUST be directed towards a release candidate branch.
Release candidates are merged into `develop` and then into `master` once they've been fully deployed.
We may sometimes have more than one active `release/X.X.X` branch if we're in the middle of a deployment.
See table in the **Active Branches** section above to find the right branch to target.

## Releases

### Changesets

We use [changesets](https://github.com/changesets/changesets) to mark packages for new releases.
When merging commits to the `develop` branch you MUST include a changeset file if your change would require that a new version of a package be released.

To add a changeset, run the command `yarn changeset` in the root of this monorepo.
You will be presented with a small prompt to select the packages to be released, the scope of the release (major, minor, or patch), and the reason for the release.
Comments within changeset files will be automatically included in the changelog of the package.

### Triggering Releases

Releases can be triggered using the following process:

1. Create a PR that merges the `develop` branch into the `master` branch.
2. Wait for the auto-generated `Version Packages` PR to be opened (may take several minutes).
3. Change the base branch of the auto-generated `Version Packages` PR from `master` to `develop` and merge into `develop`.
4. Create a second PR to merge the `develop` branch into the `master` branch.

After merging the second PR into the `master` branch, packages will be automatically released to their respective locations according to the set of changeset files in the `develop` branch at the start of the process.
Please carry this process out exactly as listed to avoid `develop` and `master` falling out of sync.

**NOTE**: PRs containing changeset files merged into `develop` during the release process can cause issues with changesets that can require manual intervention to fix.
It's strongly recommended to avoid merging PRs into develop during an active release.

## License

Code forked from [`go-ethereum`](https://github.com/ethereum/go-ethereum) under the name [`l2geth`](https://github.com/ethereum-optimism/optimism/tree/master/l2geth) is licensed under the [GNU GPLv3](https://gist.github.com/kn9ts/cbe95340d29fc1aaeaa5dd5c059d2e60) in accordance with the [original license](https://github.com/ethereum/go-ethereum/blob/master/COPYING).

All other files within this repository are licensed under the [MIT License](https://github.com/ethereum-optimism/optimism/blob/master/LICENSE) unless stated otherwise.
