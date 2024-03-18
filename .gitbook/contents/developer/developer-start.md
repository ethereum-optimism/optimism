---
description: Introduction to Boba network for Developers
---

# Boba is built for developers

<figure><img src="../../.gitbook/assets/basics.png" alt=""><figcaption></figcaption></figure>

Welcome to Boba. Boba is a compute-focused L2. We believe that L2s can play a unique role in augmenting the base _compute_ capabilities of the Ethereum ecosystem. You can learn more about Turing hybrid compute [here](broken-reference/). Boba is built on the Optimistic Rollup developed by [Optimism](https://optimism.io). Aside from its main focus, augmenting compute, Boba differs from Optimism by:

* providing additional cross-chain messaging such as a `message-relayer-fast`
* using different gas pricing logic
* providing a swap-based system for rapid L2->L1 exits (without the 7 day delay)
* providing a community fraud-detector that allows transactions to be independently verified by anyone
* interacting with L2 ETH using the normal ETH methods (`msg.value`, `send eth_sendTransaction`, and `provider.getBalance(address)`) rather than as WETH
* being organized as a DAO
* native NFT bridging
* automatically relaying classical 7-day exit messages to L1 for you, rather than this being a separate step

<figure><img src="../../.gitbook/assets/deploying standard contracts.png" alt=""><figcaption></figcaption></figure>

For most contracts, the deploy experience is exactly like deploying on Ethereum. You will need to have some ETH (or Goerli ETH) on Boba and you will have to change your RPC endpoint to either `https://mainnet.boba.network` or `https://goerli.boba.network`. That's it!

The [Mainnet blockexplorer](https://bobascan.com) and the [Goerli blockexplorer](https://testnet.bobascan.com) are similar to Etherscan. The [Gateway](https://gateway.boba.network) allows you to see your balances and bridge funds, among many other functions.

<figure><img src="../../.gitbook/assets/example contracts ready to deploy.png" alt=""><figcaption></figcaption></figure>

1. [Turing Monsters](../../boba\_community/turing-monsters/) _NFTs with on-chain svg and using the Turing random number generator_
2. [Truffle ERC20](../../boba\_examples/truffle-erc20/) _A basic ERC20 deployment using Truffle_
3. [Bitcoin Price Feeds](../../packages/boba/turing/test/005\_lending.ts) _A smart contract that pulls price data from a commercial off-chain endpoint_
4. [Stableswap using off-chain compute](../../packages/boba/turing/test/003\_stable\_swap.ts) _A smart contract using an off-chain compute endpoint to solve the stableswap quadratic using floating point math_

<figure><img src="../../.gitbook/assets/feature using hybridcompute.png" alt=""><figcaption></figcaption></figure>

Turing is a system for interacting with the outside world from within solidity smart contracts. All data returned from external APIs, such as random numbers and real-time financial data, are deposited into a public data-storage contract on Ethereum Mainnet. This extra data allows replicas, verifiers, and fraud-detectors to reproduce and validate the Boba L2 blockchain, block by block.

[Turing Getting Started - NFTs](broken-reference/)

[Turing Getting Started - External API](broken-reference/)

<figure><img src="../../.gitbook/assets/feature obtaining on-chain price data.png" alt=""><figcaption></figcaption></figure>

Price Feed oracles are an essential part of Boba, which allow smart contracts to work with external data and open the path to many more use cases. Currently Boba has several options to get real world price data directly into your contracts - each different in the way they operate to procure data for smart contracts to consume:

1. [Boba Straw](../../for-developers/features/price-feeds.md#1.-Boba-Straw)
2. [Witnet](https://docs.witnet.io/smart-contracts/supported-chains)
3. [Turing](broken-reference/)

[Full Price Feed documentation](../../for-developers/features/price-feeds.md)

<figure><img src="../../.gitbook/assets/bridging NFTs from l2 to l1.png" alt=""><figcaption></figcaption></figure>

NFTs can be minted on Boba and can also be exported to Ethereum, if desired. The minting process is identical to Ethereum. The Boba-specific interchain NFT bridging system and contracts are [documented here](../../boba\_examples/nft\_bridging/).

<figure><img src="../../.gitbook/assets/running a boba RPC node.png" alt=""><figcaption></figcaption></figure>

The [boba-node repo](../../boba\_community/boba-node/) runs a local replica of the Boba L2geth, which is useful for generating analytics for blockexplorers. A Boba node can also relay transactions to the sequencer.

<figure><img src="../../.gitbook/assets/running a community verifier and fraud detector.png" alt=""><figcaption></figcaption></figure>

The [fraud-detector repo](../../boba\_community/fraud-detector/) runs a `Verfier` geth and a _fraud-detector_ service on your computer. In `Verifier` mode, the geth will sync from L1 and use the transaction data from the L1 contracts to compute what the state roots should be, _if the operator is honest_. A separate service, the _fraud-detector_, can then be used to discover potential fraud. Fraud detection consists of requesting a state root from Boba and requesting a state root from your Verifier. If those state roots match, then the operator has been honest. If they do not match, then, that _might_ be due to fraud, or, could also indicate indexing errors, timestamp errors, or chain configuration errors. The central idea is that if two (or more) geths injects the same transactions, then they should write the same blocks with the same state roots. If they don't, then there is a problem somewhere. Fundamentally, the security of rollups has little to do with math or cryptography - rather, security arises from the operator publicly depositing transactions and their corresponding state roots, and then having many independent nodes check those data for possible discrepancies.

<figure><img src="../../.gitbook/assets/helping to develop boba on-chain price data.png" alt=""><figcaption></figcaption></figure>

If you would like to help develop Boba, it is straightforward to run the entire system locally, with [just a few commands](local-stack.md). Note: this is only relevant to developers who wish to develop Boba core services. For most test uses, it's simpler to use the [live testnet](https://goerli.boba.network).
