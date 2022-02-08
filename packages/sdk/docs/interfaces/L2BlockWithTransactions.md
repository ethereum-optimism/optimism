[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / L2BlockWithTransactions

# Interface: L2BlockWithTransactions

JSON block representation when returned by L2Geth nodes. Just a normal block but with
L2Transaction objects instead of the standard transaction response object.

## Hierarchy

- `BlockWithTransactions`

  ↳ **`L2BlockWithTransactions`**

## Table of contents

### Properties

- [\_difficulty](L2BlockWithTransactions.md#_difficulty)
- [baseFeePerGas](L2BlockWithTransactions.md#basefeepergas)
- [difficulty](L2BlockWithTransactions.md#difficulty)
- [extraData](L2BlockWithTransactions.md#extradata)
- [gasLimit](L2BlockWithTransactions.md#gaslimit)
- [gasUsed](L2BlockWithTransactions.md#gasused)
- [hash](L2BlockWithTransactions.md#hash)
- [miner](L2BlockWithTransactions.md#miner)
- [nonce](L2BlockWithTransactions.md#nonce)
- [number](L2BlockWithTransactions.md#number)
- [parentHash](L2BlockWithTransactions.md#parenthash)
- [stateRoot](L2BlockWithTransactions.md#stateroot)
- [timestamp](L2BlockWithTransactions.md#timestamp)
- [transactions](L2BlockWithTransactions.md#transactions)

## Properties

### \_difficulty

• **\_difficulty**: `BigNumber`

#### Inherited from

BlockWithTransactions.\_difficulty

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:40

___

### baseFeePerGas

• `Optional` **baseFeePerGas**: `BigNumber`

#### Inherited from

BlockWithTransactions.baseFeePerGas

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:45

___

### difficulty

• **difficulty**: `number`

#### Inherited from

BlockWithTransactions.difficulty

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:39

___

### extraData

• **extraData**: `string`

#### Inherited from

BlockWithTransactions.extraData

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:44

___

### gasLimit

• **gasLimit**: `BigNumber`

#### Inherited from

BlockWithTransactions.gasLimit

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:41

___

### gasUsed

• **gasUsed**: `BigNumber`

#### Inherited from

BlockWithTransactions.gasUsed

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:42

___

### hash

• **hash**: `string`

#### Inherited from

BlockWithTransactions.hash

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:34

___

### miner

• **miner**: `string`

#### Inherited from

BlockWithTransactions.miner

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:43

___

### nonce

• **nonce**: `string`

#### Inherited from

BlockWithTransactions.nonce

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:38

___

### number

• **number**: `number`

#### Inherited from

BlockWithTransactions.number

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:36

___

### parentHash

• **parentHash**: `string`

#### Inherited from

BlockWithTransactions.parentHash

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:35

___

### stateRoot

• **stateRoot**: `string`

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:35](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L35)

___

### timestamp

• **timestamp**: `number`

#### Inherited from

BlockWithTransactions.timestamp

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:37

___

### transactions

• **transactions**: [[`L2Transaction`](L2Transaction.md)]

#### Overrides

BlockWithTransactions.transactions

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:36](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L36)
