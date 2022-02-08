[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / CrossChainMessage

# Interface: CrossChainMessage

Describes a message that is sent between L1 and L2. Direction determines where the message was
sent from and where it's being sent to.

## Hierarchy

- [`CoreCrossChainMessage`](CoreCrossChainMessage.md)

  ↳ **`CrossChainMessage`**

## Table of contents

### Properties

- [blockNumber](CrossChainMessage.md#blocknumber)
- [direction](CrossChainMessage.md#direction)
- [gasLimit](CrossChainMessage.md#gaslimit)
- [logIndex](CrossChainMessage.md#logindex)
- [message](CrossChainMessage.md#message)
- [messageNonce](CrossChainMessage.md#messagenonce)
- [sender](CrossChainMessage.md#sender)
- [target](CrossChainMessage.md#target)
- [transactionHash](CrossChainMessage.md#transactionhash)

## Properties

### blockNumber

• **blockNumber**: `number`

#### Defined in

[packages/sdk/src/interfaces/types.ts:167](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L167)

___

### direction

• **direction**: [`MessageDirection`](../enums/MessageDirection.md)

#### Defined in

[packages/sdk/src/interfaces/types.ts:164](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L164)

___

### gasLimit

• **gasLimit**: `number`

#### Defined in

[packages/sdk/src/interfaces/types.ts:165](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L165)

___

### logIndex

• **logIndex**: `number`

#### Defined in

[packages/sdk/src/interfaces/types.ts:166](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L166)

___

### message

• **message**: `string`

#### Inherited from

[CoreCrossChainMessage](CoreCrossChainMessage.md).[message](CoreCrossChainMessage.md#message)

#### Defined in

[packages/sdk/src/interfaces/types.ts:155](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L155)

___

### messageNonce

• **messageNonce**: `number`

#### Inherited from

[CoreCrossChainMessage](CoreCrossChainMessage.md).[messageNonce](CoreCrossChainMessage.md#messagenonce)

#### Defined in

[packages/sdk/src/interfaces/types.ts:156](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L156)

___

### sender

• **sender**: `string`

#### Inherited from

[CoreCrossChainMessage](CoreCrossChainMessage.md).[sender](CoreCrossChainMessage.md#sender)

#### Defined in

[packages/sdk/src/interfaces/types.ts:153](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L153)

___

### target

• **target**: `string`

#### Inherited from

[CoreCrossChainMessage](CoreCrossChainMessage.md).[target](CoreCrossChainMessage.md#target)

#### Defined in

[packages/sdk/src/interfaces/types.ts:154](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L154)

___

### transactionHash

• **transactionHash**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:168](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L168)
