[**@eth-optimism/viem**](../../../README.md) • **Docs**

***

[@eth-optimism/viem](../../../README.md) / [actions/interop](../README.md) / buildExecutingMessage

# buildExecutingMessage()

> **buildExecutingMessage**\<`TChain`, `TAccount`\>(`client`, `params`): `Promise`\<[`BuildExecutingMessageReturnType`](../type-aliases/BuildExecutingMessageReturnType.md)\>

Build an executing message from a log

## Type Parameters

• **TChain** *extends* `undefined` \| `Chain`

• **TAccount** *extends* `undefined` \| `Account`

## Parameters

• **client**: `Client`\<`Transport`, `TChain`, `TAccount`\>

client to the chain that emitted the log

• **params**: [`BuildExecutingMessageParameters`](../type-aliases/BuildExecutingMessageParameters.md)

[BuildExecutingMessageParameters](../type-aliases/BuildExecutingMessageParameters.md)

## Returns

`Promise`\<[`BuildExecutingMessageReturnType`](../type-aliases/BuildExecutingMessageReturnType.md)\>

- [BuildExecutingMessageReturnType](../type-aliases/BuildExecutingMessageReturnType.md)

## Example

```ts
import { createPublicClient } from 'viem'
import { http } from 'viem/transports'
import { op } from '@eth-optimism/viem/chains'

const publicClientOp = createPublicClient({ chain: op, transport: http() })
const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
const params = await buildExecutingMessage(publicClientOp, { log: receipt.logs[0] })
```

## Defined in

[packages/viem/src/actions/interop/buildExecutingMessage.ts:58](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/actions/interop/buildExecutingMessage.ts#L58)
