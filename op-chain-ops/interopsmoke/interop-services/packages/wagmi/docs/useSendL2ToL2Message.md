# useSendL2ToL2Message

Initiates an L2 to L2 message. Used in the interop flow.

## Usage

```ts [example.ts]
const {
    sendMessage,
    isError,
    isPending,
} = useSendL2ToL2Message()

const hash = await sendMessage({
    account: address,
    destinationChainId: optimism.id,
    target: contractAddress,
    message: encodedMessage,
    chain: originChain,
})
```

## Returns

### `sendMessage`

- **Type:** `(params: SendL2ToL2MessageParameters) => Promise<`0x${string}`>`

Call this function with the [SendL2ToL2MessageParameters](https://github.com/ethereum-optimism/ecosystem/blob/main/packages/viem/docs/actions/sendL2ToL2Message.md#parameters) to initiate a cross chain message

### `isError`

- **Type:** `boolean`

This will be true if there was an error attempting the cross chain message 

### `isPending`

- **Type:** `boolean`

This will be true while the request is in flight

### `isSuccess`

- **Type:** `boolean`

This will be true after the request has returned successfully

