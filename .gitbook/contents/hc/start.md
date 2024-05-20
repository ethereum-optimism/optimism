---
description: HybridCompute Example - Getting started with HybridCompute
---

<figure><img src="../../assets/hc-under-upgrade.png" alt=""><figcaption></figcaption></figure>

# Wizard

Use HybridCompute with ease by navigating through a minimal web-app.

<figure><img src="../../assets/Artboard 4 (10).png" alt=""><figcaption></figcaption></figure>

<figure><img src="../../assets/how does this help me.png" alt=""><figcaption></figcaption></figure>

Using HybridCompute basically requires 3 things:

1. Deploying the HybridComputeHelper contract
2. Fund the HybridComputeHelper with some BOBA tokens
3. Deploy an AWS endpoint which receives the smart contract call and returns the result to your contract in a readable way within the same transaction.

This process can be tedious for new fellow coders, so we decided to build this HybridCompute Starter app, which basically guides you through the process and automatically deploys and funds a new HybridComputeHelper just for you.

<figure><img src="../../assets/project structure.png" alt=""><figcaption></figcaption></figure>

### [contracts](https://github.com/bobanetwork/boba/tree/develop/boba\_community/hc-start/packages/contracts)

This package is used within the react app (referenced via package.json) and contains the deployed contract addresses as well as the ABIs etc.

### [dapp-contracts](https://github.com/bobanetwork/boba/tree/develop/boba\_community/hc-start/packages/dapp-contracts)

In this package you'll find the actual solidity smart contracts which have been deployed for this DApp to work.

### [react-app](https://github.com/bobanetwork/boba/tree/develop/boba\_community/hc-start/packages/react-app)

This package contains the react app the user actually navigates through.

* **Mainnet**: [hcb.boba.network](https://hcb.boba.network/)
