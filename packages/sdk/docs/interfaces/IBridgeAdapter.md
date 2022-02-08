[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / IBridgeAdapter

# Interface: IBridgeAdapter

Represents an adapter for an L1<>L2 token bridge. Each custom bridge currently needs its own
adapter because the bridge interface is not standardized. This may change in the future.

## Implemented by

- [`StandardBridgeAdapter`](../classes/StandardBridgeAdapter.md)

## Table of contents

### Properties

- [estimateGas](IBridgeAdapter.md#estimategas)
- [l1Bridge](IBridgeAdapter.md#l1bridge)
- [l2Bridge](IBridgeAdapter.md#l2bridge)
- [messenger](IBridgeAdapter.md#messenger)
- [populateTransaction](IBridgeAdapter.md#populatetransaction)

### Methods

- [approval](IBridgeAdapter.md#approval)
- [approve](IBridgeAdapter.md#approve)
- [deposit](IBridgeAdapter.md#deposit)
- [getDepositsByAddress](IBridgeAdapter.md#getdepositsbyaddress)
- [getWithdrawalsByAddress](IBridgeAdapter.md#getwithdrawalsbyaddress)
- [supportsTokenPair](IBridgeAdapter.md#supportstokenpair)
- [withdraw](IBridgeAdapter.md#withdraw)

## Properties

### estimateGas

• **estimateGas**: `Object`

Object that holds the functions that estimates the gas required for a given transaction.
Follows the pattern used by ethers.js.

#### Type declaration

| Name | Type |
| :------ | :------ |
| `approve` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`BigNumber`\> |
| `deposit` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |
| `withdraw` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:237](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L237)

___

### l1Bridge

• **l1Bridge**: `Contract`

L1 bridge contract.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:24](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L24)

___

### l2Bridge

• **l2Bridge**: `Contract`

L2 bridge contract.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:29](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L29)

___

### messenger

• **messenger**: [`ICrossChainMessenger`](ICrossChainMessenger.md)

Provider used to make queries related to cross-chain interactions.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:19](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L19)

___

### populateTransaction

• **populateTransaction**: `Object`

Object that holds the functions that generate transactions to be signed by the user.
Follows the pattern used by ethers.js.

#### Type declaration

| Name | Type |
| :------ | :------ |
| `approve` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`TransactionRequest`\> |
| `deposit` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |
| `withdraw` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:168](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L168)

## Methods

### approval

▸ **approval**(`l1Token`, `l2Token`, `signer`): `Promise`<`BigNumber`\>

Queries the account's approval amount for a given L1 token.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |
| `signer` | `Signer` | Signer to query the approval for. |

#### Returns

`Promise`<`BigNumber`\>

Amount of tokens approved for deposits from the account.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:89](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L89)

___

### approve

▸ **approve**(`l1Token`, `l2Token`, `amount`, `signer`, `opts?`): `Promise`<`TransactionResponse`\>

Approves a deposit into the L2 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount of the token to approve. |
| `signer` | `Signer` | Signer used to sign and send the transaction. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the approval transaction.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:106](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L106)

___

### deposit

▸ **deposit**(`l1Token`, `l2Token`, `amount`, `signer`, `opts?`): `Promise`<`TransactionResponse`\>

Deposits some tokens into the L2 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount of the token to deposit. |
| `signer` | `Signer` | Signer used to sign and send the transaction. |
| `opts?` | `Object` | Additional options. |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) | Optional gas limit to use for the transaction on L2. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) | Optional address to receive the funds on L2. Defaults to sender. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the deposit transaction.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:129](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L129)

___

### getDepositsByAddress

▸ **getDepositsByAddress**(`address`, `opts?`): `Promise`<[`TokenBridgeMessage`](TokenBridgeMessage.md)[]\>

Gets all deposits for a given address.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `address` | [`AddressLike`](../modules.md#addresslike) | Address to search for messages from. |
| `opts?` | `Object` | Options object. |
| `opts.fromBlock?` | `BlockTag` | Block to start searching for messages from. If not provided, will start from the first block (block #0). |
| `opts.toBlock?` | `BlockTag` | Block to stop searching for messages at. If not provided, will stop at the latest known block ("latest"). |

#### Returns

`Promise`<[`TokenBridgeMessage`](TokenBridgeMessage.md)[]\>

All deposit token bridge messages sent by the given address.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:42](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L42)

___

### getWithdrawalsByAddress

▸ **getWithdrawalsByAddress**(`address`, `opts?`): `Promise`<[`TokenBridgeMessage`](TokenBridgeMessage.md)[]\>

Gets all withdrawals for a given address.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `address` | [`AddressLike`](../modules.md#addresslike) | Address to search for messages from. |
| `opts?` | `Object` | Options object. |
| `opts.fromBlock?` | `BlockTag` | Block to start searching for messages from. If not provided, will start from the first block (block #0). |
| `opts.toBlock?` | `BlockTag` | Block to stop searching for messages at. If not provided, will stop at the latest known block ("latest"). |

#### Returns

`Promise`<[`TokenBridgeMessage`](TokenBridgeMessage.md)[]\>

All withdrawal token bridge messages sent by the given address.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:61](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L61)

___

### supportsTokenPair

▸ **supportsTokenPair**(`l1Token`, `l2Token`): `Promise`<`boolean`\>

Checks whether the given token pair is supported by the bridge.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |

#### Returns

`Promise`<`boolean`\>

Whether the given token pair is supported by the bridge.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:76](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L76)

___

### withdraw

▸ **withdraw**(`l1Token`, `l2Token`, `amount`, `signer`, `opts?`): `Promise`<`TransactionResponse`\>

Withdraws some tokens back to the L1 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount of the token to withdraw. |
| `signer` | `Signer` | Signer used to sign and send the transaction. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) | Optional address to receive the funds on L1. Defaults to sender. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the withdraw transaction.

#### Defined in

[packages/sdk/src/interfaces/bridge-adapter.ts:153](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/bridge-adapter.ts#L153)
