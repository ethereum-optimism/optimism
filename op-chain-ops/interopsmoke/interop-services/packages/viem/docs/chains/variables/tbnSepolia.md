[**@eth-optimism/viem**](../../README.md) • **Docs**

***

[@eth-optimism/viem](../../README.md) / [chains](../README.md) / tbnSepolia

# tbnSepolia

> `const` **tbnSepolia**: `object`

Chain Definition for Binary Sepolia

## Type declaration

### blockExplorers

> **blockExplorers**: `object`

Collection of block explorers

### blockExplorers.default

> `readonly` **default**: `object`

### blockExplorers.default.name

> `readonly` **name**: `"Binary Sepolia Explorer"` = `'Binary Sepolia Explorer'`

### blockExplorers.default.url

> `readonly` **url**: `"https://explorer.sepolia.thebinaryholdings.com"` = `'https://explorer.sepolia.thebinaryholdings.com'`

### contracts

> **contracts**: `object`

Collection of contracts

### contracts.disputeGameFactory

> `readonly` **disputeGameFactory**: `object`

### contracts.disputeGameFactory.11155111

> `readonly` **11155111**: `object`

### contracts.disputeGameFactory.11155111.address

> `readonly` **address**: `"0x096b38bDC80B5BF5B5Fb4e1A75Ae38BDa520474A"` = `'0x096b38bDC80B5BF5B5Fb4e1A75Ae38BDa520474A'`

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

### contracts.l1CrossDomainMessenger.11155111

> `readonly` **11155111**: `object`

### contracts.l1CrossDomainMessenger.11155111.address

> `readonly` **address**: `"0x33Dc556A5df0B8998dC2640c78E531Ae1dB7925d"` = `'0x33Dc556A5df0B8998dC2640c78E531Ae1dB7925d'`

### contracts.l1Erc721Bridge

> `readonly` **l1Erc721Bridge**: `object`

### contracts.l1Erc721Bridge.11155111

> `readonly` **11155111**: `object`

### contracts.l1Erc721Bridge.11155111.address

> `readonly` **address**: `"0x7c95EebEA6f68875b4093D9c2211Fd26067a808F"` = `'0x7c95EebEA6f68875b4093D9c2211Fd26067a808F'`

### contracts.l1StandardBridge

> `readonly` **l1StandardBridge**: `object`

### contracts.l1StandardBridge.11155111

> `readonly` **11155111**: `object`

### contracts.l1StandardBridge.11155111.address

> `readonly` **address**: `"0x3B78C3B41b3e3fC6bdf0bD3060C9E2471401C098"` = `'0x3B78C3B41b3e3fC6bdf0bD3060C9E2471401C098'`

### contracts.l2CrossDomainMessenger

> `readonly` **l2CrossDomainMessenger**: `object`

### contracts.l2CrossDomainMessenger.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000007"`

### contracts.l2Erc721Bridge

> `readonly` **l2Erc721Bridge**: `object`

### contracts.l2Erc721Bridge.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000014"`

### contracts.l2OutputOracle

> `readonly` **l2OutputOracle**: `object`

### contracts.l2OutputOracle.11155111

> `readonly` **11155111**: `object`

### contracts.l2OutputOracle.11155111.address

> `readonly` **address**: `"0xF3699f96cBdD3868B64352669805D96d1Fb6431d"` = `'0xF3699f96cBdD3868B64352669805D96d1Fb6431d'`

### contracts.l2StandardBridge

> `readonly` **l2StandardBridge**: `object`

### contracts.l2StandardBridge.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000010"`

### contracts.l2ToL1MessagePasser

> `readonly` **l2ToL1MessagePasser**: `object`

### contracts.l2ToL1MessagePasser.address

> `readonly` **address**: `"0x4200000000000000000000000000000000000016"`

### contracts.portal

> `readonly` **portal**: `object`

### contracts.portal.11155111

> `readonly` **11155111**: `object`

### contracts.portal.11155111.address

> `readonly` **address**: `"0xFBEd910ca54F013bfeA67Bd4DC836263bdd0b46C"` = `'0xFBEd910ca54F013bfeA67Bd4DC836263bdd0b46C'`

### contracts.systemConfig

> `readonly` **systemConfig**: `object`

### contracts.systemConfig.11155111

> `readonly` **11155111**: `object`

### contracts.systemConfig.11155111.address

> `readonly` **address**: `"0x1a6D0312FaaaCa2BF818660F164450176C6205C9"` = `'0x1a6D0312FaaaCa2BF818660F164450176C6205C9'`

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

> **id**: `625`

ID in number form

### name

> **name**: `"Binary Sepolia"`

Human-readable name

### nativeCurrency

> **nativeCurrency**: `object`

Currency used by chain

### nativeCurrency.decimals

> `readonly` **decimals**: `18` = `18`

### nativeCurrency.name

> `readonly` **name**: `"Test BNRY"` = `'Test BNRY'`

### nativeCurrency.symbol

> `readonly` **symbol**: `"TBNRY"` = `'TBNRY'`

### rpcUrls

> **rpcUrls**: `object`

Collection of RPC endpoints

### rpcUrls.default

> `readonly` **default**: `object`

### rpcUrls.default.http

> `readonly` **http**: readonly [`"https://rpc.testnet.thebinaryholdings.com"`]

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

> **sourceId**: `11155111`

Source Chain ID (ie. the L1 chain)

### testnet?

> `optional` **testnet**: `boolean`

Flag for test networks

## Defined in

[packages/viem/src/chains/sepolia.ts:810](https://github.com/ethereum-optimism/ecosystem/blob/11bb27f871c202b93ad6dc93c86c82f0c754075f/packages/viem/src/chains/sepolia.ts#L810)
