# Preinstalls

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Safe](#safe)
- [SafeL2](#safel2)
- [MultiSend](#multisend)
- [MultiSendCallOnly](#multisendcallonly)
- [SafeSingletonFactory](#safesingletonfactory)
- [Multicall3](#multicall3)
- [Arachnid's Deterministic Deployment Proxy](#arachnids-deterministic-deployment-proxy)
- [Permit2](#permit2)
- [ERC-4337 EntryPoint](#erc-4337-entrypoint)
- [ERC-4337 SenderCreator](#erc-4337-sendercreator)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

[Preinstalled smart contracts](./glossary.md#preinstalled-contract-preinstall) exist on Optimism
at predetermined addresses in the genesis state. They are similar to precompiles but instead run
directly in the EVM instead of running native code outside of the EVM and are developed by third
parties unaffiliated with the Optimism Collective.

These preinstalls are commonly deployed smart contracts that are being placed at genesis for convenience.
It's important to note that these contracts do not have the same security guarantees
as [Predeployed smart contracts](./glossary.md#predeployed-contract-predeploy).

The following table includes each of the preinstalls.

| Name                                      | Address                                    |
| ----------------------------------------- | ------------------------------------------ |
| Safe                                      | 0x69f4D1788e39c87893C980c06EdF4b7f686e2938 |
| SafeL2                                    | 0xfb1bffC9d739B8D520DaF37dF666da4C687191EA |
| MultiSend                                 | 0x998739BFdAAdde7C933B942a68053933098f9EDa |
| MultiSendCallOnly                         | 0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B |
| SafeSingletonFactory                      | 0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7 |
| Multicall3                                | 0xcA11bde05977b3631167028862bE2a173976CA11 |
| Arachnid's Deterministic Deployment Proxy | 0x4e59b44847b379578588920cA78FbF26c0B4956C |
| Permit2                                   | 0x000000000022D473030F116dDEE9F6B43aC78BA3 |
| ERC-4337 EntryPoint                       | 0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789 |
| ERC-4337 SenderCreator                    | 0x7fc98430eaedbb6070b35b39d798725049088348 |

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

[Implementation](https://github.com/safe-global/safe-singleton-factory/blob/v1.0.17/source/deterministic-deployment-proxy.yul)

Address: `0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7`

Singleton factory used by Safe-related contracts based on
[Arachnid's Deterministic Deployment Proxy](#arachnids-deterministic-deployment-proxy).

The original library used a pre-signed transaction without a chain ID to allow deployment on different chains.
Some chains do not allow such transactions to be submitted; therefore, this contract will provide the same factory
that can be deployed via a pre-signed transaction that includes the chain ID. The key that is used to sign is
controlled by the Safe team.

## Multicall3

[Implementation](https://github.com/mds1/multicall/blob/v3.1.0/src/Multicall3.sol)

Address: `0xcA11bde05977b3631167028862bE2a173976CA11`

`Multicall3` has two main use cases:

- Aggregate results from multiple contract reads into a single JSON-RPC request.
- Execute multiple state-changing calls in a single transaction.

## Arachnid's Deterministic Deployment Proxy

[Implementation](https://github.com/Arachnid/deterministic-deployment-proxy/blob/v1.0.0/source/deterministic-deployment-proxy.yul)

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

[Implementation](https://github.com/Uniswap/permit2/blob/0x000000000022D473030F116dDEE9F6B43aC78BA3/src/Permit2.sol)

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
