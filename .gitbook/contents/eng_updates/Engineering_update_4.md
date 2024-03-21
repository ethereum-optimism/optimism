# DRAFT Engineering update #4

- [Engineering update #4](#engineering-update--4)
  * [1. Optimism Platform/Foundation](#1-optimism-platform-foundation)
  * [2. Production / webscale](#2-production---webscale)
  * [3. User experience / front end](#3-user-experience---front-end)
  * [4. Security](#4-security)
  * [5. Hybrid Compute](#5-hybrid-compute)
  * [6. User and Developer Experience](#6-user-and-developer-experience)

Thu June 24

Greetings from your engineering team. We have been heads' down the last few weeks, to get many basic pieces for Boba into place and stable. The most visible parts of this are an increasingly stable rinkeby testnet (https://rinkeby.boba.network) and a working webwallet (aka gateway) with swap on/off and farming at https://webwallet.rinkeby.boba.network.

## 1. Optimism Platform/Foundation

The Optimism team continues to move rapidly, whilst producing very high quality code and many new useful features and improvements, such as a simplified and more elegant approach to token bridges, as well as extensive infrastructure dedicated to making L2s financially viable, for example by providing accurate price data for the sequencer and to all users.

## 2. Production / webscale

* Under the hood, much effort has been devoted to web-scale production deployment of rollups - you can follow along here: https://github.com/bobanetwork/boba/pull/46. The production rinkeby testnet currently runs on AWS EC2, which is stable but less scalable. A major next step will be to flip the switch, and transition to the AWS ECS Cloudformation system and all its associated CI/CD infrastructure and real time health monitoring. If response times start to degrade, e.g. when users are transferring funds across the chains, then we need to know about that within a few seconds, ideally.

* On a more basic level, our major goal is to have a system that (1) gracefully responds to restart of one or more subservices, without getting confused, and (2) that can recover to last-known-good state automatically after unexpected failure of one of more subservices.

## 3. User experience / front end

* Although the webwallet works (https://webwallet.rinkeby.boba.network), the user experience remains rudimentary. This is not just a challenge for us, but for all L2s and multichain situations. The key question is - how do we make using an L2 as easy and intuitive as possible? Two front-end engineers started last week, and are working with community members and UI designers to develop a good way to make L2s intuitive. The good news is that many people have more than one bank account (savings and checking, for example), so they are already used to the notion of using different accounts for different purposes. However, they are *not* used to having to reconfigure their banks's settings when they e.g. want to transfer funds from one account to another - this should just really happen with minimal user reconfiguration. You can follow along on the Pull Requests (https://github.com/bobanetwork/boba/pulls); front-end focused work is growing quickly.

* Closely related to the UI, is our solution for fast on/off, which consists of deploying liquidity pools across the chains - i.e. we represent moving funds to L2 as a swap, where a person swaps L1 ETH for L2 wETH, and where ETH and wETH never actually move across the chain boundaries. It's just like Uniswap, except that the two liquidity pools are not in one contract, but in two contracts, one on the L1 and the other one on the L2. This feature is live, and you can already earn Rinkeby ETH by staking, which provides liquidity for easy swap on/off: https://webwallet.rinkeby.boba.network - click *Farm* on the top tab. Note that contract improvements are pending, and that the contracts have not yet been audited.

* Blockexplorer. A blockexplorer is vital, and there is a simple one you can look at here: https://blockexplorer.rinkeby.boba.network. However, we will soon transition to an industry-leading blockexplorer, so you have all the functions you expect and we do not have to maintain our own blockexplorer. Related to the blockexplorer is a feature everyone expects - reliable transaction history, so they can easily see their transactions and get reliable updates about their deposits and withdrawals.

* Hardware wallet support. One of the gaping holes in the current webwallet/gateway is the complete lack of support for hardware wallets - our first step will be to add Ledger Nano support (https://github.com/bobanetwork/boba/issues/117).

## 4. Security

* Verifiers
* Automatic Fraud Proving and Recovery

## 5. Hybrid Compute

* Welcome 4 interns
* Hello world example working - next step is to write example code for real-time updating stable-swap like system

## 6. User and Developer Experience

* Working examples
* Updated documentation
