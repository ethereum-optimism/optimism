[**@eth-optimism/viem**](../../../README.md) • **Docs**

***

[@eth-optimism/viem](../../../README.md) / [actions/interop](../README.md) / getCrossDomainMessages

# getCrossDomainMessages()

> **getCrossDomainMessages**\<`TChain`, `TAccount`\>(`client`, `parameters`): `Promise`\<[`GetCrossDomainMessagesReturnType`](../type-aliases/GetCrossDomainMessagesReturnType.md)\>

Get all cross domain messages from a set of logs

## Type Parameters

• **TChain** *extends* `undefined` \| `Chain`

• **TAccount** *extends* `undefined` \| `Account`

## Parameters

• **client**: `Client`\<`Transport`, `TChain`, `TAccount`\>

The client to use

• **parameters**: [`GetCrossDomainMessagesParameters`](../type-aliases/GetCrossDomainMessagesParameters.md)

[GetCrossDomainMessagesParameters](../type-aliases/GetCrossDomainMessagesParameters.md)

## Returns

`Promise`\<[`GetCrossDomainMessagesReturnType`](../type-aliases/GetCrossDomainMessagesReturnType.md)\>

cross domain messages - [GetCrossDomainMessagesReturnType](../type-aliases/GetCrossDomainMessagesReturnType.md)

## Example

```ts
import { createPublicClient } from 'viem'
import { http } from 'viem/transports'
import { op } from '@eth-optimism/viem/chains'
import { getCrossDomainMessages } from '@eth-optimism/viem/actions/interop'

const publicClientOp = createPublicClient({ chain: op, transport: http() })
const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
const messages = await getCrossDomainMessages(publicClientOp, { logs: receipt.logs })
```

## Defined in

[packages/viem/src/actions/interop/getCrossDomainMessages.ts:33](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/actions/interop/getCrossDomainMessages.ts#L33)
