# Engineering Update #9

- [1. Roadmap and Whitepaper - update](#1-roadmap-and-whitepaper---update)
- [2. Data are flowing](#2-data-are-flowing)
- [3. Gateway updates](#3-gateway-updates)
- [4. Under the hood - updated L2Geth / Turing](#4-under-the-hood---updated-l2geth---turing)
- [5. WAGMIv0 and WAGMIv1](#5-wagmiv0-and-wagmiv1)
- [6. Boba3 (prelude)](#6-boba3--prelude-)

February 13, 2022

Greetings from your engineering team. As noted, we have a transitioned to a 7 day tech update cycle. The main goal is to support a quickly-growing messaging infrastructure, which will take this and other material and make sure that it is more broadly distributed compared to what Boba did in the past.

## 1. Roadmap and Whitepaper - update

A graphical roadmap for 2022 was released [today](https://github.com/bobanetwork/boba/blob/develop/boba_documentation/roadmaps/RoadmapFeb13_2022.svg). As noted, this will be a 'living' document, subject to major change as needed. Boba research will be presenting a technical talk at ETH-Denver, and the slides for that - focusing on Turing, will be released after the talk (two days from now). In terms of other materials, please also see a first schematic for [Turing](https://github.com/bobanetwork/boba/blob/develop/boba_documentation/diagrams/TuringOverview.pdf).

## 2. Data are flowing

**Boba-Straw** - Boba's own price-feed oracle is now up and running on both Boba Mainnet and Boba Rinkeby. The price feed, based on Chainlink's implementation, can handle price data aggregation from multiple oracles, on-chain. To get us started - **Folkvang** has started streaming data and is acting as our first data source. While Folkvang has their own price-data aggregation strategy, to further increase reliability we will keep adding more data sources to Boba-Straw in the future. And, of course - BOBA is what keeps driving the price-feeds in the form of providing incentives for price data-submission and as a fee token to utilise/subscribe to these feeds.**Witnet**, a decentralised oracle network is also live on Boba (Mainnet and Rinkeby) with their witness-backed decentralised price feeds. While multiple options gives more to chose from, it also might just present the opportunity to have your own personal level of data-aggregation - by combining these for consistency and reliability!
Check out the various active feeds [here](https://feeds.witnet.io/), for example.

## 3. Gateway updates

The revised gateway is on track with new pages for the DAO and NFTs and a completely rewritten back-end. The main change there is to allow people to see as much as possible without needing to connect MetaMask, to make it easy to get a sense of the Boba ecosystem right away. New features include a BOBA faucet secured by a CAPCHA, which in turn is using Turing to connect real world events (solving a game/CAPTCHA) to an an on-chain event - obtaining some BOBA. Additional features include an entirely new `Bridge` page designed to simplify the bridging experience greatly.

## 4. Under the hood - updated L2Geth / Turing

We are currently testing an updated version of the L2Geth developed by Optimism, with a view to updating our Geth this coming week. This will be a hot-update with minimal downtime (less than 2 hours). The updated L2Geth will also contain an updated version of Turing, with a new caching and gasEstimation system to provide better performance under load.

## 5. WAGMIv0 and WAGMIv1

As part of the gateway updates, expect an interface to redeem your WAGMIv0, and to handle WAGMIv1, which will target new metrics and therefore needs a new oracle, a new LSP contract, and a new token (WAGMIv1).

## 6. Boba3 (prelude)

As noted on the roadmap, see above, we are preparing to transition to an alternative L2 architecture, but to do so safely and smoothly we will run the two systems side-by-side, with Optimism-style messages tunneling through the new system - provisionally called a **Zipup**. Hence the code name for Boba3 - prelude - it's a stepping stone to Boba4. The overall goal of these major architectural changes is to (**1**) end up with a highly secure and cost efficient system that is also (**2**) easier to maintain and update than the current approach. Both Boba3 and Boba4 will have similarities to Optimism at the smart contract level but will take a substantially different approach to cross-chain communications and bridging. In large part this transition is driven by anticipated changes in Ethereum's architecture - all L2s should already be preparing for those changes, to ensure a smooth migration, excellent user experience, and substantial cost savings relative to Ethereum. Documentation on the new system remains to be written, but that is underway as well.
