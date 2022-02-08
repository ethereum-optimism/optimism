[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / L2Transaction

# Interface: L2Transaction

JSON transaction representation when returned by L2Geth nodes. This is simply an extension to
the standard transaction response type. You do NOT need to use this type unless you care about
having typed access to L2-specific fields.

## Hierarchy

- `TransactionResponse`

  ↳ **`L2Transaction`**

## Table of contents

### Properties

- [accessList](L2Transaction.md#accesslist)
- [blockHash](L2Transaction.md#blockhash)
- [blockNumber](L2Transaction.md#blocknumber)
- [chainId](L2Transaction.md#chainid)
- [confirmations](L2Transaction.md#confirmations)
- [data](L2Transaction.md#data)
- [from](L2Transaction.md#from)
- [gasLimit](L2Transaction.md#gaslimit)
- [gasPrice](L2Transaction.md#gasprice)
- [hash](L2Transaction.md#hash)
- [l1BlockNumber](L2Transaction.md#l1blocknumber)
- [l1TxOrigin](L2Transaction.md#l1txorigin)
- [maxFeePerGas](L2Transaction.md#maxfeepergas)
- [maxPriorityFeePerGas](L2Transaction.md#maxpriorityfeepergas)
- [nonce](L2Transaction.md#nonce)
- [queueOrigin](L2Transaction.md#queueorigin)
- [r](L2Transaction.md#r)
- [raw](L2Transaction.md#raw)
- [rawTransaction](L2Transaction.md#rawtransaction)
- [s](L2Transaction.md#s)
- [timestamp](L2Transaction.md#timestamp)
- [to](L2Transaction.md#to)
- [type](L2Transaction.md#type)
- [v](L2Transaction.md#v)
- [value](L2Transaction.md#value)

### Methods

- [wait](L2Transaction.md#wait)

## Properties

### accessList

• `Optional` **accessList**: `AccessList`

#### Inherited from

TransactionResponse.accessList

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:40

___

### blockHash

• `Optional` **blockHash**: `string`

#### Inherited from

TransactionResponse.blockHash

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:25

___

### blockNumber

• `Optional` **blockNumber**: `number`

#### Inherited from

TransactionResponse.blockNumber

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:24

___

### chainId

• **chainId**: `number`

#### Inherited from

TransactionResponse.chainId

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:35

___

### confirmations

• **confirmations**: `number`

#### Inherited from

TransactionResponse.confirmations

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:27

___

### data

• **data**: `string`

#### Inherited from

TransactionResponse.data

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:33

___

### from

• **from**: `string`

#### Inherited from

TransactionResponse.from

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:28

___

### gasLimit

• **gasLimit**: `BigNumber`

#### Inherited from

TransactionResponse.gasLimit

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:31

___

### gasPrice

• `Optional` **gasPrice**: `BigNumber`

#### Inherited from

TransactionResponse.gasPrice

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:32

___

### hash

• **hash**: `string`

#### Inherited from

TransactionResponse.hash

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:23

___

### l1BlockNumber

• **l1BlockNumber**: `number`

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:16](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L16)

___

### l1TxOrigin

• **l1TxOrigin**: `string`

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:17](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L17)

___

### maxFeePerGas

• `Optional` **maxFeePerGas**: `BigNumber`

#### Inherited from

TransactionResponse.maxFeePerGas

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:42

___

### maxPriorityFeePerGas

• `Optional` **maxPriorityFeePerGas**: `BigNumber`

#### Inherited from

TransactionResponse.maxPriorityFeePerGas

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:41

___

### nonce

• **nonce**: `number`

#### Inherited from

TransactionResponse.nonce

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:30

___

### queueOrigin

• **queueOrigin**: `string`

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:18](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L18)

___

### r

• `Optional` **r**: `string`

#### Inherited from

TransactionResponse.r

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:36

___

### raw

• `Optional` **raw**: `string`

#### Inherited from

TransactionResponse.raw

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:29

___

### rawTransaction

• **rawTransaction**: `string`

#### Defined in

[packages/sdk/src/interfaces/l2-provider.ts:19](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/l2-provider.ts#L19)

___

### s

• `Optional` **s**: `string`

#### Inherited from

TransactionResponse.s

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:37

___

### timestamp

• `Optional` **timestamp**: `number`

#### Inherited from

TransactionResponse.timestamp

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:26

___

### to

• `Optional` **to**: `string`

#### Inherited from

TransactionResponse.to

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:28

___

### type

• `Optional` **type**: `number`

#### Inherited from

TransactionResponse.type

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:39

___

### v

• `Optional` **v**: `number`

#### Inherited from

TransactionResponse.v

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:38

___

### value

• **value**: `BigNumber`

#### Inherited from

TransactionResponse.value

#### Defined in

node_modules/@ethersproject/abstract-provider/node_modules/@ethersproject/transactions/lib/index.d.ts:34

## Methods

### wait

▸ **wait**(`confirmations?`): `Promise`<`TransactionReceipt`\>

#### Parameters

| Name | Type |
| :------ | :------ |
| `confirmations?` | `number` |

#### Returns

`Promise`<`TransactionReceipt`\>

#### Inherited from

TransactionResponse.wait

#### Defined in

node_modules/@ethersproject/abstract-provider/lib/index.d.ts:30
