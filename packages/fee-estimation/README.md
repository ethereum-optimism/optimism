# @eth-optimism/fee-estimation

Tools for estimating gas on OP chains

- **Tip** the [specs file](./src/estimateFees.spec.ts) has usage examples of every method in this library.

## Overview

This package is designed to provide an easy way to estimate gas on OP chains.

Fee estimation on OP-chains has both an l2 and l1 component. By default tools such as Viem, Wagmi, Ethers, and Web3.js do not support the l1 component. They will support this soon but in meantime, this library can help estimate fees for transactions, or act as a reference.
As these tools add support for gas estimation natively this README will be updated with framework specific instructions.

For more detailed information about gas fees on Optimism's Layer 2, you can visit their [official documentation](https://community.optimism.io/docs/developers/build/transaction-fees/#the-l2-execution-fee).

## GasPriceOracle contract

- The l2 contract that can estimate l1Fees is called [GasPriceOracle](../contracts-bedrock/contracts/l2/GasPriceOracle.sol) contract. This library provides utils for interacting with it at a high level.
- The GasPriceOracle is [deployed to Optimism](https://optimistic.etherscan.io/address/0x420000000000000000000000000000000000000F) and other OP chains at a predeployed address of `0x420000000000000000000000000000000000000F`

This library provides a higher level abstraction over the gasPriceOracle

## Installation

```bash
pnpm install @eth-optimism/fee-estimation
```

```bash
npm install @eth-optimism/fee-estimation
```

```bash
yarn add @eth-optimism/fee-estimation
```

### Basic Usage

```ts
import { estimateFees } from '@eth-optimism/fee-estimation'
import { optimistABI } from '@eth-optimism/contracts-ts'
import { viemClient } from './viem-client'

const optimistOwnerAddress =
  '0x77194aa25a06f932c10c0f25090f3046af2c85a6' as const
const tokenId = BigInt(optimistOwnerAddress)

const fees = await estimateFees({
  client: viemClient,
  // If not using in viem can pass in this instead
  /*
  client: {
    chainId: 10,
    rpcUrl: 'https://mainnet.optimism.io',
  },
  */
  functionName: 'burn',
  abi: optimistABI,
  args: [tokenid],
  account: optimistOwnerAddress,
  to: '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5',
})
```

## API

### `estimateFees` function

```ts
estimateFees(options: OracleTransactionParameters<TAbi, TFunctionName> & GasPriceOracleOptions & Omit<EstimateGasParameters, 'data'>): Promise<bigint>
```

#### Parameters

`options`: An object with the following fields:

- `abi`: A JSON object ABI of contract.

- `account`: A hex address of the account making the transaction.

- `args`: Array of arguments to contract function. The types of this will be inferred from the ABI

- `blockNumber`(optional): A BigInt representing the block number at which you want to estimate the fees.

- `chainId`: An integer chain id.

- `client`: An object with rpcUrl field, or an instance of a Viem PublicClient.

- `functionName`: A string representing the function name for the transaction call data.

- `maxFeePerGas`(optional): A BigInt representing the maximum fee per gas that the user is willing to pay.

- `maxPriorityFeePerGas`(optional): A BigInt representing the maximum priority fee per gas that the user is willing to pay.

- `to`: A hex address of the recipient of the transaction.

- `value`(optional): A BigInt representing the value in wei sent along with the transaction.

#### Returns

A Promise that resolves to a BigInt representing the estimated fee in wei.

## Other methods

This package also provides lower level methods for estimating gas

### getL2Client()

This method returns a Layer 2 (L2) client that communicates with an L2 network.

```ts
getL2Client(options: ClientOptions): PublicClient;
```

#### Parameters

- `options: ClientOptions` - The options required to initialize the L2 client.

#### Returns

- `PublicClient` - Returns a public client that can interact with the L2 network.

#### Example

```ts
const clientParams = {
  chainId: 10,
  rpcUrl: process.env.VITE_L2_RPC_URL ?? 'https://mainnet.optimism.io',
} as const

const client = getL2Client(clientParams)
```

---

### baseFee()

Returns the base fee.

```ts
baseFee({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - The options required to fetch the base fee.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the base fee.

#### Example

```ts
const blockNumber = BigInt(106889079)
const paramsWithClient = {
  client: clientParams,
  blockNumber,
}
const baseFeeValue = await baseFee(paramsWithClient)
```

---

### decimals()

Returns the decimals used in the scalar.

```ts
decimals({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - The options required to fetch the decimals.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the decimals used in the scalar.

#### Example

```ts
const decimalsValue = await decimals(paramsWithClient)
```

---

### gasPrice()

Returns the gas price.

```ts
gasPrice({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - The options required to fetch the gas price.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the gas price.

#### Example

```ts
const gasPriceValue = await gasPrice(paramsWithClient)
```

---

### getL1Fee()

Computes the L1 portion of the fee based on the size of the rlp encoded input transaction, the current L1 base fee, and the various dynamic parameters.

```ts
getL1Fee(data: Bytes, { client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `data: Bytes` - The transaction call data as a 0x-prefixed hex string.
- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the L1 portion of the fee.

#### Example

```ts
const data =
  '0x5c19a95c00000000000000000000000046abfe1c972fca43766d6ad70e1c1df72f4bb4d1'
const l1FeeValue = await getL1Fee(data, paramsWithClient)
```

### getL1GasUsed()

This method returns the amount of gas used on the L1 network for a given transaction.

```ts
getL1GasUsed(data: Bytes, { client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `data: Bytes` - The transaction call data as a 0x-prefixed hex string.
- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the amount of gas used on the L1 network for the given transaction.

#### Example

```ts
const data =
  '0x5c19a95c00000000000000000000000046abfe1c972fca43766d6ad70e1c1df72f4bb4d1'
const l1GasUsed = await getL1GasUsed(data, paramsWithClient)
```

---

### l1BaseFee()

Returns the base fee on the L1 network.

```ts
l1BaseFee({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the base fee on the L1 network.

#### Example

```ts
const l1BaseFeeValue = await l1BaseFee(paramsWithClient)
```

---

### overhead()

Returns the overhead for the given transaction.

```ts
overhead({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the overhead for the given transaction.

#### Example

```ts
const overheadValue = await overhead(paramsWithClient)
```

---

### scalar()

Returns the scalar value for the gas estimation.

```ts
scalar({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the scalar value for the gas estimation.

#### Example

```ts
const scalarValue = await scalar(paramsWithClient)
```

---

### version()

Returns the version of the fee estimation library.

```ts
version({ client, ...params }: GasPriceOracleOptions): Promise<string>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<string>` - Returns a promise that resolves to the version of the fee estimation library.

#### Example

```ts
const libraryVersion = await version(paramsWithClient)
```

---
