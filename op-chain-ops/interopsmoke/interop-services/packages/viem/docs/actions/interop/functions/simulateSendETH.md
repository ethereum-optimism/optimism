[**@eth-optimism/viem**](../../../README.md) • **Docs**

***

[@eth-optimism/viem](../../../README.md) / [actions/interop](../README.md) / simulateSendETH

# simulateSendETH()

> **simulateSendETH**\<`TChain`, `TAccount`, `TChainOverride`\>(`client`, `parameters`): `Promise`\<[`SendETHContractReturnType`](../type-aliases/SendETHContractReturnType.md)\>

Simulate contract call for [sendETH](sendETH.md)

## Type Parameters

• **TChain** *extends* `undefined` \| `Chain`

• **TAccount** *extends* `undefined` \| `Account`

• **TChainOverride** *extends* `undefined` \| `Chain` = `undefined`

## Parameters

• **client**: `Client`\<`Transport`, `TChain`, `TAccount`\>

L2 Client

• **parameters**: [`SendETHParameters`](../type-aliases/SendETHParameters.md)\<`TChain`, `TAccount`, `TChainOverride`, `DeriveChain`\<`TChain`, `TChainOverride`\>\>

[SendETHParameters](../type-aliases/SendETHParameters.md)

## Returns

`Promise`\<[`SendETHContractReturnType`](../type-aliases/SendETHContractReturnType.md)\>

contract return value - [SendETHContractReturnType](../type-aliases/SendETHContractReturnType.md)

## Defined in

[packages/viem/src/actions/interop/sendETH.ts:122](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/actions/interop/sendETH.ts#L122)
