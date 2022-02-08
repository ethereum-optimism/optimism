[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / CrossChainMessageProof

# Interface: CrossChainMessageProof

Proof data required to finalize an L2 to L1 message.

## Table of contents

### Properties

- [stateRoot](CrossChainMessageProof.md#stateroot)
- [stateRootBatchHeader](CrossChainMessageProof.md#staterootbatchheader)
- [stateRootProof](CrossChainMessageProof.md#staterootproof)
- [stateTrieWitness](CrossChainMessageProof.md#statetriewitness)
- [storageTrieWitness](CrossChainMessageProof.md#storagetriewitness)

## Properties

### stateRoot

• **stateRoot**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:237](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L237)

___

### stateRootBatchHeader

• **stateRootBatchHeader**: [`StateRootBatchHeader`](StateRootBatchHeader.md)

#### Defined in

[packages/sdk/src/interfaces/types.ts:238](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L238)

___

### stateRootProof

• **stateRootProof**: `Object`

#### Type declaration

| Name | Type |
| :------ | :------ |
| `index` | `number` |
| `siblings` | `string`[] |

#### Defined in

[packages/sdk/src/interfaces/types.ts:239](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L239)

___

### stateTrieWitness

• **stateTrieWitness**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:243](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L243)

___

### storageTrieWitness

• **storageTrieWitness**: `string`

#### Defined in

[packages/sdk/src/interfaces/types.ts:244](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L244)
