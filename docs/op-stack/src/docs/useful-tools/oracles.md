---
title: Oracles 
lang: en-US
---


[Oracles](https://ethereum.org/en/developers/docs/oracles/) provide off-chain data on chain. 
This allows code running on a blockchain to access a wide variety of information.
For example, a [stablecoin](https://ethereum.org/en/stablecoins/) that accepts ETH as collateral needs to know the ETH/USD exchange rate:

- How many stablecoins can we give a user for a given amount of ETH?
- Do we need to liquidate any deposits because they are under collateralized?

Different oracles have different security assumptions and different levels of decentralization.
Usually they are either run by the organization that produces the information, or have a mechanism to reward entites that provide accurate information and penalize those that provide incorrect information.

## Types of oracles

There are two types of oracles:

1. **Push oracles** are updated continously and always have up to date information available on chain.

1. **Pull oracles** are only updated when information is requested by a contract.
   Pull oracles are themselves divided into two types:
   1. Double-transaction oracles, which require two transactions. 
      The first transaction is the request for information, which usually causes the oracle to emit an event that triggers some off-chain mechanism to provide the answer (through its own transaction).
      The second transaction actually reads on-chain the result from the oracle and uses it.
   1. Single-transaction oracles, which only require one transaction, such as [Chainlink's random nunber generator](https://docs.chain.link/docs/get-a-random-number/#request-random-values).
      The way this works is that the transaction that requests the information includes a callback (address and the call data to provide it). 
      When the oracle is updated (which also happens through a transaction, but one that is not sent by the user), the oracle uses the callback to inform a contract of the result.

## Gas Oracle

Optimism provides a [Gas Price Oracle](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_GasPriceOracle.sol) that provides information about [gas prices and related parameters](../developers/build/transaction-fees.md).
It can also calculate the total cost of a transaction for you before you send it.

This contract is a predeploy at address `0x420000000000000000000000000000000000000F`:

- [On the production Optimism network](https://explorer.optimism.io/address/0x420000000000000000000000000000000000000F#readContract)
- [On the Optimism Goerli test network](https://goerli-explorer.optimism.io/address/0x420000000000000000000000000000000000000F)

This is a push Oracle. 
Optimism updates the gas price parameters on chain whenever those parameters change. 
The L1 gas price, which can be volatile, is only pushed once every 5 minutes, and each time can change only by up to 20%.

## Chainlink

On Optimism [Chainlink](https://chain.link/) provides a number of [price feeds](https://docs.chain.link/docs/optimism-price-feeds/).
Those feeds are available on the production network.

This is a push Oracle. 
You can always get up to date information (see, for example, [here (scroll down to **latestAnswer**)](https://explorer.optimism.io/address/0x13e3Ee699D1909E989722E753853AE30b17e08c5#readContract)).

[See this guide to learn how to use the Chainlink feeds](https://docs.chain.link/docs/get-the-latest-price/).



## Portal

[Portal](https://portal-docs.readthedocs.io/en/latest/index.html) offers users private, scalable, and fast compute power at low costs.

### Verifiable Randomness Function (VRF)

#### Portal
Portal providers a source of randomness on chain (for now on Optimism Goerli, but eventually also on the Optimism mainnet).
[You can learn how to use it here](https://portal-docs.readthedocs.io/en/latest/user-vrf-docs.html). 
It is a single-transaction pull oracle.

#### Band
[Band](https://bandprotocol.com/vrf) provides a source of [on-chain randomness](https://bandprotocol.com/vrf). 
[You can learn how to use it here](https://docs.bandchain.org/vrf/getting-started.html).
It is a single-transaction pull oracle.



## Universal Market Access (UMA)

[UMA](https://umaproject.org/) is a generic oracle.
It lets any contract request information (ask a question), and any staked entity can provide an answer.
Other external entities can dispute the proposed answer by providing their own answer and a putting up their own stake.
In the case of dispute the question goes to a vote of token holders.
The token holders that vote with the majority are assumed to be truthful and get rewarded.
The external entities that proposed the correct answer are rewarded.
Those that proposed the wrong answer lose their stake.

[See here for the UMA addresses on Optimism](https://github.com/UMAprotocol/protocol/blob/master/packages/core/networks/10.json). 

[See here for instructions how to use UMA](https://docs.umaproject.org/build-walkthrough/build-process).

UMA is a pull Oracle, it does not get information until it is requested by a contract. 
This means that an decentralized application needs to issue two transactions.
First, a transaction that causes a contract on the blockchain to ask for the information.
Later (in the case of UMA 48 hours later if there is no dispute, longer if there is), a second transaction need to be triggered to causes the contract to read from the Oracle and see the response to the request.

## Uniswap

Technically speaking [Uniswap](https://uniswap.io/) is not an oracle, because the information comes from on-chain sources.
However, Uniswap pools do provide [quotes that give the relative costs of assets](https://docs.uniswap.org/protocol/concepts/V3-overview/oracle).

::: warning

Using onchain asset prices, especially those in low liquidity pools, makes you vulnerable to price manipulation. 

:::

To use Uniswap as an Oracle:

1. See [the list of pools on Optimism](https://info.uniswap.org/#/optimism/).
1. To find the pool address, [look at the Uniswap factory](https://explorer.optimism.io/address/0x1f98431c8ad98523631ae4a59f267346ea31f984#readContract).
   Use **getPool** with these parameters:

      | Parameter           | Meaning                             |
      | ------------------- | ----------------------------------- | 
      | One token address   | [Address of the ERC-20 contract for that token on Optimism (chainId 10)](https://static.optimism.io/optimism.tokenlist.json) |
      | Other token address | [Address of the ERC-20 contract for that token on Optimism (chainId 10)](https://static.optimism.io/optimism.tokenlist.json) |      
      | Pool fee            | The pool fee percentage times ten thousand. For example, for 0.3% enter `3000` |

1. In your contract, use [IUniswapV3PoolState](https://github.com/Uniswap/v3-core/blob/main/contracts/interfaces/pool/IUniswapV3PoolState.sol) and [IUniswapV3PoolDerivedState](https://github.com/Uniswap/v3-core/blob/main/contracts/interfaces/pool/IUniswapV3PoolDerivedState.sol) to get the pool state.
