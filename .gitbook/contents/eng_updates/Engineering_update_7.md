- [Engineering update #7](#engineering-update--7)
  * [1. Engineering Priorities for 2022](#1-engineering-priorities-for-2022)
  * [2. Turing - released to Rinkeby today](#2-turing---released-to-rinkeby-today)
  * [3. Boba Straw - On-chain Price Oracles](#3-boba-straw---on-chain-price-oracles)
  * [4. Additional Data Sources for Boba](#4-additional-data-sources-for-boba)
  * [5. Minimally, 50% cost reduction of multi-token bridging](#5-minimally--50--cost-reduction-of-multi-token-bridging)
  * [6. Updated Blockexplorer](#6-updated-blockexplorer)
  * [7. Gnosis Safe is live on Boba Mainnet](#7-gnosis-safe-is-live-on-boba-mainnet)
  * [8. NFT Bridging is live on Mainnet](#8-nft-bridging-is-live-on-mainnet)

# Engineering update #7

January 24 2024

Greetings from your engineering team. Boba Mainnet has been running stably since early November 2021. Our main focus in the last 45 days has been to (**1**) add critical missing functionality on the ecosystem level (oracles, gnosis, better blockexplorer, ...) and (**2**) to address major friction points on the user-experience side (e.g. cost/speed of bridging). We are happy to be able to report more than 10 significant new features/advances that have either been deployed over the weekend, or are being deployed this week. Before going into more detail, let's first touch on the main engineering priorities for this year, 2022.

## 1. Engineering Priorities for 2022

* *fully decentralizing the sequencer*
* *strategy for keeping L2Geth in sync with Mainnet Geth*

Our main focus for 2022 will be *fully decentralizing the sequencer*. As is clear to everyone in the L2 space, the notion of a single sequencer is inherently problematic and is only a temporary stop-gap while longer term solutions are being built and tested. A secondary focus of 2022 is to deal with *L2Geth updating*. Several Optimistic Rollups are built on [Geth Rojo Loco v1.9.10](https://github.com/ethereum/go-ethereum/releases/tag/v1.9.10) which was released on Jan 20, 2020. A diff of Geth 1.9.10 with a current Geth reveals changes in >1000 files and with every passing day the code gap increases. Already, callers need to enable 'legacy support' to interact with rollups built on Rojo Loco.

This is clearly not a sustainable approach to maintaining secure and powerful rollups - approaches are needed where the rollup has minimal, or better-yet zero, footprint within Geth, so that it can be kept in better sync with the main Ethereum EIPs and implementations. Interestingly, sequencer decentralization and L2Geth upgrades are closely related technically. For example, a reduced-Geth footprint rollup could also make it easier to more people to participate in the system as sequencers.

As a first step in these directions, please see our `next-generation` develop branch which will the core of Boba3, which features major changes to core services, messaging, security, and foot-print reduction. You can follow along with our Boba3 work at [ng-prototype](https://github.com/bobanetwork/boba/tree/mm/ng-prototype). Boba3 will remain L1 contract-compatible with Optimism but represents a transition to ground-up Boba-specific services and architecture.

## 2. Turing - released to Rinkeby today

Turing is a system for interacting with the outside world from within solidity smart contracts. Turing is *atomic*, meaning that everything is completed within a single transaction. Moreover, the data returned from external APIs, such as random numbers and real-time financial data, are deposited into a public data-storage contract on Ethereum Mainnet. This extra data allows replicas, verifiers, and fraud-detectors to reproduce and validate the Boba L2 blockchain, block by block. The major limitation of Turing is that it is (currently) centralized, since it uses a modified Geth sequencer, of which there is only one at present.

The new Turing-enabled L2Geth was released to [Rinkeby](rinkeby.boba.network) today, and all updated core services (`core-utils`, `data-translation-layer`, `batch-submitter`, `replica`, `verifier`, and `fraud detector`) are also all up and working correctly, and the `fraud-detector` is correctly handling the new system, the new block metadata, and the API response data that are being written to L1.

[Examples](https://github.com/bobanetwork/boba/tree/turing-hybrid-compute/packages/boba/turing) include using Turing to do the math for a stableswap contract and pulling example *BTC-USD* price data from an off-chain data provider. Using Turing involves registering your own `turingHelper` contract, funding this contract via a new `TuringCredit` contract, and then calling Turing functions, such as:

```javascript

  rate = lending.getCurrentQuote('https://i9iznmo33e.execute-api.us-east-1.amazonaws.com/quote', "BTC/USD")

  // test response
  Bitcoin to usd price is 42406.68
  timestamp 1642104413221
  ✓ should get the current Bitcoin - USD price (327ms)

```

BOBA is the native fee token for Turing. Each Turing call costs 0.01 BOBA, which is debited from the exchange's or deployer's balance in the `TuringCredit` contract. The end users of the DEX or DAPP will typically not see these transactions, since they are billed to the deployer of the calling contract.

## 3. Boba Straw - On-chain Price Oracles

[Boba Straw](https://github.com/bobanetwork/boba/tree/develop/packages/boba/contracts/contracts/oracle) is built on the Chainlink contracts and represents a traditional on-chain price oracle. Boba Straw was deployed to Rinkeby today and we are assisting data providers with early testing as they prepare to push data. BOBA is the native fee token for Boba Straw.

## 4. Additional Data Sources for Boba

In addition to Turing and Boba Straw, a system for decentralized collection and verification of real-world data will go live in February. Details will be announced separately by the developers of this new system.

## 5. Minimally, 50% cost reduction of multi-token bridging

Currently, all tokens (e.g. ETH and BOBA) must be bridged separately. This week, the gateway will begin to allow multi-token bridging, such that as many as 4 tokens are bridged in *one* bridging operation. This means that for many users, the total cost of bridging will be reduced significantly, and the overall experience will be faster and less confusing as well.

## 6. Updated Blockexplorer

The updated [blockexplorer](https://blockexplorer.rinkeby.boba.network/) was launched today on Rinkeby. Enjoy. Next steps - enable support for mobile; fix dark mode; improve display of contract verification.

## 7. Gnosis Safe is live on Mainnet

[Boba Gnosis Safe](https://safe.boba.network/app/welcome) in on Mainnet. Enjoy - this functionality is critical for teams wishing to manage their treasuries on Boba, as well as for large DeFi and NFT projects. Next steps - add links to e.g. the gateway to make it easier to find and use.

## 8. NFT Bridging is live on Mainnet

NFTs minted on Boba can now be bridged to Ethereum Mainnet for use/sale there. See [NFT Bridges](https://github.com/bobanetwork/boba/blob/develop/packages/boba/contracts/contracts/bridges/README.md) for the technical details. **NOTE** - this functionality needs to be enabled/supported by the NFT minters and exchanges.
