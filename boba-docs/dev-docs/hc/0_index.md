# Introduction

This guide will walk you through the necessary steps to get started and provide you with the information you need to successfully integrate our system into your project. Additionally, you can view a [full example](https://github.com/bobanetwork/aa-hc-example) on our Github.

## Hybrid Compute

Hybrid Compute, built on top of [Account Abstraction](../account-abstraction/index), allows smart contracts to interact with external data and services. Typically, smart contracts on blockchains like Ethereum are limited to the data available on the blockchain, unable to access outside information directly. Hybrid Compute changes this by enabling smart contracts to make API calls to external services. This interaction allows smart contracts to both access data and perform complex computations off-chain. 

The results of these computations can then be used on-chain, enhancing the functionality and efficiency of smart contracts. By doing so, Hybrid Compute reduces gas costs associated with complex computations and broadens the scope of what decentralized applications (dApps) can achieve. In essence, it bridges the gap between the blockchain and the real world, allowing for more sophisticated and dynamic applications.

Hybrid Compute's workflow generally looks something like this:

```mermaid
graph LR
A[Bundler] -- method --> B[Off-Chain Handler] -- update --> D[Smart Contract]
```

More specifically, here are representations of real operations Hybrid Compute can run:

![HC Technical Details 1](../../assets/Estimate-Gas.png)
![HC Technical Details 2](../../assets/Submit-Operation.png)

## Prerequisites

Before you begin, make sure you have the following prerequisites in place:

- Python
- Docker

Once your environment is ready, proceed through the following steps to implement your Hybrid Compute examples.
