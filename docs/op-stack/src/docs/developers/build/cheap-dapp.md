---
title: Making Optimism Dapps Even Cheaper
lang: en-US
---

The cost of using a decentralized application in Optimism is much lower than the cost of the equivalent application on L1 Ethereum.
[See here](https://l2fees.info/) for the current values.
However, with proper optimization, we can make our decentralized applications even cheaper.
Here are some strategies.


## Background

This is a basic introduction into some of the concepts you need to understand to fully optimise your contracts in the Optimism L2 environment.

### What are the transaction fees?

The cost of an L2 transaction on Optimism is composed of two components:

- L2 execution fee, which is proportional to the gas actually used in processing the transaction.
  Normally the cost of L2 gas is 0.001 gwei, but this may increase when the system is extremely congested. 
  Do not hardcode this value. 
  
- L1 data fee, which is proportional to:
  - The gas cost of writing the transaction's data to L1 (roughly equal to the transaction's length)
  - The cost of gas on L1.
    The cost of gas on L1 can be extremely volatile. 
  
To view the current gas costs as a user, [see here](https://public-grafana.optimism.io/). To retrieve them programatically, [see here](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/sdk-estimate-gas).

For a more in depth look at how transaction fees are calculated see our [fee documentation](transaction-fees.md).

### Optimization tradeoffs

In almost all cases, the L1 data fee is the vast majority of the transaction's cost.
The L2 execution fee is, comparatively speaking, negligible.
This means that the optimization tradeoffs are very different in Optimism than they are in Ethereum.

Transaction call data is *expensive*.
The cost of writing a byte to L1 is approximately 16 gas.
At a cost of 45 gwei per L1 gas unit, writing one byte to L1 on Optimism costs 720 gwei, or 720,000 units of L2 gas (at the non-congested price of 0.001 gwei per L2 gas unit).

In comparison, on-chain processing and storage are cheap.
The worst case for writing to storage (previously uninitialized storage) is a cost of [22100 L2 gas per EVM word, which contains 32 bytes of data](https://www.evm.codes/#55), which averages out to less than 700 L2 gas / byte.
At a cost of 45 gwei per L1 gas unit, this means it is cheaper to write a whole kilobyte to storage, rather than add one byte to the transaction call data. 

## Modify the [ABI (application binary interface)](https://docs.soliditylang.org/en/latest/abi-spec.html)

[The standard ABI](https://docs.soliditylang.org/en/latest/abi-spec.html) was designed with L1 tradeoffs in mind. 
It uses four byte function selectors and pads values to a 32 byte size. 
Neither is optimal when using Optimism.

It is much more efficient to [create a shorter ABI with just the required bytes, and decode it onchain](https://ethereum.org/en/developers/tutorials/short-abi/).
All of your [`view`](https://docs.soliditylang.org/en/latest/contracts.html#view-functions) and [`pure`](https://docs.soliditylang.org/en/latest/contracts.html#pure-functions) functions can use the standard ABI at no cost.


## Use smaller values when possible

Your modified ABI is not going to pad values, so the less bytes you use the better.
For example, it is standard to use `uint256` for amounts.
This means that the highest number we can represent is 2<sup>256</sup>-1, or about 1.2*10<sup>77</sup>. 
When storing ETH balances, for example, using `uint256` is overkill as there are only [120 million ETH](https://ycharts.com/indicators/ethereum_supply). Thus, we can safely store ETH balances in `uint88` which is just eleven bytes.

Go through your contracts and identify any values that will never reach 32 bytes and reduce them to logical sizes. You can do this same process for ints, bytes and [other Solidity data types](https://docs.soliditylang.org/en/develop/types.html#types).

