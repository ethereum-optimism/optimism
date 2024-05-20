# Engineering Update #10

- [1. Emerging Turing Uses](#1-emerging-turing-uses)
- [2. Data are flowing](#2-data-are-flowing)
- [3. Gateway updates](#3-gateway-updates)
- [4. Under the hood - updated L2Geth / Turing](#4-under-the-hood---updated-l2geth---turing)
- [5. New documentation](#5-new-documentation)

February 21, 2022

Greetings from your engineering team. Many team members were at ETH Denver this week. We came back with many new ideas and feedback from teams wishing to deploy on Boba. A common motivation for looking at Boba was that their current chain was congested and they needed a more responsive. Our main focus for this week is to complete a cross-stack update of many services, continue to test Turing on Rinkeby, and prepare to release a new gateway.

## 1. Emerging Turing Uses

We are increasingly using Turing on Rinkeby for a variety of purposes. The new [ETH/BOBA fountain on Rinkeby](https://faucets.boba.network) is secured by a CAPTCHA, which uses Turing. Second, we have been [minting TuringMonsters NFTs on Rinkeby](https://github.com/bobanetwork/boba/blob/develop/boba_community/turing-monsters/README.md). When Turing goes Mainnet, the gateway will have a `mint` button that allows all of you to mint your own TuringMonster. Based on discussions with traders and fintech, we are seeing interest in the notion of using Turing as a way to protect algorithms and data - sensitive code can live entirely off-chain with smart contracts on L2 providing a settlement layer.

## 2. Data are flowing

[**Boba-Straw**](https://github.com/bobanetwork/boba/blob/develop/boba_documentation/Price_Data_Feeds_Overview.md), [**Witnet**](https://feeds.witnet.io/), and [**Turing**](https://github.com/bobanetwork/boba/blob/develop/packages/boba/turing/README.md) are now available for price feeds. Folkvang has pushed hundreds of price updates to Boba Straw and $BOBA are flowing back to Folkvang. A new [data overview is available](https://github.com/bobanetwork/boba/blob/develop/boba_documentation/Price_Data_Feeds_Overview.md) to help new data providers join the system, and to have DeFi teams use the new data feeds.

## 3. Gateway updates

The revised gateway will be released for testing on Rinkeby within the next 24 hours. We welcome your feedback. The only thing still missing there is a new *Bridge* page designed to make it easier to bridge to Boba.

## 4. Under the hood - updated L2Geth / Turing

Significantly updated code for various services is now running on Rinkeby, in preparation for launching Turing on Mainnet on March 1. We ran into issues with the new caching system for Turing, which did not play well with the L2Geth in replica mode. Another problem we encountered was that the new L2 blocks contain a Turing meta-data field, which is not present in the 'legacy' blocks on Rinkeby and Mainnet. Additional work was therefore needed to ensure that all systems can smoothly handle both types of blocks ('legacy' and 'turing').

## 5. New documentation

So many new features have been recently released (**Boba-Straw**, **Turing**, **ETH/BOBA fountain on Rinkeby**, **TuringMonsters**, ...) that our documentation is outdated in places; a secondary focus of this week has been to update and check all documentation. That process will easily take another week since more than ~50 pages of material must be hand-checked and updated.
