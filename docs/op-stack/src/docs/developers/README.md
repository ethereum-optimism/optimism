---
title: Developer docs
lang: en-US
---

Welcome to the Optimism developer docs!

Whether you're just looking to [deploy a basic contract](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/getting-started) or you're ready to [build a cross-chain app](./bridge/messaging.md), you'll be able to find everything you need to start building on Optimism within this section.

If you're looking for third-party tools that make building on Optimism easier, check out the [Tools for Developers](../useful-tools) section.

## Where should I start?

### Just getting started with Optimism?

If you're brand new to Optimism, we recommend checking out the [guide to deploying a basic contract](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/getting-started).
It'll get you familiar with the core steps required to get a contract deployed to the network.
Luckily, Optimism is [EVM equivalent](https://medium.com/ethereum-optimism/introducing-evm-equivalence-5c2021deb306), so it's 100% the same as deploying a contract to Ethereum.

If you're a bit more familiar with Optimism and Ethereum, you can try walking through one of the various [tutorials](https://github.com/ethereum-optimism/optimism-tutorial) put together by the Optimism community.
They'll help you get a headstart when building your first Optimistic project.

### Ready to deploy a contract?

If you're looking to deploy your contracts to the Optimism mainnet or the Optimism Goerli testnet, take a look at our page on [using your favorite tools](./build/using-tools.md).
It contains sample configuration files for deploying your contracts from common frameworks like Hardhat, Truffle, and Brownie.

You might also want to check out our guides for [running a local development environment](./build/dev-node.md) or [running your own Optimism node](./build/run-a-node.md).
These guides are designed to help you feel totally confident in your Optimism deployment.

### Want to explore the cross-chain frontier?

We've got detailed guides for that.
If you want to bridge a token from Ethereum to Optimism (or vice versa!), you should learn more about our [Standard Token Bridge](./bridge/standard-bridge.md).
The Standard Token Bridge makes the process of moving tokens between chains as easy as possible.

If you're looking for something more advanced, we recommend reading through our page on [sending data between L1 and L2](./bridge/messaging.md).
Contracts on one chain can trigger contract functions on the other chain, it's pretty cool!
We even dogfood the same infrastructure and use it under the hood of the Standard Token Bridge.

## Still don't know where to look?

If you can't find the content you're looking for you've got a few options to get extra help.
We recommend first searching through this documentation (search bar at the top right).
If you've already done this and come up short, you can try [asking us a question in Discord](https://discord-gateway.optimism.io), [checking the Help Center](https://help.optimism.io/hc/en-us), or [making an issue on GitHub](https://github.com/ethereum-optimism/community-hub/issues).
