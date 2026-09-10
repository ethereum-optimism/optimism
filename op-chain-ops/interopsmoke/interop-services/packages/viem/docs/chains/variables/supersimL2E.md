[**@eth-optimism/viem**](../../README.md) • **Docs**

***

[@eth-optimism/viem](../../README.md) / [chains](../README.md) / supersimL2E

# supersimL2E

> `const` **supersimL2E**: `object`

L2 chain E definition for supersim in non-forked mode

## Type declaration

### blockExplorers?

> `optional` **blockExplorers**: `object`

Collection of block explorers

#### Index Signature

 \[`key`: `string`\]: `ChainBlockExplorer`

### blockExplorers.default

> **default**: `ChainBlockExplorer`

### contracts

> **contracts**: `object`

Collection of contracts

### contracts.addressManager

> `readonly` **addressManager**: `object`

### contracts.anchorStateRegistry

> `readonly` **anchorStateRegistry**: `object`

### contracts.disputeGameFactory

> `readonly` **disputeGameFactory**: `object`

### contracts.faultDisputeGame

> `readonly` **faultDisputeGame**: `object`

### contracts.gasPriceOracle

> `readonly` **gasPriceOracle**: `object`

### contracts.gasPriceOracle.address

> `readonly` **address**: `"0x420000000000000000000000000000000000000F"`

### contracts.l1Block

> `readonly` **l1Block**: `object`

### contracts.l1Block.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000015"`

### contracts.l1CrossDomainMessenger

> `readonly` **l1CrossDomainMessenger**: `object`

### contracts.l1Erc721BridgeProxy

> `readonly` **l1Erc721BridgeProxy**: `object`

### contracts.l1StandardBridge

> `readonly` **l1StandardBridge**: `object`

### contracts.l2CrossDomainMessenger

> `readonly` **l2CrossDomainMessenger**: `object`

### contracts.l2CrossDomainMessenger.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000007"`

### contracts.l2Erc721Bridge

> `readonly` **l2Erc721Bridge**: `object`

### contracts.l2Erc721Bridge.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000014"`

### contracts.l2StandardBridge

> `readonly` **l2StandardBridge**: `object`

### contracts.l2StandardBridge.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000010"`

### contracts.l2ToL1MessagePasser

> `readonly` **l2ToL1MessagePasser**: `object`

### contracts.l2ToL1MessagePasser.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000016"`

### contracts.opChainProxyAdmin

> `readonly` **opChainProxyAdmin**: `object`

### contracts.optimismMintableErc20FactoryProxy

> `readonly` **optimismMintableErc20FactoryProxy**: `object`

### contracts.optimismPortal

> `readonly` **optimismPortal**: `object`

### contracts.permissionedDisputeGame

> `readonly` **permissionedDisputeGame**: `object`

### contracts.systemConfig

> `readonly` **systemConfig**: `object`

### custom?

> `optional` **custom**: `Record`\<`string`, `unknown`\>

Custom chain data.

### fees?

> `optional` **fees**: `ChainFees`\<`undefined`\>

Modifies how fees are derived.

### formatters

> `readonly` **formatters**: `object`

Modifies how data is formatted and typed (e.g. blocks and transactions)

### formatters.block

> `readonly` **block**: `object`

### formatters.block.exclude

> **exclude**: `undefined` \| []

### formatters.block.format()

> **format**: (`args`) => `object`

#### Parameters

• **args**: `OpStackRpcBlock`\<`BlockTag`, `boolean`\>

#### Returns

`object`

##### baseFeePerGas

> **baseFeePerGas**: `null` \| `bigint`

##### blobGasUsed

> **blobGasUsed**: `bigint`

##### difficulty

> **difficulty**: `bigint`

##### excessBlobGas

> **excessBlobGas**: `bigint`

##### extraData

> **extraData**: \`0x$\{string\}\`

##### gasLimit

> **gasLimit**: `bigint`

##### gasUsed

> **gasUsed**: `bigint`

##### hash

> **hash**: `null` \| \`0x$\{string\}\`

##### logsBloom

> **logsBloom**: `null` \| \`0x$\{string\}\`

##### miner

> **miner**: \`0x$\{string\}\`

##### mixHash

> **mixHash**: \`0x$\{string\}\`

##### nonce

> **nonce**: `null` \| \`0x$\{string\}\`

##### number

> **number**: `null` \| `bigint`

##### parentHash

> **parentHash**: \`0x$\{string\}\`

##### receiptsRoot

> **receiptsRoot**: \`0x$\{string\}\`

##### sealFields

> **sealFields**: \`0x$\{string\}\`[]

##### sha3Uncles

> **sha3Uncles**: \`0x$\{string\}\`

##### size

> **size**: `bigint`

##### stateRoot

> **stateRoot**: \`0x$\{string\}\`

##### timestamp

> **timestamp**: `bigint`

##### totalDifficulty

> **totalDifficulty**: `null` \| `bigint`

##### transactions

> **transactions**: \`0x$\{string\}\`[] \| `OpStackTransaction`\<`boolean`\>[]

##### transactionsRoot

> **transactionsRoot**: \`0x$\{string\}\`

##### uncles

> **uncles**: \`0x$\{string\}\`[]

##### withdrawals?

> `optional` **withdrawals**: `Withdrawal`[]

##### withdrawalsRoot?

> `optional` **withdrawalsRoot**: \`0x$\{string\}\`

### formatters.block.type

> **type**: `"block"`

### formatters.transaction

> `readonly` **transaction**: `object`

### formatters.transaction.exclude

> **exclude**: `undefined` \| []

### formatters.transaction.format()

> **format**: (`args`) => `object` \| `object` \| `object` \| `object` \| `object`

#### Parameters

• **args**: `OpStackRpcTransaction`\<`boolean`\>

#### Returns

`object` \| `object` \| `object` \| `object` \| `object`

### formatters.transaction.type

> **type**: `"transaction"`

### formatters.transactionReceipt

> `readonly` **transactionReceipt**: `object`

### formatters.transactionReceipt.exclude

> **exclude**: `undefined` \| []

### formatters.transactionReceipt.format()

> **format**: (`args`) => `object`

#### Parameters

• **args**: `OpStackRpcTransactionReceipt`

#### Returns

`object`

##### blobGasPrice?

> `optional` **blobGasPrice**: `bigint`

##### blobGasUsed?

> `optional` **blobGasUsed**: `bigint`

##### blockHash

> **blockHash**: \`0x$\{string\}\`

##### blockNumber

> **blockNumber**: `bigint`

##### contractAddress

> **contractAddress**: `undefined` \| `null` \| \`0x$\{string\}\`

##### cumulativeGasUsed

> **cumulativeGasUsed**: `bigint`

##### effectiveGasPrice

> **effectiveGasPrice**: `bigint`

##### from

> **from**: \`0x$\{string\}\`

##### gasUsed

> **gasUsed**: `bigint`

##### l1Fee

> **l1Fee**: `null` \| `bigint`

##### l1FeeScalar

> **l1FeeScalar**: `null` \| `number`

##### l1GasPrice

> **l1GasPrice**: `null` \| `bigint`

##### l1GasUsed

> **l1GasUsed**: `null` \| `bigint`

##### logs

> **logs**: `Log`\<`bigint`, `number`, `false`, `undefined`, `undefined`, `undefined`, `undefined`\>[]

##### logsBloom

> **logsBloom**: \`0x$\{string\}\`

##### root?

> `optional` **root**: \`0x$\{string\}\`

##### status

> **status**: `"success"` \| `"reverted"`

##### to

> **to**: `null` \| \`0x$\{string\}\`

##### transactionHash

> **transactionHash**: \`0x$\{string\}\`

##### transactionIndex

> **transactionIndex**: `number`

##### type

> **type**: `TransactionType`

### formatters.transactionReceipt.type

> **type**: `"transactionReceipt"`

### id

> **id**: `905`

ID in number form

### name

> **name**: `"Supersim L2 E"`

Human-readable name

### nativeCurrency

> **nativeCurrency**: `object`

Currency used by chain

### nativeCurrency.decimals

> `readonly` **decimals**: `18` = `18`

### nativeCurrency.name

> `readonly` **name**: `"Ether"` = `'Ether'`

### nativeCurrency.symbol

> `readonly` **symbol**: `"ETH"` = `'ETH'`

### rpcUrls

> **rpcUrls**: `object`

Collection of RPC endpoints

### rpcUrls.default

> `readonly` **default**: `object`

### rpcUrls.default.http

> `readonly` **http**: readonly [`"http://127.0.0.1:9549"`]

### serializers

> `readonly` **serializers**: `object`

Modifies how data is serialized (e.g. transactions).

### serializers.transaction()

> `readonly` **transaction**: (`transaction`, `signature`?) => \`0x02$\{string\}\` \| \`0x01$\{string\}\` \| \`0x03$\{string\}\` \| `TransactionSerializedLegacy` \| \`0x7e$\{string\}\`

#### Parameters

• **transaction**: `OpStackTransactionSerializable`

• **signature?**: `Signature`

#### Returns

\`0x02$\{string\}\` \| \`0x01$\{string\}\` \| \`0x03$\{string\}\` \| `TransactionSerializedLegacy` \| \`0x7e$\{string\}\`

### sourceId

> **sourceId**: `900`

Source Chain ID (ie. the L1 chain)

### testnet

> **testnet**: `true`

Flag for test networks

## Defined in

[packages/viem/src/chains/supersim.ts:124](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/chains/supersim.ts#L124)
