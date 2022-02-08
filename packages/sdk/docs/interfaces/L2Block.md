[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / L2Block

# Interface: L2Block

JSON block representation when returned by L2Geth nodes. Just a normal block but with
an added stateRoot field.

## Hierarchy

- `Block`

  ↳ **`L2Block`**

## Table of contents

### Properties

- [\_difficulty](L2Block.md#_difficulty)
- [baseFeePerGas](L2Block.md#basefeepergas)
- [difficulty](L2Block.md#difficulty)
- [extraData](L2Block.md#extradata)
- [gasLimit](L2Block.md#gaslimit)
- [gasUsed](L2Block.md#gasused)
- [hash](L2Block.md#hash)
- [miner](L2Block.md#miner)
- [nonce](L2Block.md#nonce)
- [number](L2Block.md#number)
- [parentHash](L2Block.md#parenthash)
- [stateRoot](L2Block.md#stateroot)
- [timestamp](L2Block.md#timestamp)
- [transactions](L2Block.md#transactions)

## Properties

### \_difficulty

• **\_difficulty**: `BigNumber`

#### Inherited from

Block.\_difficulty

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:40

___

### baseFeePerGas

• `Optional` **baseFeePerGas**: `BigNumber`

#### Inherited from

Block.baseFeePerGas

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:45

___

### difficulty

• **difficulty**: `number`

#### Inherited from

Block.difficulty

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:39

___

### extraData

• **extraData**: `string`

#### Inherited from

Block.extraData

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:44

___

### gasLimit

• **gasLimit**: `BigNumber`

#### Inherited from

Block.gasLimit

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:41

___

### gasUsed

• **gasUsed**: `BigNumber`

#### Inherited from

Block.gasUsed

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:42

___

### hash

• **hash**: `string`

#### Inherited from

Block.hash

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:34

___

### miner

• **miner**: `string`

#### Inherited from

Block.miner

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:43

___

### nonce

• **nonce**: `string`

#### Inherited from

Block.nonce

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:38

___

### number

• **number**: `number`

#### Inherited from

Block.number

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:36

___

### parentHash

• **parentHash**: `string`

#### Inherited from

Block.parentHash

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:35

___

### stateRoot

• **stateRoot**: `string`

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:27](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L27)

___

### timestamp

• **timestamp**: `number`

#### Inherited from

Block.timestamp

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:37

___

### transactions

• **transactions**: `string`[]

#### Inherited from

Block.transactions

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:48
