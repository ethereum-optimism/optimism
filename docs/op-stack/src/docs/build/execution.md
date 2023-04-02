---
title: Execution Hacks
lang: en-US
---


::: warning 🚧 OP Stack Hacks are explicitly things that you can do with the OP Stack that are *not* currently intended for production use

OP Stack Hacks are not for the faint of heart. You will not be able to receive significant developer support for OP Stack Hacks — be prepared to get your hands dirty and to work without support.

:::

## Overview

The Execution Layer is responsible for defining the format of state and the state transition function on L2. It is expected to trigger the state transition function when it receives a payload via the [Engine API](https://github.com/ethereum/execution-apis/tree/main/src/engine). Although the default Execution Layer module is the EVM, you can replace the EVM with any alternative VM as long as it sits behind the Engine API.

## Default

The default Execution Layer module is the Rollup EVM module. The Rollup EVM module utilizes a very lightly modified EVM that adds support for transactions that are triggered by smart contracts on L1 and introduces an L1 data fee to each transaction that accounts for the cost of publishing user transactions to L1. You can find the full set of differences between the standard EVM and the Rollup EVM [on this page](https://op-geth.optimism.io/).

## Security

As with modifications to the Derivation Layer, modifications to the Execution Layer can have unintended consequences. For instance, modifications to the EVM may break existing tooling or may open the door to denial of service attacks. Consider the impact of each modification carefully on a case-by-case basis.

## Modding

### EVM Tweaks

The default Execution Layer module is the EVM. It’s possible to modify the EVM in many different ways like adding new precompiles or inserting predeployed smart contracts into the genesis state. Precompiles can help make common smart contract operations cheaper and can therefore further reduce the cost of execution for your specific use-case. These modifications should be made directly to [the execution client](https://github.com/ethereum-optimism/op-geth). 

It’s also possible to create alternative execution client implementations to improve the security properties of your chain. Note that if you modify the EVM, you must apply the same modifications to every execution client that you would like to support.

### Alternative VMs

The OP Stack allows you to replace the EVM with *any* state transition function, as long as the transition can be triggered via the Engine API. This has, for example, been used to implement an OP Stack chain that runs a GameBoy emulator rather than the EVM.

[Tutorial: Adding a precompile](./tutorials/new-precomp.md).