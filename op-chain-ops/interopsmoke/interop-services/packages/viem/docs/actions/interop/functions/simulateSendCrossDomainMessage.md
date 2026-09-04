[**@eth-optimism/viem**](../../../README.md) • **Docs**

***

[@eth-optimism/viem](../../../README.md) / [actions/interop](../README.md) / simulateSendCrossDomainMessage

# simulateSendCrossDomainMessage()

> **simulateSendCrossDomainMessage**\<`TChain`, `TAccount`, `TChainOverride`\>(`client`, `parameters`): `Promise`\<[`SendCrossDomainMessageContractReturnType`](../type-aliases/SendCrossDomainMessageContractReturnType.md)\>

Simulate contract call for sendMessage

## Type Parameters

• **TChain** *extends* `undefined` \| `Chain`

• **TAccount** *extends* `undefined` \| `Account`

• **TChainOverride** *extends* `undefined` \| `Chain` = `undefined`

## Parameters

• **client**: `Client`\<`Transport`, `TChain`, `TAccount`\>

L2 Client

• **parameters**: [`SendCrossDomainMessageParameters`](../type-aliases/SendCrossDomainMessageParameters.md)\<`TChain`, `TAccount`, `TChainOverride`, `DeriveChain`\<`TChain`, `TChainOverride`\>\>

[SendCrossDomainMessageParameters](../type-aliases/SendCrossDomainMessageParameters.md)

## Returns

`Promise`\<[`SendCrossDomainMessageContractReturnType`](../type-aliases/SendCrossDomainMessageContractReturnType.md)\>

contract return value - [SendCrossDomainMessageContractReturnType](../type-aliases/SendCrossDomainMessageContractReturnType.md)

## Defined in

[packages/viem/src/actions/interop/sendCrossDomainMessage.ts:136](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/actions/interop/sendCrossDomainMessage.ts#L136)
