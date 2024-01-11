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
- [SchemaRegistry](#schemaregistry)
- [EAS](#eas)
- [create2Deployer](#create2deployer)
- [Safe](#safe)
- [SafeL2](#safel2)
- [MultiSend](#multisend)
- [MultiSendCallOnly](#multisendcallonly)
- [SafeSingletonFactory](#safesingletonfactory)
- [MultiCall3](#multicall3)
- [Arachnid's Deterministic Deployment Proxy](#arachnids-deterministic-deployment-proxy)
- [Permit2](#permit2)
- [ERC-4337 EntryPoint](#erc-4337-entrypoint)
- [ERC-4337 SenderCreator](#erc-4337-sendercreator)

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
or `Bedrock` or `Canyon`. Deprecated contracts should not be used.

| Name                                      | Address                                    | Introduced  | Deprecated | Proxied |
| ----------------------------------------- | ------------------------------------------ | ----------- | ---------- | ------- |
| LegacyMessagePasser                       | 0x4200000000000000000000000000000000000000 | Legacy      | Yes        | Yes     |
| DeployerWhitelist                         | 0x4200000000000000000000000000000000000002 | Legacy      | Yes        | Yes     |
| LegacyERC20ETH                            | 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000 | Legacy      | Yes        | No      |
| WETH9                                     | 0x4200000000000000000000000000000000000006 | Legacy      | No         | No      |
| L2CrossDomainMessenger                    | 0x4200000000000000000000000000000000000007 | Legacy      | No         | Yes     |
| L2StandardBridge                          | 0x4200000000000000000000000000000000000010 | Legacy      | No         | Yes     |
| SequencerFeeVault                         | 0x4200000000000000000000000000000000000011 | Legacy      | No         | Yes     |
| OptimismMintableERC20Factory              | 0x4200000000000000000000000000000000000012 | Legacy      | No         | Yes     |
| L1BlockNumber                             | 0x4200000000000000000000000000000000000013 | Legacy      | Yes        | Yes     |
| GasPriceOracle                            | 0x420000000000000000000000000000000000000F | Legacy      | No         | Yes     |
| GovernanceToken                           | 0x4200000000000000000000000000000000000042 | Legacy      | No         | No      |
| L1Block                                   | 0x4200000000000000000000000000000000000015 | Bedrock     | No         | Yes     |
| L2ToL1MessagePasser                       | 0x4200000000000000000000000000000000000016 | Bedrock     | No         | Yes     |
| L2ERC721Bridge                            | 0x4200000000000000000000000000000000000014 | Legacy      | No         | Yes     |
| OptimismMintableERC721Factory             | 0x4200000000000000000000000000000000000017 | Bedrock     | No         | Yes     |
| ProxyAdmin                                | 0x4200000000000000000000000000000000000018 | Bedrock     | No         | Yes     |
| BaseFeeVault                              | 0x4200000000000000000000000000000000000019 | Bedrock     | No         | Yes     |
| L1FeeVault                                | 0x420000000000000000000000000000000000001a | Bedrock     | No         | Yes     |
| SchemaRegistry                            | 0x4200000000000000000000000000000000000020 | Bedrock     | No         | Yes     |
| EAS                                       | 0x4200000000000000000000000000000000000021 | Bedrock     | No         | Yes     |
| create2Deployer                           | 0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2 | Canyon      | No         | No      |
| Safe                                      | 0x69f4D1788e39c87893C980c06EdF4b7f686e2938 | Bedrock*    | No         | No      |
| SafeL2                                    | 0xfb1bffC9d739B8D520DaF37dF666da4C687191EA | Bedrock*    | No         | No      |
| MultiSend                                 | 0x998739BFdAAdde7C933B942a68053933098f9EDa | Bedrock*    | No         | No      |
| MultiSendCallOnly                         | 0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B | Bedrock*    | No         | No      |
| SafeSingletonFactory                      | 0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7 | Bedrock*    | No         | No      |
| MultiCall3                                | 0xcA11bde05977b3631167028862bE2a173976CA11 | Bedrock*    | No         | No      |
| Arachnid's Deterministic Deployment Proxy | 0x4e59b44847b379578588920cA78FbF26c0B4956C | Bedrock*    | No         | No      |
| Permit2                                   | 0x000000000022D473030F116dDEE9F6B43aC78BA3 | Bedrock*    | No         | No      |
| ERC-4337 EntryPoint                       | 0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789 | Bedrock*    | No         | No      |
| ERC-4337 SenderCreator                    | 0x7fc98430eaedbb6070b35b39d798725049088348 | Bedrock*    | No         | No      |

\* Early Bedrock chains do not include these contracts at genesis,
but all chains going forward will have them at genesis

## LegacyMessagePasser

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/legacy/LegacyMessagePasser.sol)

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

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2ToL1MessagePasser.sol)

Address: `0x4200000000000000000000000000000000000016`

The `L2ToL1MessagePasser` stores commitments to withdrawal transactions.
When a user is submitting the withdrawing transaction on L1, they provide a
proof that the transaction that they withdrew on L2 is in the `sentMessages`
mapping of this contract.

Any withdrawn ETH accumulates into this contract on L2 and can be
permissionlessly removed from the L2 supply by calling the `burn()` function.

## DeployerWhitelist

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/legacy/DeployerWhitelist.sol)

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

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/legacy/LegacyERC20ETH.sol)

Address: `0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000`

The `LegacyERC20ETH` predeploy represents all ether in the system before the
Bedrock upgrade. All ETH was represented as an ERC20 token and users could opt
into the ERC20 interface or the native ETH interface.

The upgrade to Bedrock migrates all ether out of this contract and moves it to
its native representation. All of the stateful methods in this contract will
revert after the Bedrock upgrade.

This contract is deprecated and its usage should be avoided.

## WETH9

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/vendor/WETH9.sol)

Address: `0x4200000000000000000000000000000000000006`

`WETH9` is the standard implementation of Wrapped Ether on Optimism. It is a
commonly used contract and is placed as a predeploy so that it is at a
deterministic address across Optimism based networks.

## L2CrossDomainMessenger

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol)

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

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2StandardBridge.sol)

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

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/legacy/L1BlockNumber.sol)

Address: `0x4200000000000000000000000000000000000013`

The `L1BlockNumber` returns the last known L1 block number. This contract was
introduced in the legacy system and should be backwards compatible by calling
out to the `L1Block` contract under the hood.

It is recommended to use the `L1Block` contract for getting information about
L1 on L2.

## GasPriceOracle

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/GasPriceOracle.sol)

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

The values used to compute the L1 portion of the fee prior to the Ecotone upgrade are:

- scalar
- overhead
- decimals

After the Bedrock upgrade, these values are instead managed by the
`SystemConfig` contract on L1. The `scalar` and `overhead` values
are sent to the `L1Block` contract each block and the `decimals` value
has been hardcoded to 6.

Following the Ecotone upgrade, the values used for L1 fee computation are:

- l1BasefeeScalar
- l1BlobBasefeeScalar
- decimals

These values are managed by the `SystemConfig` contract on the L1. The`decimals` remains hardcoded
to 6, and the old `scalar` and `overhead` values are ignored.

## L1Block

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L1Block.sol)

Address: `0x4200000000000000000000000000000000000015`

[l1-block-predeploy]: glossary.md#l1-attributes-predeployed-contract

The [L1Block][l1-block-predeploy] was introduced in Bedrock and is responsible for
maintaining L1 context in L2. This allows for L1 state to be accessed in L2.

## ProxyAdmin

[ProxyAdmin](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/universal/ProxyAdmin.sol)
Address: `0x4200000000000000000000000000000000000018`

The `ProxyAdmin` is the owner of all of the proxy contracts set at the
predeploys. It is itself behind a proxy. The owner of the `ProxyAdmin` will
have the ability to upgrade any of the other predeploy contracts.

## SequencerFeeVault

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/SequencerFeeVault.sol)

Address: `0x4200000000000000000000000000000000000011`

The `SequencerFeeVault` accumulates any transaction priority fee and is the value of
`block.coinbase`.
When enough fees accumulate in this account, they can be withdrawn to an immutable L1 address.

To change the L1 address that fees are withdrawn to, the contract must be
upgraded by changing its proxy's implementation key.

## OptimismMintableERC20Factory

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/universal/OptimismMintableERC20Factory.sol)

Address: `0x4200000000000000000000000000000000000012`

The `OptimismMintableERC20Factory` is responsible for creating ERC20 contracts on L2 that can be
used for depositing native L1 tokens into. These ERC20 contracts can be created permisionlessly
and implement the interface required by the `StandardBridge` to just work with deposits and withdrawals.

Each ERC20 contract that is created by the `OptimismMintableERC20Factory` allows for the `L2StandardBridge` to mint
and burn tokens, depending on if the user is depositing from L1 to L2 or withdrawing from L2 to L1.

## OptimismMintableERC721Factory

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/universal/OptimismMintableERC721Factory.sol)

Address: `0x4200000000000000000000000000000000000017`

The `OptimismMintableERC721Factory` is responsible for creating ERC721 contracts on L2 that can be used for
depositing native L1 NFTs into.

## BaseFeeVault

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/BaseFeeVault.sol)

Address: `0x4200000000000000000000000000000000000019`

The `BaseFeeVault` predeploy receives the basefees on L2. The basefee is not
burnt on L2 like it is on L1. Once the contract has received a certain amount
of fees, the ETH can be withdrawn to an immutable address on
L1.

## L1FeeVault

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L1FeeVault.sol)

Address: `0x420000000000000000000000000000000000001a`

The `L1FeeVault` predeploy receives the L1 portion of the transaction fees.
Once the contract has received a certain amount of fees, the ETH can be
withdrawn to an immutable address on L1.

## SchemaRegistry

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/EAS/SchemaRegistry.sol)

Address: `0x4200000000000000000000000000000000000020`

The `SchemaRegistry` predeploy implements the global attestation schemas for the `Ethereum Attestation Service`
protocol.

## EAS

[Implementation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/EAS/EAS.sol)

Address: `0x4200000000000000000000000000000000000021`

The `EAS` predeploy implements the `Ethereum Attestation Service` protocol.

## create2Deployer

[Implementation](https://github.com/mdehoog/create2deployer/blob/69b9a8e112b15f9257ce8c62b70a09914e7be29c/contracts/Create2Deployer.sol)

The create2Deployer is a nice Solidity wrapper around the CREATE2 opcode. It provides the following ABI.

```solidity
    /**
     * @dev Deploys a contract using `CREATE2`. The address where the
     * contract will be deployed can be known in advance via {computeAddress}.
     *
     * The bytecode for a contract can be obtained from Solidity with
     * `type(contractName).creationCode`.
     *
     * Requirements:
     * - `bytecode` must not be empty.
     * - `salt` must have not been used for `bytecode` already.
     * - the factory must have a balance of at least `value`.
     * - if `value` is non-zero, `bytecode` must have a `payable` constructor.
     */
    function deploy(uint256 value, bytes32 salt, bytes memory code) public

    /**
     * @dev Deployment of the {ERC1820Implementer}.
     * Further information: https://eips.ethereum.org/EIPS/eip-1820
     */
    function deployERC1820Implementer(uint256 value, bytes32 salt)

    /**
     * @dev Returns the address where a contract will be stored if deployed via {deploy}.
     * Any change in the `bytecodeHash` or `salt` will result in a new destination address.
     */
    function computeAddress(bytes32 salt, bytes32 codeHash) public view returns (address)

    /**
     * @dev Returns the address where a contract will be stored if deployed via {deploy} from a
     * contract located at `deployer`. If `deployer` is this contract's address, returns the
     * same value as {computeAddress}.
     */
    function computeAddressWithDeployer(
        bytes32 salt,
        bytes32 codeHash,
        address deployer
    ) public pure returns (address)
```

Address: `0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2`

When Canyon activates, the contract code at `0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2` is set to
`0x6080604052600436106100435760003560e01c8063076c37b21461004f578063481286e61461007157806356299481146100ba57806366cfa057146100da57600080fd5b3661004a57005b600080fd5b34801561005b57600080fd5b5061006f61006a366004610327565b6100fa565b005b34801561007d57600080fd5b5061009161008c366004610327565b61014a565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b3480156100c657600080fd5b506100916100d5366004610349565b61015d565b3480156100e657600080fd5b5061006f6100f53660046103ca565b610172565b61014582826040518060200161010f9061031a565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe082820381018352601f90910116604052610183565b505050565b600061015683836102e7565b9392505050565b600061016a8484846102f0565b949350505050565b61017d838383610183565b50505050565b6000834710156101f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f437265617465323a20696e73756666696369656e742062616c616e636500000060448201526064015b60405180910390fd5b815160000361025f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f437265617465323a2062797465636f6465206c656e677468206973207a65726f60448201526064016101eb565b8282516020840186f5905073ffffffffffffffffffffffffffffffffffffffff8116610156576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f437265617465323a204661696c6564206f6e206465706c6f790000000000000060448201526064016101eb565b60006101568383305b6000604051836040820152846020820152828152600b8101905060ff815360559020949350505050565b61014e806104ad83390190565b6000806040838503121561033a57600080fd5b50508035926020909101359150565b60008060006060848603121561035e57600080fd5b8335925060208401359150604084013573ffffffffffffffffffffffffffffffffffffffff8116811461039057600080fd5b809150509250925092565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000806000606084860312156103df57600080fd5b8335925060208401359150604084013567ffffffffffffffff8082111561040557600080fd5b818601915086601f83011261041957600080fd5b81358181111561042b5761042b61039b565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156104715761047161039b565b8160405282815289602084870101111561048a57600080fd5b826020860160208301376000602084830101528095505050505050925092509256fe608060405234801561001057600080fd5b5061012e806100206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063249cb3fa14602d575b600080fd5b603c603836600460b1565b604e565b60405190815260200160405180910390f35b60008281526020818152604080832073ffffffffffffffffffffffffffffffffffffffff8516845290915281205460ff16608857600060aa565b7fa2ef4600d742022d532d4747cb3547474667d6f13804902513b2ec01c848f4b45b9392505050565b6000806040838503121560c357600080fd5b82359150602083013573ffffffffffffffffffffffffffffffffffffffff8116811460ed57600080fd5b80915050925092905056fea26469706673582212205ffd4e6cede7d06a5daf93d48d0541fc68189eeb16608c1999a82063b666eb1164736f6c63430008130033a2646970667358221220fdc4a0fe96e3b21c108ca155438d37c9143fb01278a3c1d274948bad89c564ba64736f6c63430008130033`.

## Safe

[Implementation](https://github.com/safe-global/safe-contracts/blob/v1.3.0/contracts/GnosisSafe.sol)

Address: `0x69f4D1788e39c87893C980c06EdF4b7f686e2938`

A multisignature wallet with support for confirmations using signed messages based on ERC191.
Differs from [SafeL2](#safel2) by not emitting events to save gas.

## SafeL2

[Implementation](https://github.com/safe-global/safe-contracts/blob/v1.3.0/contracts/GnosisSafeL2.sol)

Address: `0xfb1bffC9d739B8D520DaF37dF666da4C687191EA`

A multisignature wallet with support for confirmations using signed messages based on ERC191.
Differs from [Safe](#safe) by emitting events.

## MultiSend

[Implementation](https://github.com/safe-global/safe-contracts/blob/v1.3.0/contracts/libraries/MultiSend.sol)

Address: `0x998739BFdAAdde7C933B942a68053933098f9EDa`

Allows to batch multiple transactions into one.

## MultiSendCallOnly

[Implementation](https://github.com/safe-global/safe-contracts/blob/v1.3.0/contracts/libraries/MultiSendCallOnly.sol)

Address: `0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B`

Allows to batch multiple transactions into one, but only calls.

## SafeSingletonFactory

[Implementation](https://github.com/safe-global/safe-singleton-factory/blob/main/source/deterministic-deployment-proxy.yul)

Address: `0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7`

Singleton factory used by Safe-related contracts based on
[Arachnid's Deterministic Deployment Proxy](#arachnids-deterministic-deployment-proxy).

The original library used a pre-signed transaction without a chain ID to allow deployment on different chains.
Some chains do not allow such transactions to be submitted; therefore, this contract will provide the same factory
that can be deployed via a pre-signed transaction that includes the chain ID. The key that is used to sign is
controlled by the Safe team.

## MultiCall3

[Implementation](https://github.com/mds1/multicall/blob/main/src/Multicall3.sol)

Address: `0xcA11bde05977b3631167028862bE2a173976CA11`

`MultiCall3` has two main use cases:

- Aggregate results from multiple contract reads into a single JSON-RPC request.
- Execute multiple state-changing calls in a single transaction.

## Arachnid's Deterministic Deployment Proxy

[Implementation](https://github.com/Arachnid/deterministic-deployment-proxy/blob/master/source/deterministic-deployment-proxy.yul)

Address: `0x4e59b44847b379578588920cA78FbF26c0B4956C`

This contract can deploy other contracts with a deterministic address on any chain using `CREATE2`. The `CREATE2`
call will deploy a contract (like `CREATE` opcode) but instead of the address being
`keccak256(rlp([deployer_address, nonce]))` it instead uses the hash of the contract's bytecode and a salt.
This means that a given deployer address will deploy the
same code to the same address no matter when or where they issue the deployment. The deployer is deployed
ith a one-time-use-account, so no matter what chain the deployer is on, its address will always be the same. This
means the only variables in determining the address of your contract are its bytecode hash and the provided salt.

Between the use of `CREATE2` opcode and the one-time-use-account for the deployer, this contracts ensures
that a given contract will exist at the exact same address on every chain, but without having to use the
same gas pricing or limits every time.

## Permit2

[Implementation](https://github.com/Uniswap/permit2/blob/main/src/Permit2.sol)

Address: `0x000000000022D473030F116dDEE9F6B43aC78BA3`

Permit2 introduces a low-overhead, next-generation token approval/meta-tx system to make token approvals easier,
more secure, and more consistent across applications.

## ERC-4337 EntryPoint

[Implementation](https://github.com/eth-infinitism/account-abstraction/blob/v0.6.0/contracts/core/EntryPoint.sol)

Address: `0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789`

This contract verifies and executes the bundles of ERC-4337
[UserOperations](https://www.erc4337.io/docs/understanding-ERC-4337/user-operation) sent to it.

## ERC-4337 SenderCreator

[Implementation](https://github.com/eth-infinitism/account-abstraction/blob/v0.6.0/contracts/core/SenderCreator.sol)

Address: `0x7fc98430eaedbb6070b35b39d798725049088348`

Helper contract for [EntryPoint](#erc-4337-entrypoint), to call `userOp.initCode` from a "neutral" address,
which is explicitly not `EntryPoint` itself.
