[**@eth-optimism/viem**](../../../README.md) • **Docs**

***

[@eth-optimism/viem](../../../README.md) / [actions/interop](../README.md) / relayCrossDomainMessage

# relayCrossDomainMessage()

> **relayCrossDomainMessage**\<`TChain`, `TAccount`, `TChainOverride`\>(`client`, `parameters`): `Promise`\<[`RelayCrossDomainMessageReturnType`](../type-aliases/RelayCrossDomainMessageReturnType.md)\>

Relays a message emitted by the CrossDomainMessenger

## Type Parameters

• **TChain** *extends* `undefined` \| `Chain`

• **TAccount** *extends* `undefined` \| `Account`

• **TChainOverride** *extends* `undefined` \| `Chain` = `undefined`

## Parameters

• **client**: `Client`\<`Transport`, `TChain`, `TAccount`\>

L2 Client

• **parameters**: [`RelayCrossDomainMessageParameters`](../type-aliases/RelayCrossDomainMessageParameters.md)\<`TChain`, `TAccount`, `TChainOverride`, `DeriveChain`\<`TChain`, `TChainOverride`\>\>

[RelayCrossDomainMessageParameters](../type-aliases/RelayCrossDomainMessageParameters.md)

## Returns

`Promise`\<[`RelayCrossDomainMessageReturnType`](../type-aliases/RelayCrossDomainMessageReturnType.md)\>

transaction hash - [RelayCrossDomainMessageReturnType](../type-aliases/RelayCrossDomainMessageReturnType.md)

## Example

```ts
import { createPublicClient } from 'viem'
import { http } from 'viem/transports'
import { op, unichain } from '@eth-optimism/viem/chains'

const publicClientOp = createPublicClient({ chain: op, transport: http() })
const publicClientUnichain = createPublicClient({ chain: unichain, transport: http() })

const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
const messages = await getCrossDomainMessages(publicClientOp, { logs: receipt.logs })

const message = messages.filter((message) => message.destination === unichain.id)[0]
const params = await buildExecutingMessage(publicClientOp, { log: message.log })

const hash = await relayCrossDomainMessage(publicClientUnichain, params)
```

## Defined in

[packages/viem/src/actions/interop/relayCrossDomainMessage.ts:87](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/actions/interop/relayCrossDomainMessage.ts#L87)
