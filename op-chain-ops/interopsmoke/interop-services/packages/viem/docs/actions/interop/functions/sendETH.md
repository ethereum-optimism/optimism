[**@eth-optimism/viem**](../../../README.md) • **Docs**

***

[@eth-optimism/viem](../../../README.md) / [actions/interop](../README.md) / sendETH

# sendETH()

> **sendETH**\<`chain`, `account`, `chainOverride`\>(`client`, `parameters`): `Promise`\<[`SendETHContractReturnType`](../type-aliases/SendETHContractReturnType.md)\>

Sends ETH to the specified recipient on the destination chain

## Type Parameters

• **chain** *extends* `undefined` \| `Chain`

• **account** *extends* `undefined` \| `Account`

• **chainOverride** *extends* `undefined` \| `Chain` = `undefined`

## Parameters

• **client**: `Client`\<`Transport`, `chain`, `account`\>

L2 Client

• **parameters**: [`SendETHParameters`](../type-aliases/SendETHParameters.md)\<`chain`, `account`, `chainOverride`, `DeriveChain`\<`chain`, `chainOverride`\>\>

[SendETHParameters](../type-aliases/SendETHParameters.md)

## Returns

`Promise`\<[`SendETHContractReturnType`](../type-aliases/SendETHContractReturnType.md)\>

transaction hash - [SendETHContractReturnType](../type-aliases/SendETHContractReturnType.md)

## Defined in

[packages/viem/src/actions/interop/sendETH.ts:67](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/actions/interop/sendETH.ts#L67)
