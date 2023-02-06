---
title: Upgrade Guide
lang: en-US
---

::: warning This guide is for bedrock
This guide is for the *bedrock* upgrade, which is coming in Q1, 2023, subject to approval by Optimism governance.
Do not attempt to use this in production prior to that upgrade. Keep an eye on these docs or [our official Twitter](https://twitter.com/OPLabsPBC) for announcements.
:::

This document provides an overview of how the Bedrock upgrade will be executed and how it will impact node operators and dApp developers.

## Upgrade Overview

Unlike previous upgrades, the Bedrock upgrade will not be a "regenesis" event where historical transaction data is lost and the chain resets at block zero. Instead, the Bedrock upgrade will resemble a hard fork where the new Bedrock chain will be a continuation of the old one. This ensures that the upgrade is as seamless as possible.

The upgrade will proceed as follows on upgrade day:

1. Deposits and withdrawals on the legacy network will be paused.
2. Transactions on the legacy sequencer will no longer be accepted.
3. The smart contracts on L1 will be upgraded and an irregular state transition on L2 will be performed.
4. The Bedrock sequencer will start up.
5. Deposits and withdrawals will be re-enabled.
6. The contract addresses, binaries, and data directories required to interact with the new system will be distributed.

Backwards compatibility is one of the upgrade's key design goals. 
Potential incompatibilities and their workarounds are highlighted in the sections below.

## Optimism Mainnet Upgrade
The Bedrock upgrade to the Optimism Mainnet Network is yet to be scheduled. 
We plan to announce an official date and time in February 2023 at least 3 weeks in advance of the upgrade, subject to approval by Optimism governance.

## For Node Operators

:::tip Prerequisites
This section assumes that you have read and understood our [Node Operator Guide](./node-operator-guide.md). Please read that first if you haven't already.
:::

From a node operator perspective, the old system will be completely _replaced_ on upgrade day. This means that rather than upgrading legacy infrastructure, node operators will be standing up entirely new infrastructure to run the Bedrock network.

On upgrade day, we will provide node operators with the following information:

1. The correct `op-node` and `op-geth` images and binaries to use.
2. A URL to an upgraded data directory containing the genesis state for the new system.
3. A URL to a legacy data directory containing data for Legacy Geth. 
4. A set of bootnodes to use as part of the peer-to-peer network.

We will embed the rollup config into the `op-node` itself. Then, on upgrade day, you will need to:

1. Initialize `op-geth`'s data directory using the upgraded genesis state from the provided URL. See the [Initialization via Data Directory](./node-operator-guide.md#initialization-via-data-directory) section of the Node Operator Guide for more information.
2. Specify the `op-node`'s network via the `--network` flag or `OP_NODE_NETWORK` environment variable. Its value will be `goerli` for the Goerli upgrade, or `mainnet` for the mainnet upgrade.
3. Initialize Legacy Geth's data directory using the legacy genesis state from the provided URL. See the [Initialization via Data Directory](./node-operator-guide.md#initialization-via-data-directory) and [Legacy Geth](./node-operator-guide.md#legacy-geth) sections of the Node Operator Guide for more information.
4. Set the `op-geth` `--rollup.historicalrpc` parameter to point to Legacy Geth's RPC endpoint.
5. Start `op-geth` and `op-node` as usual.

The best way to prepare for the upgrade is to participate in one of our public testnets. Please see the [public testnets](./public-testnets.md) page for how to connect to our testnet.

## For dApp and Wallet Developers

On upgrade day, deposits and withdrawals will be paused, along with ingress on the sequencer. This means that all transactions on Optimism will be halted for the duration of the upgrade.

Once the upgrade is complete, everything should be identical to how it was before the upgrade. All balances, contract addresses, transaction data, block data, and historical execution traces will be preserved. The new network is EVM-equivalent, so all existing Ethereum tooling will continue to work with the new system. Differences are described in [How is Bedrock Different?](./how-is-bedrock-different.md).

## FAQs

### When is the upgrade taking place?

The Goerli upgrade is tentatively scheduled for January 2023. The mainnet upgrade is tentatively scheduled for February 2023. The Goerli upgrade will be live for at least a month before the mainnet upgrade.

### Is this a hard fork, or a new network?

This is a hard fork. The network will retain the same chain ID, transaction history, and state. The first block of the new network will be the last block of the new network + 1.

### How long will the upgrade take?

We expect the upgrade to take less than 4 hours.

### How can I best prepare for the upgrade?

The best way to prepare for the upgrade is to participate in one of our public testnets. Please see the [Beta Testnet](https://www.notion.so/External-Optimism-Bedrock-Beta-Testnet-454a37e469af4658b89a9d766334e331) page for how to connect to our current testnet.

### Why is Legacy Geth necessary?

The upgraded data directory used to initialize `op-geth` contains the current state of the network as well as all historical block, transaction, and receipt data. However, providing historical execution would require bundling the legacy system's EVM implementation with `op-geth`. In an effort to keep our diff between `op-geth` and upstream `go-ethereum` small, we instead route requests for historical execution traces to Legacy Geth which already contains the correct execution engine.

You only need to run Legacy Geth if you need historical execution traces.

### What version of upstream Geth is op-geth based off of?

`op-geth` is currently based off of the `1.10.x` version of `go-ethereum`. 
`op-geth` is periodically updated include the latest upstream changes.

### How can I see the difference between upstream Geth and op-geth?

A single-commit diff is maintained in the `op-geth` repository. See [here](https://github.com/ethereum-optimism/op-geth/compare/master...optimism) for the comparison.

### Will transaction tracing for post-Bedrock data be faster?

Yes. `op-geth` uses the latest transaction tracers from upstream, which have much better performance than tracers legacy `l2geth` uses.  