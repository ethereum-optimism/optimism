[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / CrossChainMessenger

# Class: CrossChainMessenger

## Implements

- [`ICrossChainMessenger`](../interfaces/ICrossChainMessenger.md)

## Table of contents

### Constructors

- [constructor](CrossChainMessenger.md#constructor)

### Properties

- [bridges](CrossChainMessenger.md#bridges)
- [contracts](CrossChainMessenger.md#contracts)
- [depositConfirmationBlocks](CrossChainMessenger.md#depositconfirmationblocks)
- [estimateGas](CrossChainMessenger.md#estimategas)
- [l1BlockTimeSeconds](CrossChainMessenger.md#l1blocktimeseconds)
- [l1ChainId](CrossChainMessenger.md#l1chainid)
- [l1SignerOrProvider](CrossChainMessenger.md#l1signerorprovider)
- [l2SignerOrProvider](CrossChainMessenger.md#l2signerorprovider)
- [populateTransaction](CrossChainMessenger.md#populatetransaction)

### Accessors

- [l1Provider](CrossChainMessenger.md#l1provider)
- [l1Signer](CrossChainMessenger.md#l1signer)
- [l2Provider](CrossChainMessenger.md#l2provider)
- [l2Signer](CrossChainMessenger.md#l2signer)

### Methods

- [approval](CrossChainMessenger.md#approval)
- [approveERC20](CrossChainMessenger.md#approveerc20)
- [depositERC20](CrossChainMessenger.md#depositerc20)
- [depositETH](CrossChainMessenger.md#depositeth)
- [estimateL2MessageGasLimit](CrossChainMessenger.md#estimatel2messagegaslimit)
- [estimateMessageWaitTimeSeconds](CrossChainMessenger.md#estimatemessagewaittimeseconds)
- [finalizeMessage](CrossChainMessenger.md#finalizemessage)
- [getBridgeForTokenPair](CrossChainMessenger.md#getbridgefortokenpair)
- [getChallengePeriodSeconds](CrossChainMessenger.md#getchallengeperiodseconds)
- [getDepositsByAddress](CrossChainMessenger.md#getdepositsbyaddress)
- [getMessageProof](CrossChainMessenger.md#getmessageproof)
- [getMessageReceipt](CrossChainMessenger.md#getmessagereceipt)
- [getMessageStateRoot](CrossChainMessenger.md#getmessagestateroot)
- [getMessageStatus](CrossChainMessenger.md#getmessagestatus)
- [getMessagesByAddress](CrossChainMessenger.md#getmessagesbyaddress)
- [getMessagesByTransaction](CrossChainMessenger.md#getmessagesbytransaction)
- [getStateBatchAppendedEventByBatchIndex](CrossChainMessenger.md#getstatebatchappendedeventbybatchindex)
- [getStateBatchAppendedEventByTransactionIndex](CrossChainMessenger.md#getstatebatchappendedeventbytransactionindex)
- [getStateRootBatchByTransactionIndex](CrossChainMessenger.md#getstaterootbatchbytransactionindex)
- [getWithdrawalsByAddress](CrossChainMessenger.md#getwithdrawalsbyaddress)
- [resendMessage](CrossChainMessenger.md#resendmessage)
- [sendMessage](CrossChainMessenger.md#sendmessage)
- [toCrossChainMessage](CrossChainMessenger.md#tocrosschainmessage)
- [waitForMessageReceipt](CrossChainMessenger.md#waitformessagereceipt)
- [waitForMessageStatus](CrossChainMessenger.md#waitformessagestatus)
- [withdrawERC20](CrossChainMessenger.md#withdrawerc20)
- [withdrawETH](CrossChainMessenger.md#withdraweth)

## Constructors

### constructor

• **new CrossChainMessenger**(`opts`)

Creates a new CrossChainProvider instance.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `opts` | `Object` | Options for the provider. |
| `opts.bridges?` | [`BridgeAdapterData`](../interfaces/BridgeAdapterData.md) | Optional bridge address list. |
| `opts.contracts?` | [`DeepPartial`](../modules.md#deeppartial)<[`OEContractsLike`](../interfaces/OEContractsLike.md)\> | Optional contract address overrides. |
| `opts.depositConfirmationBlocks?` | [`NumberLike`](../modules.md#numberlike) | Optional number of blocks before a deposit is confirmed. |
| `opts.l1BlockTimeSeconds?` | [`NumberLike`](../modules.md#numberlike) | Optional estimated block time in seconds for the L1 chain. |
| `opts.l1ChainId` | [`NumberLike`](../modules.md#numberlike) | Chain ID for the L1 chain. |
| `opts.l1SignerOrProvider` | `any` | Signer or Provider for the L1 chain, or a JSON-RPC url. |
| `opts.l2SignerOrProvider` | `any` | Signer or Provider for the L2 chain, or a JSON-RPC url. |

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:74](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L74)

## Properties

### bridges

• **bridges**: [`BridgeAdapters`](../interfaces/BridgeAdapters.md)

List of custom bridges for the given network.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[bridges](../interfaces/ICrossChainMessenger.md#bridges)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:58](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L58)

___

### contracts

• **contracts**: [`OEContracts`](../interfaces/OEContracts.md)

Contract objects attached to their respective providers and addresses.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[contracts](../interfaces/ICrossChainMessenger.md#contracts)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:57](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L57)

___

### depositConfirmationBlocks

• **depositConfirmationBlocks**: `number`

Number of blocks before a deposit is considered confirmed.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[depositConfirmationBlocks](../interfaces/ICrossChainMessenger.md#depositconfirmationblocks)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:59](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L59)

___

### estimateGas

• **estimateGas**: `Object`

Object that holds the functions that estimates the gas required for a given transaction.
Follows the pattern used by ethers.js.

#### Type declaration

| Name | Type |
| :------ | :------ |
| `approveERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`BigNumber`\> |
| `depositERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |
| `depositETH` | (`amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |
| `finalizeMessage` | (`message`: [`MessageLike`](../modules.md#messagelike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`BigNumber`\> |
| `resendMessage` | (`message`: [`MessageLike`](../modules.md#messagelike), `messageGasLimit`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`BigNumber`\> |
| `sendMessage` | (`message`: [`CrossChainMessageRequest`](../interfaces/CrossChainMessageRequest.md), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides`  }) => `Promise`<`BigNumber`\> |
| `withdrawERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |
| `withdrawETH` | (`amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[estimateGas](../interfaces/ICrossChainMessenger.md#estimategas)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:1129](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L1129)

___

### l1BlockTimeSeconds

• **l1BlockTimeSeconds**: `number`

Estimated average L1 block time in seconds.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l1BlockTimeSeconds](../interfaces/ICrossChainMessenger.md#l1blocktimeseconds)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:60](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L60)

___

### l1ChainId

• **l1ChainId**: `number`

Chain ID for the L1 network.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l1ChainId](../interfaces/ICrossChainMessenger.md#l1chainid)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:56](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L56)

___

### l1SignerOrProvider

• **l1SignerOrProvider**: `Signer` \| `Provider`

Provider connected to the L1 chain.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l1SignerOrProvider](../interfaces/ICrossChainMessenger.md#l1signerorprovider)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:54](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L54)

___

### l2SignerOrProvider

• **l2SignerOrProvider**: `Signer` \| `Provider`

Provider connected to the L2 chain.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l2SignerOrProvider](../interfaces/ICrossChainMessenger.md#l2signerorprovider)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:55](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L55)

___

### populateTransaction

• **populateTransaction**: `Object`

Object that holds the functions that generate transactions to be signed by the user.
Follows the pattern used by ethers.js.

#### Type declaration

| Name | Type |
| :------ | :------ |
| `approveERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`TransactionRequest`\> |
| `depositERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |
| `depositETH` | (`amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |
| `finalizeMessage` | (`message`: [`MessageLike`](../modules.md#messagelike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`TransactionRequest`\> |
| `resendMessage` | (`message`: [`MessageLike`](../modules.md#messagelike), `messageGasLimit`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides`  }) => `Promise`<`TransactionRequest`\> |
| `sendMessage` | (`message`: [`CrossChainMessageRequest`](../interfaces/CrossChainMessageRequest.md), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides`  }) => `Promise`<`TransactionRequest`\> |
| `withdrawERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |
| `withdrawETH` | (`amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[populateTransaction](../interfaces/ICrossChainMessenger.md#populatetransaction)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:988](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L988)

## Accessors

### l1Provider

• `get` **l1Provider**(): `Provider`

Provider connected to the L1 chain.

#### Returns

`Provider`

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l1Provider](../interfaces/ICrossChainMessenger.md#l1provider)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:108](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L108)

___

### l1Signer

• `get` **l1Signer**(): `Signer`

Signer connected to the L1 chain.

#### Returns

`Signer`

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l1Signer](../interfaces/ICrossChainMessenger.md#l1signer)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:124](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L124)

___

### l2Provider

• `get` **l2Provider**(): `Provider`

Provider connected to the L2 chain.

#### Returns

`Provider`

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l2Provider](../interfaces/ICrossChainMessenger.md#l2provider)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:116](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L116)

___

### l2Signer

• `get` **l2Signer**(): `Signer`

Signer connected to the L2 chain.

#### Returns

`Signer`

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[l2Signer](../interfaces/ICrossChainMessenger.md#l2signer)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:132](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L132)

## Methods

### approval

▸ **approval**(`l1Token`, `l2Token`, `opts?`): `Promise`<`BigNumber`\>

Queries the account's approval amount for a given L1 token.

#### Parameters

| Name | Type |
| :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) |
| `opts?` | `Object` |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`BigNumber`\>

Amount of tokens approved for deposits from the account.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[approval](../interfaces/ICrossChainMessenger.md#approval)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:917](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L917)

___

### approveERC20

▸ **approveERC20**(`l1Token`, `l2Token`, `amount`, `opts?`): `Promise`<`TransactionResponse`\>

Approves a deposit into the L2 chain.

#### Parameters

| Name | Type |
| :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) |
| `amount` | [`NumberLike`](../modules.md#numberlike) |
| `opts?` | `Object` |
| `opts.overrides?` | `Overrides` |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the approval transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[approveERC20](../interfaces/ICrossChainMessenger.md#approveerc20)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:928](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L928)

___

### depositERC20

▸ **depositERC20**(`l1Token`, `l2Token`, `amount`, `opts?`): `Promise`<`TransactionResponse`\>

Deposits some ERC20 tokens into the L2 chain.

#### Parameters

| Name | Type |
| :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) |
| `amount` | [`NumberLike`](../modules.md#numberlike) |
| `opts?` | `Object` |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) |
| `opts.overrides?` | `Overrides` |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the deposit transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[depositERC20](../interfaces/ICrossChainMessenger.md#depositerc20)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:947](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L947)

___

### depositETH

▸ **depositETH**(`amount`, `opts?`): `Promise`<`TransactionResponse`\>

Deposits some ETH into the L2 chain.

#### Parameters

| Name | Type |
| :------ | :------ |
| `amount` | [`NumberLike`](../modules.md#numberlike) |
| `opts?` | `Object` |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) |
| `opts.overrides?` | `Overrides` |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the deposit transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[depositETH](../interfaces/ICrossChainMessenger.md#depositeth)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:890](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L890)

___

### estimateL2MessageGasLimit

▸ **estimateL2MessageGasLimit**(`message`, `opts?`): `Promise`<`BigNumber`\>

Estimates the amount of gas required to fully execute a given message on L2. Only applies to
L1 => L2 messages. You would supply this gas limit when sending the message to L2.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageRequestLike`](../modules.md#messagerequestlike) |
| `opts?` | `Object` |
| `opts.bufferPercent?` | `number` |
| `opts.from?` | `string` |

#### Returns

`Promise`<`BigNumber`\>

Estimates L2 gas limit.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[estimateL2MessageGasLimit](../interfaces/ICrossChainMessenger.md#estimatel2messagegaslimit)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:541](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L541)

___

### estimateMessageWaitTimeSeconds

▸ **estimateMessageWaitTimeSeconds**(`message`): `Promise`<`number`\>

Returns the estimated amount of time before the message can be executed. When this is a
message being sent to L1, this will return the estimated time until the message will complete
its challenge period. When this is a message being sent to L2, this will return the estimated
amount of time until the message will be picked up and executed on L2.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |

#### Returns

`Promise`<`number`\>

Estimated amount of time remaining (in seconds) before the message can be executed.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[estimateMessageWaitTimeSeconds](../interfaces/ICrossChainMessenger.md#estimatemessagewaittimeseconds)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:574](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L574)

___

### finalizeMessage

▸ **finalizeMessage**(`message`, `opts?`): `Promise`<`TransactionResponse`\>

Finalizes a cross chain message that was sent from L2 to L1. Only applicable for L2 to L1
messages. Will throw an error if the message has not completed its challenge period yet.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |
| `opts?` | `Object` |
| `opts.overrides?` | `Overrides` |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the finalization transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[finalizeMessage](../interfaces/ICrossChainMessenger.md#finalizemessage)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:878](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L878)

___

### getBridgeForTokenPair

▸ **getBridgeForTokenPair**(`l1Token`, `l2Token`): `Promise`<[`IBridgeAdapter`](../interfaces/IBridgeAdapter.md)\>

Finds the appropriate bridge adapter for a given L1<>L2 token pair. Will throw if no bridges
support the token pair or if more than one bridge supports the token pair.

#### Parameters

| Name | Type |
| :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) |

#### Returns

`Promise`<[`IBridgeAdapter`](../interfaces/IBridgeAdapter.md)\>

The appropriate bridge adapter for the given token pair.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getBridgeForTokenPair](../interfaces/ICrossChainMessenger.md#getbridgefortokenpair)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:228](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L228)

___

### getChallengePeriodSeconds

▸ **getChallengePeriodSeconds**(): `Promise`<`number`\>

Queries the current challenge period in seconds from the StateCommitmentChain.

#### Returns

`Promise`<`number`\>

Current challenge period in seconds.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getChallengePeriodSeconds](../interfaces/ICrossChainMessenger.md#getchallengeperiodseconds)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:633](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L633)

___

### getDepositsByAddress

▸ **getDepositsByAddress**(`address`, `opts?`): `Promise`<[`TokenBridgeMessage`](../interfaces/TokenBridgeMessage.md)[]\>

Gets all deposits for a given address.

#### Parameters

| Name | Type |
| :------ | :------ |
| `address` | [`AddressLike`](../modules.md#addresslike) |
| `opts` | `Object` |
| `opts.fromBlock?` | `BlockTag` |
| `opts.toBlock?` | `BlockTag` |

#### Returns

`Promise`<[`TokenBridgeMessage`](../interfaces/TokenBridgeMessage.md)[]\>

All deposit token bridge messages sent by the given address.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getDepositsByAddress](../interfaces/ICrossChainMessenger.md#getdepositsbyaddress)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:250](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L250)

___

### getMessageProof

▸ **getMessageProof**(`message`): `Promise`<[`CrossChainMessageProof`](../interfaces/CrossChainMessageProof.md)\>

Generates the proof required to finalize an L2 to L1 message.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |

#### Returns

`Promise`<[`CrossChainMessageProof`](../interfaces/CrossChainMessageProof.md)\>

Proof that can be used to finalize the message.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getMessageProof](../interfaces/ICrossChainMessenger.md#getmessageproof)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:797](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L797)

___

### getMessageReceipt

▸ **getMessageReceipt**(`message`): `Promise`<[`MessageReceipt`](../interfaces/MessageReceipt.md)\>

Finds the receipt of the transaction that executed a particular cross chain message.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |

#### Returns

`Promise`<[`MessageReceipt`](../interfaces/MessageReceipt.md)\>

CrossChainMessage receipt including receipt of the transaction that relayed the
given message.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getMessageReceipt](../interfaces/ICrossChainMessenger.md#getmessagereceipt)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:390](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L390)

___

### getMessageStateRoot

▸ **getMessageStateRoot**(`message`): `Promise`<[`StateRoot`](../interfaces/StateRoot.md)\>

Returns the state root that corresponds to a given message. This is the state root for the
block in which the transaction was included, as published to the StateCommitmentChain. If the
state root for the given message has not been published yet, this function returns null.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |

#### Returns

`Promise`<[`StateRoot`](../interfaces/StateRoot.md)\>

State root for the block in which the message was created.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getMessageStateRoot](../interfaces/ICrossChainMessenger.md#getmessagestateroot)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:639](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L639)

___

### getMessageStatus

▸ **getMessageStatus**(`message`): `Promise`<[`MessageStatus`](../enums/MessageStatus.md)\>

Retrieves the status of a particular message as an enum.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |

#### Returns

`Promise`<[`MessageStatus`](../enums/MessageStatus.md)\>

Status of the message.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getMessageStatus](../interfaces/ICrossChainMessenger.md#getmessagestatus)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:349](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L349)

___

### getMessagesByAddress

▸ **getMessagesByAddress**(`address`, `opts?`): `Promise`<[`CrossChainMessage`](../interfaces/CrossChainMessage.md)[]\>

Retrieves all cross chain messages sent by a particular address.

#### Parameters

| Name | Type |
| :------ | :------ |
| `address` | [`AddressLike`](../modules.md#addresslike) |
| `opts?` | `Object` |
| `opts.direction?` | [`MessageDirection`](../enums/MessageDirection.md) |
| `opts.fromBlock?` | [`NumberLike`](../modules.md#numberlike) |
| `opts.toBlock?` | [`NumberLike`](../modules.md#numberlike) |

#### Returns

`Promise`<[`CrossChainMessage`](../interfaces/CrossChainMessage.md)[]\>

All cross chain messages sent by the particular address.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getMessagesByAddress](../interfaces/ICrossChainMessenger.md#getmessagesbyaddress)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:211](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L211)

___

### getMessagesByTransaction

▸ **getMessagesByTransaction**(`transaction`, `opts?`): `Promise`<[`CrossChainMessage`](../interfaces/CrossChainMessage.md)[]\>

Retrieves all cross chain messages sent within a given transaction.

#### Parameters

| Name | Type |
| :------ | :------ |
| `transaction` | [`TransactionLike`](../modules.md#transactionlike) |
| `opts` | `Object` |
| `opts.direction?` | [`MessageDirection`](../enums/MessageDirection.md) |

#### Returns

`Promise`<[`CrossChainMessage`](../interfaces/CrossChainMessage.md)[]\>

All cross chain messages sent within the transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getMessagesByTransaction](../interfaces/ICrossChainMessenger.md#getmessagesbytransaction)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:140](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L140)

___

### getStateBatchAppendedEventByBatchIndex

▸ **getStateBatchAppendedEventByBatchIndex**(`batchIndex`): `Promise`<`Event`\>

Returns the StateBatchAppended event that was emitted when the batch with a given index was
created. Returns null if no such event exists (the batch has not been submitted).

#### Parameters

| Name | Type |
| :------ | :------ |
| `batchIndex` | `number` |

#### Returns

`Promise`<`Event`\>

StateBatchAppended event for the batch, or null if no such batch exists.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getStateBatchAppendedEventByBatchIndex](../interfaces/ICrossChainMessenger.md#getstatebatchappendedeventbybatchindex)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:690](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L690)

___

### getStateBatchAppendedEventByTransactionIndex

▸ **getStateBatchAppendedEventByTransactionIndex**(`transactionIndex`): `Promise`<`Event`\>

Returns the StateBatchAppended event for the batch that includes the transaction with the
given index. Returns null if no such event exists.

#### Parameters

| Name | Type |
| :------ | :------ |
| `transactionIndex` | `number` |

#### Returns

`Promise`<`Event`\>

StateBatchAppended event for the batch that includes the given transaction by index.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getStateBatchAppendedEventByTransactionIndex](../interfaces/ICrossChainMessenger.md#getstatebatchappendedeventbytransactionindex)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:709](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L709)

___

### getStateRootBatchByTransactionIndex

▸ **getStateRootBatchByTransactionIndex**(`transactionIndex`): `Promise`<[`StateRootBatch`](../interfaces/StateRootBatch.md)\>

Returns information about the state root batch that included the state root for the given
transaction by index. Returns null if no such state root has been published yet.

#### Parameters

| Name | Type |
| :------ | :------ |
| `transactionIndex` | `number` |

#### Returns

`Promise`<[`StateRootBatch`](../interfaces/StateRootBatch.md)\>

State root batch for the given transaction index, or null if none exists yet.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getStateRootBatchByTransactionIndex](../interfaces/ICrossChainMessenger.md#getstaterootbatchbytransactionindex)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:768](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L768)

___

### getWithdrawalsByAddress

▸ **getWithdrawalsByAddress**(`address`, `opts?`): `Promise`<[`TokenBridgeMessage`](../interfaces/TokenBridgeMessage.md)[]\>

Gets all withdrawals for a given address.

#### Parameters

| Name | Type |
| :------ | :------ |
| `address` | [`AddressLike`](../modules.md#addresslike) |
| `opts` | `Object` |
| `opts.fromBlock?` | `BlockTag` |
| `opts.toBlock?` | `BlockTag` |

#### Returns

`Promise`<[`TokenBridgeMessage`](../interfaces/TokenBridgeMessage.md)[]\>

All withdrawal token bridge messages sent by the given address.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[getWithdrawalsByAddress](../interfaces/ICrossChainMessenger.md#getwithdrawalsbyaddress)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:273](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L273)

___

### resendMessage

▸ **resendMessage**(`message`, `messageGasLimit`, `opts?`): `Promise`<`TransactionResponse`\>

Resends a given cross chain message with a different gas limit. Only applies to L1 to L2
messages. If provided an L2 to L1 message, this function will throw an error.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |
| `messageGasLimit` | [`NumberLike`](../modules.md#numberlike) |
| `opts?` | `Object` |
| `opts.overrides?` | `Overrides` |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the message resending transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[resendMessage](../interfaces/ICrossChainMessenger.md#resendmessage)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:861](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L861)

___

### sendMessage

▸ **sendMessage**(`message`, `opts?`): `Promise`<`TransactionResponse`\>

Sends a given cross chain message. Where the message is sent depends on the direction attached
to the message itself.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`CrossChainMessageRequest`](../interfaces/CrossChainMessageRequest.md) |
| `opts?` | `Object` |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) |
| `opts.overrides?` | `Overrides` |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the message sending transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[sendMessage](../interfaces/ICrossChainMessenger.md#sendmessage)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:845](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L845)

___

### toCrossChainMessage

▸ **toCrossChainMessage**(`message`): `Promise`<[`CrossChainMessage`](../interfaces/CrossChainMessage.md)\>

Resolves a MessageLike into a CrossChainMessage object.
Unlike other coercion functions, this function is stateful and requires making additional
requests. For now I'm going to keep this function here, but we could consider putting a
similar function inside of utils/coercion.ts if people want to use this without having to
create an entire CrossChainProvider object.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |

#### Returns

`Promise`<[`CrossChainMessage`](../interfaces/CrossChainMessage.md)\>

Message coerced into a CrossChainMessage.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[toCrossChainMessage](../interfaces/ICrossChainMessenger.md#tocrosschainmessage)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:296](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L296)

___

### waitForMessageReceipt

▸ **waitForMessageReceipt**(`message`, `opts?`): `Promise`<[`MessageReceipt`](../interfaces/MessageReceipt.md)\>

Waits for a message to be executed and returns the receipt of the transaction that executed
the given message.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |
| `opts` | `Object` |
| `opts.confirmations?` | `number` |
| `opts.pollIntervalMs?` | `number` |
| `opts.timeoutMs?` | `number` |

#### Returns

`Promise`<[`MessageReceipt`](../interfaces/MessageReceipt.md)\>

CrossChainMessage receipt including receipt of the transaction that relayed the
given message.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[waitForMessageReceipt](../interfaces/ICrossChainMessenger.md#waitformessagereceipt)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:448](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L448)

___

### waitForMessageStatus

▸ **waitForMessageStatus**(`message`, `status`, `opts?`): `Promise`<`void`\>

Waits until the status of a given message changes to the expected status. Note that if the
status of the given message changes to a status that implies the expected status, this will
still return. If the status of the message changes to a status that exclues the expected
status, this will throw an error.

#### Parameters

| Name | Type |
| :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) |
| `status` | [`MessageStatus`](../enums/MessageStatus.md) |
| `opts` | `Object` |
| `opts.pollIntervalMs?` | `number` |
| `opts.timeoutMs?` | `number` |

#### Returns

`Promise`<`void`\>

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[waitForMessageStatus](../interfaces/ICrossChainMessenger.md#waitformessagestatus)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:474](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L474)

___

### withdrawERC20

▸ **withdrawERC20**(`l1Token`, `l2Token`, `amount`, `opts?`): `Promise`<`TransactionResponse`\>

Withdraws some ERC20 tokens back to the L1 chain.

#### Parameters

| Name | Type |
| :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) |
| `amount` | [`NumberLike`](../modules.md#numberlike) |
| `opts?` | `Object` |
| `opts.overrides?` | `Overrides` |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the withdraw transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[withdrawERC20](../interfaces/ICrossChainMessenger.md#withdrawerc20)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:968](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L968)

___

### withdrawETH

▸ **withdrawETH**(`amount`, `opts?`): `Promise`<`TransactionResponse`\>

Withdraws some ETH back to the L1 chain.

#### Parameters

| Name | Type |
| :------ | :------ |
| `amount` | [`NumberLike`](../modules.md#numberlike) |
| `opts?` | `Object` |
| `opts.overrides?` | `Overrides` |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) |
| `opts.signer?` | `Signer` |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the withdraw transaction.

#### Implementation of

[ICrossChainMessenger](../interfaces/ICrossChainMessenger.md).[withdrawETH](../interfaces/ICrossChainMessenger.md#withdraweth)

#### Defined in

[packages/sdk/src/cross-chain-messenger.ts:904](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts#L904)
