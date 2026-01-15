<div align="center">
  <br />
  <br />
  <a href="https://optimism.io"><img alt="Optimism" src="./logo.svg" width=600></a>
  <br />
  <h3><a href="https://optimism.io">Optimism</a> is Ethereum, scaled.</h3>
  <br />
</div>

**Table of Contents**

<!--TOC-->

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
  - [Development branch](#development-branch)
- [License](#license)

<!--TOC-->

What is Optimism?

Optimism
 is a public-good–driven project focused on scaling Ethereum and expanding its ability to coordinate people globally to build decentralized economies, applications, and governance systems.

At its core, Optimism believes that Ethereum’s future depends not just on better technology, but on better incentives. This philosophy is captured in the principle of:

Impact = Profit
Those who create positive impact for the Collective should be proportionally rewarded.
Change the incentives, and you change the world.

Optimism is stewarded by the Optimism Collective
 — a governance and economic system that funds public goods, supports builders, and maintains open-source infrastructure that benefits the entire Ethereum ecosystem.

The OP Stack

This repository contains the core implementation of the OP Stack, a modular, open-source, and production-ready Layer 2 blockchain stack maintained by the Optimism Collective.

The OP Stack powers:

OP Mainnet

Base

And other Optimism-aligned chains

The stack is designed to be:

Modular – components can be swapped or extended

Open-source by default – permissive licensing and public development

Production-grade – used by real networks securing billions in value

Ethereum-aligned – inherits Ethereum’s security and values

You are encouraged to explore, fork, modify, and build on this codebase.

Who This Repository Is For

This repo is relevant if you are:

Building dApps or infrastructure on OP Mainnet

Launching your own OP Stack–based chain

Operating nodes, sequencers, or validators

Contributing to Ethereum scaling infrastructure

Researching rollups, fault proofs, or modular blockchain design

Documentation

Start here depending on your goals:

Build on OP Mainnet
→ Optimism Documentation

Launch or customize an OP Stack chain
→ OP Stack Guide

Contribute to this repository
→ Read the Development and Release Process
 below

Specifications

Formal technical specifications for the OP Stack are maintained separately in the
OP Stack Specs
 repository.

These specs define:

Protocol behavior

Fault proof systems

Cross-chain messaging

Consensus and derivation rules

If you are implementing or auditing OP Stack components, this repo is essential.

Community & Governance

Optimism is governed and built in public.

General discussion & developer support
→ Optimism Discord

Governance proposals & deliberation
→ Optimism Governance Forum

Participation is open — builders, researchers, and community members are encouraged to contribute.

Contributing

The OP Stack is a collaborative, community-driven project.

By working on shared standards and open infrastructure, the Optimism Collective aims to:

Avoid siloed development

Accelerate Ethereum scaling

Fund and sustain public goods

Getting Started

Read CONTRIBUTING.md
 for contribution guidelines

Follow the Developer Quick Start
 to set up your environment

Look for Good First Issues
 if you’re new

Larger initiatives and long-term projects are also documented in CONTRIBUTING.md.

Security Policy & Vulnerability Reporting

Security is taken seriously across the OP Stack.

Report vulnerabilities according to the official
Security Policy

Bug bounty hunters should see the
Optimism Immunefi Program

with rewards of up to $2,000,042 for critical, in-scope vulnerabilities

Repository Structure

Below is a high-level overview of the major components in this monorepo:

<pre> ├── cannon : Onchain MIPS emulator for fault proofs ├── devnet-sdk : Toolkit for standardized devnet interactions ├── docs : Audits, post-mortems, and technical documents ├── kurtosis-devnet : Kurtosis-based OP Stack devnet ├── op-acceptance-tests : Acceptance tests and configurations ├── op-alt-da : Alternative Data Availability (beta) ├── op-batcher : Submits L2 transaction batches to L1 ├── op-chain-ops : State surgery and chain utilities ├── op-challenger : Dispute game challenge agent ├── op-conductor : High-availability sequencer service ├── op-deployer : CLI for deploying & upgrading contracts ├── op-devstack : Flexible integration testing frontend ├── op-dispute-mon : Dispute monitoring service ├── op-dripper : Controlled token distribution service ├── op-e2e : End-to-end tests (Go) ├── op-faucet : Multi-chain development faucet ├── op-node : Rollup consensus-layer client ├── op-program : Fault proof program ├── op-proposer : Submits L2 outputs to L1 ├── op-service : Shared utilities and libraries ├── op-supervisor : Cross-chain message safety monitor ├── ops : Operational tooling └── packages └── contracts-bedrock : Core OP Stack smart contracts </pre>
Development & Release Process
Overview

If you plan to fork, run nodes, or submit frequent PRs, read this section carefully.

Production Releases

Releases are tag-based, versioned as:

<component-name>/v<semver>


Example:

op-node/v1.1.2

op-contracts/v1.0.0

Release candidates use:

<component-name>/v<semver>-rc.X


Tags like v1.1.4 (without component prefix):

Include all Go-based op-* components

Exclude smart contracts

Required due to Golang versioning rules

Smart Contracts

Not all contracts in a release are production-ready

Refer to GitHub release notes for deployed contracts

Most contracts are under active development

op-geth Versioning

op-geth embeds upstream Geth versions:

vMAJOR.GETH_MAJOR GETH_MINOR GETH_PATCH.PATCH


Example:

Geth v1.12.0

op-geth → v1.101200.0

Released Components

Only the following components have official releases:

op-batcher

op-contracts

op-challenger

op-node

op-proposer

All others should be considered development-only.

Development Branch

Primary branch: develop

Contains the latest backwards-compatible changes

Compatible with current experimental networks

⚠️ Contracts in packages/contracts-bedrock/src are usually NOT backwards compatible.

If unsure:

Use a feature branch

Avoid breaking develop

License

Unless otherwise stated, all files in this repository are licensed under the
MIT License
.
