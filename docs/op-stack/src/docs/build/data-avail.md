---
title: Data Availability Hacks
lang: en-US
---

::: warning 🚧 OP Stack Hacks are explicitly things that you can do with the OP Stack that are *not* currently intended for production use

OP Stack Hacks are not for the faint of heart. You will not be able to receive significant developer support for OP Stack Hacks — be prepared to get your hands dirty and to work without support.

:::


## Overview

The Data Availability Layer is responsible for the *ordering* and *storage* of the raw input data that forms the backbone of an OP Stack based chain (transactions, state roots, calls from other blockchains, etc.). You can conceptually think of this as an array of inputs — the ordering of this array should remain stable and the contents of this array should remain available. Unstable ordering of inputs will lead to reorgs of the OP Stack chain, while unavailable inputs will cause the OP Stack chain to halt entirely.

## Default

The default Data Availability Layer module for an OP Stack chain is the Ethereum DA module. When using the Ethereum DA module, all raw input data is expected to be found on Ethereum. Any data that is accessible on Ethereum can be queried when using this module, including calldata, events, and other block data.

## Security

OP Stack based chains are functions of the raw input data found on the Data Availability Layer module(s) used. If a required piece of data is not available, nodes will not be able to properly sync the chain. This also means that these nodes will not be able to dispute any invalid state proposals made to a Settlement Layer module. An OP Stack based chain cannot be safer than the Data Availability module.

You should be careful to understand the security properties of any Data Availability module(s) that you use. The standard Ethereum DA module generally provides the best security guarantees at the cost of higher transaction fees. Alternative DA modules may be appropriate depending on your particular use-case and risk tolerance.

## Modding

### Alternative EVM DA

A simple modification is to use an EVM-based blockchain other than Ethereum as the Data Availability Layer. Doing so simply requires using an L1 RPC other than Ethereum.

### EVM-Ordered Alternative DA

A more involved modification to the Data Availability Layer is an "EVM-Ordered" Alternative DA module. This involves using an EVM-based chain to maintain the *ordering* of transaction data while using a different data storage system to host the underlying data. Generally, ordering is maintained by publishing hashes of the data to the EVM-based chain while publishing the preimages to those hashes to the alternative data source.

An EVM-Ordered Alternative DA module significantly reduces costs by only publishing hashes and not full input data to the EVM chain. Using an EVM chain for ordering also reduces the number of changes that must be made to the standard Rollup configuration to achieve this result.

An example of an EVM-Ordered Alternative DA module can be found within [this modification to the OP Stack](https://github.com/celestiaorg/optimism/pull/3) that uses the Celestia blockchain as a third-party data availability provider.

### Non-EVM DA

A non-EVM DA module uses a chain not based on the EVM to manage both the ordering and storage of raw input data. Such a modification would require relatively significant modifications to the [derivation portion](https://github.com/ethereum-optimism/optimism/tree/develop/op-node/rollup/derive) of the `op-node`. No such fully-independent DA modules have been developed yet — be the first!

### Multiple DA

It is possible to use multiple Data Availability Layer modules at the same time. For instance, one could source data from two EVM-based chains simultaneously in order to form a bridge between the two chains. When using multiple Data Availability Layer modules, it is imperative to establish a global ordering between the two chains. One option for establishing this ordering is to use the timestamps of blocks from each chain.

Like a non-EVM DA module, a system with multiple Data Availability modules would need to make significant modifications to the [derivation portion](https://github.com/ethereum-optimism/optimism/tree/develop/op-node/rollup/derive) of the `op-node`. No such projects have been constructed yet.