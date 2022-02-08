[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / ICrossChainMessenger

# Interface: ICrossChainMessenger

Handles L1/L2 interactions.

## Implemented by

- [`CrossChainMessenger`](../classes/CrossChainMessenger.md)

## Table of contents

### Properties

- [bridges](ICrossChainMessenger.md#bridges)
- [contracts](ICrossChainMessenger.md#contracts)
- [depositConfirmationBlocks](ICrossChainMessenger.md#depositconfirmationblocks)
- [estimateGas](ICrossChainMessenger.md#estimategas)
- [l1BlockTimeSeconds](ICrossChainMessenger.md#l1blocktimeseconds)
- [l1ChainId](ICrossChainMessenger.md#l1chainid)
- [l1Provider](ICrossChainMessenger.md#l1provider)
- [l1Signer](ICrossChainMessenger.md#l1signer)
- [l1SignerOrProvider](ICrossChainMessenger.md#l1signerorprovider)
- [l2Provider](ICrossChainMessenger.md#l2provider)
- [l2Signer](ICrossChainMessenger.md#l2signer)
- [l2SignerOrProvider](ICrossChainMessenger.md#l2signerorprovider)
- [populateTransaction](ICrossChainMessenger.md#populatetransaction)

### Methods

- [approval](ICrossChainMessenger.md#approval)
- [approveERC20](ICrossChainMessenger.md#approveerc20)
- [depositERC20](ICrossChainMessenger.md#depositerc20)
- [depositETH](ICrossChainMessenger.md#depositeth)
- [estimateL2MessageGasLimit](ICrossChainMessenger.md#estimatel2messagegaslimit)
- [estimateMessageWaitTimeSeconds](ICrossChainMessenger.md#estimatemessagewaittimeseconds)
- [finalizeMessage](ICrossChainMessenger.md#finalizemessage)
- [getBridgeForTokenPair](ICrossChainMessenger.md#getbridgefortokenpair)
- [getChallengePeriodSeconds](ICrossChainMessenger.md#getchallengeperiodseconds)
- [getDepositsByAddress](ICrossChainMessenger.md#getdepositsbyaddress)
- [getMessageProof](ICrossChainMessenger.md#getmessageproof)
- [getMessageReceipt](ICrossChainMessenger.md#getmessagereceipt)
- [getMessageStateRoot](ICrossChainMessenger.md#getmessagestateroot)
- [getMessageStatus](ICrossChainMessenger.md#getmessagestatus)
- [getMessagesByAddress](ICrossChainMessenger.md#getmessagesbyaddress)
- [getMessagesByTransaction](ICrossChainMessenger.md#getmessagesbytransaction)
- [getStateBatchAppendedEventByBatchIndex](ICrossChainMessenger.md#getstatebatchappendedeventbybatchindex)
- [getStateBatchAppendedEventByTransactionIndex](ICrossChainMessenger.md#getstatebatchappendedeventbytransactionindex)
- [getStateRootBatchByTransactionIndex](ICrossChainMessenger.md#getstaterootbatchbytransactionindex)
- [getWithdrawalsByAddress](ICrossChainMessenger.md#getwithdrawalsbyaddress)
- [resendMessage](ICrossChainMessenger.md#resendmessage)
- [sendMessage](ICrossChainMessenger.md#sendmessage)
- [toCrossChainMessage](ICrossChainMessenger.md#tocrosschainmessage)
- [waitForMessageReceipt](ICrossChainMessenger.md#waitformessagereceipt)
- [waitForMessageStatus](ICrossChainMessenger.md#waitformessagestatus)
- [withdrawERC20](ICrossChainMessenger.md#withdrawerc20)
- [withdrawETH](ICrossChainMessenger.md#withdraweth)

## Properties

### bridges

• **bridges**: [`BridgeAdapters`](BridgeAdapters.md)

List of custom bridges for the given network.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:57](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L57)

___

### contracts

• **contracts**: [`OEContracts`](OEContracts.md)

Contract objects attached to their respective providers and addresses.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:52](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L52)

___

### depositConfirmationBlocks

• **depositConfirmationBlocks**: `number`

Number of blocks before a deposit is considered confirmed.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:82](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L82)

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
| `sendMessage` | (`message`: [`CrossChainMessageRequest`](CrossChainMessageRequest.md), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides`  }) => `Promise`<`BigNumber`\> |
| `withdrawERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |
| `withdrawETH` | (`amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`BigNumber`\> |

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:683](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L683)

___

### l1BlockTimeSeconds

• **l1BlockTimeSeconds**: `number`

Estimated average L1 block time in seconds.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:87](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L87)

___

### l1ChainId

• **l1ChainId**: `number`

Chain ID for the L1 network.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:47](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L47)

___

### l1Provider

• **l1Provider**: `Provider`

Provider connected to the L1 chain.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:62](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L62)

___

### l1Signer

• **l1Signer**: `Signer`

Signer connected to the L1 chain.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:72](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L72)

___

### l1SignerOrProvider

• **l1SignerOrProvider**: `Signer` \| `Provider`

Provider connected to the L1 chain.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:37](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L37)

___

### l2Provider

• **l2Provider**: `Provider`

Provider connected to the L2 chain.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:67](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L67)

___

### l2Signer

• **l2Signer**: `Signer`

Signer connected to the L2 chain.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:77](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L77)

___

### l2SignerOrProvider

• **l2SignerOrProvider**: `Signer` \| `Provider`

Provider connected to the L2 chain.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:42](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L42)

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
| `sendMessage` | (`message`: [`CrossChainMessageRequest`](CrossChainMessageRequest.md), `opts?`: { `l2GasLimit?`: [`NumberLike`](../modules.md#numberlike) ; `overrides?`: `Overrides`  }) => `Promise`<`TransactionRequest`\> |
| `withdrawERC20` | (`l1Token`: [`AddressLike`](../modules.md#addresslike), `l2Token`: [`AddressLike`](../modules.md#addresslike), `amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |
| `withdrawETH` | (`amount`: [`NumberLike`](../modules.md#numberlike), `opts?`: { `overrides?`: `Overrides` ; `recipient?`: [`AddressLike`](../modules.md#addresslike)  }) => `Promise`<`TransactionRequest`\> |

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:525](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L525)

## Methods

### approval

▸ **approval**(`l1Token`, `l2Token`, `opts?`): `Promise`<`BigNumber`\>

Queries the account's approval amount for a given L1 token.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |
| `opts?` | `Object` | Additional options. |
| `opts.signer?` | `Signer` | Optional signer to get the approval for. |

#### Returns

`Promise`<`BigNumber`\>

Amount of tokens approved for deposits from the account.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:444](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L444)

___

### approveERC20

▸ **approveERC20**(`l1Token`, `l2Token`, `amount`, `opts?`): `Promise`<`TransactionResponse`\>

Approves a deposit into the L2 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | The L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | The L2 token address. |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount of the token to approve. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the approval transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:463](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L463)

___

### depositERC20

▸ **depositERC20**(`l1Token`, `l2Token`, `amount`, `opts?`): `Promise`<`TransactionResponse`\>

Deposits some ERC20 tokens into the L2 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | Address of the L1 token. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | Address of the L2 token. |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount to deposit. |
| `opts?` | `Object` | Additional options. |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) | Optional gas limit to use for the transaction on L2. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) | Optional address to receive the funds on L2. Defaults to sender. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the deposit transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:486](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L486)

___

### depositETH

▸ **depositETH**(`amount`, `opts?`): `Promise`<`TransactionResponse`\>

Deposits some ETH into the L2 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount of ETH to deposit (in wei). |
| `opts?` | `Object` | Additional options. |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) | Optional gas limit to use for the transaction on L2. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) | Optional address to receive the funds on L2. Defaults to sender. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the deposit transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:406](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L406)

___

### estimateL2MessageGasLimit

▸ **estimateL2MessageGasLimit**(`message`, `opts?`): `Promise`<`BigNumber`\>

Estimates the amount of gas required to fully execute a given message on L2. Only applies to
L1 => L2 messages. You would supply this gas limit when sending the message to L2.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageRequestLike`](../modules.md#messagerequestlike) | Message get a gas estimate for. |
| `opts?` | `Object` | Options object. |
| `opts.bufferPercent?` | `number` | Percentage of gas to add to the estimate. Defaults to 20. |
| `opts.from?` | `string` | Address to use as the sender. |

#### Returns

`Promise`<`BigNumber`\>

Estimates L2 gas limit.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:260](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L260)

___

### estimateMessageWaitTimeSeconds

▸ **estimateMessageWaitTimeSeconds**(`message`): `Promise`<`number`\>

Returns the estimated amount of time before the message can be executed. When this is a
message being sent to L1, this will return the estimated time until the message will complete
its challenge period. When this is a message being sent to L2, this will return the estimated
amount of time until the message will be picked up and executed on L2.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to estimate the time remaining for. |

#### Returns

`Promise`<`number`\>

Estimated amount of time remaining (in seconds) before the message can be executed.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:277](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L277)

___

### finalizeMessage

▸ **finalizeMessage**(`message`, `opts?`): `Promise`<`TransactionResponse`\>

Finalizes a cross chain message that was sent from L2 to L1. Only applicable for L2 to L1
messages. Will throw an error if the message has not completed its challenge period yet.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to finalize. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the finalization transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:387](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L387)

___

### getBridgeForTokenPair

▸ **getBridgeForTokenPair**(`l1Token`, `l2Token`): `Promise`<[`IBridgeAdapter`](IBridgeAdapter.md)\>

Finds the appropriate bridge adapter for a given L1<>L2 token pair. Will throw if no bridges
support the token pair or if more than one bridge supports the token pair.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | L1 token address. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | L2 token address. |

#### Returns

`Promise`<[`IBridgeAdapter`](IBridgeAdapter.md)\>

The appropriate bridge adapter for the given token pair.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:136](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L136)

___

### getChallengePeriodSeconds

▸ **getChallengePeriodSeconds**(): `Promise`<`number`\>

Queries the current challenge period in seconds from the StateCommitmentChain.

#### Returns

`Promise`<`number`\>

Current challenge period in seconds.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:284](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L284)

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

[packages/sdk/src/interfaces/cross-chain-messenger.ts:152](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L152)

___

### getMessageProof

▸ **getMessageProof**(`message`): `Promise`<[`CrossChainMessageProof`](CrossChainMessageProof.md)\>

Generates the proof required to finalize an L2 to L1 message.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to generate a proof for. |

#### Returns

`Promise`<[`CrossChainMessageProof`](CrossChainMessageProof.md)\>

Proof that can be used to finalize the message.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:335](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L335)

___

### getMessageReceipt

▸ **getMessageReceipt**(`message`): `Promise`<[`MessageReceipt`](MessageReceipt.md)\>

Finds the receipt of the transaction that executed a particular cross chain message.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to find the receipt of. |

#### Returns

`Promise`<[`MessageReceipt`](MessageReceipt.md)\>

CrossChainMessage receipt including receipt of the transaction that relayed the
given message.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:206](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L206)

___

### getMessageStateRoot

▸ **getMessageStateRoot**(`message`): `Promise`<[`StateRoot`](StateRoot.md)\>

Returns the state root that corresponds to a given message. This is the state root for the
block in which the transaction was included, as published to the StateCommitmentChain. If the
state root for the given message has not been published yet, this function returns null.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to find a state root for. |

#### Returns

`Promise`<[`StateRoot`](StateRoot.md)\>

State root for the block in which the message was created.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:294](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L294)

___

### getMessageStatus

▸ **getMessageStatus**(`message`): `Promise`<[`MessageStatus`](../enums/MessageStatus.md)\>

Retrieves the status of a particular message as an enum.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Cross chain message to check the status of. |

#### Returns

`Promise`<[`MessageStatus`](../enums/MessageStatus.md)\>

Status of the message.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:197](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L197)

___

### getMessagesByAddress

▸ **getMessagesByAddress**(`address`, `opts?`): `Promise`<[`CrossChainMessage`](CrossChainMessage.md)[]\>

Retrieves all cross chain messages sent by a particular address.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `address` | [`AddressLike`](../modules.md#addresslike) | Address to search for messages from. |
| `opts?` | `Object` | Options object. |
| `opts.direction?` | [`MessageDirection`](../enums/MessageDirection.md) | Direction to search for messages in. If not provided, will attempt to find all messages in both directions. |
| `opts.fromBlock?` | [`NumberLike`](../modules.md#numberlike) | Block to start searching for messages from. If not provided, will start from the first block (block #0). |
| `opts.toBlock?` | [`NumberLike`](../modules.md#numberlike) | Block to stop searching for messages at. If not provided, will stop at the latest known block ("latest"). |

#### Returns

`Promise`<[`CrossChainMessage`](CrossChainMessage.md)[]\>

All cross chain messages sent by the particular address.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:119](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L119)

___

### getMessagesByTransaction

▸ **getMessagesByTransaction**(`transaction`, `opts?`): `Promise`<[`CrossChainMessage`](CrossChainMessage.md)[]\>

Retrieves all cross chain messages sent within a given transaction.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `transaction` | [`TransactionLike`](../modules.md#transactionlike) | Transaction hash or receipt to find messages from. |
| `opts?` | `Object` | Options object. |
| `opts.direction?` | [`MessageDirection`](../enums/MessageDirection.md) | Direction to search for messages in. If not provided, will attempt to automatically search both directions under the assumption that a transaction hash will only exist on one chain. If the hash exists on both chains, will throw an error. |

#### Returns

`Promise`<[`CrossChainMessage`](CrossChainMessage.md)[]\>

All cross chain messages sent within the transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:99](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L99)

___

### getStateBatchAppendedEventByBatchIndex

▸ **getStateBatchAppendedEventByBatchIndex**(`batchIndex`): `Promise`<`Event`\>

Returns the StateBatchAppended event that was emitted when the batch with a given index was
created. Returns null if no such event exists (the batch has not been submitted).

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `batchIndex` | `number` | Index of the batch to find an event for. |

#### Returns

`Promise`<`Event`\>

StateBatchAppended event for the batch, or null if no such batch exists.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:303](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L303)

___

### getStateBatchAppendedEventByTransactionIndex

▸ **getStateBatchAppendedEventByTransactionIndex**(`transactionIndex`): `Promise`<`Event`\>

Returns the StateBatchAppended event for the batch that includes the transaction with the
given index. Returns null if no such event exists.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `transactionIndex` | `number` | Index of the L2 transaction to find an event for. |

#### Returns

`Promise`<`Event`\>

StateBatchAppended event for the batch that includes the given transaction by index.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:314](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L314)

___

### getStateRootBatchByTransactionIndex

▸ **getStateRootBatchByTransactionIndex**(`transactionIndex`): `Promise`<[`StateRootBatch`](StateRootBatch.md)\>

Returns information about the state root batch that included the state root for the given
transaction by index. Returns null if no such state root has been published yet.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `transactionIndex` | `number` | Index of the L2 transaction to find a state root batch for. |

#### Returns

`Promise`<[`StateRootBatch`](StateRootBatch.md)\>

State root batch for the given transaction index, or null if none exists yet.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:325](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L325)

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

[packages/sdk/src/interfaces/cross-chain-messenger.ts:171](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L171)

___

### resendMessage

▸ **resendMessage**(`message`, `messageGasLimit`, `opts?`): `Promise`<`TransactionResponse`\>

Resends a given cross chain message with a different gas limit. Only applies to L1 to L2
messages. If provided an L2 to L1 message, this function will throw an error.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Cross chain message to resend. |
| `messageGasLimit` | [`NumberLike`](../modules.md#numberlike) | New gas limit to use for the message. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the message resending transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:368](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L368)

___

### sendMessage

▸ **sendMessage**(`message`, `opts?`): `Promise`<`TransactionResponse`\>

Sends a given cross chain message. Where the message is sent depends on the direction attached
to the message itself.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`CrossChainMessageRequest`](CrossChainMessageRequest.md) | Cross chain message to send. |
| `opts?` | `Object` | Additional options. |
| `opts.l2GasLimit?` | [`NumberLike`](../modules.md#numberlike) | Optional gas limit to use for the transaction on L2. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the message sending transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:348](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L348)

___

### toCrossChainMessage

▸ **toCrossChainMessage**(`message`): `Promise`<[`CrossChainMessage`](CrossChainMessage.md)\>

Resolves a MessageLike into a CrossChainMessage object.
Unlike other coercion functions, this function is stateful and requires making additional
requests. For now I'm going to keep this function here, but we could consider putting a
similar function inside of utils/coercion.ts if people want to use this without having to
create an entire CrossChainProvider object.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | MessageLike to resolve into a CrossChainMessage. |

#### Returns

`Promise`<[`CrossChainMessage`](CrossChainMessage.md)\>

Message coerced into a CrossChainMessage.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:189](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L189)

___

### waitForMessageReceipt

▸ **waitForMessageReceipt**(`message`, `opts?`): `Promise`<[`MessageReceipt`](MessageReceipt.md)\>

Waits for a message to be executed and returns the receipt of the transaction that executed
the given message.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to wait for. |
| `opts?` | `Object` | Options to pass to the waiting function. |
| `opts.confirmations?` | `number` | Number of transaction confirmations to wait for before returning. |
| `opts.pollIntervalMs?` | `number` | Number of milliseconds to wait between polling for the receipt. |
| `opts.timeoutMs?` | `number` | Milliseconds to wait before timing out. |

#### Returns

`Promise`<[`MessageReceipt`](MessageReceipt.md)\>

CrossChainMessage receipt including receipt of the transaction that relayed the
given message.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:220](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L220)

___

### waitForMessageStatus

▸ **waitForMessageStatus**(`message`, `status`, `opts?`): `Promise`<`void`\>

Waits until the status of a given message changes to the expected status. Note that if the
status of the given message changes to a status that implies the expected status, this will
still return. If the status of the message changes to a status that exclues the expected
status, this will throw an error.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `message` | [`MessageLike`](../modules.md#messagelike) | Message to wait for. |
| `status` | [`MessageStatus`](../enums/MessageStatus.md) | Expected status of the message. |
| `opts?` | `Object` | Options to pass to the waiting function. |
| `opts.pollIntervalMs?` | `number` | Number of milliseconds to wait when polling. |
| `opts.timeoutMs?` | `number` | Milliseconds to wait before timing out. |

#### Returns

`Promise`<`void`\>

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:241](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L241)

___

### withdrawERC20

▸ **withdrawERC20**(`l1Token`, `l2Token`, `amount`, `opts?`): `Promise`<`TransactionResponse`\>

Withdraws some ERC20 tokens back to the L1 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `l1Token` | [`AddressLike`](../modules.md#addresslike) | Address of the L1 token. |
| `l2Token` | [`AddressLike`](../modules.md#addresslike) | Address of the L2 token. |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount to withdraw. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) | Optional address to receive the funds on L1. Defaults to sender. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the withdraw transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:510](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L510)

___

### withdrawETH

▸ **withdrawETH**(`amount`, `opts?`): `Promise`<`TransactionResponse`\>

Withdraws some ETH back to the L1 chain.

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `amount` | [`NumberLike`](../modules.md#numberlike) | Amount of ETH to withdraw. |
| `opts?` | `Object` | Additional options. |
| `opts.overrides?` | `Overrides` | Optional transaction overrides. |
| `opts.recipient?` | [`AddressLike`](../modules.md#addresslike) | Optional address to receive the funds on L1. Defaults to sender. |
| `opts.signer?` | `Signer` | Optional signer to use to send the transaction. |

#### Returns

`Promise`<`TransactionResponse`\>

Transaction response for the withdraw transaction.

#### Defined in

[packages/sdk/src/interfaces/cross-chain-messenger.ts:426](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/cross-chain-messenger.ts#L426)
