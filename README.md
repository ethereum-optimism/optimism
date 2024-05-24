```markdown
<div align="center">
  <br />
  <br />
  <a href="https://optimism.io"><img alt="Optimism" src="https://raw.githubusercontent.com/ethereum-optimism/brand-kit/main/assets/svg/OPTIMISM-R.svg" width="600"></a>
  <br />
  <h3><a href="https://optimism.io">Optimism</a> is Ethereum, scaled.</h3>
  <br />
</div>

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [What is Optimism?](#what-is-optimism)
- [Documentation](#documentation)
- [Specification](#specification)
- [Community](#community)
- [Contributing](#contributing)
- [Security Policy and Vulnerability Reporting](#security-policy-and-vulnerability-reporting)
- [Directory Structure](#directory-structure)
- [Development and Release Process](#development-and-release-process)
  - [Overview](#overview)
  - [Production Releases](#production-releases)
  - [Development Branch](#development-branch)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## What is Optimism?

[Optimism](https://www.optimism.io/) is a project dedicated to scaling Ethereum's technology and enhancing its capacity to coordinate global communities in building decentralized economies and governance systems. The [Optimism Collective](https://app.optimism.io/announcement) develops open-source software for running L2 blockchains and aims to solve key governance and economic challenges in the cryptocurrency ecosystem. Operating on the principle of **impact=profit**, Optimism rewards those who positively impact the Collective. **Change the incentives and you change the world.**

This repository includes many core components of the OP Stack, the decentralized software stack maintained by the Optimism Collective. The OP Stack powers Optimism and other blockchains like [OP Mainnet](https://explorer.optimism.io/) and [Base](https://base.org). Designed to be "aggressively open source," the OP Stack encourages exploration, modification, extension, and testing. By collaborating on open software and shared standards, the Optimism Collective aims to prevent siloed software development and accelerate the Ethereum ecosystem's growth. Join us to contribute, build the future, and redefine power together.

## Documentation

- To build on top of OP Mainnet, refer to the [Optimism Documentation](https://docs.optimism.io).
- To build your own OP Stack-based blockchain, refer to the [OP Stack Guide](https://docs.optimism.io/stack/getting-started), and understand this repository's [Development and Release Process](#development-and-release-process).

## Specification

For technical details on how Optimism works, refer to the [Optimism Protocol Specification](https://github.com/ethereum-optimism/specs).

## Community

General discussions take place on the [Optimism Discord](https://discord.gg/optimism).
Governance discussions are held on the [Optimism Governance Forum](https://gov.optimism.io/).

## Contributing

For a general overview of the contributing process, read [CONTRIBUTING.md](./CONTRIBUTING.md).
Use the [Developer Quick Start](./CONTRIBUTING.md#development-quick-start) to set up your development environment and start working on the Optimism Monorepo.
Check out the [Good First Issues](https://github.com/ethereum-optimism/optimism/issues?q=is:open+is:issue+label:D-good-first-issue) for initial tasks.
While typo fixes are welcome, please consolidate them into a single commit and batch as many fixes as possible in one PR. Spammy PRs will be closed.

## Security Policy and Vulnerability Reporting

Refer to the [Security Policy](https://github.com/ethereum-optimism/.github/blob/master/SECURITY.md) for detailed information on reporting vulnerabilities.
Bounty hunters can check the [Optimism Immunefi bug bounty program](https://immunefi.com/bounty/optimism/), which offers up to $2,000,042 for critical vulnerabilities.

## Directory Structure

<pre>
├── <a href="./docs">docs</a>: A collection of documents, including audits and post-mortems
├── <a href="./op-batcher">op-batcher</a>: L2-Batch Submitter, submits bundles of batches to L1
├── <a href="./op-bootnode">op-bootnode</a>: Standalone op-node discovery bootnode
├── <a href="./op-chain-ops">op-chain-ops</a>: State surgery utilities
├── <a href="./op-challenger">op-challenger</a>: Dispute game challenge agent
├── <a href="./op-e2e">op-e2e</a>: End-to-End testing of all bedrock components in Go
├── <a href="./op-heartbeat">op-heartbeat</a>: Heartbeat monitor service
├── <a href="./op-node">op-node</a>: Rollup consensus-layer client
├── <a href="./op-preimage">op-preimage</a>: Go bindings for Preimage Oracle
├── <a href="./op-program">op-program</a>: Fault proof program
├── <a href="./op-proposer">op-proposer</a>: L2-Output Submitter, submits proposals to L1
├── <a href="./op-service">op-service</a>: Common codebase utilities
├── <a href="./op-ufm">op-ufm</a>: Simulations for monitoring end-to-end transaction latency
├── <a href="./op-wheel">op-wheel</a>: Database utilities
├── <a href="./ops">ops</a>: Various operational packages
├── <a href="./ops-bedrock">ops-bedrock</a>: Bedrock devnet work
├── <a href="./packages">packages</a>
│   ├── <a href="./packages/chain-mon">chain-mon</a>: Chain monitoring services
│   ├── <a href="./packages/contracts-bedrock">contracts-bedrock</a>: Bedrock smart contracts
│   ├── <a href="./packages/sdk">sdk</a>: Tools for interacting with Optimism
├── <a href="./proxyd">proxyd</a>: Configurable RPC request router and proxy
├── <a href="./specs">specs</a>: Specifications of the rollup starting at the Bedrock upgrade
└── <a href="./ufm-test-services">ufm-test-services</a>: Runs a set of tasks to generate metrics
</pre>

## Development and Release Process

### Overview

Read this section if you're planning to fork this repository or make frequent PRs.

### Production Releases

Production releases are tagged and versioned as `<component-name>/v<semver>`. For example, an `op-node` release might be `op-node/v1.1.2`, and smart contract releases might be `op-contracts/v1.0.0`. Release candidates are versioned as `op-node/v1.1.2-rc.1`.

For contract releases, refer to the GitHub release notes for specific contracts being released. Tags like `v<semver>` (e.g., `v1.1.4`) indicate releases of all Go code only, excluding smart contracts.

`op-geth` embeds upstream geth’s version in its own version: `vMAJOR.GETH_MAJOR GETH_MINOR GETH_PATCH.PATCH`. For example, if geth is at `v1.12.0`, the corresponding op-geth version would be `v1.101200.0`.

See the [Node Software Releases](https://docs.optimism.io/builders/node-operators/releases) page for more details on the latest node component releases. The full set of components with releases includes:

- `chain-mon`
- `ci-builder`
- `indexer`
- `op-batcher`
- `op-contracts`
- `op-challenger`
- `op-heartbeat`
- `op-node`
- `op-proposer`
- `op-ufm`
- `proxyd`
- `ufm-metamask`

All other components and packages are considered development-only and do not have releases.

### Development Branch

The primary development branch is [`develop`](https://github.com/ethereum-optimism/optimism/tree/develop/). It contains the most up-to-date software that remains backward compatible with the latest experimental [network deployments](https://community.optimism.io/docs/useful-tools/networks/). For backward compatible changes, direct your pull requests to `develop`.

**Changes to contracts within `packages/contracts-bedrock/src` are usually not backward compatible.** If unsure, use a feature branch for changes or additions to contracts.

## License

All files within this repository are licensed under the [MIT License](https://github.com/ethereum-optimism/optimism/blob/master/LICENSE) unless stated otherwise.
```
