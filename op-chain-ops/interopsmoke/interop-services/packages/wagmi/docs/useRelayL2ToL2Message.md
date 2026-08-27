# useRelayL2ToL2Message

Relays an cross domain messenger from the L2ToL2CrossDomainMessenger

## Usage

```ts [example.ts]
const {
    relayMessage,
    isError,
    isPending,
} = useRelayL2ToL2Message()

const hash = await relayMessage({
    account: address,
    sentMessageId: id,
    sentMessagePayload: payload,
    chain: destinationChain,
})
```

## Returns

### `relayMessage`

- **Type:** `(params: RelayL2ToL2MessageParameters) => Promise<`0x${string}`>`

Call this function with the [RelayL2ToL2MessageParameters](https://github.com/ethereum-optimism/ecosystem/blob/main/packages/viem/docs/actions/relayL2ToL2Message.md#parameters) to execute a cross chain message

### `isError`

- **Type:** `boolean`

This will be true if there was an error attempting the cross chain message 

### `isPending`

- **Type:** `boolean`

This will be true while the request is in flight

### `isSuccess`

- **Type:** `boolean`

This will be true after the request has returned successfully

