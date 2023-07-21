
<div align="center">
  <br />
  <br />
  <a href="https://optimism.io"><img alt="Optimism" src="https://raw.githubusercontent.com/ethereum-optimism/brand-kit/main/assets/svg/OPTIMISM-R.svg" width=600></a>
  <h3><a href="https://optimism.io">Optimism</a> is a low-cost and lightning-fast Ethereum L2 blockchain, built with the OP Stack.</h3>
  <br />
  <h3>+</h3>
  <a href="https://celestia.org"><img alt="Celestia" src="docs/op-stack/src/assets/docs/understand/Celestia-logo-color-color.svg" width=600></a>
  <h3><a href="https://celestia.org">Celestia</a> is a modular consensus and data network, built to enable anyone to easily deploy their own blockchain with minimal overhead.</h3>
  <br />
</div>

## Celestia + OP Stack tutorial

If you're looking to run the OP Stack + Celestia setup for this repository, please visit the [Optimism & Celestia guides and tutorials](https://docs.celestia.org/developers/intro-to-op-stack/) to get started.

## What are Optimism and the OP Stack?

Optimism is a low-cost and lightning-fast Ethereum L2 blockchain, **but it's also so much more than that**.

The OP Stack powers Optimism, an Ethereum L2 blockchain, and forms the technical foundation for the [the Optimism Collective](https://app.optimism.io/announcement)—a group committed to the **impact=profit** principle. This principle rewards individuals for their positive contributions to the collective.

Optimism addresses critical coordination failures in the crypto ecosystem, such as funding public goods and infrastructure. The OP Stack focuses on creating a shared, open-source system for developing new L2 blockchains within the proposed Superchain ecosystem, promoting collaboration and preventing redundant efforts.

As Optimism evolves, the OP Stack will adapt, encompassing components ranging from blockchain infrastructure to governance systems. This software suite aims to simplify L2 blockchain creation while supporting the growth and development of the Optimism ecosystem.

## What is Celestia?

Celestia is a modular consensus and data network, built to enable anyone to easily deploy their own blockchain with minimal overhead.

Celestia is a minimal blockchain that only orders and publishes transactions and does not execute them. By decoupling the consensus and application execution layers, Celestia modularizes the blockchain technology stack and unlocks new possibilities for decentralized application builders. Lean more at [Celestia.org](https://celestia.org).

## Maintenance

The maintenance guide for this repository can be found in the Wiki tab of the repository or [here](https://github.com/celestiaorg/optimism/wiki).

## Documentation

If you want to build on top of Celestia, take a look at the documentation at [docs.celestia.org](https://docs.celestia.org).

If you want to learn more about the OP Stack, check out the documentation at [stack.optimism.io](https://stack.optimism.io/).

## Community

### Optimism

General discussion happens most frequently on the [Optimism discord](https://discord.gg/optimism).
Governance discussion can also be found on the [Optimism Governance Forum](https://gov.optimism.io/).

### Celestia

General discussion happens most frequently on the [Celestia discord](https://discord.com/invite/YsnTPcSfWQ).
Other discussions can be found on the [Celestia forum](https://forum.celestia.org).

<!-- ## Contributing

Read through [CONTRIBUTING.md](./CONTRIBUTING.md) for a general overview of our contribution process.
Use the [Developer Quick Start](./CONTRIBUTING.md#development-quick-start) to get your development environment set up to start working on the Optimism Monorepo.
Then check out our list of [good first issues](https://github.com/ethereum-optimism/optimism/contribute) to find something fun to work on! -->

## e2e testing

This repository has updated end-to-end tests in the `op-e2e` package to work with
Celestia as the data availability (DA) layer.

Currently, the tests assume a working [Celestia devnet](https://github.com/rollkit/local-celestia-devnet) running locally:

```bash
docker run --platform linux/amd64 -p 26658:26658 ghcr.io/rollkit/local-celestia-devnet:v0.11.0-rc8
```

The e2e tests can be triggered with:

```bash
cd $HOME/optimism
cd op-e2e
make test
```

## Bridging

If you have the OP Stack + Celestia setup running, you can test out bridging from the L1
to the L2.

To do this, first navigate to the `packages/contracts-bedrock` directory and create a
`.env` file with the following contents:

```bash
L1_PROVIDER_URL=http://localhost:8545
L2_PROVIDER_URL=http://localhost:9545
PRIVATE_KEY=bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7
```

Then, run the following from the same directory:

```bash
npx hardhat deposit --network devnetL1 --l1-provider-url http://localhost:8545 --l2-provider-url http://localhost:9545 --amount-eth <AMOUNT> --to <ADDRESS>
```

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
├── <a href="./l2geth">l2geth</a>: Optimism client software, a fork of <a href="https://github.com/ethereum/go-ethereum/tree/v1.9.10">geth v1.9.10</a>  (deprecated for BEDROCK upgrade)
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
<!--
## Branching Model

### Active Branches

| Branch          | Status                                                                           |
| --------------- | -------------------------------------------------------------------------------- |
| [master](https://github.com/ethereum-optimism/optimism/tree/master/)                   | Accepts PRs from `develop` when we intend to deploy to mainnet.                                      |
| [develop](https://github.com/ethereum-optimism/optimism/tree/develop/)                 | Accepts PRs that are compatible with `master` OR from `release/X.X.X` branches.                    |
| release/X.X.X                                                                          | Accepts PRs for all changes, particularly those not backwards compatible with `develop` and `master`. |

### Overview

We generally follow [this Git branching model](https://nvie.com/posts/a-successful-git-branching-model/).
Please read the linked post if you're planning to make frequent PRs into this repository (e.g., people working at/with Optimism).

### Production branch

Our production branch is `master`.
The `master` branch contains the code for our latest "stable" releases.
Updates from `master` **always** come from the `develop` branch.
We only ever update the `master` branch when we intend to deploy code within the `develop` to the Optimism mainnet.
Our update process takes the form of a PR merging the `develop` branch into the `master` branch.

### Development branch

Our primary development branch is [`develop`](https://github.com/ethereum-optimism/optimism/tree/develop/).
`develop` contains the most up-to-date software that remains backwards compatible with our latest experimental [network deployments](https://community.optimism.io/docs/useful-tools/networks/).
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

All other files within this repository are licensed under the [MIT License](https://github.com/ethereum-optimism/optimism/blob/master/LICENSE) unless stated otherwise. -->
