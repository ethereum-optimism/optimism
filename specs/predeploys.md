# Predeploys

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [LegacyMessagePasser](#legacymessagepasser)
- [L2ToL1MessagePasser](#l2tol1messagepasser)
- [DeployerWhitelist](#deployerwhitelist)
- [LegacyERC20ETH](#legacyerc20eth)
- [WETH9](#weth9)
- [L2CrossDomainMessenger](#l2crossdomainmessenger)
- [L2StandardBridge](#l2standardbridge)
- [L1BlockNumber](#l1blocknumber)
- [GasPriceOracle](#gaspriceoracle)
- [L1Block](#l1block)
- [ProxyAdmin](#proxyadmin)
- [SequencerFeeVault](#sequencerfeevault)
- [OptimismMintableERC20Factory](#optimismmintableerc20factory)
- [OptimismMintableERC721Factory](#optimismmintableerc721factory)
- [BaseFeeVault](#basefeevault)
- [L1FeeVault](#l1feevault)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

[Predeployed smart contracts](./glossary.md#predeployed-contract-predeploy) exist on Optimism
at predetermined addresses in the genesis state. They are  similar to precompiles but instead run
directly in the EVM instead of running  native code outside of the EVM.

Predeploys are used instead of precompiles to make it easier for multiclient
implementations as well as allowing for more integration with hardhat/foundry
network forking.

Predeploy addresses exist in 1 byte namespace `0x42000000000000000000000000000000000000xx`.
Proxies are set at each possible predeploy address except for the
`GovernanceToken` and the `ProxyAdmin`.

The `LegacyERC20ETH` predeploy lives at a special address `0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000`
and there is no proxy deployed at that account.

The following table includes each of the predeploys. The system version
indicates when the predeploy was introduced. The possible values are `Legacy`
or `Bedrock`. Deprecated contracts should not be used.

| Name                          | Address                                    | Introduced | Deprecated | Proxied |
| ----------------------------- | ------------------------------------------ | ---------- | ---------- |---------|
| LegacyMessagePasser           | 0x4200000000000000000000000000000000000000 | Legacy     | Yes        | Yes     |
| DeployerWhitelist             | 0x4200000000000000000000000000000000000002 | Legacy     | Yes        | Yes     |
| LegacyERC20ETH                | 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000 | Legacy     | Yes        | No      |
| WETH9                         | 0x4200000000000000000000000000000000000006 | Legacy     | No         | No      |
| L2CrossDomainMessenger        | 0x4200000000000000000000000000000000000007 | Legacy     | No         | Yes     |
| L2StandardBridge              | 0x4200000000000000000000000000000000000010 | Legacy     | No         | Yes     |
| SequencerFeeVault             | 0x4200000000000000000000000000000000000011 | Legacy     | No         | Yes     |
| OptimismMintableERC20Factory  | 0x4200000000000000000000000000000000000012 | Legacy     | No         | Yes     |
| L1BlockNumber                 | 0x4200000000000000000000000000000000000013 | Legacy     | Yes        | Yes     |
| GasPriceOracle                | 0x420000000000000000000000000000000000000F | Legacy     | No         | Yes     |
| GovernanceToken               | 0x4200000000000000000000000000000000000042 | Legacy     | No         | No      |
| L1Block                       | 0x4200000000000000000000000000000000000015 | Bedrock    | No         | Yes     |
| L2ToL1MessagePasser           | 0x4200000000000000000000000000000000000016 | Bedrock    | No         | Yes     |
| L2ERC721Bridge                | 0x4200000000000000000000000000000000000014 | Legacy     | No         | Yes     |
| OptimismMintableERC721Factory | 0x4200000000000000000000000000000000000017 | Bedrock    | No         | Yes     |
| ProxyAdmin                    | 0x4200000000000000000000000000000000000018 | Bedrock    | No         | No      |
| BaseFeeVault                  | 0x4200000000000000000000000000000000000019 | Bedrock    | No         | Yes     |
| L1FeeVault                    | 0x420000000000000000000000000000000000001a | Bedrock    | No         | Yes     |

## LegacyMessagePasser

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/legacy/LegacyMessagePasser.sol)

Address: `0x4200000000000000000000000000000000000000`

The `LegacyMessagePasser` contract stores commitments to withdrawal
transactions before the Bedrock upgrade. A merkle proof to a particular
storage slot that commits to the withdrawal transaction is used as part
of the withdrawing transaction on L1. The expected account that includes
the storage slot is hardcoded into the L1 logic. After the bedrock upgrade,
the `L2ToL1MessagePasser` is used instead. Finalizing withdrawals from this
contract will no longer be supported after the Bedrock and is only left
to allow for alternative bridges that may depend on it. This contract does
not forward calls to the `L2ToL1MessagePasser` and calling it is considered
a no-op in context of doing withdrawals through the `CrossDomainMessenger`
system.

Any pending withdrawals that have not been finalized are migrated to the
`L2ToL1MessagePasser` as part of the upgrade so that they can still be
finalized.

## L2ToL1MessagePasser

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/L2ToL1MessagePasser.sol)

Address: `0x4200000000000000000000000000000000000016`

The `L2ToL1MessagePasser` stores commitments to withdrawal transactions.
When a user is submitting the withdrawing transaction on L1, they provide a
proof that the transaction that they withdrew on L2 is in the `sentMessages`
mapping of this contract.

Any withdrawn ETH accumulates into this contract on L2 and can be
permissionlessly removed from the L2 supply by calling the `burn()` function.

## DeployerWhitelist

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/legacy/DeployerWhitelist.sol)

Address: `0x4200000000000000000000000000000000000002`

The `DeployerWhitelist` is a predeploy that was used to provide additional safety
during the initial phases of Optimism.
It previously defined the accounts that are allowed to deploy contracts to the network.

Arbitrary contract deployment was subsequently enabled and it is not possible to turn
off. In the legacy system, this contract was hooked into `CREATE` and
`CREATE2` to ensure that the deployer was allowlisted.

In the Bedrock system, this contract will no longer be used as part of the
`CREATE` codepath.

This contract is deprecated and its usage should be avoided.

## LegacyERC20ETH

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/legacy/LegacyERC20ETH.sol)

Address: `0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000`

The `LegacyERC20ETH` predeploy represents all ether in the system before the
Bedrock upgrade. All ETH was represented as an ERC20 token and users could opt
into the ERC20 interface or the native ETH interface.

The upgrade to Bedrock migrates all ether out of this contract and moves it to
its native representation. All of the stateful methods in this contract will
revert after the Bedrock upgrade.

This contract is deprecated and its usage should be avoided.

## WETH9

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/vendor/WETH9.sol)

Address: `0x4200000000000000000000000000000000000006`

`WETH9` is the standard implementation of Wrapped Ether on Optimism. It is a
commonly used contract and is placed as a predeploy so that it is at a
deterministic address across Optimism based networks.

## L2CrossDomainMessenger

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/L2CrossDomainMessenger.sol)

Address: `0x4200000000000000000000000000000000000007`

The `L2CrossDomainMessenger` gives a higher level API for sending cross domain
messages compared to directly calling the `L2ToL1MessagePasser`.
It maintains a mapping of L1 messages that have been relayed to L2
to prevent replay attacks and also allows for replayability if the L1 to L2
transaction reverts on L2.

Any calls to the `L1CrossDomainMessenger` on L1 are serialized such that they
go through the `L2CrossDomainMessenger` on L2.

The `relayMessage` function executes a transaction from the remote domain while
the `sendMessage` function sends a transaction to be executed on the remote
domain through the remote domain's `relayMessage` function.

## L2StandardBridge

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/L2StandardBridge.sol)

Address: `0x4200000000000000000000000000000000000010`

The `L2StandardBridge` is a higher level API built on top of the
`L2CrossDomainMessenger` that gives a standard interface for sending ETH or
ERC20 tokens across domains.

To deposit a token from L1 to L2, the `L1StandardBridge` locks the token and
sends a cross domain message to the `L2StandardBridge` which then mints the
token to the specified account.

To withdraw a token from L2 to L1, the user will burn the token on L2 and the
`L2StandardBridge` will send a message to the `L1StandardBridge` which will
unlock the underlying token and transfer it to the specified account.

The `OptimismMintableERC20Factory` can be used to create an ERC20 token contract
on a remote domain that maps to an ERC20 token contract on the local domain
where tokens can be deposited to the remote domain. It deploys an
`OptimismMintableERC20` which has the interface that works with the
`StandardBridge`.

This contract can also be deployed on L1 to allow for L2 native tokens to be
withdrawn to L1.

## L1BlockNumber

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/legacy/L1BlockNumber.sol)

Address: `0x4200000000000000000000000000000000000013`

The `L1BlockNumber` returns the last known L1 block number. This contract was
introduced in the legacy system and should be backwards compatible by calling
out to the `L1Block` contract under the hood.

It is recommended to use the `L1Block` contract for getting information about
L1 on L2.

## GasPriceOracle

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/GasPriceOracle.sol)

Address: `0x420000000000000000000000000000000000000F`

In the legacy system, the `GasPriceOracle` was a permissioned contract
that was pushed the L1 basefee and the L2 gas price by an offchain actor.
The offchain actor observes the L1 blockheaders to get the
L1 basefee as well as the gas usage on L2 to compute what the L2 gas price
should be based on a congestion control algorithm.

After Bedrock, the `GasPriceOracle` is no longer a permissioned contract
and only exists to preserve the API for offchain gas estimation. The
function `getL1Fee(bytes)` accepts an unsigned RLP transaction and will return
the L1 portion of the fee. This fee pays for using L1 as a data availability
layer and should be added to the L2 portion of the fee, which pays for
execution, to compute the total transaction fee.

The values used to compute the L2 portion of the fee are:

- scalar
- overhead
- decimals

After the Bedrock upgrade, these values are instead managed by the
`SystemConfig` contract on L2. The `scalar` and `overhead` values
are sent to the `L1Block` contract each block and the `decimals` value
has been hardcoded to 6.

## L1Block

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/L1Block.sol)

Address: `0x4200000000000000000000000000000000000015`

[l1-block-predeploy]: glossary.md#l1-attributes-predeployed-contract

The [L1Block][l1-block-predeploy] was introduced in Bedrock and is responsible for
maintaining L1 context in L2. This allows for L1 state to be accessed in L2.

## ProxyAdmin

[ProxyAdmin](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/ProxyAdmin.sol)
Address: `0x4200000000000000000000000000000000000018`

The `ProxyAdmin` is the owner of all of the proxy contracts set at the
predeploys. It is itself behind a proxy. The owner of the `ProxyAdmin` will
have the ability to upgrade any of the other predeploy contracts.

## SequencerFeeVault

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/SequencerFeeVault.sol)

Address: `0x4200000000000000000000000000000000000011`

The `SequencerFeeVault` accumulates any transaction priority fee and is the value of
`block.coinbase`.
When enough fees accumulate in this account, they can be withdrawn to an immutable L1 address.

To change the L1 address that fees are withdrawn to, the contract must be
upgraded by changing its proxy's implementation key.

## OptimismMintableERC20Factory

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/OptimismMintableERC20Factory.sol)

Address: `0x4200000000000000000000000000000000000012`

The `OptimismMintableERC20Factory` is responsible for creating ERC20 contracts on L2 that can be
used for depositing native L1 tokens into. These ERC20 contracts can be created permisionlessly
and implement the interface required by the `StandardBridge` to just work with deposits and withdrawals.

Each ERC20 contract that is created by the `OptimismMintableERC20Factory` allows for the `L2StandardBridge` to mint
and burn tokens, depending on if the user is depositing from L1 to L2 or withdrawaing from L2 to L1.

## OptimismMintableERC721Factory

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/OptimismMintableERC721Factory.sol)

Address: `0x4200000000000000000000000000000000000017`

The `OptimismMintableERC721Factory` is responsible for creating ERC721 contracts on L2 that can be used for
depositing native L1 NFTs into.

## BaseFeeVault

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/BaseFeeVault.sol)

Address: `0x4200000000000000000000000000000000000019`

The `BaseFeeVault` predeploy receives the basefees on L2. The basefee is not
burnt on L2 like it is on L1. Once the contract has received a certain amount
of fees, the ETH can be withdrawn to an immutable address on
L1.

## L1FeeVault

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L2/L1FeeVault.sol)

Address: `0x420000000000000000000000000000000000001a`

The `L1FeeVault` predeploy receives the L1 portion of the transaction fees.
Once the contract has received a certain amount of fees, the ETH can be
withdrawn to an immutable address on L1.
