[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / TokenBridgeMessage

# Interface: TokenBridgeMessage

Describes a token withdrawal or deposit, along with the underlying raw cross chain message
behind the deposit or withdrawal.

## Table of contents

### Properties

- [amount](TokenBridgeMessage.md#amount)
- [blockNumber](TokenBridgeMessage.md#blocknumber)
- [data](TokenBridgeMessage.md#data)
- [direction](TokenBridgeMessage.md#direction)
- [from](TokenBridgeMessage.md#from)
- [l1Token](TokenBridgeMessage.md#l1token)
- [l2Token](TokenBridgeMessage.md#l2token)
- [logIndex](TokenBridgeMessage.md#logindex)
- [to](TokenBridgeMessage.md#to)
- [transactionHash](TokenBridgeMessage.md#transactionhash)

## Properties

### amount

• **amount**: `BigNumber`

#### Defined in

[packages/sdk/src/interfaces/types.ts:181](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L181)

___

### blockNumber

• **blockNumber**: `number`

#### Defined in

[packages/sdk/src/interfaces/types.ts:184](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L184)

___

### data

• **data**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:182](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L182)

___

### direction

• **direction**: [`MessageDirection`](../enums/MessageDirection.md)

#### Defined in

[packages/sdk/src/interfaces/types.ts:176](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L176)

___

### from

• **from**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:177](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L177)

___

### l1Token

• **l1Token**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:179](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L179)

___

### l2Token

• **l2Token**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:180](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L180)

___

### logIndex

• **logIndex**: `number`

#### Defined in

[packages/sdk/src/interfaces/types.ts:183](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L183)

___

### to

• **to**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:178](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L178)

___

### transactionHash

• **transactionHash**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:185](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L185)
