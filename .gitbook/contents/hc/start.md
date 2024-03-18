---
description: HybridCompute Example - Getting started with HybridCompute
---

# Wizard

Use HybridCompute with ease by navigating through a minimal web-app.

<figure><img src="../../.gitbook/assets/Artboard 4 (10).png" alt=""><figcaption></figcaption></figure>

<figure><img src="../../.gitbook/assets/how does this help me.png" alt=""><figcaption></figcaption></figure>

Using HybridCompute basically requires 3 things:

1. Deploying the HybridComputeHelper contract
2. Fund the HybridComputeHelper with some BOBA tokens
3. Deploy an AWS endpoint which receives the smart contract call and returns the result to your contract in a readable way within the same transaction.

This process can be tedious for new fellow coders, so we decided to build this HybridCompute Starter app, which basically guides you through the process and automatically deploys and funds a new HybridComputeHelper just for you.



<figure><img src="../../.gitbook/assets/project structure.png" alt=""><figcaption></figcaption></figure>

### [contracts](https://github.com/bobanetwork/boba/tree/develop/boba\_community/hc-start/packages/contracts)

This package is used within the react app (referenced via package.json) and contains the deployed contract addresses as well as the ABIs etc.

### [dapp-contracts](https://github.com/bobanetwork/boba/tree/develop/boba\_community/hc-start/packages/dapp-contracts)

In this package you'll find the actual solidity smart contracts which have been deployed for this DApp to work.

### [react-app](https://github.com/bobanetwork/boba/tree/develop/boba\_community/hc-start/packages/react-app)

This package contains the react app the user actually navigates through.

* **Rinkeby**: [hcb.goerli.boba.network](https://hcb.goerli.boba.network/)
* **Mainnet**: [hcb.boba.network](https://hcb.boba.network/)



<figure><img src="../../.gitbook/assets/goerli deployment.png" alt=""><figcaption></figcaption></figure>

Helper contract deployed as `0xbfE85eabF9429A5Bf706a313a44817327062Fe28` Implementation of Helper at `0x3ecc1B250449B9f3f33037BfE0CB29aF7dC49FF8` Factory contract deployed as `0xfB61e269905d67339dD263309c420c19Dc730E6F`
